// Package runtimeapplication resolves frozen per-agent invocation inputs and authors local Runtime configuration.
package runtimeapplication

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// ErrStale remains a diagnostic category for changed frozen inputs.
var ErrStale = errors.New("project runtime resolution stale")

type Request struct {
	Role           string   `json:"role"`
	Complexity     string   `json:"complexity"`
	Capabilities   []string `json:"capabilities"`
	RuntimeID      string   `json:"runtimeId,omitempty"`
	ModelProfileID string   `json:"modelProfileId,omitempty"`
}

type Preview struct {
	ProjectID             string                  `json:"projectId"`
	Request               Request                 `json:"request"`
	Choice                *runtimeprofile.Choice  `json:"choice,omitempty"`
	Blocker               *runtimeprofile.Blocker `json:"blocker,omitempty"`
	ProjectDigest         string                  `json:"projectDigest,omitempty"`
	ConfigurationDigest   string                  `json:"configurationDigest,omitempty"`
	ObservationDigest     string                  `json:"observationDigest,omitempty"`
	ConfigurationRevision uint64                  `json:"configurationRevision,omitempty"`
	ObservationRevisions  map[string]uint64       `json:"observationRevisions,omitempty"`
}

// Digest is retained for the presentation envelope of historical preview DTOs.
func (p Preview) Digest() string { wire, _ := json.Marshal(p); return digest(wire) }
func digest(wire []byte) string  { d := sha256.Sum256(wire); return hex.EncodeToString(d[:]) }
