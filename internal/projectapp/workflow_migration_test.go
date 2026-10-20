package projectapp_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	fx "github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type migrationPort struct {
	*workflowPort
	cfg       runtimeprofile.ConfigurationV3
	published *workflowdefinition.Document
	selected  int
	token     string
}

var _ projectapp.WorkflowProfilePort = (*migrationPort)(nil)

// ObserveWorkflowProfiles re-reads the canonical format-4 record on every call.
func (p *migrationPort) ObserveWorkflowProfiles(context.Context, string) (runtimeprofile.ConsumerV4, string, string) {
	consumer, err := runtimeprofile.NewConsumerV4(fx.Current(p.cfg))
	if err != nil {
		return runtimeprofile.ConsumerV4{}, "", "profile_effort_unconfigured"
	}
	return consumer, p.token, ""
}
func (p *migrationPort) PublishWorkflowRevision(_ context.Context, _ projectapp.WorkflowObservation, doc *workflowdefinition.Document, idx workflowdefinition.Index) string {
	p.writes++
	p.published = doc
	if doc != nil {
		p.observed.Catalog.Documents["custom/2.json"] = *doc
	}
	p.observed.Catalog.Index = idx
	return "applied"
}
func (p *migrationPort) PublishWorkflowSelection(context.Context, projectapp.WorkflowObservation, project.Project) string {
	p.selected++
	return "applied"
}
func migrationFixture(t *testing.T) (*migrationPort, workflowdefinition.Document, projectapp.WorkflowRequest) {
	t.Helper()
	port, source := workflowFixture(t)
	state := port.observed.Selection.Portable.Project().State()
	state.Runtimes = project.Configured([]project.Runtime{{ID: "codex"}})
	state.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "worker", RuntimeRef: project.Configured("codex"), Model: project.Configured("opaque-model")}})
	p, issues := project.New(state)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	wire, issues := manifest.Encode(p)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	snapshot, problems := projectapp.ReadSnapshot(manifest.Codec{}, wire, nil)
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	port.observed.Selection.Portable = snapshot
	cfg := runtimeprofile.ConfigurationV3{FormatVersion: runtimeprofile.FormatVersionV3, Revision: 5, Runtimes: []runtimeprofile.RuntimeV3{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"worker"}}}, ModelProfiles: []runtimeprofile.ModelProfile{{ID: "worker", RuntimeID: "codex", Model: "opaque-model", Effort: runtimeprofile.Effort{Mode: "explicit", Value: "opaque-high"}, Capabilities: []string{"read", "repository-write", "process", "artifact-publish", "integration"}}}}
	request := projectapp.WorkflowRequest{Operation: "migrate", Project: "sample", Prior: source.Ref("project")}
	for _, stage := range source.Definition.Stages {
		for _, agent := range stage.Agents {
			request.Mapping = append(request.Mapping, projectapp.WorkflowAgentProfileMapping{StageID: stage.ID, AgentID: agent.ID, ProfileID: "worker"})
		}
	}
	return &migrationPort{workflowPort: port, cfg: cfg, token: "synthetic-identity-1"}, source, request
}
func TestWorkflowMigrationComparesEffortAndSeparatesPublicationSelection(t *testing.T) {
	ctx := context.Background()
	p, source, request := migrationFixture(t)
	original := append([]byte(nil), source.Canonical...)
	preview := projectapp.ApplyWorkflow(ctx, p, request)
	if preview.Category != "previewed" || p.writes != 0 || p.selected != 0 {
		t.Fatalf("preview: %+v", preview)
	}
	if preview.Report.MigrationSource == nil || *preview.Report.MigrationSource != request.Prior || preview.Report.ProfileConfigurationDigest == "" || len(preview.Report.EffortComparisons) != len(request.Mapping) || preview.Report.Reference.Revision != 2 || preview.Report.Definition.SchemaVersion != 2 {
		t.Fatalf("missing migration projection %+v", preview)
	}
	for _, comparison := range preview.Report.EffortComparisons {
		if comparison.LegacyRequestedEffort.Mode != "runtime-default" || comparison.ProfileEffort.Value != "opaque-high" || !comparison.ChangedBehavior {
			t.Fatalf("missing changed behavior %+v", comparison)
		}
	}
	request.AuthorizeLocal = true
	request.PreviewDigest = preview.Report.PreviewDigest
	request.ExpectedRevision = p.observed.Selection.PortableRevision
	result := projectapp.ApplyWorkflow(ctx, p, request)
	if result.Category != "applied" || p.writes != 1 || p.selected != 0 || p.published == nil {
		t.Fatalf("apply %+v", result)
	}
	if !bytes.Equal(original, p.observed.Catalog.Documents["custom/1.json"].Canonical) || p.observed.Catalog.Documents["custom/1.json"].Digest != source.Digest {
		t.Fatal("source changed")
	}
	if p.published.Definition.SchemaVersion != 2 || p.published.Definition.WorkflowID != source.Definition.WorkflowID {
		t.Fatal("wrong identity")
	}
}
func TestWorkflowMigrationRejectsIncompleteMappingDriftAndPolicy(t *testing.T) {
	cases := []struct {
		name     string
		change   func(*migrationPort, *projectapp.WorkflowRequest)
		expected string
	}{
		{"partial", func(_ *migrationPort, r *projectapp.WorkflowRequest) { r.Mapping = r.Mapping[:1] }, "invalid_profile_mapping"},
		{"duplicate", func(_ *migrationPort, r *projectapp.WorkflowRequest) { r.Mapping = append(r.Mapping, r.Mapping[0]) }, "invalid_profile_mapping"},
		{"unknown", func(_ *migrationPort, r *projectapp.WorkflowRequest) { r.Mapping[0].ProfileID = "missing" }, "profile_effort_unconfigured"},
		{"allowlist", func(p *migrationPort, _ *projectapp.WorkflowRequest) {
			p.cfg.Runtimes[0].Enabled = false
		}, "project_policy_update_required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, _, r := migrationFixture(t)
			tc.change(p, &r)
			out := projectapp.ApplyWorkflow(context.Background(), p, r)
			if out.Category != tc.expected || p.writes != 0 || p.selected != 0 {
				t.Fatalf("got %+v", out)
			}
		})
	}
	for _, drift := range []string{"effort", "generation", "policy", "identity"} {
		t.Run(drift, func(t *testing.T) {
			p, _, r := migrationFixture(t)
			preview := projectapp.ApplyWorkflow(context.Background(), p, r)
			if preview.Category != "previewed" {
				t.Fatal(preview)
			}
			r.AuthorizeLocal = true
			r.PreviewDigest = preview.Report.PreviewDigest
			r.ExpectedRevision = p.observed.Selection.PortableRevision
			switch drift {
			case "identity":
				p.token = "synthetic-identity-2"
			case "effort":
				p.cfg.ModelProfiles[0].Effort.Value = "opaque-low"
			case "generation":
				p.cfg.Revision++
			case "policy":
				p.observed.Selection.PortableRevision = strings.Repeat("c", 64)
			}
			out := projectapp.ApplyWorkflow(context.Background(), p, r)
			if out.Category != "stale_authority" || p.writes != 0 || p.selected != 0 {
				t.Fatalf("drift accepted %+v", out)
			}
		})
	}
}
