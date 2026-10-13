package main

import (
	"context"
	"encoding/json"
	"github.com/rgomids/axiom/internal/capabilityproof"
	"io"
)

// The hidden entrypoint accepts correlation, never host targets. T16 must
// provide the envelope loader and T22 the verified child-inbox effects.
type probeCanaryLoader interface {
	Load(context.Context, string, string) (capabilityproof.CanaryContext, capabilityproof.CanaryEffects, error)
}

func probeCanaryCommand(ctx context.Context, args []string, loader probeCanaryLoader, out io.Writer) (bool, int) {
	if len(args) == 0 || args[0] != "__probe-canary" {
		return false, 0
	}
	if len(args) != 3 || loader == nil {
		return true, 1
	}
	c, e, err := loader.Load(ctx, args[1], args[2])
	if err != nil || c.ProbeID != args[1] || c.ManifestDigest != args[2] {
		return true, 1
	}
	r, err := capabilityproof.RunCanary(ctx, c, e)
	if err != nil {
		return true, 1
	}
	if json.NewEncoder(out).Encode(r) != nil {
		return true, 1
	}
	return true, 0
}
