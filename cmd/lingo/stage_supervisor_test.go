package main

import (
	"context"
	"testing"
)

func TestHiddenSupervisorShellInvocationFailsClosed(t *testing.T) {
	for _, args := range [][]string{{"__stage-supervisor"}, {"__stage-supervisor", "project", "parent", "preview", "authority"}, {"__stage-supervisor", "project", "foreign", "preview", "forged"}, {"__stage-supervisor", "project", "parent", "preview", "authority", "--authorize-local"}} {
		if handled, code := stageSupervisorCommand(context.Background(), args, nil); !handled || code == 0 {
			t.Fatal("shell admitted", args)
		}
	}
	if handled, _ := stageSupervisorCommand(context.Background(), []string{"workflow", "stage", "status"}, nil); handled {
		t.Fatal("intercepted public command")
	}
}
