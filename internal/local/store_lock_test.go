//go:build linux || darwin

package local

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Readers that always overlap keep a shared flock held continuously; a writer
// in the same process must still get the exclusive lock within its wait
// instead of timing out with ErrConflict.
func TestLockRootsWriterIsNotStarvedByOverlappingReaders(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	var stop atomic.Bool
	var readers sync.WaitGroup
	for range 8 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for !stop.Load() {
				locks, err := lockRoots(false, root)
				if err != nil {
					continue
				}
				time.Sleep(5 * time.Millisecond)
				closeFiles(locks)
			}
		}()
	}
	defer func() {
		stop.Store(true)
		readers.Wait()
	}()
	time.Sleep(20 * time.Millisecond) // let the readers overlap
	for i := range 10 {
		locks, err := lockRoots(true, root)
		if err != nil {
			t.Fatalf("write %d starved: %v", i, err)
		}
		closeFiles(locks)
	}
}
