// Package executiongraph retains inert historical graph codecs and shared
// process contracts. It has no operational Plan producer.
package executiongraph

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

const (
	OperationalProposalVersion = 3
	GraphFormatVersion         = 1
	maxNodes                   = 32
	maxListItems               = 64
	maxTextBytes               = 512
)

var ErrInvalidGraph = errors.New("invalid execution graph")

type CapabilityRequest struct {
	Role         string   `json:"role"`
	Complexity   string   `json:"complexity"`
	Capabilities []string `json:"capabilities"`
}

type Scope struct {
	ProjectID     string   `json:"projectId"`
	RepositoryKey string   `json:"repositoryKey"`
	Paths         []string `json:"paths"`
}

type Effect struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
}

type ExecutionControls struct {
	Timeout         time.Duration `json:"timeout"`
	MaximumAttempts uint32        `json:"maximumAttempts"`
	ReasoningEffort string        `json:"reasoningEffort,omitempty"`
}

func validEnvelopeFields(node ChildEnvelope) bool {
	if !validCapability(node.Capability) || !validToken(node.Scope.ProjectID) || !validToken(node.Scope.RepositoryKey) || len(node.Inputs) == 0 || len(node.Inputs) > maxListItems || len(node.Outputs) == 0 || len(node.Outputs) > maxListItems || len(node.Scope.Paths) == 0 || len(node.Scope.Paths) > maxListItems || len(node.AllowedEffects) > maxListItems {
		return false
	}
	if node.Controls.Timeout <= 0 || node.Controls.Timeout > 24*time.Hour || node.Controls.MaximumAttempts == 0 || node.Controls.MaximumAttempts > 10 || node.Controls.ReasoningEffort != "" && !validToken(node.Controls.ReasoningEffort) {
		return false
	}
	if node.Controls.ReasoningEffort != "" && !hasCapability(node.Capability.Capabilities, "reasoning-effort-"+node.Controls.ReasoningEffort) {
		return false
	}
	for _, values := range [][]string{node.Inputs, node.Outputs} {
		for _, value := range values {
			if !validToken(value) {
				return false
			}
		}
	}
	for _, value := range node.Scope.Paths {
		if !validRelativePath(value) {
			return false
		}
	}
	for _, effect := range node.AllowedEffects {
		if effect.Kind != "read" && effect.Kind != "repository-write" && effect.Kind != "process" && effect.Kind != "artifact-publish" && effect.Kind != "integration" {
			return false
		}
		if !validTarget(effect.Target) {
			return false
		}
	}
	return true
}

func hasCapability(capabilities []string, value string) bool {
	for _, capability := range capabilities {
		if capability == value {
			return true
		}
	}
	return false
}

func validCapability(request CapabilityRequest) bool {
	if !validToken(request.Role) || !validToken(request.Complexity) || len(request.Capabilities) == 0 || len(request.Capabilities) > maxListItems {
		return false
	}
	for _, capability := range request.Capabilities {
		if !validToken(capability) {
			return false
		}
	}
	return true
}

func validRelativePath(value string) bool {
	// Portable Repository-relative slash paths only: Windows volume (`C:`),
	// drive-relative, stream (`:`) and backslash forms never stay confined.
	return validTarget(value) && value != "." && value != ".." && !strings.HasPrefix(value, "/") && path.Clean(value) == value && !strings.HasPrefix(value, "../") && !strings.ContainsAny(value, `\:`)
}

func validTarget(value string) bool {
	return value != "" && len(value) <= maxTextBytes && !strings.ContainsAny(value, "\x00\r\n") && strings.TrimSpace(value) == value
}

func validToken(value string) bool {
	if value == "" || len(value) > maxTextBytes {
		return false
	}
	for _, char := range value {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' || char == ':') {
			return false
		}
	}
	return true
}

func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

// ValidProfileControls verifies the frozen sole-owner projection without acquiring proof.
func ValidProfileControls(profile *runtimeprofile.ResolvedProfile, controls ExecutionControls, capability CapabilityRequest) bool {
	return profile != nil && runtimeprofile.ValidateResolvedProfile(*profile) == nil && validEffortControls(profile.Effort, controls, capability)
}

func ValidFrozenProfileControls(profile *runtimeprofile.ResolvedProfileV4, controls ExecutionControls, capability CapabilityRequest) bool {
	return profile != nil && runtimeprofile.ValidateResolvedProfileV4(*profile) == nil && validEffortControls(profile.Effort, controls, capability)
}

func validNodeProfile(version int, historical *runtimeprofile.ResolvedProfile, frozen *runtimeprofile.ResolvedProfileV4, controls ExecutionControls, capability CapabilityRequest) bool {
	switch version {
	case 1:
		return historical == nil && frozen == nil
	case 2:
		return frozen == nil && ValidProfileControls(historical, controls, capability)
	case OperationalProposalVersion:
		return historical == nil && ValidFrozenProfileControls(frozen, controls, capability)
	}
	return false
}

func validEffortControls(effort runtimeprofile.Effort, controls ExecutionControls, capability CapabilityRequest) bool {
	expected := ""
	if effort.Mode == "explicit" {
		expected = effort.Value
	}
	if controls.ReasoningEffort != expected {
		return false
	}
	found := false
	for _, value := range capability.Capabilities {
		if strings.HasPrefix(value, "reasoning-effort-") {
			if expected == "" || value != "reasoning-effort-"+expected || found {
				return false
			}
			found = true
		}
	}
	return expected == "" || found
}
