package runtimeadapter

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestI369PinnedParserBounds(t *testing.T) {
	arrays := func(n int) string { return "a=" + strings.Repeat("[", n) + "0" + strings.Repeat("]", n) }
	for _, c := range []struct {
		name, wire string
		refuse     bool
	}{
		{"array64", arrays(64), false}, {"array65", arrays(65), true}, {"array129", arrays(129), true}, {"array500000", arrays(500000), true},
		{"inline129", "a=" + strings.Repeat("{k=", 129) + "0" + strings.Repeat("}", 129), true},
		{"mixed129", "a=" + strings.Repeat("[{k=", 65) + "0" + strings.Repeat("}]", 65), true},
		{"dotted129", strings.Repeat("a.", 128) + "a=1", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			var v map[string]any
			d := toml.NewDecoder(strings.NewReader(c.wire))
			d.MaxDepth(64)
			_, err := d.Decode(&v)
			if (err != nil) != c.refuse {
				t.Fatal("parser bound changed", err)
			}
		})
	}
}

func TestI369TOMLSyntaxAndSelectors(t *testing.T) {
	for _, wire := range []string{
		"notify=[\"synthetic\",\"arg\"]\nx={a=1}\n",
		"notify=[\n\"x\",\n]\nx=\"\"\"line\nline\"\"\"",
		"[[items]]\nvalue=1\n[[items]]\nvalue=2",
		"text='model_reasoning_effort' # model_reasoning_effort\ndate=2026-11-04",
	} {
		if keys, err := tomlEffortSelectorsBounded([]byte(wire), true); err != nil || len(keys) != 0 {
			t.Fatal("unrelated valid TOML refused", err)
		}
	}
	for _, wire := range []string{
		"notify=['x']\nmodel_reasoning_effort='high'",
		"a={model_reasoning_effort='high'}",
		"a=[{model_reasoning_effort='high'}]",
		"[[items]]\nmodel_reasoning_effort='high'",
		"a.'model_reasoning_effort'='high'",
	} {
		if keys, err := tomlEffortSelectorsBounded([]byte(wire), true); err != nil || len(keys) != 1 {
			t.Fatalf("selector missed %s: %v", wire, err)
		}
	}
	for _, wire := range []string{
		"a=1\na=2", "[a]\n[a]", "a=1\n[a]", "a=[", string([]byte{'a', '=', 0xff}),
		"a=" + strings.Repeat("[", 33) + "0" + strings.Repeat("]", 33),
		"a=[" + strings.Repeat("0,", 65536) + "]",
	} {
		if _, err := tomlEffortSelectorsBounded([]byte(wire), true); err == nil {
			t.Fatal("unsafe/overbound TOML accepted")
		}
	}
}

func TestI369JoinedErrorBaseline(t *testing.T) {
	for _, err := range []error{errors.Join(ErrEffortConfigurationInvalid, ErrEffortConfigurationUnsafe), errors.Join(ErrEffortConfigurationUnsafe, ErrEffortConfigurationInvalid)} {
		if !errors.Is(err, ErrEffortConfigurationInvalid) || !errors.Is(err, ErrEffortConfigurationUnsafe) {
			t.Fatal("joined sentinel compatibility lost")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("cancellation baseline")
	}
}
