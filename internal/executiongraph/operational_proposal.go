package executiongraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/strictjson"
)

const OperationalProposalVersion = 3

type OperationalProposedNode struct {
	ResolvedProfile  runtimeprofile.ResolvedProfileV4 `json:"resolvedProfile"`
	Key              string                           `json:"key"`
	Capability       CapabilityRequest                `json:"capability"`
	Dependencies     []string                         `json:"dependencies"`
	Inputs           []string                         `json:"inputs"`
	Outputs          []string                         `json:"outputs"`
	Scope            Scope                            `json:"scope"`
	Effects          []Effect                         `json:"effects"`
	ValidationOwner  bool                             `json:"validationOwner"`
	IntegrationOwner bool                             `json:"integrationOwner"`
	Optional         bool                             `json:"optional"`
	Controls         ExecutionControls                `json:"controls"`
}

type OperationalProposal struct {
	FormatVersion int                       `json:"formatVersion"`
	PlanRevision  string                    `json:"planRevision"`
	PlanDigest    string                    `json:"planDigest"`
	Nodes         []OperationalProposedNode `json:"nodes"`
	Digest        string                    `json:"digest"`
}

func validateOperationalProposalTopology(p OperationalProposal) error {
	if p.FormatVersion != OperationalProposalVersion || len(p.Nodes) == 0 || len(p.Nodes) > maxNodes {
		return ErrInvalidGraph
	}
	topology := Proposal{PlanRevision: p.PlanRevision, PlanDigest: p.PlanDigest}
	for _, n := range p.Nodes {
		if !ValidOperationalProfileControls(n.ResolvedProfile, n.Controls, n.Capability) {
			return ErrInvalidGraph
		}
		topology.Nodes = append(topology.Nodes, ProposedNode{Key: n.Key, Capability: n.Capability, Dependencies: n.Dependencies, Inputs: n.Inputs, Outputs: n.Outputs, Scope: n.Scope, Effects: n.Effects, ValidationOwner: n.ValidationOwner, IntegrationOwner: n.IntegrationOwner, Optional: n.Optional, Controls: n.Controls})
	}
	return validateProposalTopology(topology, maxNodes)
}

func OperationalProposalDigest(p OperationalProposal) (string, error) {
	if validateOperationalProposalTopology(p) != nil {
		return "", ErrInvalidGraph
	}
	p.Digest = ""
	wire, err := json.Marshal(p)
	if err != nil || len(wire)+1 > MaxGraphBytes {
		return "", ErrInvalidGraph
	}
	sum := sha256.Sum256(wire)
	return hex.EncodeToString(sum[:]), nil
}

func ValidateOperationalProposal(p OperationalProposal) error {
	digest, err := OperationalProposalDigest(p)
	if err != nil || p.Digest != digest {
		return ErrInvalidGraph
	}
	return nil
}

func EncodeOperationalProposal(p OperationalProposal) ([]byte, error) {
	if ValidateOperationalProposal(p) != nil {
		return nil, ErrInvalidGraph
	}
	wire, err := json.Marshal(p)
	if err != nil || len(wire)+1 > MaxGraphBytes {
		return nil, ErrInvalidGraph
	}
	return append(wire, '\n'), nil
}

func DecodeOperationalProposal(wire []byte) (OperationalProposal, error) {
	var p OperationalProposal
	if strictjson.Decode(wire, &p, MaxGraphBytes) != nil || ValidateOperationalProposal(p) != nil {
		return OperationalProposal{}, ErrInvalidGraph
	}
	return p, nil
}
