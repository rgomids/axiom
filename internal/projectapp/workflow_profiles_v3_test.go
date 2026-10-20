package projectapp_test

import (
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	fx "github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// schema2Workflow binds every agent of the built-in Workflow to profileID with
// Profile-owned effort, through the canonical schema-2 encoder.
func schema2Workflow(t *testing.T, profileID string) workflowdefinition.Definition {
	t.Helper()
	copied, issues := workflowdefinition.Decode(workflowdefinition.Builtin().Canonical)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	d := copied.Definition
	d.SchemaVersion = 2
	for i := range d.Stages {
		for j := range d.Stages[i].Agents {
			d.Stages[i].Agents[j].ProfileRef = profileID
			d.Stages[i].Agents[j].Effort = workflowdefinition.Effort{}
		}
	}
	doc, issues := workflowdefinition.Encode(d)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return doc.Definition
}

func v3PolicyState(t *testing.T, profiles ...project.ModelProfile) project.State {
	t.Helper()
	if profiles == nil {
		profiles = []project.ModelProfile{
			{Key: fx.CodexHighProfileID, RuntimeRef: project.Configured(fx.CodexRuntimeID), Model: project.Configured("synthetic-codex-model")},
			{Key: fx.CodexDefaultProfileID, RuntimeRef: project.Configured(fx.CodexRuntimeID), Model: project.Configured("synthetic-codex-model")},
			{Key: fx.CodexUnlistedProfileID, RuntimeRef: project.Configured(fx.CodexRuntimeID), Model: project.Configured("synthetic-codex-model")},
		}
	}
	state := project.State{SchemaVersion: 2, ID: "123e4567-e89b-42d3-a456-426614174000", Slug: "example", Name: "Example",
		Runtimes: project.Configured([]project.Runtime{{ID: fx.CodexRuntimeID}}), ModelProfiles: project.Configured(profiles)}
	p, issues := project.New(state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return p.State()
}

func TestWorkflowProfilesV3BindsExactProfileIntent(t *testing.T) {
	cfg := fx.CodexOnly()
	consumer := fx.Consumer(t, cfg)
	for _, tc := range []struct {
		id     string
		effort runtimeprofile.Effort
	}{{fx.CodexHighProfileID, fx.ExplicitEffort("high")}, {fx.CodexDefaultProfileID, fx.RuntimeDefaultEffort()}} {
		d := schema2Workflow(t, tc.id)
		bindings, code := projectapp.ValidateWorkflowProfilesV3(d, consumer, v3PolicyState(t))
		if code != "" || len(bindings) == 0 {
			t.Fatalf("%s refused: %s", tc.id, code)
		}
		again, _ := projectapp.ValidateWorkflowProfilesV3(d, consumer, v3PolicyState(t))
		if !reflect.DeepEqual(bindings, again) {
			t.Fatal("validation is not deterministic")
		}
		for _, b := range bindings {
			if b.Profile.ProfileID != tc.id || b.Profile.RuntimeID != fx.CodexRuntimeID || b.Profile.Effort != tc.effort || b.Profile.ConfigurationDigest != fx.CurrentDigest(t, cfg) || b.Profile.ConfigurationRevision != cfg.Revision || b.Profile.ProfileDigest == "" {
				t.Fatalf("binding lost exact identity: %+v", b)
			}
		}
	}
}

func TestWorkflowProfilesV3FailsClosed(t *testing.T) {
	codex := fx.CodexOnly()
	withEffort := schema2Workflow(t, fx.CodexHighProfileID)
	withEffort.Stages[0].Agents[0].Effort = workflowdefinition.Effort{Mode: "explicit", Value: "low"}
	legacy := schema2Workflow(t, fx.CodexHighProfileID)
	legacy.SchemaVersion = 1
	constrained := schema2Workflow(t, fx.CodexHighProfileID)
	constrained.Stages[0].Agents[0].RuntimeConstraints = []string{fx.ClaudeRuntimeID}
	empty := schema2Workflow(t, fx.CodexHighProfileID)
	empty.Stages[0].Agents[0].ProfileRef = ""
	otherModel := []project.ModelProfile{{Key: fx.CodexHighProfileID, RuntimeRef: project.Configured(fx.CodexRuntimeID), Model: project.Configured("other-model")}}
	for _, tc := range []struct {
		name     string
		d        workflowdefinition.Definition
		consumer runtimeprofile.ConsumerV4
		profiles []project.ModelProfile
		want     string
	}{
		{"workflow effort override", withEffort, fx.Consumer(t, codex), nil, "forbidden_effort"},
		{"legacy schema not reinterpreted", legacy, fx.Consumer(t, codex), nil, "workflow_migration_required"},
		{"missing profile", schema2Workflow(t, fx.UnknownProfileID), fx.Consumer(t, codex), nil, "profile_effort_unconfigured"},
		{"missing profile reference", empty, fx.Consumer(t, codex), nil, "invalid_profile_mapping"},
		{"disabled runtime", schema2Workflow(t, fx.CodexHighProfileID), fx.Consumer(t, fx.WithRuntimeDisabled(codex, fx.CodexRuntimeID)), nil, "project_policy_update_required"},
		{"not allowlisted", schema2Workflow(t, fx.CodexUnlistedProfileID), fx.Consumer(t, fx.MultiRuntime()), nil, "project_policy_update_required"},
		{"project model drift", schema2Workflow(t, fx.CodexHighProfileID), fx.Consumer(t, codex), otherModel, "project_policy_update_required"},
		{"runtime constraint mismatch", constrained, fx.Consumer(t, codex), nil, "profile_runtime_mismatch"},
		{"unconsumed configuration", schema2Workflow(t, fx.CodexHighProfileID), runtimeprofile.ConsumerV4{}, nil, "profile_effort_unconfigured"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bindings, code := projectapp.ValidateWorkflowProfilesV3(tc.d, tc.consumer, v3PolicyState(t, tc.profiles...))
			if code != tc.want || bindings != nil {
				t.Fatalf("got %q want %q", code, tc.want)
			}
		})
	}
}

func TestWorkflowProfilesV3RetainsMultiRuntimeAuthoringGate(t *testing.T) {
	state := v3PolicyState(t)
	state.SchemaVersion = 1
	state.Runtimes = project.Declaration[[]project.Runtime]{}
	state.Runtime = project.Configured(project.Runtime{ID: fx.CodexRuntimeID})
	p, issues := project.New(state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	bindings, code := projectapp.ValidateWorkflowProfilesV3(schema2Workflow(t, fx.CodexHighProfileID), fx.Consumer(t, fx.CodexOnly()), p.State())
	if bindings != nil || code != "project_policy_update_required" {
		t.Fatal("legacy Project widened Workflow authoring permission")
	}
}
