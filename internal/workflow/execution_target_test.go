package workflow

import (
	"context"
	"github.com/rgomids/axiom/internal/provenance"
	"testing"
)

type linkedTargetResolver struct{}

func (linkedTargetResolver) Resolve(context.Context, string) (Project, string) {
	return Project{ID: "project", Repositories: []Repository{{Key: "main", Path: "/fixture"}}}, ""
}

type linkedTargetItems struct{}

func (linkedTargetItems) Load(context.Context, string, string, string, string, string) (WorkItem, error) {
	return WorkItem{Provider: "github", Resource: "owner/repo", ExternalID: "7", URL: "https://github.com/owner/repo/issues/7", State: "OPEN"}, nil
}
func linkedTargetService(t *testing.T) Service {
	t.Helper()
	source, err := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: "fixture", SourceState: provenance.Clean}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return New(linkedTargetResolver{}, linkedTargetItems{}, source)
}
func TestLinkedExecutionTargetRejectsMalformedAndMismatchedSelectors(t *testing.T) {
	for _, selector := range []string{"", "0", "-1", "07", "word", "github:owner/repo#7"} {
		t.Run(selector, func(t *testing.T) {
			_, failed := linkedTargetService(t).ResolveLinked(t.Context(), Target{ProjectSelector: "sample", RepositoryKey: "main", WorkItem: selector})
			if failed.Status != ValidationFailed {
				t.Fatalf("accepted invalid selector: %+v", failed)
			}
		})
	}
	for name, mutate := range map[string]func(*Target){
		"wrong item":         func(t *Target) { t.WorkItem = "8" },
		"wrong provider":     func(t *Target) { t.WorkItemProvider = "other" },
		"wrong resource":     func(t *Target) { t.WorkItemResource = "other/repo" },
		"wrong repository":   func(t *Target) { t.RepositoryKey = "other" },
		"existing execution": func(t *Target) { t.ExecutionID = "other" },
	} {
		t.Run(name, func(t *testing.T) {
			target := Target{ProjectSelector: "sample", RepositoryKey: "main", WorkItem: "7"}
			mutate(&target)
			_, failed := linkedTargetService(t).ResolveLinked(t.Context(), target)
			if failed.Status != ValidationFailed {
				t.Fatalf("accepted mismatched linkage: %+v", failed)
			}
		})
	}
}
func TestLinkedExecutionTargetResolvesWithoutWorkflowAndHonorsCancellation(t *testing.T) {
	service := linkedTargetService(t)
	target := Target{ProjectSelector: "sample", RepositoryKey: "main", WorkItem: "7"}
	resolved, failed := service.ResolveLinked(t.Context(), target)
	if failed.Category != "" || resolved.ProjectID != "project" || resolved.Repository.Key != "main" || resolved.WorkItem.ExternalID != "7" {
		t.Fatal(resolved, failed)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, failed = service.ResolveLinked(ctx, target); failed.Status != Interrupted {
		t.Fatal(failed)
	}
}
