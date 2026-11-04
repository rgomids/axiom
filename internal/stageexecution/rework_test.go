//go:build linux || darwin

package stageexecution

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
)

func TestLinkedReworkResolvesFreshAuthorityAndInvocationOnSharedEngine(t *testing.T) {
	f, store := startFixture(t, "OUTPUT result\n")
	ctx := context.Background()
	source, err := f.engine.Start(ctx, store, linkedRequest())
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := source.StageResultDigest(workflow.DefaultStageID)
	source, err = store.UpdateFenced(ctx, source.ProjectID, source.ExecutionID, nil, func(x *workflow.Execution) error {
		return x.RejectResult(workflow.DefaultStageID, "human:operator", digest, workflow.ArtifactRef{Name: "human-review", RetainedID: "evidence", RepositoryKey: "app", Digest: workflow.DigestText([]byte("review")), Size: 1}, f.engine.now())
	})
	if err != nil {
		t.Fatal(err)
	}
	link, err := source.RejectedSource()
	if err != nil {
		t.Fatal(err)
	}
	request := linkedRequest()
	request.ExecutionID = "00000000-0000-4000-8000-000000000099"
	request.ReworkOf = &link
	resolve := f.engine.Resolve
	f.engine.Resolve = func(context.Context, workflow.Execution, workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
		return workflow.Invocation{}, workflow.AttemptAuthority{}, workflow.ErrUnauthorizedEffect
	}
	if _, err = f.engine.Start(ctx, store, request); !errors.Is(err, workflow.ErrUnauthorizedEffect) || f.runner.spawns.Load() != 1 {
		t.Fatal("rework inherited source authority", err)
	}
	if _, err = store.Load(ctx, request.ProjectID, request.ExecutionID); err == nil {
		t.Fatal("denied rework persisted")
	}
	f.engine.Resolve = func(ctx context.Context, x workflow.Execution, r workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
		inv, auth, err := resolve(ctx, x, r)
		inv.Model = "fresh-selected-model"
		auth.Actor = "user:fresh-grant"
		return inv, auth, err
	}
	next, err := f.engine.Start(ctx, store, request)
	if err != nil || next.Status != workflow.RunCompleted || f.runner.spawns.Load() != 2 {
		t.Fatal("linked lifecycle", err, next.Status)
	}
	attempt := next.Stages[0].Tasks[0].Attempts[0]
	if attempt.Authority.Actor != "user:fresh-grant" || attempt.Invocation.Model != "fresh-selected-model" || attempt.AttemptID == source.Stages[0].Tasks[0].Attempts[0].AttemptID || next.Cancel != nil || next.Rejection != nil || len(next.Deliveries) != 0 || len(next.Messages) != 0 || len(next.Integrations) != 0 {
		t.Fatal("source state inherited")
	}
	if _, err = f.engine.Run(ctx, source.ProjectID, source.ExecutionID, RunOptions{}); !errors.Is(err, workflow.ErrTransition) || f.runner.spawns.Load() != 2 {
		t.Fatal("rejected source replayed", err)
	}
	after, err := store.Load(ctx, source.ProjectID, source.ExecutionID)
	if err != nil || !reflect.DeepEqual(source, after) {
		t.Fatal("source changed", err)
	}
}
