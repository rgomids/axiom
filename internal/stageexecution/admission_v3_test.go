package stageexecution

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowcompiler"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// currentContractV3 reuses the current admission fixture and replaces only
// the configuration with a canonical format-3 record holding the frozen
// Profile, plus one unrelated allowlisted Profile on the same Runtime.
func currentContractV3(t *testing.T) (ContractV3, runtimeprofile.ConfigurationV3) {
	t.Helper()
	c := currentContract(t)
	p := c.Execution.Binding.ResolvedProfile
	cfg := runtimeprofile.ConfigurationV3{FormatVersion: runtimeprofile.FormatVersionV3, Revision: 4,
		Runtimes: []runtimeprofile.RuntimeV3{{ID: p.RuntimeID, Adapter: p.RuntimeID, Enabled: true, AllowlistedProfileIDs: []string{p.ProfileID, "other-profile"}}},
		ModelProfiles: []runtimeprofile.ModelProfile{
			{ID: p.ProfileID, RuntimeID: p.RuntimeID, Model: p.Model, Effort: p.Effort, Capabilities: []string{"read"}},
			{ID: "other-profile", RuntimeID: p.RuntimeID, Model: "other-model", Effort: runtimeprofile.Effort{Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"}, Capabilities: []string{"read"}},
		}}
	if err := runtimeprofile.ValidateV3(cfg); err != nil {
		t.Fatal(err)
	}
	v3 := cfg
	return ContractV3{Definition: c.Definition, Execution: c.Execution, Graph: c.Graph, Record: runtimeprofile.RecordActiveV3,
		Configuration: runtimeprofile.VersionedConfiguration{FormatVersion: runtimeprofile.FormatVersionV3, V3: &v3}, WorkflowSelected: true, PolicyAllowed: true}, cfg
}

func TestAdmitV3ActiveBindsCurrentIdentity(t *testing.T) {
	c, cfg := currentContractV3(t)
	before, err := runtimeprofile.EncodeV3(cfg)
	if err != nil {
		t.Fatal(err)
	}
	frozen := *c.Execution.Binding.ResolvedProfile
	for _, op := range []Operation{StagePlan, StageRun, StageRetry, StageResume, StageCancel, StageReconcile, StageIntegrate, CapabilityProbe} {
		admission, err := AdmitV3(op, c)
		if err != nil || admission.Next != "" || admission.Profile == nil {
			t.Fatalf("%s refused: %v", op, err)
		}
		digest, _ := runtimeprofile.DigestV3(cfg)
		profileDigest, _ := runtimeprofile.ProfileDigest(cfg.ModelProfiles[0])
		got := *admission.Profile
		if got.RuntimeID != frozen.RuntimeID || got.ProfileID != frozen.ProfileID || got.Model != frozen.Model || got.Effort != frozen.Effort ||
			got.ConfigurationDigest != digest || got.ConfigurationRevision != cfg.Revision || got.ProfileDigest != profileDigest {
			t.Fatalf("%s did not bind exact v3 identity: %+v", op, got)
		}
	}
	after, _ := runtimeprofile.EncodeV3(*c.Configuration.V3)
	if !bytes.Equal(before, after) || !reflect.DeepEqual(*c.Execution.Binding.ResolvedProfile, frozen) {
		t.Fatal("admission mutated its inputs")
	}
	c.PolicyAllowed = false
	if _, err := AdmitV3(StageRun, c); !errors.Is(err, ErrPolicyUpdateRequired) {
		t.Fatal("policy authority bypassed", err)
	}
}

func TestAdmitV3RefusesNonCurrentConfiguration(t *testing.T) {
	base, cfg := currentContractV3(t)
	p := base.Execution.Binding.ResolvedProfile
	invalid := cfg
	invalid.ModelProfiles = append([]runtimeprofile.ModelProfile(nil), cfg.ModelProfiles...)
	invalid.ModelProfiles[0].Effort = runtimeprofile.Effort{}
	disabled := cfg
	disabled.Runtimes = []runtimeprofile.RuntimeV3{cfg.Runtimes[0]}
	disabled.Runtimes[0].Enabled = false
	unlisted := cfg
	unlisted.Runtimes = []runtimeprofile.RuntimeV3{cfg.Runtimes[0]}
	unlisted.Runtimes[0].AllowlistedProfileIDs = []string{"other-profile"}
	// The frozen Profile is gone; the remaining Profile is never substituted.
	removed := unlisted
	removed.ModelProfiles = []runtimeprofile.ModelProfile{cfg.ModelProfiles[1]}
	for _, tc := range []struct {
		name   string
		record runtimeprofile.RecordStateV3
		v      runtimeprofile.VersionedConfiguration
		want   error
	}{
		{"missing record", runtimeprofile.RecordMissingV3, runtimeprofile.VersionedConfiguration{}, ErrProfileEffortUnconfigured},
		{"retired record", runtimeprofile.RecordRetiredV3, runtimeprofile.VersionedConfiguration{}, ErrProfileEffortUnconfigured},
		{"historical v1", runtimeprofile.RecordActiveV3, runtimeprofile.VersionedConfiguration{FormatVersion: runtimeprofile.FormatVersion, Configuration: &runtimeprofile.Configuration{FormatVersion: runtimeprofile.FormatVersion}}, ErrProfileEffortUnconfigured},
		{"historical v2", runtimeprofile.RecordActiveV3, runtimeprofile.VersionedConfiguration{FormatVersion: runtimeprofile.FormatVersionV2, Configuration: &runtimeprofile.Configuration{FormatVersion: runtimeprofile.FormatVersionV2}}, ErrProfileEffortUnconfigured},
		{"invalid v3", runtimeprofile.RecordActiveV3, runtimeprofile.VersionedConfiguration{FormatVersion: runtimeprofile.FormatVersionV3, V3: &invalid}, ErrAdmission},
		{"inconsistent retired with configuration", runtimeprofile.RecordRetiredV3, runtimeprofile.VersionedConfiguration{FormatVersion: runtimeprofile.FormatVersionV3, V3: &cfg}, ErrAdmission},
		{"disabled runtime", runtimeprofile.RecordActiveV3, runtimeprofile.VersionedConfiguration{FormatVersion: runtimeprofile.FormatVersionV3, V3: &disabled}, ErrProfileEffortUnconfigured},
		{"not allowlisted", runtimeprofile.RecordActiveV3, runtimeprofile.VersionedConfiguration{FormatVersion: runtimeprofile.FormatVersionV3, V3: &unlisted}, ErrProfileEffortUnconfigured},
		{"frozen profile removed", runtimeprofile.RecordActiveV3, runtimeprofile.VersionedConfiguration{FormatVersion: runtimeprofile.FormatVersionV3, V3: &removed}, ErrProfileEffortUnconfigured},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			c.Record, c.Configuration = tc.record, tc.v
			admission, err := AdmitV3(StageRun, c)
			if !errors.Is(err, tc.want) || admission.Profile != nil {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
	mismatch := base
	frozen := *p
	frozen.RuntimeID = "other-runtime"
	binding := *base.Execution.Binding
	binding.ResolvedProfile = &frozen
	mismatch.Execution.Binding = &binding
	if _, err := AdmitV3(CapabilityProbe, mismatch); !errors.Is(err, ErrProfileEffortUnconfigured) {
		t.Fatal("Runtime/Profile relationship not exact", err)
	}
}

// Execution, graph, Workflow and historical inspection semantics are Admit's
// own: for every non-configuration row both evaluators decide identically,
// whatever the format-3 record state is.
func TestAdmitV3PreservesExecutionAndInspectionSemantics(t *testing.T) {
	historical := historicalGraph(t, 2)
	rows := []struct {
		name   string
		op     Operation
		change func(*Contract)
	}{
		{"schema1", StagePlan, func(c *Contract) { c.Definition, _ = workflowdefinition.BuiltinRevision(1) }},
		{"unselected", StageRun, func(c *Contract) { c.WorkflowSelected = false }},
		{"legacy run", StageRun, func(c *Contract) { c.Execution.FormatVersion = 2 }},
		{"legacy binding", StageResume, func(c *Contract) { c.Execution.Binding.ResolvedProfile = nil }},
		{"legacy inspection", StageStatus, func(c *Contract) { c.Execution.FormatVersion = 2 }},
		{"historical graph inspection", CoordinationRead, func(c *Contract) { c.Graph = &historical }},
		{"historical graph mutation", StageRun, func(c *Contract) { c.Graph = &historical }},
		{"current inspection", StageStatus, func(*Contract) {}},
		{"unknown operation", Operation("stage.other"), func(*Contract) {}},
	}
	for _, row := range rows {
		for _, record := range []runtimeprofile.RecordStateV3{runtimeprofile.RecordActiveV3, runtimeprofile.RecordMissingV3, runtimeprofile.RecordRetiredV3} {
			legacy := currentContract(t)
			b := *legacy.Execution.Binding
			legacy.Execution.Binding = &b
			row.change(&legacy)
			v3, _ := currentContractV3(t)
			v3.Definition, v3.Execution, v3.Graph, v3.WorkflowSelected = legacy.Definition, legacy.Execution, legacy.Graph, legacy.WorkflowSelected
			if record != runtimeprofile.RecordActiveV3 {
				v3.Record, v3.Configuration = record, runtimeprofile.VersionedConfiguration{}
			}
			want, wantErr := Admit(row.op, legacy)
			got, gotErr := AdmitV3(row.op, v3)
			if wantErr == nil && row.op != StageStatus && row.op != CoordinationRead {
				continue // configuration-dependent rows are covered above
			}
			if got.Admission != want || !errors.Is(gotErr, wantErr) && gotErr != wantErr || got.Profile != nil {
				t.Fatalf("%s/%d: got %+v %v want %+v %v", row.name, record, got, gotErr, want, wantErr)
			}
		}
	}
	if _, err := AdmitV3(StagePlan, func() ContractV3 {
		c, _ := currentContractV3(t)
		c.Definition, _ = workflowdefinition.BuiltinRevision(1)
		return c
	}()); !errors.Is(err, workflowcompiler.ErrWorkflowMigrationRequired) {
		t.Fatal("schema-1 Workflow reinterpreted", err)
	}
	if _, err := AdmitV3(CoordinationRead, func() ContractV3 {
		c, _ := currentContractV3(t)
		c.Graph = &historical
		return c
	}()); err != nil {
		t.Fatal("historical graph inspection refused", err)
	}
	if _, err := AdmitV3(StageRun, func() ContractV3 {
		c, _ := currentContractV3(t)
		c.Graph = &historical
		return c
	}()); !errors.Is(err, executiongraph.ErrUnsupportedGraphFormat) {
		t.Fatal("historical graph mutation admitted", err)
	}
}
