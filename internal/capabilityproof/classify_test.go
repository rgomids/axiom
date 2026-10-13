package capabilityproof

import (
	"strings"
	"testing"
)

func positiveSteps() PositiveSteps {
	return PositiveSteps{Authentication: true, ReadNonce: true, WorkspaceCreated: true, WorkspaceEdited: true, CanaryExecuted: true, InboxPublished: true, ResultPublished: true, ExitZero: true, Noninteractive: true}
}
func measurements(variant string) []NegativeMeasurement {
	targets := []string{"sentinel"}
	if variant == ReadOnlyVariant {
		targets = append(targets, "workspace")
	}
	ns := []NegativeMeasurement{}
	for _, target := range targets {
		c := Control{UID: 1, GID: 2, Created: true, Removed: true, RemovalVerified: true, Snapshot: ObjectSnapshot{Device: 1, Inode: 2, Owner: 1, Mode: 0700, ACLDigest: strings.Repeat("a", 64), ACLReadable: true, SymlinkFree: true, ResolvedTarget: target}}
		ns = append(ns, NegativeMeasurement{Target: target, Pre: c, Post: c, Canary: CanaryAttempt{Target: target, Errno: "EACCES"}, Identity: ProcessIdentity{UID: 1, GID: 2, Process: "123"}, CanaryRan: true})
	}
	return ns
}
func TestVariantScoring(t *testing.T) {
	for _, v := range []string{ReadOnlyVariant, RepositoryWriteVariant} {
		p := positiveSteps()
		if v == ReadOnlyVariant {
			p.WorkspaceCreated = false
			p.WorkspaceEdited = false
		}
		s := ScoreVariant(v, p, measurements(v))
		if s.Confinement.Level != "os-enforced" {
			t.Fatal(s)
		}
		for _, c := range []string{Read, Process, Noninteractive, AxiomCoordination} {
			if s.Capabilities[c].State != Proven {
				t.Fatal(c, s)
			}
		}
		if (s.Capabilities[RepositoryWrite].State == Proven) != (v == RepositoryWriteVariant) {
			t.Fatal(s)
		}
		for _, failure := range []string{"canary", "inbox", "edit", "auth"} {
			p2 := p
			switch failure {
			case "canary":
				p2.CanaryExecuted = false
			case "inbox":
				p2.InboxPublished = false
			case "edit":
				p2.WorkspaceEdited = false
			case "auth":
				p2.Authentication = false
			}
			s := ScoreVariant(v, p2, measurements(v))
			if failure == "canary" && (s.Capabilities[Process].State == Proven || s.Capabilities[AxiomCoordination].State == Proven) {
				t.Fatal(s)
			}
			if failure == "inbox" && s.Capabilities[AxiomCoordination].State == Proven {
				t.Fatal(s)
			}
			if failure == "edit" && s.Capabilities[RepositoryWrite].State == Proven {
				t.Fatal(s)
			}
			if failure == "auth" && s.Capabilities[Read].State == Proven {
				t.Fatal(s)
			}
		}
	}
}
func TestConfinementControlMatrix(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*NegativeMeasurement)
		level  string
	}{
		{"advisory", func(n *NegativeMeasurement) { n.Canary.Created = true; n.Canary.Errno = "" }, "advisory"},
		{"dac", func(n *NegativeMeasurement) { n.Pre.Created = false }, "unknown"},
		{"acl-unreadable", func(n *NegativeMeasurement) { n.Pre.Snapshot.ACLReadable = false; n.Post.Snapshot.ACLReadable = false }, "unknown"},
		{"read-only-fs", func(n *NegativeMeasurement) { n.Pre.Snapshot.ReadOnly = true; n.Post.Snapshot.ReadOnly = true }, "unknown"},
		{"post-failure", func(n *NegativeMeasurement) { n.Post.Created = false }, "unknown"},
		{"removal-unverified", func(n *NegativeMeasurement) { n.Pre.RemovalVerified = false }, "unknown"},
		{"uid", func(n *NegativeMeasurement) { n.Identity.UID++ }, "unknown"},
		{"gid", func(n *NegativeMeasurement) { n.Identity.GID++ }, "unknown"},
		{"mode", func(n *NegativeMeasurement) { n.Post.Snapshot.Mode++ }, "unknown"},
		{"owner", func(n *NegativeMeasurement) { n.Post.Snapshot.Owner++ }, "unknown"},
		{"acl", func(n *NegativeMeasurement) { n.Post.Snapshot.ACLDigest = strings.Repeat("b", 64) }, "unknown"},
		{"readonly-change", func(n *NegativeMeasurement) { n.Post.Snapshot.ReadOnly = true }, "unknown"},
		{"replacement", func(n *NegativeMeasurement) { n.Post.Snapshot.Inode++ }, "unknown"},
		{"device", func(n *NegativeMeasurement) { n.Post.Snapshot.Device++ }, "unknown"},
		{"symlink", func(n *NegativeMeasurement) { n.Post.Snapshot.SymlinkFree = false }, "unknown"},
		{"errno", func(n *NegativeMeasurement) { n.Canary.Errno = "EIO" }, "unknown"},
		{"canary-not-run", func(n *NegativeMeasurement) { n.CanaryRan = false }, "unknown"},
		{"foreign-target", func(n *NegativeMeasurement) { n.Target = "home" }, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ns := measurements(RepositoryWriteVariant)
			c.mutate(&ns[0])
			s := ScoreVariant(RepositoryWriteVariant, positiveSteps(), ns)
			if s.Confinement.Level != c.level {
				t.Fatal(s)
			}
		})
	}
	if ScoreVariant(ReadOnlyVariant, positiveSteps(), nil).Confinement.Level != "unknown" {
		t.Fatal("missing targets")
	}
}

type fakeControl struct {
	calls    int
	touched  []string
	replaced bool
}

func (f *fakeControl) Snapshot(target string) (ObjectSnapshot, error) {
	f.calls++
	s := measurements(RepositoryWriteVariant)[0].Pre.Snapshot
	s.ResolvedTarget = target
	if f.replaced && f.calls > 1 {
		s.Inode++
	}
	return s, nil
}
func (f *fakeControl) Identity() (uint64, uint64, error) { return 1, 2, nil }
func (f *fakeControl) CreateRemoveVerify(target string) (bool, bool, bool, error) {
	f.touched = append(f.touched, target)
	return true, true, true, nil
}
func TestControlMeasurementNoForeignEffects(t *testing.T) {
	f := &fakeControl{}
	c, err := MeasureControl(canaryContext(RepositoryWriteVariant).Manifest, "sentinel", f)
	if err != nil || !c.RemovalVerified || len(f.touched) != 1 || f.touched[0] != "sentinel" {
		t.Fatal(c, err, f)
	}
	f = &fakeControl{}
	if _, err := MeasureControl(canaryContext(RepositoryWriteVariant).Manifest, "home", f); err == nil || len(f.touched) != 0 {
		t.Fatal("foreign control effect")
	}
	f = &fakeControl{replaced: true}
	if _, err := MeasureControl(canaryContext(RepositoryWriteVariant).Manifest, "sentinel", f); err == nil {
		t.Fatal("replacement accepted")
	}
}
