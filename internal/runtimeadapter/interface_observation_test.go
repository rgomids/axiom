package runtimeadapter

import (
	"context"
	"github.com/rgomids/axiom/internal/capabilityproof"
	"reflect"
	"strings"
	"testing"
)

type fakeInterface struct {
	identities []string
	calls      int
	args, envs [][]string
}

func (f *fakeInterface) Identity(context.Context, string) (string, error) {
	i := f.identities[f.calls]
	f.calls++
	return i, nil
}
func (f *fakeInterface) Output(_ context.Context, _ string, args, env []string, _ int) (string, error) {
	f.args = append(f.args, args)
	f.envs = append(f.envs, append([]string(nil), env...))
	if args[0] == "--version" {
		return "codex 0.159.1", nil
	}
	return "Usage: codex exec\nOptions:\n  --model M\n  --ignore-user-config\n  --ephemeral\n  --sandbox MODE\n  --add-dir DIR", nil
}
func TestInterfaceExactEnvironmentAndReplacement(t *testing.T) {
	skipPOSIXStagePaths(t)
	for _, ids := range [][]string{{"a", "a", "a"}, {"a", "b"}, {"a", "a", "b"}} {
		for i := range ids {
			ids[i] = strings.Repeat(ids[i], 64)
		}
		f := &fakeInterface{identities: ids}
		o, err := ObserveInterface(context.Background(), f, "/vendor/codex", "codex", capabilityproof.ReadOnlyVariant, []string{"HOME=/login", "PATH=/bin", "OPENAI_API_KEY=synthetic", "UNLISTED=value"})
		replaced := ids[len(ids)-1] != strings.Repeat("a", 64)
		if (err != nil) != replaced {
			t.Fatal(o, err)
		}
		for _, env := range f.envs {
			if !reflect.DeepEqual(env, []string{"HOME=/login", "PATH=/bin"}) {
				t.Fatal(env)
			}
		}
		if !replaced {
			for _, r := range o.Capabilities {
				if r.State != capabilityproof.Unknown {
					t.Fatal(o)
				}
			}
			if strings.Join(f.args[1], " ") != "exec --help" {
				t.Fatal(f.args)
			}
		}
	}
}
