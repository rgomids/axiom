package executiongraph

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/strictjson"
)

func decodeProposalV3(wire []byte) (Proposal, error) {
	var p Proposal
	if strictjson.Decode(wire, &p, MaxGraphBytes) != nil || p.FormatVersion != OperationalProposalVersion || validateProposalV3(p) != nil {
		return Proposal{}, ErrInvalidGraph
	}
	return p, nil
}

func validateProposalV3(p Proposal) error {
	digest, err := signProposalV3(p)
	if err != nil || digest != p.Digest {
		return ErrInvalidGraph
	}
	return nil
}

func signProposalV3(p Proposal) (string, error) {
	if p.FormatVersion != OperationalProposalVersion || NewPlanner(&capabilitySpy{}).Validate(context.Background(), p, maxNodes) != nil {
		return "", ErrInvalidGraph
	}
	return proposalDigest(p)
}

func encodeProposalV3(p Proposal) ([]byte, error) {
	if validateProposalV3(p) != nil {
		return nil, errors.New("invalid")
	}
	wire, err := json.Marshal(p)
	return append(wire, '\n'), err
}

func operationalProposalFixture(t *testing.T) Proposal {
	t.Helper()
	p, err := decodeProposalV3(mustRead(t, "testdata/axm36/proposal-v3.json"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOperationalProposalGolden(t *testing.T) {
	p := operationalProposalFixture(t)
	wire, err := encodeProposalV3(p)
	if err != nil || !bytes.Equal(wire, mustRead(t, "testdata/axm36/proposal-v3.json")) {
		t.Fatal("unstable proposal bytes", err)
	}
	want := fmt.Sprintf("proposal %s\nwire %x\n", p.Digest, sha256.Sum256(wire))
	if want != string(mustRead(t, "testdata/axm36/proposal-v3.digests")) {
		t.Fatal("proposal digest changed")
	}
	old := p
	old.FormatVersion = 2
	if NewPlanner(&capabilitySpy{}).Validate(context.Background(), old, maxNodes) == nil {
		t.Fatal("new tuple accepted as historical Proposal 2")
	}
	for _, retired := range []string{"capabilityProofReference", "runtimeVersion", `"resolvedProfile"`} {
		if bytes.Contains(wire, []byte(retired)) {
			t.Fatalf("retired field %s", retired)
		}
	}
}

func TestOperationalProposalDigestAndTopology(t *testing.T) {
	for name, mutate := range map[string]func(*Proposal){
		"future": func(p *Proposal) { p.FormatVersion++ },
		"node limit": func(p *Proposal) {
			for len(p.Nodes) <= maxNodes {
				p.Nodes = append(p.Nodes, p.Nodes[0])
			}
		},
		"duplicate":          func(p *Proposal) { p.Nodes[1].Key = p.Nodes[0].Key },
		"missing dependency": func(p *Proposal) { p.Nodes[0].Dependencies = []string{"absent"} },
		"cycle":              func(p *Proposal) { p.Nodes[0].Dependencies = []string{p.Nodes[1].Key} },
		"integration": func(p *Proposal) {
			for i := range p.Nodes {
				p.Nodes[i].IntegrationOwner = false
			}
		},
		"validation owner": func(p *Proposal) {
			for i := range p.Nodes {
				p.Nodes[i].ValidationOwner = false
			}
		},
		"unsafe path":   func(p *Proposal) { p.Nodes[0].Scope.Paths = []string{"../escape"} },
		"effort":        func(p *Proposal) { p.Nodes[0].Controls.ReasoningEffort = "arbitrary" },
		"invalid tuple": func(p *Proposal) { p.Nodes[0].FrozenProfile.EffortSourceDigest = "bad" },
	} {
		t.Run(name, func(t *testing.T) {
			p := operationalProposalFixture(t)
			mutate(&p)
			if _, err := signProposalV3(p); err == nil {
				t.Fatal("unsafe topology can be signed")
			}
		})
	}
	p := operationalProposalFixture(t)
	p.Nodes[0].FrozenProfile.ExecutableDigest = strings.Repeat("f", 64)
	if _, err := encodeProposalV3(p); err == nil {
		t.Fatal("stale digest accepted after identity drift")
	}
	digest, err := signProposalV3(p)
	if err != nil || digest == p.Digest {
		t.Fatal("identity drift did not change digest")
	}
	p.Digest = digest
	if _, err := encodeProposalV3(p); err != nil {
		t.Fatal("fresh proposal refused", err)
	}
}

func TestOperationalProposalClosedSchema(t *testing.T) {
	wire := mustRead(t, "testdata/axm36/proposal-v3.json")
	for name, bad := range map[string][]byte{
		"unknown":   append([]byte(`{"unknown":true,`), wire[1:]...),
		"duplicate": append([]byte(`{"formatVersion":3,`), wire[1:]...),
		"case":      bytes.Replace(wire, []byte(`"formatVersion"`), []byte(`"FORMATVERSION"`), 1),
		"proof":     bytes.Replace(wire, []byte(`"profileId":`), []byte(`"capabilityProofReference":"retired","profileId":`), 1),
		"oversize":  bytes.Repeat([]byte(" "), MaxGraphBytes+1),
		"truncated": wire[:len(wire)/2],
	} {
		if _, err := decodeProposalV3(bad); err == nil {
			t.Errorf("admitted %s", name)
		}
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(wire, &object) != nil {
		t.Fatal("fixture")
	}
	for key, value := range object {
		delete(object, key)
		bad, _ := json.Marshal(object)
		if _, err := decodeProposalV3(bad); err == nil {
			t.Errorf("missing %s admitted", key)
		}
		object[key] = value
	}
}
