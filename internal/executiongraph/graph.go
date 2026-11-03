package executiongraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

const MaxGraphBytes = 1 << 20

type ParentExecution struct {
	ExecutionID      string    `json:"executionId"`
	GraphRevision    uint64    `json:"graphRevision"`
	ProposalDigest   string    `json:"proposalDigest"`
	AuthorityDigest  string    `json:"authorityDigest"`
	AuthorityEffects []Effect  `json:"authorityEffects"`
	IntegrationChild string    `json:"integrationChild"`
	CreatedAt        time.Time `json:"createdAt"`
}

type Resolution struct {
	ResolvedProfile       *runtimeprofile.ResolvedProfile   `json:"resolvedProfile,omitempty"`
	FrozenProfile         *runtimeprofile.ResolvedProfileV4 `json:"frozenProfile,omitempty"`
	RuntimeID             string                            `json:"runtimeId"`
	ModelProfileID        string                            `json:"modelProfileId"`
	ConfigurationRevision uint64                            `json:"configurationRevision"`
	ObservationRevision   uint64                            `json:"observationRevision"`
}

type ChildEnvelope struct {
	Capability         CapabilityRequest `json:"capability"`
	Inputs             []string          `json:"inputs"`
	Outputs            []string          `json:"outputs"`
	Scope              Scope             `json:"scope"`
	Workspace          string            `json:"workspace"`
	Dependencies       []string          `json:"dependencies"`
	AllowedEffects     []Effect          `json:"allowedEffects"`
	AuthorityReference string            `json:"authorityReference"`
	Controls           ExecutionControls `json:"controls"`
	ValidationOwner    bool              `json:"validationOwner"`
	IntegrationOwner   bool              `json:"integrationOwner"`
	Optional           bool              `json:"optional"`
	Resolution         Resolution        `json:"resolution"`
}

type ChildExecution struct {
	ExecutionID   string        `json:"executionId"`
	ParentID      string        `json:"parentId"`
	GraphRevision uint64        `json:"graphRevision"`
	NodeKey       string        `json:"nodeKey"`
	Envelope      ChildEnvelope `json:"envelope"`
	Attempts      []Attempt     `json:"attempts"`
}

type AttemptStatus string

const (
	AttemptNotStarted AttemptStatus = "not-started"
	AttemptRunning    AttemptStatus = "running"
	AttemptBlocked    AttemptStatus = "blocked"
	AttemptSucceeded  AttemptStatus = "succeeded"
	AttemptFailed     AttemptStatus = "failed"
	AttemptCancelled  AttemptStatus = "cancelled"
	AttemptSkipped    AttemptStatus = "skipped"
	AttemptUnknown    AttemptStatus = "unknown"
)

type Attempt struct {
	AttemptID        string        `json:"attemptId"`
	Number           uint32        `json:"number"`
	Status           AttemptStatus `json:"status"`
	StartedAt        *time.Time    `json:"startedAt,omitempty"`
	FinishedAt       *time.Time    `json:"finishedAt,omitempty"`
	ResultReference  string        `json:"resultReference,omitempty"`
	OutputDigest     string        `json:"outputDigest,omitempty"`
	ExitCode         int           `json:"exitCode,omitempty"`
	AmbiguousEffect  bool          `json:"ambiguousEffect"`
	CancellationSeen bool          `json:"cancellationSeen,omitempty"`
}

type Graph struct {
	FormatVersion   int               `json:"formatVersion"`
	Parent          ParentExecution   `json:"parent"`
	Children        []ChildExecution  `json:"children"`
	StorageRevision [sha256.Size]byte `json:"-"`
}

func ValidGraph(graph Graph) bool {
	for _, child := range graph.Children {
		resolution := child.Envelope.Resolution
		if !validNodeProfile(graph.FormatVersion, resolution.ResolvedProfile, resolution.FrozenProfile, child.Envelope.Controls, child.Envelope.Capability) {
			return false
		}
	}
	return len(graph.Children) > 0 && validGraphTopology(graph)
}

