package capabilityproof

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// Paths are fake roots under t.TempDir; this adapter performs no filesystem IO.
type fakeExtendedEffects struct {
	controls, canaries, removals     int
	residue, changed, controlFailure bool
	names                            []string
}

func (f *fakeExtendedEffects) Inspect(_ context.Context, path string) (ObjectSnapshot, error) {
	kind := "tmpdir"
	if filepath.Base(path) == "system-temp" {
		kind = "system-temp"
	}
	inode := uint64(2)
	if f.changed {
		inode++
	}
	return ObjectSnapshot{Device: 1, Inode: inode, Owner: 1, Mode: 0700, ACLDigest: strings.Repeat("a", 64), ACLReadable: true, SymlinkFree: true, ResolvedTarget: kind}, nil
}
func (f *fakeExtendedEffects) Control(_ context.Context, root ExtendedRoot, name string) (Control, error) {
	f.controls++
	f.names = append(f.names, name)
	return Control{UID: 1, GID: 2, Created: true, Removed: true, RemovalVerified: !f.controlFailure, Snapshot: root.Snapshot}, nil
}
func (f *fakeExtendedEffects) Canary(_ context.Context, root ExtendedRoot, name string) (CanaryAttempt, ProcessIdentity, error) {
	f.canaries++
	f.names = append(f.names, name)
	return CanaryAttempt{Target: root.Kind, Errno: "EACCES"}, ProcessIdentity{UID: 1, GID: 2, Process: "fake-process"}, nil
}
func (f *fakeExtendedEffects) RemoveVerify(context.Context, ExtendedRoot, string) error {
	f.removals++
	if f.residue {
		return ErrProbeResidue
	}
	return nil
}
func extendedFixture(t *testing.T) (ExtendedService, *fakeExtendedEffects) {
	t.Helper()
	root := t.TempDir()
	effects := &fakeExtendedEffects{}
	return ExtendedService{Policy: ExtendedPolicy{TempDir: filepath.Join(root, "tmpdir"), SystemTemp: filepath.Join(root, "system-temp"), Home: filepath.Join(root, "fake-home"), VendorHomes: []string{filepath.Join(root, "fake-vendor-home")}}, Effects: effects}, effects
}
func TestExtendedAuthorizedFakeRoots(t *testing.T) {
	s, f := extendedFixture(t)
	p, err := s.Preview(context.Background(), "unique-probe", []string{"tmpdir", "system-temp"})
	if err != nil {
		t.Fatal(err)
	}
	a := ExtendedAuthority{Actor: "operator", Local: true, External: true, Preview: p}
	results, err := s.Apply(context.Background(), p, a)
	if err != nil || len(results) != 2 || f.canaries != 2 || f.controls != 4 || f.removals != 4 {
		t.Fatal(results, err)
	}
	for _, r := range results {
		if r.State != "denied" {
			t.Fatal(r)
		}
	}
	for _, name := range f.names {
		if name != p.Filename {
			t.Fatal("more than one unique filename")
		}
	}
}
func TestExtendedMissingAuthority(t *testing.T) {
	for _, missing := range []string{"local", "external", "actor", "preview"} {
		t.Run(missing, func(t *testing.T) {
			s, f := extendedFixture(t)
			p, err := s.Preview(context.Background(), "unique-probe", []string{"tmpdir"})
			if err != nil {
				t.Fatal(err)
			}
			a := ExtendedAuthority{Actor: "operator", Local: true, External: true, Preview: p}
			switch missing {
			case "local":
				a.Local = false
			case "external":
				a.External = false
			case "actor":
				a.Actor = ""
			case "preview":
				a.Preview = ExtendedPreview{}
			}
			if _, err := s.Apply(context.Background(), p, a); !errors.Is(err, ErrAuthority) || f.canaries != 0 || f.controls != 0 {
				t.Fatal(err)
			}
		})
	}
}
func TestExtendedResidue(t *testing.T) {
	s, f := extendedFixture(t)
	p, err := s.Preview(context.Background(), "unique-probe", []string{"tmpdir"})
	if err != nil {
		t.Fatal(err)
	}
	f.residue = true
	_, err = s.Apply(context.Background(), p, ExtendedAuthority{Actor: "operator", Local: true, External: true, Preview: p})
	if !errors.Is(err, ErrProbeResidue) || f.canaries != 1 || f.controls != 1 {
		t.Fatal(err)
	}
}
func TestExtendedForbiddenRoots(t *testing.T) {
	for _, kind := range []string{"home", "vendor-home", "external-network", "home-as-temp", "vendor-as-temp", "traversal", "duplicate"} {
		t.Run(kind, func(t *testing.T) {
			s, f := extendedFixture(t)
			kinds := []string{kind}
			switch kind {
			case "home-as-temp":
				s.Policy.TempDir = s.Policy.Home
				kinds = []string{"tmpdir"}
			case "vendor-as-temp":
				s.Policy.TempDir = s.Policy.VendorHomes[0]
				kinds = []string{"tmpdir"}
			case "traversal":
				s.Policy.TempDir = s.Policy.TempDir + string(filepath.Separator) + ".."
				kinds = []string{"tmpdir"}
			case "duplicate":
				kinds = []string{"tmpdir", "tmpdir"}
			}
			if _, err := s.Preview(context.Background(), "unique-probe", kinds); err == nil || f.controls != 0 || f.canaries != 0 {
				t.Fatal(err)
			}
		})
	}
}
func TestExtendedStaleRootAndControlFailure(t *testing.T) {
	for _, mode := range []string{"stale", "control"} {
		t.Run(mode, func(t *testing.T) {
			s, f := extendedFixture(t)
			p, err := s.Preview(context.Background(), "unique-probe", []string{"tmpdir"})
			if err != nil {
				t.Fatal(err)
			}
			if mode == "stale" {
				f.changed = true
			} else {
				f.controlFailure = true
			}
			if _, err = s.Apply(context.Background(), p, ExtendedAuthority{Actor: "operator", Local: true, External: true, Preview: p}); err == nil || f.canaries != 0 {
				t.Fatal(err)
			}
		})
	}
}

