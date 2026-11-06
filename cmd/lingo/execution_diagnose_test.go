//go:build linux || darwin

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/workflow"
)

func TestI369PublicDiagnosisReadOnly(t *testing.T) {
	f := newExecutionRunFixture(t)
	started := f.run(t, cli.ExitSuccess, minimalRun("--authorize-external")...)
	dir := filepath.Join(os.Getenv("HOME"), ".codex")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, conflict := range []bool{false, true} {
		wire := "notify=['synthetic']\n"
		if conflict {
			wire += "model_reasoning_effort=['runtime-default'\n"
		}
		if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(wire), 0600); err != nil {
			t.Fatal(err)
		}
		for _, prospective := range []bool{false, true} {
			before := snapshot303Trees(t, f.env.root, f.env.state, f.env.elsewhere)
			args := []string{"workflow", "stage", "diagnose", "--project", "external", "--execution", started.Stage.ExecutionID, "--stage", workflow.DefaultStageID, "--agent", workflow.DefaultAgentID}
			if prospective {
				args = append(args, "--new-selection", "--select", "agent.model=prospective-model")
			}
			var out bytes.Buffer
			if code := cli.Run(context.Background(), args, f.env.service, currentProvenance(), &out); code != cli.ExitSuccess {
				t.Fatal(code, out.String())
			}
			var event struct {
				Stage invocationDiagnosis `json:"stage"`
			}
			if err := json.Unmarshal(out.Bytes(), &event); err != nil {
				t.Fatal(err)
			}
			d := event.Stage
			want := "admissible"
			if conflict {
				want = "refused"
			}
			if d.InvocationAdmissibility != want || d.Authentication != "not_evaluated" || d.Inference != "not_run" || d.OutputDirectoryMaterialization != "not_evaluated" {
				t.Fatalf("untruthful diagnosis: %s", out.String())
			}
			if conflict && (d.LaunchFailure == nil || d.LaunchFailure.Reason != "configuration_invalid") {
				t.Fatal("reason lost")
			}
			if prospective != (d.SelectionMode == "prospective_preview") {
				t.Fatal("preview not labeled")
			}
			if !bytes.Equal(before, snapshot303Trees(t, f.env.root, f.env.state, f.env.elsewhere)) {
				t.Fatal("diagnose changed persistent state")
			}
			if strings.Contains(out.String(), os.Getenv("HOME")) {
				t.Fatal("private path disclosed")
			}
		}
	}
}
