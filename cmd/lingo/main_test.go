package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
)

func TestComposedCLICompletesMinimalPortableLifecycle(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	service := compose()

	runCLI(t, service, []string{"project", "init", "--slug", "sample", "--name", "Sample"}, cli.ExitSuccess, "applied")
	runCLI(t, service, []string{"project", "validate", "--slug", "sample"}, cli.ExitSuccess, "valid")
	runCLI(t, service, []string{"project", "reopen", "--slug", "sample"}, cli.ExitSuccess, "reopened")
	runCLI(t, service, []string{"project", "update", "--slug", "sample", "--name", "Changed"}, cli.ExitSuccess, "applied")

	manifest, err := os.ReadFile(filepath.Join(root, "sample", "axiom.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), "name: Changed") {
		t.Fatalf("updated manifest does not contain new name: %q", manifest)
	}
}

func TestComposedCLIRejectsRelativeRoot(t *testing.T) {
	t.Setenv("LINGO_PROJECTS_ROOT", "relative")
	var output bytes.Buffer
	if code := cli.Run(context.Background(), []string{"project", "init", "--slug", "sample", "--name", "Sample"}, compose(), &output); code != cli.ExitFailure {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(output.String(), "application_unavailable") {
		t.Fatalf("unexpected error event: %q", output.String())
	}
}

func runCLI(t *testing.T, service cli.Service, args []string, wantCode int, wantCategory string) {
	t.Helper()
	var output bytes.Buffer
	if code := cli.Run(context.Background(), args, service, &output); code != wantCode {
		t.Fatalf("%v: exit code = %d, output=%q", args, code, output.String())
	}
	if !strings.Contains(output.String(), `"category":"`+wantCategory+`"`) {
		t.Fatalf("%v: category absent from %q", args, output.String())
	}
}