func validGraphTopology(graph Graph) bool {
	if !validOpaqueID(graph.Parent.ExecutionID) || graph.Parent.GraphRevision == 0 || !validDigest(graph.Parent.ProposalDigest) || !validDigest(graph.Parent.AuthorityDigest) || graph.Parent.CreatedAt.IsZero() || len(graph.Children) == 0 || len(graph.Children) > maxNodes {
		return false
	}
	ids := map[string]bool{graph.Parent.ExecutionID: true}
	attemptIDs := map[string]bool{}
	childrenByID := make(map[string]ChildExecution, len(graph.Children))
	projectID := ""
	integrationCount := 0
	for _, child := range graph.Children {
		if !validOpaqueID(child.ExecutionID) || ids[child.ExecutionID] || attemptIDs[child.ExecutionID] || child.ParentID != graph.Parent.ExecutionID || child.GraphRevision != graph.Parent.GraphRevision || !validToken(child.NodeKey) || !validEnvelope(child.Envelope) || len(child.Attempts) > int(child.Envelope.Controls.MaximumAttempts) {
			return false
		}
		ids[child.ExecutionID] = true
		childrenByID[child.ExecutionID] = child
		if projectID == "" {
			projectID = child.Envelope.Scope.ProjectID
		} else if child.Envelope.Scope.ProjectID != projectID {
			return false
		}
		if !effectSubset(child.Envelope.AllowedEffects, graph.Parent.AuthorityEffects) {
			return false
		}
		for index, attempt := range child.Attempts {
			if !validOpaqueID(attempt.AttemptID) || ids[attempt.AttemptID] || attemptIDs[attempt.AttemptID] || attempt.Number != uint32(index+1) || !validAttemptStatus(attempt.Status) || attempt.StartedAt == nil {
				return false
			}
			attemptIDs[attempt.AttemptID] = true
			if attempt.Status != AttemptRunning && attempt.FinishedAt == nil {
				return false
			}
			if attempt.OutputDigest != "" && !validDigest(attempt.OutputDigest) {
				return false
			}
		}
		if child.Envelope.IntegrationOwner {
			integrationCount++
			if graph.Parent.IntegrationChild != child.ExecutionID || !child.Envelope.ValidationOwner {
				return false
			}
		}
	}
	for _, child := range graph.Children {
		for _, dependency := range child.Envelope.Dependencies {
			if !ids[dependency] || dependency == graph.Parent.ExecutionID || dependency == child.ExecutionID {
				return false
			}
		}
	}
	if childDependencyCycle(childrenByID) {
		return false
	}
	for childID := range childrenByID {
		if childID != graph.Parent.IntegrationChild && !childDependsTransitively(childrenByID, graph.Parent.IntegrationChild, childID) {
			return false
		}
	}
	digest, err := effectsDigest(graph.Parent.AuthorityEffects)
	return integrationCount == 1 && err == nil && digest == graph.Parent.AuthorityDigest
}

func validAttemptStatus(status AttemptStatus) bool {
	switch status {
	case AttemptNotStarted, AttemptRunning, AttemptBlocked, AttemptSucceeded, AttemptFailed, AttemptCancelled, AttemptSkipped, AttemptUnknown:
		return true
	default:
		return false
	}
}

func EncodeGraph(graph Graph) ([]byte, error) {
	if graph.FormatVersion == OperationalProposalVersion || !ValidGraph(graph) {
		return nil, ErrInvalidGraph
	}
	graph.StorageRevision = [sha256.Size]byte{}
	wire, err := json.Marshal(graph)
	if err != nil || len(wire)+1 > MaxGraphBytes {
		return nil, ErrInvalidGraph
	}
	return append(wire, '\n'), nil
}

