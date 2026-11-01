package local

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	fx "github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

type policySelection struct {
	fixture editPublicationFixture
}

func (s policySelection) SelectForEdit(ctx context.Context, _ string) (projectapp.EditSelection, projectapp.EditFailure) {
	f := s.fixture
	o, code := f.installation.Inspect(ctx, editTestProjectID)
	if code != "" {
		return projectapp.EditSelection{}, projectapp.EditRecoveryRequired
	}
	p, err := f.portable.Inspect(ctx, "sample")
	if err != nil {
		return projectapp.EditSelection{}, projectapp.EditStateUnsafe
	}
	return projectapp.EditSelection{Portable: p.Snapshot, Local: ApplicationRecord(o.Record), LocalWire: o.Wire, PortableDestination: f.source, LocalDestination: filepath.Dir(f.recordPath()), PortableRevision: p.Revision, LocalRevision: o.Revision}, projectapp.EditOK
}

func newPolicyEditFixture(t *testing.T) editPublicationFixture {
	return newPolicyEditVersionFixture(t, 3)
}

func newPolicyEditVersionFixture(t *testing.T, version int) editPublicationFixture {
	t.Helper()
	f := editPublicationFixture{projects: privateTestRoot(t), state: privateTestRoot(t)}
	var err error
	f.portable, err = NewPortableStore(f.projects)
	if err != nil {
		t.Fatal(err)
	}
	f.installation, err = NewInstallationStore(f.state)
	if err != nil {
		t.Fatal(err)
	}
	p, issues := project.New(project.State{SchemaVersion: version, ID: editTestProjectID, Slug: "sample", Name: "SENTINEL-NAME", Runtimes: project.Configured([]project.Runtime{{ID: fx.CodexRuntimeID}}), ModelProfiles: project.Configured([]project.ModelProfile{{Key: fx.CodexHighProfileID, RuntimeRef: project.Configured(fx.CodexRuntimeID), Model: project.Configured("synthetic-codex-model")}}), BusinessContext: project.Configured(project.BusinessContext{Text: project.Configured("SENTINEL-CONTEXT")}), CredentialReferences: project.Configured([]project.CredentialReference{{Key: "SENTINEL-CREDENTIAL"}})})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	wire, issues := manifest.Encode(p)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if err := f.portable.Create(context.Background(), "sample", wire); err != nil {
		t.Fatal(err)
	}
	f.source = filepath.Join(f.projects, "sample")
	if got := f.installation.Install(context.Background(), f.source); got.Status != InstallationApplied {
		t.Fatal(got)
	}
	return f
}

func (f editPublicationFixture) policyPorts(t *testing.T) projectapp.PolicyEditPorts {
	t.Helper()
	return projectapp.PolicyEditPorts{Edit: projectapp.EditPorts{Source: policySelection{fixture: f}, Checkouts: DirectoryObserver{}, Manifest: manifest.Codec{}, Local: RecordCodec{}}, Facts: PolicyEditObserver{Installation: f.installation, Portable: f.portable}}
}

func policyIntent(t *testing.T, wire string) projectapp.EditIntent {
	t.Helper()
	draft, failure := projectapp.DecodePolicyDraft([]byte(wire))
	if failure != projectapp.EditOK {
		t.Fatal(failure)
	}
	return projectapp.EditIntent{Selector: "sample", Policy: &draft}
}

