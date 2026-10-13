package capabilityproof

import (
	"context"
	"errors"
	"net/netip"
	"reflect"
	"strings"
	"testing"
)

type fakeCanary struct {
	touched                      []string
	deny, symlink, publishDenied bool
	network                      string
}

func (f *fakeCanary) Verify(context.Context, CanaryContext) error {
	if f.symlink {
		return errors.New("symlink")
	}
	return nil
}
func (f *fakeCanary) Identity(context.Context) (ProcessIdentity, error) {
	return ProcessIdentity{UID: 1, GID: 2, Process: "123"}, nil
}
func (f *fakeCanary) CreateEmptyExclusive(_ context.Context, target, name string) (bool, string, error) {
	f.touched = append(f.touched, target+"/"+name)
	if f.deny {
		return false, "EACCES", nil
	}
	return true, "", nil
}
func (f *fakeCanary) ConnectOwnedLoopback(context.Context, netip.AddrPort) (string, error) {
	return f.network, nil
}
func (f *fakeCanary) Publish(context.Context, CanaryReport) error {
	if f.publishDenied {
		return errors.New("denied")
	}
	f.touched = append(f.touched, "inbox/result")
	return nil
}
func canaryContext(variant string) CanaryContext {
	targets := []string{"sentinel"}
	allowed := []string{"workspace", "inbox"}
	if variant == ReadOnlyVariant {
		targets = append(targets, "workspace")
		allowed = []string{"inbox"}
	}
	nonce := strings.Repeat("a", 64)
	return CanaryContext{ProbeID: "probe-1", Nonce: nonce, ManifestDigest: strings.Repeat("b", 64), Loopback: netip.MustParseAddrPort("127.0.0.1:1234"), ListenerOwned: true, Manifest: EnvelopeManifest{Version: 1, Phase: "ready", ProbeID: "probe-1", Nonce: nonce, Variant: variant, NegativeTargets: targets, AllowedRoots: allowed, Sentinel: "sentinel"}}
}
func TestCanaryManifestTargetsAndRedaction(t *testing.T) {
	for _, v := range []string{ReadOnlyVariant, RepositoryWriteVariant} {
		for _, deny := range []bool{true, false} {
			c := canaryContext(v)
			f := &fakeCanary{deny: deny, network: "reached"}
			if deny {
				f.network = "denied"
			}
			r, err := RunCanary(context.Background(), c, f)
			if err != nil || !r.Published {
				t.Fatal(r, err)
			}
			want := []string{}
			for _, target := range c.Manifest.NegativeTargets {
				want = append(want, target+"/canary-"+c.Nonce)
			}
			want = append(want, "inbox/result")
			if !reflect.DeepEqual(f.touched, want) {
				t.Fatal(f.touched)
			}
		}
	}
}
func TestCanaryRefusesBeforeEffects(t *testing.T) {
	for _, name := range []string{"foreign-target", "missing-manifest", "foreign-probe", "external-host", "unowned-listener", "symlink"} {
		t.Run(name, func(t *testing.T) {
			c := canaryContext(ReadOnlyVariant)
			f := &fakeCanary{network: "reached"}
			switch name {
			case "foreign-target":
				c.Manifest.NegativeTargets = []string{"home"}
			case "missing-manifest":
				c.Manifest = EnvelopeManifest{}
			case "foreign-probe":
				c.ProbeID = "foreign"
			case "external-host":
				c.Loopback = netip.MustParseAddrPort("192.0.2.1:1234")
			case "unowned-listener":
				c.ListenerOwned = false
			case "symlink":
				f.symlink = true
			}
			if _, err := RunCanary(context.Background(), c, f); err == nil || len(f.touched) != 0 {
				t.Fatal(err, f.touched)
			}
		})
	}
	c := canaryContext(ReadOnlyVariant)
	f := &fakeCanary{network: "reached", publishDenied: true}
	r, err := RunCanary(context.Background(), c, f)
	if err == nil || r.Published {
		t.Fatal(r, err)
	}
}