func DecodeGraph(wire []byte) (Graph, error) {
	if len(wire) == 0 || len(wire) > MaxGraphBytes {
		return Graph{}, ErrInvalidGraph
	}
	decoder := json.NewDecoder(strings.NewReader(string(wire)))
	decoder.DisallowUnknownFields()
	var graph Graph
	if err := decoder.Decode(&graph); err != nil {
		return Graph{}, ErrInvalidGraph
	}
	if graph.FormatVersion == 2 && workflowdefinition.StrictJSONBounded(wire, &graph, MaxGraphBytes) != nil {
		return Graph{}, ErrInvalidGraph
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) || graph.FormatVersion == OperationalProposalVersion || !ValidGraph(graph) {
		return Graph{}, ErrInvalidGraph
	}
	graph.StorageRevision = sha256.Sum256(wire)
	return graph, nil
}

func validEnvelope(envelope ChildEnvelope) bool {
	return validEnvelopeFields(envelope) && validTarget(envelope.Workspace) && validTarget(envelope.AuthorityReference) && validResolution(envelope.Resolution)
}

func validResolution(resolution Resolution) bool {
	if profile := resolution.ResolvedProfile; profile != nil {
		if resolution.FrozenProfile != nil || runtimeprofile.ValidateResolvedProfile(*profile) != nil || profile.RuntimeID != resolution.RuntimeID || profile.ProfileID != resolution.ModelProfileID || profile.ConfigurationRevision != resolution.ConfigurationRevision || profile.ObservationRevision != resolution.ObservationRevision {
			return false
		}
	}
	if profile := resolution.FrozenProfile; profile != nil {
		if runtimeprofile.ValidateResolvedProfileV4(*profile) != nil || profile.RuntimeID != resolution.RuntimeID || profile.ProfileID != resolution.ModelProfileID || profile.ConfigurationRevision != resolution.ConfigurationRevision || profile.ObservationRevision != resolution.ObservationRevision {
			return false
		}
	}
	return (resolution.RuntimeID == "codex" || resolution.RuntimeID == "claude") && validToken(resolution.ModelProfileID) && resolution.ConfigurationRevision > 0 && resolution.ObservationRevision > 0
}

func childDependencyCycle(children map[string]ChildExecution) bool {
	state := map[string]uint8{}
	var visit func(string) bool
	visit = func(childID string) bool {
		if state[childID] == 1 {
			return true
		}
		if state[childID] == 2 {
			return false
		}
		state[childID] = 1
		for _, dependency := range children[childID].Envelope.Dependencies {
			if visit(dependency) {
				return true
			}
		}
		state[childID] = 2
		return false
	}
	for childID := range children {
		if visit(childID) {
			return true
		}
	}
	return false
}

func childDependsTransitively(children map[string]ChildExecution, childID, dependencyID string) bool {
	seen := map[string]bool{}
	var visit func(string) bool
	visit = func(current string) bool {
		if seen[current] {
			return false
		}
		seen[current] = true
		for _, dependency := range children[current].Envelope.Dependencies {
			if dependency == dependencyID || visit(dependency) {
				return true
			}
		}
		return false
	}
	return visit(childID)
}

func effectSubset(child, parent []Effect) bool {
	allowed := map[string]bool{}
	for _, effect := range parent {
		allowed[effect.Kind+"\x00"+effect.Target] = true
	}
	for _, effect := range child {
		if !allowed[effect.Kind+"\x00"+effect.Target] {
			return false
		}
	}
	return true
}

func sortedEffects(effects []Effect) []Effect {
	result := append([]Effect(nil), effects...)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Kind+"\x00"+result[i].Target < result[j].Kind+"\x00"+result[j].Target
	})
	return result
}

func effectsDigest(effects []Effect) (string, error) {
	wire, err := json.Marshal(sortedEffects(effects))
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:]), nil
}

func validOpaqueID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if char != '-' {
				return false
			}
			continue
		}
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return true
}
