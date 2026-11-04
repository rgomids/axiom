//go:build linux || darwin

package stageexecution

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestI369InspectionDoesNotMaterialize(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			l, r := runtimeLauncherFixture(t, id, "runtime-default")
			before, err := os.ReadDir(l.InboxRoot)
			if err != nil {
				t.Fatal(err)
			}
			first, err := l.Inspect(context.Background(), r)
			if err != nil {
				t.Fatal(err)
			}
			after, err := os.ReadDir(l.InboxRoot)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("inspection wrote")
			}
			built, err := l.Build(context.Background(), r)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(first, built) {
				t.Fatal("inspection/build drift")
			}
			// A missing planned parent stays absent.
			l.InboxRoot = filepath.Join(l.InboxRoot, "planned", "missing")
			if _, err = l.Inspect(context.Background(), r); err != nil {
				t.Fatal(err)
			}
			if _, err = os.Lstat(l.InboxRoot); !os.IsNotExist(err) {
				t.Fatal("planned parent created")
			}
		})
	}
}
