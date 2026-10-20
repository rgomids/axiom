package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	fx "github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

func TestRuntimeProfileValidityV3(t *testing.T) {
	valid := operationalCLIConfig(fx.CodexOnly())
	invalid := operationalCLIConfig(fx.CodexOnly())
	invalid.ModelProfiles[0].Effort = runtimeprofile.Effort{}
	historical := runtimeprofile.Configuration{FormatVersion: runtimeprofile.FormatVersionV2}
	for _, tc := range []struct {
		name     string
		snapshot local.RuntimeProfileSnapshot
		err      error
		want     string
	}{
		{"active v4", local.RuntimeProfileSnapshot{State: "active-v4", V4: &valid}, nil, ""},
		{"v1 unsupported", local.RuntimeProfileSnapshot{State: "active-v1", Legacy: runtimeprofile.Configuration{FormatVersion: runtimeprofile.FormatVersion}}, nil, "unsupported_configuration_format"},
		{"v2 unsupported", local.RuntimeProfileSnapshot{State: "active-v2", Legacy: historical}, nil, "unsupported_configuration_format"},
		{"invalid v4", local.RuntimeProfileSnapshot{State: "active-v4", V4: &invalid}, nil, "invalid_configuration"},
		{"missing", local.RuntimeProfileSnapshot{State: "fresh"}, nil, "configuration_missing"},
		{"retired", local.RuntimeProfileSnapshot{State: "retired", Revision: 3}, nil, "configuration_retired"},
		{"unsafe", local.RuntimeProfileSnapshot{State: "unsafe"}, local.ErrUnsafe, "configuration_unsafe"},
		{"recovery required", local.RuntimeProfileSnapshot{State: "pending"}, local.ErrRecoveryRequired, "configuration_recovery_required"},
		{"read failure", local.RuntimeProfileSnapshot{}, errors.New("synthetic"), "configuration_unsafe"},
	} {
		if got := runtimeProfileValidity(tc.snapshot, tc.err); got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}

}

func TestRuntimeProfileValidityV3FreshStoreZeroWrite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	store, err := local.NewRuntimeProfileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Inspect(context.Background())
	if got := runtimeProfileValidity(snapshot, err); got != "configuration_missing" {
		t.Fatalf("fresh store: got %q", got)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("validation created local state")
	}
}