func TestPolicyPreviewDeterministicSafeAndReadOnly(t *testing.T) {
	f := newPolicyEditFixture(t)
	intent := policyIntent(t, `{"runtimes":{"mode":"members","add":[{"id":"another"}]}}`)
	before := map[string][]byte{f.manifestPath(): fileBytes(t, f.manifestPath()), f.recordPath(): fileBytes(t, f.recordPath())}
	ports := f.policyPorts(t)
	first, failure := projectapp.PreviewPolicyEdit(context.Background(), ports, intent)
	if failure != projectapp.EditOK {
		t.Fatal(failure)
	}
	second, failure := projectapp.PreviewPolicyEdit(context.Background(), ports, intent)
	if failure != projectapp.EditOK || !reflect.DeepEqual(first.Preview(), second.Preview()) {
		t.Fatal("unstable preview", failure)
	}
	preview := first.Preview()
	if preview.Runtime[0].Code != projectapp.RuntimeSelectedAtDispatch || !preview.Workflow.Compatible || !preview.Admission.Allowed || preview.Admission.GrantsAuthority || len(preview.Diff) != 1 || len(preview.Effects) != 2 {
		t.Fatalf("incorrect preview: %+v", preview)
	}
	// Per-agent policy coverage is authoring guidance, never a predictive gate.
	if preview.Runtime[0].Blocker != "" || preview.Runtime[0].Next == "" || len(preview.Readiness) != 2 {
		t.Fatal("lost scoped guidance or kept predictive blocker", preview)
	}
	for _, operation := range preview.Readiness {
		for _, blocker := range operation.Blockers {
			if blocker.Code == projectapp.BlockerRuntimeResolution || blocker.Code == projectapp.BlockerRuntimePolicyUnavailable {
				t.Fatal("Runtime prediction blocked an operation", operation)
			}
		}
	}
	wire, err := json.Marshal(preview)
	if err != nil {
		t.Fatal(err)
	}
	for _, sentinel := range []string{f.projects, f.state, "SENTINEL-NAME", "SENTINEL-CONTEXT", "SENTINEL-CREDENTIAL", "portableManifest", "localDestination"} {
		if bytes.Contains(wire, []byte(sentinel)) {
			t.Fatalf("unsafe projection: %s", sentinel)
		}
	}
	preview.Runtime[0].Code = "mutated"
	preview.Policy.Runtimes.Values = nil
	if reflect.DeepEqual(preview, first.Preview()) {
		t.Fatal("mutable preview aliases proposal")
	}
	for path, original := range before {
		if !bytes.Equal(original, fileBytes(t, path)) {
			t.Fatal("preview wrote state")
		}
	}
	for _, path := range []string{filepath.Join(f.state, "runtime-profiles"), filepath.Join(filepath.Dir(f.recordPath()), operationalRecordName)} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatal("preview created absent dependency", err)
		}
	}
	if len(f.editStates(t)) != 0 {
		t.Fatal("preview created recovery")
	}
}

