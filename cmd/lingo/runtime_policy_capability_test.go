package main

import (
	"context"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type fakeVersionBase struct{ digest string }

func (f *fakeVersionBase) Observe(context.Context, string) (runtimeprofile.Observation, error) {
	return runtimeprofile.Observation{RuntimeID: "codex", Installed: true, Available: true, ExecutableDigest: f.digest}, nil
}
func TestRuntimeVersionObserverUsesCurrentFacts(t *testing.T) {
	base := &fakeVersionBase{digest: strings.Repeat("a", 64)}
	observer := runtimeVersionObserver{base: base}
	observed, err := observer.Observe(context.Background(), "codex")
	if err != nil || observed.Version != "" {
		t.Fatal(observed, err)
	}
	observer.version = func(_ context.Context, id, digest string) (string, error) {
		if id != "codex" || digest != base.digest {
			t.Fatal("wrong identity")
		}
		return "1.0", nil
	}
	observed, err = observer.Observe(context.Background(), "codex")
	if err != nil || observed.Version != "1.0" {
		t.Fatal(observed, err)
	}
	observer.version = func(context.Context, string, string) (string, error) {
		base.digest = strings.Repeat("b", 64)
		return "1.0", nil
	}
	if _, err = observer.Observe(context.Background(), "codex"); err == nil {
		t.Fatal("interface-time replacement admitted")
	}
}