func TestExtendedPreviewDigestAndMutationRefusals(t *testing.T) {
	changes := []struct {
		name   string
		change func(*ExtendedPreview)
	}{
		{"digest", func(p *ExtendedPreview) { p.Digest = strings.Repeat("b", 64) }},
		{"probe", func(p *ExtendedPreview) { p.ProbeID = "another-probe" }},
		{"filename", func(p *ExtendedPreview) { p.Filename += "-other" }},
		{"limit", func(p *ExtendedPreview) { p.Limit++ }},
		{"timeout", func(p *ExtendedPreview) { p.Timeout++ }},
		{"path", func(p *ExtendedPreview) { p.Roots[0].Path += "-other" }},
		{"kind", func(p *ExtendedPreview) { p.Roots[0].Kind = "system-temp" }},
		{"device", func(p *ExtendedPreview) { p.Roots[0].Snapshot.Device++ }},
		{"inode", func(p *ExtendedPreview) { p.Roots[0].Snapshot.Inode++ }},
		{"owner", func(p *ExtendedPreview) { p.Roots[0].Snapshot.Owner++ }},
		{"mode", func(p *ExtendedPreview) { p.Roots[0].Snapshot.Mode++ }},
		{"acl", func(p *ExtendedPreview) { p.Roots[0].Snapshot.ACLDigest = strings.Repeat("c", 64) }},
		{"unreadable acl", func(p *ExtendedPreview) { p.Roots[0].Snapshot.ACLReadable = false }},
		{"read-only", func(p *ExtendedPreview) { p.Roots[0].Snapshot.ReadOnly = true }},
		{"symlink", func(p *ExtendedPreview) { p.Roots[0].Snapshot.SymlinkFree = false }},
		{"target", func(p *ExtendedPreview) { p.Roots[0].Snapshot.ResolvedTarget = "home" }},
		{"root set", func(p *ExtendedPreview) { p.Roots = p.Roots[:1] }},
	}
	for _, tc := range changes {
		t.Run(tc.name, func(t *testing.T) {
			s, effects := extendedFixture(t)
			p, err := s.Preview(context.Background(), "unique-probe", []string{"tmpdir", "system-temp"})
			if err != nil || !digest.MatchString(p.Digest) {
				t.Fatalf("preview=%+v err=%v", p, err)
			}
			repeated, err := s.Preview(context.Background(), "unique-probe", []string{"tmpdir", "system-temp"})
			if err != nil || p.Digest != repeated.Digest {
				t.Fatal("preview digest is not deterministic", err)
			}
			if effects.controls != 0 || effects.canaries != 0 || effects.removals != 0 {
				t.Fatal("preview performed effects")
			}
			tc.change(&p)
			if _, err = s.Apply(context.Background(), p, ExtendedAuthority{Actor: "operator", Local: true, External: true, Preview: p}); !errors.Is(err, ErrStalePreview) {
				t.Fatal("changed preview accepted", err)
			}
			if effects.controls != 0 || effects.canaries != 0 || effects.removals != 0 {
				t.Fatal("refusal performed effects")
			}
		})
	}
}

func TestExtendedPreviewDigestTracksCurrentIdentity(t *testing.T) {
	s, effects := extendedFixture(t)
	p, err := s.Preview(context.Background(), "unique-probe", []string{"tmpdir"})
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"probe", "path", "identity", "root set"} {
		t.Run(change, func(t *testing.T) {
			changed := s
			changedEffects := *effects
			changed.Effects = &changedEffects
			id, kinds := "unique-probe", []string{"tmpdir"}
			switch change {
			case "probe":
				id = "another-probe"
			case "path":
				changed.Policy.TempDir += "-other"
			case "identity":
				changedEffects.changed = true
			case "root set":
				kinds = append(kinds, "system-temp")
			}
			fresh, err := changed.Preview(context.Background(), id, kinds)
			if err != nil || !digest.MatchString(fresh.Digest) || fresh.Digest == p.Digest {
				t.Fatalf("digest did not bind %s: %+v, %v", change, fresh, err)
			}
			if changedEffects.controls != 0 || changedEffects.canaries != 0 || changedEffects.removals != 0 {
				t.Fatal("preview performed effects")
			}
		})
	}
}
