package local

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/executiongraph"
)

func TestPreparedEnvelopeReadOnlyReopen(t *testing.T) {
	e := envelopeFixture(t)
	before, err := os.ReadDir(e.path)
	if err != nil {
		t.Fatal(err)
	}
	p, err := OpenPreparedEnvelope(e.policy, e.manifest.ProbeID, e.digest)
	if err != nil || p.InspectReady() != nil {
		t.Fatal("unused ready envelope refused", err)
	}
	after, err := os.ReadDir(e.path)
	if err != nil || len(before) != len(after) {
		t.Fatal("reopen wrote state", err)
	}
	if _, err := OpenPreparedEnvelope(e.policy, e.manifest.ProbeID, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong manifest accepted")
	}
	q, err := OpenQuarantinedEnvelope(e.policy, e.manifest.ProbeID, e.digest)
	if err != nil || q.InspectReady() == nil {
		t.Fatal("cleanup opener acquired execution handle", err)
	}
	preparation := capabilityproof.ProbePreparation{Manifest: p.Manifest(), ManifestDigest: p.digest, Invocation: executiongraph.Invocation{Stdin: []byte("synthetic-packet")}}
	s, err := (PreparedProbeEnvelopes{Envelope: p}).Open(context.Background(), preparation)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Quarantine(context.Background()) })
	if _, err := OpenPreparedEnvelope(e.policy, e.manifest.ProbeID, e.digest); err == nil {
		t.Fatal("consumed preparation reopened")
	}
	if _, err := (PreparedProbeEnvelopes{Envelope: e}).Open(context.Background(), preparation); err == nil {
		t.Fatal("second handle launched from the same preparation")
	}
	if p.InspectReady() != nil {
		t.Fatal("foreign handle revoked the active owner")
	}
}

func TestPreparedEnvelopeClaimIsExclusiveAcrossHandles(t *testing.T) {
	e := envelopeFixture(t)
	const handles = 8
	opened := make([]*Envelope, handles)
	for i := range opened {
		var err error
		opened[i], err = OpenPreparedEnvelope(e.policy, e.manifest.ProbeID, e.digest)
		if err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	results := make(chan bool, handles)
	for _, p := range opened {
		wg.Go(func() { results <- p.claim() == nil })
	}
	wg.Wait()
	close(results)
	claimed := 0
	for success := range results {
		if success {
			claimed++
		}
	}
	if claimed != 1 {
		t.Fatalf("claims=%d", claimed)
	}
	if _, err := OpenPreparedEnvelope(e.policy, e.manifest.ProbeID, e.digest); err == nil {
		t.Fatal("crash after claim reopened for dispatch")
	}
	q, err := OpenQuarantinedEnvelope(e.policy, e.manifest.ProbeID, e.digest)
	if err != nil {
		t.Fatal(err)
	}
	p, err := q.PreviewCleanup()
	if err != nil || p.Apply() != nil {
		t.Fatal("claimed envelope not recoverable", err)
	}
}

func TestQuarantinedEnvelopeCannotBeRearmed(t *testing.T) {
	e := envelopeFixture(t)
	if err := e.Quarantine(); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenPreparedEnvelope(e.policy, e.manifest.ProbeID, e.digest); err == nil {
		t.Fatal("quarantined envelope reopened")
	}
	q, err := OpenQuarantinedEnvelope(e.policy, e.manifest.ProbeID, e.digest)
	if err != nil {
		t.Fatal(err)
	}
	p, err := q.PreviewCleanup()
	if err != nil || p.Apply() != nil {
		t.Fatal("quarantine marker blocked exact cleanup", err)
	}
	if _, err := os.Lstat(filepath.Join(e.path, probeClaimName)); !os.IsNotExist(err) {
		t.Fatal("quarantine marker survived cleanup", err)
	}
}