func TestPolicyPreviewUsesCandidateAndBindsLocalConfiguration(t *testing.T) {
	f := newPolicyEditFixture(t)
	ctx := context.Background()
	store, err := NewRuntimeProfileStore(f.state)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := store.Inspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg := fx.CodexOnly()
	wire, err := runtimeprofile.EncodeV4(operationalTestConfig(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(ctx, fresh.Token, wire); err != nil {
		t.Fatal(err)
	}
	ports := f.policyPorts(t)
	noop := policyIntent(t, `{}`)
	first, failure := projectapp.PreviewPolicyEdit(ctx, ports, noop)
	if failure != projectapp.EditOK || first.Preview().Runtime[0].Code != projectapp.RuntimeSelectedAtDispatch {
		t.Fatal("unexpected readiness", failure, first.Preview())
	}
	intent := policyIntent(t, `{"modelProfiles":{"mode":"members","replace":[{"key":"`+fx.CodexHighProfileID+`","runtimeRef":"`+fx.CodexRuntimeID+`","model":"different-model"}]}}`)
	next, failure := projectapp.PreviewPolicyEdit(ctx, ports, intent)
	if failure != projectapp.EditOK || next.Preview().Runtime[0].Code != projectapp.RuntimeSelectedAtDispatch {
		t.Fatal("reloaded installed policy instead of candidate", failure)
	}
	if first.Preview().Digest == next.Preview().Digest {
		t.Fatal("candidate not bound")
	}
	if err := os.WriteFile(filepath.Join(f.state, "runtime-profiles", "v1", runtimeProfileObject), []byte("unsafe SENTINEL-ERROR"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, failure := projectapp.PreviewPolicyEdit(ctx, ports, noop); failure == projectapp.EditOK {
		t.Fatal("unsafe configuration treated as absent")
	}
}

type unsafePolicyFacts struct {
	base   projectapp.PolicyEditObserver
	mutate func(*projectapp.PolicyEditFacts)
}

func (s unsafePolicyFacts) ObservePolicyEdit(ctx context.Context, selection projectapp.EditSelection, candidate project.Project) (projectapp.PolicyEditFacts, projectapp.EditFailure) {
	facts, failure := s.base.ObservePolicyEdit(ctx, selection, candidate)
	s.mutate(&facts)
	return facts, failure
}

func TestPolicyPreviewRejectsUnsafeOrUnsealedFacts(t *testing.T) {
	f := newPolicyEditFixture(t)
	for _, mutate := range []func(*projectapp.PolicyEditFacts){
		func(f *projectapp.PolicyEditFacts) { f.Dependencies.Runtime = "" },
		func(f *projectapp.PolicyEditFacts) { f.Dependencies.Source = strings.Repeat("A", 64) },
		func(f *projectapp.PolicyEditFacts) { f.Operational.Revision = "" },
		func(f *projectapp.PolicyEditFacts) { f.Runtime[0].Code = "SENTINEL-ERROR /private/path" },
		func(f *projectapp.PolicyEditFacts) { f.Runtime[0].Code = "no_allowed_match" },
		func(f *projectapp.PolicyEditFacts) { f.Runtime[0].Blocker = "runtime_resolution_blocked" },
		func(f *projectapp.PolicyEditFacts) { f.Runtime[0].Stage = "SENTINEL-STAGE" },
	} {
		ports := f.policyPorts(t)
		ports.Facts = unsafePolicyFacts{base: ports.Facts, mutate: mutate}
		if _, failure := projectapp.PreviewPolicyEdit(context.Background(), ports, policyIntent(t, `{}`)); failure == projectapp.EditOK {
			t.Fatal("unsafe observation accepted")
		}
	}
}

func TestPolicyPreviewUnconfiguredPolicyIsGuidanceNotBlocker(t *testing.T) {
	f := newPolicyEditFixture(t)
	intent := policyIntent(t, `{"runtimes":{"mode":"absent"},"modelProfiles":{"mode":"absent"},"runtimePreferences":{"mode":"absent"}}`)
	proposal, failure := projectapp.PreviewPolicyEdit(context.Background(), f.policyPorts(t), intent)
	if failure != projectapp.EditOK {
		t.Fatal(failure)
	}
	row := proposal.Preview().Runtime[0]
	if row.Code != projectapp.RuntimeSelectedAtDispatch || row.Blocker != "" || row.Next != projectapp.PolicyRuntimeGuidance(projectapp.RuntimeSelectedAtDispatch) {
		t.Fatal("unconfigured policy guidance changed or became a gate", row)
	}
	for _, operation := range proposal.Preview().Readiness {
		if operation.Operation == projectapp.OperationExecution && len(operation.Blockers) != 0 && operation.Blockers[0].Code == projectapp.BlockerRuntimePolicyUnavailable {
			t.Fatal("absent optional policy blocked Execution", operation)
		}
	}
}

func TestPolicyReadinessSnapshotParityAndGuidance(t *testing.T) {
	for _, scenario := range []struct {
		name, code string
		config     *runtimeprofile.ConfigurationV3
	}{
		{"missing", projectapp.RuntimeSelectedAtDispatch, nil},
		{"matching_unproven", projectapp.RuntimeSelectedAtDispatch, configPointer(fx.CodexOnly())},
		{"disabled", projectapp.RuntimeSelectedAtDispatch, configPointer(fx.WithRuntimeDisabled(fx.CodexOnly(), fx.CodexRuntimeID))},
		{"wrong_runtime", projectapp.RuntimeSelectedAtDispatch, configPointer(fx.ClaudeOnly())},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newPolicyEditFixture(t)
			if scenario.config != nil {
				store, err := NewRuntimeProfileStore(f.state)
				if err != nil {
					t.Fatal(err)
				}
				snapshot, err := store.Inspect(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				wire, err := runtimeprofile.EncodeV4(operationalTestConfig(*scenario.config))
				if err != nil {
					t.Fatal(err)
				}
				if _, err := store.Publish(context.Background(), snapshot.Token, wire); err != nil {
					t.Fatal(err)
				}
			}
			ports := f.policyPorts(t)
			proposal, failure := projectapp.PreviewPolicyEdit(context.Background(), ports, policyIntent(t, `{}`))
			if failure != projectapp.EditOK {
				t.Fatal(failure)
			}
			selection, _ := ports.Edit.Source.SelectForEdit(context.Background(), "sample")
			subject := projectapp.ReadinessSubject{Project: selection.Portable.Project(), Local: selection.Local}
			code := proposal.Preview().Runtime[0].Code
			if code != scenario.code || code != proposal.Preview().Runtime[0].Code {
				t.Fatalf("parity: %s %+v", code, proposal.Preview().Runtime)
			}
			// Readiness parity uses the canonical non-predictive basis; the
			// per-agent rows share that non-predictive basis and its next step.
			report := projectapp.EvaluateReadiness(projectapp.ReadinessInput{Project: subject.Project, Local: subject.Local, RuntimeCode: projectapp.RuntimeSelectedAtDispatch, Catalog: projectapp.SupportedProviders})
			if !reflect.DeepEqual(report.Operations, proposal.Preview().Readiness) || report.Runtime.Status != projectapp.RuntimeSelectedAtDispatch || proposal.Preview().Runtime[0].Next != projectapp.PolicyRuntimeGuidance(code) || proposal.Preview().Runtime[0].Next == "" {
				t.Fatal("canonical readiness differs", report, proposal.Preview())
			}
		})
	}
}

func configPointer(config runtimeprofile.ConfigurationV3) *runtimeprofile.ConfigurationV3 {
	return &config
}
