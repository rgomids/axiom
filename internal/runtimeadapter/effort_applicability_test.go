package runtimeadapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// BI-3.2 item 5: applicability is observed passively from the resolved
// executable identity and the version observer keyed by it; the synthetic
// executable is never run, and drift or absence is unproven.
func TestObserveEffortApplicability(t *testing.T) {
	ctx := context.Background()
	executable := filepath.Join(t.TempDir(), "codex")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nexit 97\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := ExecutableIdentity(executable)
	if err != nil {
		t.Fatal(err)
	}
	look := func(string) (string, error) { return executable, nil }
	version := func(_ context.Context, id, observed string) (string, error) {
		if id != "codex" || observed != digest {
			return "", errors.New("unobserved")
		}
		return "synthetic-codex-1", nil
	}
	got, err := ObserveEffortApplicability(ctx, look, version, "codex")
	if err != nil || got.RuntimeID != "codex" || got.RuntimeVersion != "synthetic-codex-1" || got.ExecutableDigest != digest || got.InvocationMode != "noninteractive" {
		t.Fatalf("%+v %v", got, err)
	}
	drift := func(_ context.Context, _, _ string) (string, error) {
		if err := os.WriteFile(executable, []byte("#!/bin/sh\nexit 98\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		return "synthetic-codex-1", nil
	}
	cases := map[string]struct {
		look    func(string) (string, error)
		version func(context.Context, string, string) (string, error)
		id      string
	}{
		"no-version-observer": {look, nil, "codex"},
		"version-unobserved":  {look, func(context.Context, string, string) (string, error) { return "", errors.New("none") }, "codex"},
		"unsafe-version":      {look, func(context.Context, string, string) (string, error) { return "1.0\n", nil }, "codex"},
		"absent":              {func(string) (string, error) { return "", errors.New("absent") }, version, "codex"},
		"relative":            {func(string) (string, error) { return "codex", nil }, version, "codex"},
		"unsupported-runtime": {look, version, "other"},
		"drift":               {look, drift, "codex"},
	}
	for name, c := range cases {
		if _, err := ObserveEffortApplicability(ctx, c.look, c.version, c.id); !errors.Is(err, ErrEffortConfigurationUnproven) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
