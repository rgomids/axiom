package runtimeadapter

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

func (f operationalEffortFixture) frozen(t *testing.T) runtimeprofile.ResolvedProfileV4 {
	t.Helper()
	profile := f.request.Configuration.ModelProfiles[0]
	profileDigest, err := runtimeprofile.ProfileDigest(profile)
	if err != nil {
		t.Fatal(err)
	}
	configurationDigest, err := runtimeprofile.DigestV4(f.request.Configuration)
	if err != nil {
		t.Fatal(err)
	}
	digest := strings.Repeat("a", 64)
	return runtimeprofile.ResolvedProfileV4{ProfileID: profile.ID, RuntimeID: profile.RuntimeID, Model: profile.Model, Effort: profile.Effort, ProfileDigest: profileDigest, ConfigurationRevision: f.request.Configuration.Revision, ConfigurationDigest: configurationDigest, ExecutableDigest: f.request.ExecutableDigest, InvocationMode: "noninteractive", ObservationRevision: 1, ObservationDigest: digest, EffortSourceDigest: digest}
}

func TestFrozenStageDeliveredEffortInspection(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		for _, explicit := range []bool{true, false} {
			t.Run(id+map[bool]string{true: "/explicit", false: "/runtime-default"}[explicit], func(t *testing.T) {
				f := newOperationalEffortFixture(t, id, explicit)
				frozen := f.frozen(t)
				inv := f.request.Invocation
				var err error
				if err != nil {
					t.Fatal(err)
				}
				inspector := StageEffortInspector{Configuration: func(context.Context) (runtimeprofile.ConfigurationV4, error) { return f.request.Configuration, nil }, Repository: f.request.Roots.Project, Workspaces: []string{f.child}, Consented: true}
				digest, err := inspector.CheckEffortSources(context.Background(), frozen, inv)
				if err != nil || digest == "" {
					t.Fatal(digest, err)
				}
				content := "model_reasoning_effort = \"high\"\n"
				if id == "claude" {
					content = `{"effortLevel":"high"}`
				}
				// A late personal selector never blocks; managed refusal is
				// covered by the source inspection tests.
				source := fixedEffortRules(id, f.request.Roots).Sources[0]
				write(t, filepath.Join(source.Root, source.Relative), content)
				if _, err := inspector.CheckEffortSources(context.Background(), frozen, inv); err != nil {
					t.Fatal("late personal selector blocked the run", err)
				}
			})
		}
	}
}
