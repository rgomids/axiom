package executiongraph

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func operationalProposalFixture(t *testing.T) OperationalProposal {
	t.Helper()
	p, err := DecodeOperationalProposal(mustRead(t, "testdata/axm36/proposal-v3.json"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOperationalProposalGolden(t *testing.T) {
	p := operationalProposalFixture(t)
	wire, err := EncodeOperationalProposal(p)
	if err != nil || !bytes.Equal(wire, mustRead(t, "testdata/axm36/proposal-v3.json")) {
		t.Fatal("unstable proposal bytes", err)
	}
	want := fmt.Sprintf("proposal %s\nwire %x\n", p.Digest, sha256.Sum256(wire))
	if want != string(mustRead(t, "testdata/axm36/proposal-v3.digests")) {
		t.Fatal("proposal digest changed")
	}
	var old Proposal
	if json.Unmarshal(wire, &old) != nil {
		t.Fatal("fixture")
	}
	if NewPlanner(nil).Validate(context.Background(), old, maxNodes) == nil {
		t.Fatal("new proposal accepted as historical authority")
	}
	old.FormatVersion = 2
	legacy, _ := json.Marshal(old)
	if _, err := DecodeOperationalProposal(legacy); err == nil {
		t.Fatal("historical proposal accepted as current")
	}
}

func TestOperationalProposalDigestAndTopology(t *testing.T) {
	for name, mutate := range map[string]func(*OperationalProposal){
		"future": func(p *OperationalProposal) { p.FormatVersion++ },
		"node limit": func(p *OperationalProposal) {
			for len(p.Nodes) <= maxNodes {
				p.Nodes = append(p.Nodes, p.Nodes[0])
			}
		},
		"duplicate":          func(p *OperationalProposal) { p.Nodes[1].Key = p.Nodes[0].Key },
		"missing dependency": func(p *OperationalProposal) { p.Nodes[0].Dependencies = []string{"absent"} },
		"cycle":              func(p *OperationalProposal) { p.Nodes[0].Dependencies = []string{p.Nodes[1].Key} },
		"integration": func(p *OperationalProposal) {
			for i := range p.Nodes {
				p.Nodes[i].IntegrationOwner = false
			}
		},
		"validation owner": func(p *OperationalProposal) {
			for i := range p.Nodes {
				p.Nodes[i].ValidationOwner = false
			}
		},
		"unsafe path":   func(p *OperationalProposal) { p.Nodes[0].Scope.Paths = []string{"../escape"} },
		"effort":        func(p *OperationalProposal) { p.Nodes[0].Controls.ReasoningEffort = "arbitrary" },
		"invalid tuple": func(p *OperationalProposal) { p.Nodes[0].ResolvedProfile.EffortSourceDigest = "bad" },
	} {
		t.Run(name, func(t *testing.T) {
			p := operationalProposalFixture(t)
			mutate(&p)
			if _, err := OperationalProposalDigest(p); err == nil {
				t.Fatal("unsafe topology can be signed")
			}
		})
	}
	p := operationalProposalFixture(t)
	p.Nodes[0].ResolvedProfile.ExecutableDigest = strings.Repeat("f", 64)
	if _, err := EncodeOperationalProposal(p); err == nil {
		t.Fatal("stale digest accepted after identity drift")
	}
	digest, err := OperationalProposalDigest(p)
	if err != nil || digest == p.Digest {
		t.Fatal("identity drift did not change digest")
	}
	p.Digest = digest
	if _, err := EncodeOperationalProposal(p); err != nil {
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
		if _, err := DecodeOperationalProposal(bad); err == nil {
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
		if _, err := DecodeOperationalProposal(bad); err == nil {
			t.Errorf("missing %s admitted", key)
		}
		object[key] = value
	}
}
