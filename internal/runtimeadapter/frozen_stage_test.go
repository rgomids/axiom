package runtimeadapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Reuses delivered X05 inspection against inert temp files, never an executable
// Runtime, ambient HOME, probe, network or inference.
func TestFrozenStageDeliveredEffortInspection(t *testing.T) {
	skipPOSIXStagePaths(t)
	for _, id := range []string{"codex", "claude"} {
		for _, mode := range []string{"explicit", "runtime-default"} {
			t.Run(id+mode, func(t *testing.T) {
				inspection, p, original := effortInspectionFixture(t, id)
				// Keep test names containing "effort" out of argv path values:
				// the delivered inspector conservatively rejects that token.
				workspace, err := os.MkdirTemp("", "axiom-stage-")
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { os.RemoveAll(workspace) })
				workspace, err = filepath.EvalSymlinks(workspace)
				if err != nil {
					t.Fatal(err)
				}
				exe := filepath.Join(workspace, id)
				if err := os.Rename(original.Argv[0], exe); err != nil {
					t.Fatal(err)
				}
				p.ExecutableDigest, err = ExecutableIdentity(exe)
				if err != nil {
					t.Fatal(err)
				}
				p.RuntimeVersion = "0.159.1"
				if id == "claude" {
					p.RuntimeVersion = "2.1.296"
				}
				p.Effort = runtimeprofile.Effort{Mode: mode, Value: "default"}
				if mode == "explicit" {
					p.Effort.Value = "high"
				}
				inspection.Declaration.ExecutableDigest = p.ExecutableDigest
				inspection.Declaration.RuntimeVersion = p.RuntimeVersion
				request := StageInvocationRequest{Profile: CommandProfile{RuntimeID: id, ModelProfileID: p.ProfileID, Executable: exe, Model: p.Model}, Version: p.RuntimeVersion, Workspace: filepath.Join(workspace, "worktree"), Inbox: filepath.Join(workspace, "fake-inbox"), AxiomPath: filepath.Join(workspace, "axiom")}
				inv, err := BuildFrozenStageInvocation(request, p)
				if err != nil {
					t.Fatal(err)
				}
				digest, err := inspection.CheckEffortSources(context.Background(), p, inv)
				if err != nil || digest == "" {
					t.Fatal(digest, err)
				}
				source := inspection.Declaration.Sources[0]
				content := []byte("model_reasoning_effort = \"high\"\n")
				if id == "claude" {
					content = []byte(`{"fixtureEffortKey":"high"}`)
				}
				if err := os.WriteFile(filepath.Join(source.Root, source.Relative), content, 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := inspection.CheckEffortSources(context.Background(), p, inv); !errors.Is(err, ErrEffortOverrideConflict) {
					t.Fatal("late selector accepted", err)
				}
			})
		}
	}
}
