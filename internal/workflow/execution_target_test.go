package workflow

import (
	"context"
	"reflect"
	"testing"
)

func TestExecutionTargetRejectsMalformedAndMismatchedSelectors(t *testing.T) {
	for _, selector := range []string{"", "0", "-1", "07", "word", "github:owner/repo#7"} {
		t.Run(selector, func(t *testing.T) {
			service, store, provider := newS4Service(t)
			target := s4Target()
			target.WorkItem = selector
			got := service.Start(context.Background(), target)
			if got.Status != ValidationFailed || store.created || store.saves != 0 || len(provider.effects) != 0 {
				t.Fatalf("accepted invalid target: %+v", got)
			}
		})
	}
	for name, mutate := range map[string]func(*Target){
		"wrong item":     func(t *Target) { t.WorkItem = "8" },
		"wrong provider": func(t *Target) { t.WorkItemProvider = "other" },
		"wrong resource": func(t *Target) { t.WorkItemResource = "other/repo" },
	} {
		t.Run(name, func(t *testing.T) {
			service, store, _ := newS4Service(t)
			target := s4Target()
			mutate(&target)
			before := store.state
			got := service.Start(context.Background(), target)
			if got.Status != ValidationFailed || store.created || !reflect.DeepEqual(before, store.state) {
				t.Fatalf("accepted mismatched linkage: %+v", got)
			}
		})
	}
}

type changingTargetItems struct{ item WorkItem }

func (i *changingTargetItems) Load(context.Context, string, string, string, string, string) (WorkItem, error) {
	return i.item, nil
}

type changingTargetResolver struct{ project Project }

func (r *changingTargetResolver) Resolve(context.Context, string) (Project, string) {
	return r.project, ""
}

func TestReviewedExecutionTargetRevalidatedBeforeAllocation(t *testing.T) {
	for _, drift := range []string{"project", "repository path", "item state", "item URL"} {
		t.Run(drift, func(t *testing.T) {
			service, store, provider := newS4Service(t)
			target := s4Target()
			resolved, failed := service.ValidateTarget(context.Background(), target)
			if failed.Category != "" {
				t.Fatal(failed)
			}
			target.ReviewedTarget = &resolved
			resolver := &changingTargetResolver{Project{ID: resolved.ProjectID, Repositories: []Repository{resolved.Repository}}}
			items := &changingTargetItems{resolved.WorkItem}
			service.resolver = resolver
			service.workItems = items
			switch drift {
			case "project":
				resolver.project.ID = "123e4567-e89b-42d3-a456-426614174001"
			case "repository path":
				resolver.project.Repositories[0].Path = "/tmp/changed"
			case "item state":
				items.item.State = "CLOSED"
			case "item URL":
				items.item.URL = "https://github.com/owner/repo/issues/7?changed"
			}
			allocations := 0
			service.allocateID = func() (string, error) { allocations++; return "", nil }
			got := service.Start(context.Background(), target)
			if got.Category != "stale_execution_target" || allocations != 0 || store.created || store.saves != 0 || len(provider.effects) != 0 {
				t.Fatalf("drift effects: %+v", got)
			}
		})
	}
}

func TestExecutionIdentityConflictsPreservePersistedState(t *testing.T) {
	for name, change := range map[string]func(*Target, *Service){
		"runtime":   func(v *Target, _ *Service) { v.RuntimeID = "claude" },
		"execution": func(v *Target, _ *Service) { v.ExecutionID = "other" },
		"project": func(_ *Target, s *Service) {
			s.resolver = &changingTargetResolver{Project{ID: "123e4567-e89b-42d3-a456-426614174001", Repositories: []Repository{{Key: "main", Path: "/tmp/repository"}}}}
		},
		"repository": func(v *Target, s *Service) {
			v.RepositoryKey = "other"
			s.resolver = &changingTargetResolver{Project{ID: "123e4567-e89b-42d3-a456-426614174000", Repositories: []Repository{{Key: "other", Path: "/tmp/repository"}}}}
		},
		"item": func(v *Target, s *Service) {
			v.WorkItem = "8"
			s.workItems = &changingTargetItems{WorkItem{Provider: "github", Resource: "owner/repo", ExternalID: "8", URL: "https://github.com/owner/repo/issues/8", State: "OPEN"}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			service, store, provider := newS4Service(t)
			target := s4Target()
			started := service.Start(context.Background(), target)
			if started.Category != "execution_started" {
				t.Fatal(started)
			}
			before := cloneState(store.state)
			change(&target, &service)
			got := service.Start(context.Background(), target)
			if got.Status != ValidationFailed || !reflect.DeepEqual(before, store.state) || store.saves != 0 || len(provider.effects) != 0 {
				t.Fatalf("scope conflict mutated persisted state: %+v", got)
			}
		})
	}
}

type recoveringTargetStore struct{ Store }

func (s recoveringTargetStore) Load(context.Context, string, string, WorkItem) (State, error) {
	return State{}, ErrRecoveryRequired
}

func TestExecutionStartCancellationAndRecoveryPreserveState(t *testing.T) {
	for _, failure := range []string{"cancellation", "recovery"} {
		t.Run(failure, func(t *testing.T) {
			service, store, provider := newS4Service(t)
			target := s4Target()
			started := service.Start(context.Background(), target)
			if started.Category != "execution_started" {
				t.Fatal(started)
			}
			before := cloneState(store.state)
			ctx := context.Background()
			want := "recovery_required"
			if failure == "cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
				want = "workflow_cancelled"
			} else {
				service.store = recoveringTargetStore{store}
			}
			got := service.Start(ctx, target)
			if got.Category != want || !reflect.DeepEqual(before, store.state) || store.saves != 0 || len(provider.effects) != 0 {
				t.Fatalf("failure changed persisted state: %+v", got)
			}
		})
	}
}
