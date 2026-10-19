package runtimeadapter

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestOperationalIdentityReobservesBytesWithoutAuthOrCapabilityClaims(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		fixture := newAuthFixture(t, runtimeID)
		observer, err := NewExecutableObserver(7, time.Unix(1, 0), []ObservedRuntime{{ID: runtimeID, Executable: fixture.executable}})
		if err != nil {
			t.Fatal(err)
		}
		before, err := observer.ObserveOperational(context.Background(), runtimeID)
		if err != nil || !before.Installed || !before.Available || before.ExecutableDigest == "" || before.SubscriptionObserved || before.EffortSourceDigest != "" || before.Revision != 7 {
			t.Fatal("identity observation asserted more than executable availability", before, err)
		}
		if err := os.WriteFile(fixture.executable, []byte("different synthetic bytes; never execute"), 0o700); err != nil {
			t.Fatal(err)
		}
		after, err := observer.ObserveOperational(context.Background(), runtimeID)
		if err != nil || before.ExecutableDigest == after.ExecutableDigest {
			t.Fatal("identity observation was cached", err)
		}
		if err := os.Remove(fixture.executable); err != nil {
			t.Fatal(err)
		}
		missing, err := observer.ObserveOperational(context.Background(), runtimeID)
		if err != nil || missing.Installed || missing.Available || missing.ExecutableDigest != "" {
			t.Fatal("missing executable retained identity", missing, err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := observer.ObserveOperational(ctx, runtimeID); err == nil {
			t.Fatal("cancelled observation succeeded")
		}
	}
}
