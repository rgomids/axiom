package stageexecution

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

func currentContractV4(t *testing.T) (Contract, runtimeprofile.ConfigurationV4) {
	t.Helper()
	c := currentContract(t)
	cfg := *c.Configuration
	copy := cfg
	c.Configuration = &copy
	return c, cfg
}

func TestAdmitActiveBindsCurrentIdentity(t *testing.T) {
	c, cfg := currentContractV4(t)
	before, err := runtimeprofile.EncodeV4(cfg)
	if err != nil {
		t.Fatal(err)
	}
	frozen := *c.Execution.Binding.FrozenProfile
	for _, op := range []Operation{StagePlan, StageRun, StageRetry, StageResume, StageCancel, StageReconcile, StageIntegrate} {
		admission, err := Admit(op, c)
		if err != nil || admission.Next != "" || admission.Profile == nil {
			t.Fatalf("%s refused: %v", op, err)
		}
		digest, _ := runtimeprofile.DigestV4(cfg)
		profileDigest, _ := runtimeprofile.ProfileDigest(cfg.ModelProfiles[0])
		got := *admission.Profile
		if got.RuntimeID != frozen.RuntimeID || got.ProfileID != frozen.ProfileID || got.Model != frozen.Model || got.Effort != frozen.Effort ||
			got.ConfigurationDigest != digest || got.ConfigurationRevision != cfg.Revision || got.ProfileDigest != profileDigest {
			t.Fatalf("%s did not bind exact v4 identity: %+v", op, got)
		}
	}
	after, _ := runtimeprofile.EncodeV4(*c.Configuration)
	if !bytes.Equal(before, after) || !reflect.DeepEqual(*c.Execution.Binding.FrozenProfile, frozen) {
		t.Fatal("admission mutated its inputs")
	}
	c.PolicyAllowed = false
	if _, err := Admit(StageRun, c); !errors.Is(err, ErrPolicyUpdateRequired) {
		t.Fatal("policy authority bypassed", err)
	}
}

func TestAdmitRefusesNonCurrentConfiguration(t *testing.T) {
	base, cfg := currentContractV4(t)
	p := base.Execution.Binding.FrozenProfile
	invalid := cfg
	invalid.ModelProfiles = append([]runtimeprofile.ModelProfile(nil), cfg.ModelProfiles...)
	invalid.ModelProfiles[0].Effort = runtimeprofile.Effort{}
	disabled := cfg
	disabled.Runtimes = []runtimeprofile.RuntimeV4{cfg.Runtimes[0]}
	disabled.Runtimes[0].Enabled = false
	unlisted := cfg
	unlisted.Runtimes = []runtimeprofile.RuntimeV4{cfg.Runtimes[0]}
	unlisted.Runtimes[0].AllowlistedProfileIDs = []string{"other-profile"}
	removed := unlisted
	removed.ModelProfiles = []runtimeprofile.ModelProfile{cfg.ModelProfiles[1]}
	for _, tc := range []struct {
		name, record string
		cfg          *runtimeprofile.ConfigurationV4
		want         error
	}{
		{"missing", "fresh", nil, ErrProfileEffortUnconfigured},
		{"retired", "retired", nil, ErrProfileEffortUnconfigured},
		{"historical1", "active-v1", nil, ErrProfileEffortUnconfigured},
		{"historical2", "active-v2", nil, ErrProfileEffortUnconfigured},
		{"historical3", "active-v3", nil, ErrProfileEffortUnconfigured},
		{"invalid", "active-v4", &invalid, ErrAdmission},
		{"disabled", "active-v4", &disabled, ErrProfileEffortUnconfigured},
		{"unlisted", "active-v4", &unlisted, ErrProfileEffortUnconfigured},
		{"removed", "active-v4", &removed, ErrProfileEffortUnconfigured},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			c.Record, c.Configuration = tc.record, tc.cfg
			a, err := Admit(StageRun, c)
			if !errors.Is(err, tc.want) || a.Profile != nil {
				t.Fatalf("got %+v %v want %v", a, err, tc.want)
			}
		})
	}

	mismatch := base
	frozen := *p
	frozen.RuntimeID = "other-runtime"
	binding := *base.Execution.Binding
	binding.FrozenProfile = &frozen
	mismatch.Execution.Binding = &binding
	if _, err := Admit(StagePlan, mismatch); !errors.Is(err, ErrProfileEffortUnconfigured) {
		t.Fatal("Runtime/Profile relationship not exact", err)
	}
}

func TestAdmissionInspectionIndependentOfConfiguration(t *testing.T) {
	historical := historicalGraph(t, 2)
	for _, record := range []string{"active-v4", "fresh", "retired", "active-v3"} {
		for _, op := range []Operation{StageStatus, CoordinationRead} {
			for _, generation := range []int{2, 3, 5} {
				c := currentContract(t)
				c.Record, c.Configuration = record, nil
				c.Execution.FormatVersion = generation
				a, err := Admit(op, c)
				next := ""
				if generation != 5 {
					next = ErrLegacyExecution.Error()
				}
				if err != nil || a.Next != next || a.Profile != nil {
					t.Fatalf("%s/%s/%d: %+v %v", record, op, generation, a, err)
				}
			}
			c := currentContract(t)
			c.Record, c.Configuration, c.Graph = record, nil, &historical
			a, err := Admit(op, c)
			if err != nil || a.Next != executiongraph.ErrUnsupportedGraphFormat.Error() || a.Profile != nil {
				t.Fatalf("historical: %+v %v", a, err)
			}
		}
	}
}
