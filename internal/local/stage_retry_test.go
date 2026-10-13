package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
)

// failLocalFirstAttempt persists one terminal first attempt of child 0 under a
// fresh lease, so the child becomes retry-eligible, and returns the saved graph.
func failLocalFirstAttempt(t *testing.T, graphs GraphStore, leases RunLeaseStore, projectID, parentID string) executiongraph.SuccessorGraph {
	t.Helper()
	ctx := context.Background()
	g, err := graphs.LoadSuccessor(ctx, projectID, parentID)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := leases.Acquire(ctx, projectID, parentID, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	started := g.Parent.CreatedAt.Add(time.Second)
	finished := started.Add(time.Second)
	next := g
	next.Children = append([]executiongraph.SuccessorChild(nil), g.Children...)
	next.Children[0].Attempts = []executiongraph.SuccessorAttempt{{AttemptID: "00000000-0000-4000-8000-000000000071", Number: 1, Status: executiongraph.AttemptFailed, StartedAt: &started, FinishedAt: &finished, ExitCode: 1, ClassificationReason: executiongraph.ProcessFailed, LeaseEpoch: lease.Epoch, Process: &executiongraph.ProcessCorrelation{PID: 42, StartIdentity: "synthetic-start", GroupIdentity: "synthetic-group"}, PacketDigest: strings.Repeat("b", 64), ReconciliationFacts: []executiongraph.ReconciliationFact{}}}
	saved, err := graphs.SaveSuccessor(ctx, next, lease)
	if err != nil {
		t.Fatal(err)
	}
	return saved
}

func localRetryAuthority(g executiongraph.SuccessorGraph) executiongraph.RetryAuthority {
	c := g.Children[0]
	disposition := executiongraph.DispositionNotRequired
	if executiongraph.RetryDispositionRequired(c) {
		disposition = executiongraph.DispositionPreserve
	}
	return executiongraph.RetryAuthority{FormatVersion: 1, ProjectID: c.Envelope.Scope.ProjectID, ParentID: g.Parent.ExecutionID, Revision: g.Parent.GraphRevision, GraphDigest: fmt.Sprintf("%x", g.StorageRevision), PreviewDigest: strings.Repeat("c", 64), DispatchDigest: strings.Repeat("d", 64), ChildID: c.ExecutionID, PriorAttemptID: c.Attempts[len(c.Attempts)-1].AttemptID, AttemptNumber: uint32(len(c.Attempts)) + 1, Effects: c.Envelope.AllowedEffects, Disposition: disposition, ActorKind: "human-operator", Actor: "operator", RecordedAt: time.Unix(20, 0).UTC()}
}

func committedLocalDispatch(t *testing.T) (string, GraphStore, RunLeaseStore, executiongraph.SuccessorGraph) {
	t.Helper()
	root := privateTestRoot(t)
	graphs, err := NewGraphStore(root)
	if err != nil {
		t.Fatal(err)
	}
	leases, err := NewRunLeaseStore(root)
	if err != nil {
		t.Fatal(err)
	}
	g, a := localDispatch(t)
	lease, err := leases.Acquire(context.Background(), a.ProjectID, a.ParentID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := graphs.CommitDispatch(context.Background(), g, a, a.PreviewDigest, lease); err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	return root, graphs, leases, failLocalFirstAttempt(t, graphs, leases, a.ProjectID, a.ParentID)
}

func TestRetryReceiptCommitLoadAndInventory(t *testing.T) {
	ctx := context.Background()
	root, graphs, _, g := committedLocalDispatch(t)
	a := localRetryAuthority(g)
	projectID := a.ProjectID
	ref, err := graphs.CommitRetry(ctx, g, a)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := executiongraph.RetryAuthorityDigest(a)
	if ref != digest {
		t.Fatalf("ref = %s, want %s", ref, digest)
	}
	// A repeated identical commit is an idempotent read-back.
	if again, err := graphs.CommitRetry(ctx, g, a); err != nil || again != ref {
		t.Fatalf("repeat = %s, %v", again, err)
	}
	loaded, err := graphs.LoadRetry(ctx, projectID, a.ParentID, ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := executiongraph.CheckRetryDispatch(g, loaded); err != nil {
		t.Fatalf("loaded receipt does not authorize its own attempt: %v", err)
	}
	// The graph itself is unchanged by the receipt commit.
	current, err := graphs.LoadSuccessor(ctx, projectID, a.ParentID)
	if err != nil || current.StorageRevision != g.StorageRevision {
		t.Fatalf("graph changed by retry receipt: %v", err)
	}
	inventory, err := InspectStateInventory(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range inventory.Entries {
		if strings.HasSuffix(entry.Relative, ref+".json") {
			found = entry.Kind == InventoryRetryAuthority
		}
		if !entry.Kind.Supported() {
			t.Errorf("%s classified %s", entry.Relative, entry.Kind)
		}
	}
	if !found {
		t.Fatal("retry receipt not inventoried as stage_retry_authority")
	}
}

func TestRetryReceiptRefusals(t *testing.T) {
	ctx := context.Background()
	t.Run("stale graph revision", func(t *testing.T) {
		_, graphs, _, g := committedLocalDispatch(t)
		a := localRetryAuthority(g)
		stale := g
		stale.StorageRevision[0] ^= 0xff
		a.GraphDigest = fmt.Sprintf("%x", stale.StorageRevision)
		if _, err := graphs.CommitRetry(ctx, stale, a); !errors.Is(err, ErrConflict) {
			t.Fatalf("stale commit = %v", err)
		}
	})
	t.Run("authority not matching graph", func(t *testing.T) {
		_, graphs, _, g := committedLocalDispatch(t)
		a := localRetryAuthority(g)
		a.AttemptNumber = 3
		if _, err := graphs.CommitRetry(ctx, g, a); err == nil {
			t.Fatal("receipt for a different attempt number committed")
		}
	})
	t.Run("load by foreign graph, tampered bytes and bad ref", func(t *testing.T) {
		root, graphs, _, g := committedLocalDispatch(t)
		a := localRetryAuthority(g)
		ref, err := graphs.CommitRetry(ctx, g, a)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := graphs.LoadRetry(ctx, a.ProjectID, "00000000-0000-4000-8000-0000000000ff", ref); err == nil {
			t.Fatal("receipt loaded for a foreign graph")
		}
		if _, err := graphs.LoadRetry(ctx, a.ProjectID, a.ParentID, "../"+ref); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("traversal ref = %v", err)
		}
		path := filepath.Join(root, "graphs", "v1", a.ProjectID, ref+".json")
		wire, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		tampered := strings.Replace(string(wire), `"actor":"operator"`, `"actor":"intruder"`, 1)
		if err := os.WriteFile(path, []byte(tampered), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := graphs.LoadRetry(ctx, a.ProjectID, a.ParentID, ref); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("tampered receipt = %v", err)
		}
		if kind := classifyRetryAuthority(a.ProjectID, ref+".json", []byte(tampered)); kind != InventoryMalformed {
			t.Fatalf("tampered receipt inventoried as %s", kind)
		}
	})
}
