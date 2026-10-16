package executiongraph

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

type fakeProcessContainment struct {
	alive, escaped      bool
	stopErr, observeErr error
	descendants         int
}

func (*fakeProcessContainment) start(*exec.Cmd) error { return nil }
func (g *fakeProcessContainment) stop() error {
	if g.stopErr != nil {
		return g.stopErr
	}
	g.alive = false
	g.descendants = 0
	return nil
}
func (g *fakeProcessContainment) gone() (bool, error) {
	return !g.alive && g.descendants == 0 && !g.escaped, g.observeErr
}
func TestConfirmedProcessGroupStop(t *testing.T) {
	for _, tc := range []struct {
		name  string
		group fakeProcessContainment
		want  bool
	}{
		{"descendants", fakeProcessContainment{alive: true, descendants: 3}, true},
		{"escaped descendant", fakeProcessContainment{alive: true, descendants: 3, escaped: true}, false},
		{"already gone", fakeProcessContainment{}, true},
		{"permission denied", fakeProcessContainment{alive: true, stopErr: errors.New("permission denied")}, false},
		{"observation denied", fakeProcessContainment{observeErr: errors.New("permission denied")}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if got := confirmProcessStop(ctx, &tc.group); got != tc.want {
				t.Fatalf("stopped=%v want %v", got, tc.want)
			}
		})
	}
}
