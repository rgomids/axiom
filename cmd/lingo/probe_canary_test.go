package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/rgomids/axiom/internal/capabilityproof"
	"net/netip"
	"strings"
	"testing"
)

type rejectedCanaryLoader struct{}

func (rejectedCanaryLoader) Load(context.Context, string, string) (capabilityproof.CanaryContext, capabilityproof.CanaryEffects, error) {
	return capabilityproof.CanaryContext{}, nil, errors.New("missing manifest")
}
func TestHiddenCanaryRequiresProbeContext(t *testing.T) {
	for _, args := range [][]string{{"__probe-canary"}, {"__probe-canary", "id", "digest"}, {"__probe-canary", "id", "digest", "/foreign"}} {
		var out bytes.Buffer
		handled, code := probeCanaryCommand(context.Background(), args, nil, &out)
		if !handled || code == 0 || out.Len() != 0 {
			t.Fatal(handled, code, out.String())
		}
	}
	var out bytes.Buffer
	_, code := probeCanaryCommand(context.Background(), []string{"__probe-canary", "id", "digest"}, rejectedCanaryLoader{}, &out)
	if code == 0 {
		t.Fatal("missing manifest accepted")
	}
	handled, _ := probeCanaryCommand(context.Background(), []string{"version"}, nil, &out)
	if handled {
		t.Fatal("intercepted public command")
	}
}

type fakeCanaryLoader struct {
	c capabilityproof.CanaryContext
	e capabilityproof.CanaryEffects
}

func (f fakeCanaryLoader) Load(context.Context, string, string) (capabilityproof.CanaryContext, capabilityproof.CanaryEffects, error) {
	return f.c, f.e, nil
}

type commandCanaryEffects struct{ writes []string }

func (f *commandCanaryEffects) Verify(context.Context, capabilityproof.CanaryContext) error {
	return nil
}
func (f *commandCanaryEffects) Identity(context.Context) (capabilityproof.ProcessIdentity, error) {
	return capabilityproof.ProcessIdentity{Process: "123", UID: 1, GID: 2}, nil
}
func (f *commandCanaryEffects) CreateEmptyExclusive(_ context.Context, target, name string) (bool, string, error) {
	f.writes = append(f.writes, target)
	return false, "EPERM", nil
}
func (f *commandCanaryEffects) ConnectOwnedLoopback(context.Context, netip.AddrPort) (string, error) {
	return "denied", nil
}
func (f *commandCanaryEffects) Publish(context.Context, capabilityproof.CanaryReport) error {
	f.writes = append(f.writes, "inbox")
	return nil
}
func TestHiddenCanaryWithFakeProbeContext(t *testing.T) {
	nonce := strings.Repeat("a", 64)
	d := strings.Repeat("b", 64)
	c := capabilityproof.CanaryContext{ProbeID: "probe-1", Nonce: nonce, ManifestDigest: d, ListenerOwned: true, Loopback: netip.MustParseAddrPort("127.0.0.1:1234"), Manifest: capabilityproof.EnvelopeManifest{ProbeID: "probe-1", Nonce: nonce, Version: 1, Phase: "ready", Variant: capabilityproof.ReadOnlyVariant, AllowedRoots: []string{"inbox"}, NegativeTargets: []string{"sentinel", "workspace"}, Sentinel: "sentinel"}}
	f := &commandCanaryEffects{}
	var out bytes.Buffer
	handled, code := probeCanaryCommand(context.Background(), []string{"__probe-canary", "probe-1", d}, fakeCanaryLoader{c: c, e: f}, &out)
	if !handled || code != 0 || len(f.writes) != 3 || !strings.Contains(out.String(), "EPERM") {
		t.Fatal(handled, code, f.writes, out.String())
	}
}
