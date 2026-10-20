package runtimeadapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func operationalSourceFixture(t *testing.T, runtimeID string, explicit bool) (OperationalSource, operationalEffortFixture) {
	t.Helper()
	f := newOperationalEffortFixture(t, runtimeID, explicit)
	executable := f.stage.Profile.Executable
	source := OperationalSource{Configuration: f.request.Configuration, LookPath: func(name string) (string, error) {
		if name != runtimeID {
			return "", os.ErrNotExist
		}
		if _, err := os.Lstat(executable); err != nil {
			return "", err
		}
		return executable, nil
	}, Repository: f.request.Roots.Project, Environment: f.stage.ParentEnvironment, Consented: true, ObservedAt: time.Unix(1, 0)}
	return source, f
}

func TestOperationalSourceIdentityWithoutAuthVersionOrProof(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		source, f := operationalSourceFixture(t, runtimeID, false)
		before, err := source.ObserveExecutable(context.Background(), runtimeID)
		if err != nil || !before.Installed || !before.Available || before.ExecutableDigest != f.request.ExecutableDigest || before.EffortSourceDigest != "" || before.Revision != 1 {
			t.Fatal("identity observation asserted more than executable availability", before, err)
		}
		if err := os.WriteFile(f.stage.Profile.Executable, []byte("different synthetic bytes; never execute"), 0o700); err != nil {
			t.Fatal(err)
		}
		after, err := source.ObserveExecutable(context.Background(), runtimeID)
		if err != nil || before.ExecutableDigest == after.ExecutableDigest {
			t.Fatal("identity observation was cached", err)
		}
		if err := os.Remove(f.stage.Profile.Executable); err != nil {
			t.Fatal(err)
		}
		if _, err := source.ObserveExecutable(context.Background(), runtimeID); !errors.Is(err, ErrExecutableUnavailable) {
			t.Fatal("missing executable observed", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := source.ObserveExecutable(ctx, runtimeID); err == nil {
			t.Fatal("cancelled observation succeeded")
		}
	}
}

func TestOperationalSourceEffortDigestMatchesRealInvocationView(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		for _, explicit := range []bool{false, true} {
			source, f := operationalSourceFixture(t, runtimeID, explicit)
			profileID := f.request.ProfileID
			unconsented := source
			unconsented.Consented = false
			if _, err := unconsented.ObserveOperational(context.Background(), runtimeID, profileID); !errors.Is(err, ErrInspectionConsentRequired) {
				t.Fatal("effort sources inspected without consent", err)
			}
			observed, err := source.ObserveOperational(context.Background(), runtimeID, profileID)
			if err != nil {
				t.Fatal(err)
			}
			roots, err := (StageEffortRoots{Repository: f.request.Roots.Project}).ResolveOperational(context.Background(), f.request.Invocation, true)
			if err != nil {
				t.Fatal(err)
			}
			real := f.request
			real.Roots = roots
			want, err := InspectEffortSourcesV4(context.Background(), real)
			if err != nil || observed.EffortSourceDigest != want {
				t.Fatal("role placeholders changed the repository-view digest", err)
			}
			conflict := filepath.Join(f.request.Roots.Project, ".codex", "config.toml")
			content := `model_reasoning_effort = "future-effort"`
			if runtimeID == "claude" {
				conflict, content = filepath.Join(f.request.Roots.Project, ".claude", "settings.json"), `{"effortLevel":"future-effort"}`
			}
			write(t, conflict, content)
			if _, err := source.ObserveOperational(context.Background(), runtimeID, profileID); !errors.Is(err, ErrEffortOverrideConflict) {
				t.Fatal("equal-value competing selector admitted", err)
			}
			relocated := source
			relocated.Environment = append([]string{"CODEX_HOME=/relocated", "CLAUDE_CONFIG_DIR=/relocated"}, source.Environment...)
			if _, err := relocated.ObserveOperational(context.Background(), runtimeID, profileID); !errors.Is(err, ErrEffortConfigurationUnsafe) {
				t.Fatal("relocated configuration admitted", err)
			}
			if _, err := source.ObserveOperational(context.Background(), runtimeID, "absent"); err == nil {
				t.Fatal("unallowed Profile observed")
			}
		}
	}
}
