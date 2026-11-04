// Compile inside an archive of 9fad368ded09fc2f70e0b88d23fef5c4588869e7.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/workflow"
)

type spy struct{ calls int }

func (s *spy) Build(context.Context, stageexecution.LaunchRequest) (executiongraph.Invocation, error) {
	s.calls++
	panic("builder called")
}
func (s *spy) Collect(context.Context, stageexecution.LaunchRequest, executiongraph.ProcessResult) (stageexecution.Outcome, error) {
	s.calls++
	panic("collector called")
}
func (s *spy) RunObserved(context.Context, executiongraph.Invocation, func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
	s.calls++
	panic("runner called")
}
func snapshot(root string) map[string]string {
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[rel] = info.Mode().String()
		if !d.IsDir() {
			w, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			out[rel] += fmt.Sprintf("%x", sha256.Sum256(w))
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return out
}
func reject(candidate []byte, original workflow.Execution) {
	root, err := os.MkdirTemp("", "i369-old-reader-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	dir := filepath.Join(root, "executions", "v2", original.ProjectID)
	if err = os.MkdirAll(dir, 0700); err != nil {
		panic(err)
	}
	if err = os.WriteFile(filepath.Join(dir, original.ExecutionID+".json"), candidate, 0600); err != nil {
		panic(err)
	}
	store, err := local.NewExecutionStore(root)
	if err != nil {
		panic(err)
	}
	before := snapshot(root)
	effects := &spy{}
	engine := stageexecution.Engine{Store: store, Runner: effects, Launcher: effects, Resolve: func(context.Context, workflow.Execution, workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
		effects.calls++
		panic("resolver called")
	}}
	if _, err = workflow.DecodeExecution(candidate); err == nil {
		panic("old decoder accepted new fields")
	}
	if _, err = engine.Run(context.Background(), original.ProjectID, original.ExecutionID, stageexecution.RunOptions{}); err == nil {
		panic("old engine accepted new fields")
	}
	if _, _, err = engine.Reconcile(context.Background(), original.ProjectID, original.ExecutionID); err == nil {
		panic("old reconcile accepted new fields")
	}
	if effects.calls != 0 || !reflect.DeepEqual(before, snapshot(root)) {
		panic("effects or directory/bytes changed")
	}
}
func main() {
	for _, path := range os.Args[1:] {
		wire, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		original, err := workflow.DecodeExecution(wire)
		if err != nil {
			panic(err)
		}
		back, err := workflow.EncodeExecution(original)
		if err != nil || !bytes.Equal(back, wire) {
			panic("old bytes changed")
		}
		reject(append([]byte(`{"attemptPolicy":"spawn-boundary-v1",`), wire[1:]...), original)
		reject(bytes.Replace(wire, []byte(`"attemptId":`), []byte(`"launchFailure":{"reason":"builder_failed","phase":"invocation_build"},"attemptId":`), 1), original)
		fmt.Println(filepath.Base(path), "PASS original roundtrip; candidate rejection through real old decoder/store/engine/reconcile; zero effects/lease/directory/byte changes")
	}
}
