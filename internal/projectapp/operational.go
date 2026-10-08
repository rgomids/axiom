package projectapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"sort"

	"github.com/rgomids/axiom/internal/portableconfig"
	"github.com/rgomids/axiom/internal/project"
)

// Issue #230 machine-local Project operational state (I230-T02). Clarifications
// C230-01 and C230-02 fix its ownership: Project archive and Integration
// disable belong to this machine's installation of the Project. They never
// change portable intent (axiom.yaml), another machine, credentials or
// Provider state. This file owns the domain value, its transitions and the
// reviewed preview/authority boundary; the local adapter owns the strict wire
// format, confinement and the ADR-0007 publication protocol.

type ProjectStatus string

const (
	ProjectActive   ProjectStatus = "active"
	ProjectArchived ProjectStatus = "archived"
)

// MaxDisabledIntegrations bounds the local disable set.
const MaxDisabledIntegrations = 64

// OperationalRevisionAbsent is the exact observed revision of a missing record.
const OperationalRevisionAbsent = "absent"

// OperationalState is the complete machine-local operational state of one
// installed Project. A missing record means DefaultOperationalState.
type OperationalState struct {
	ProjectStatus        ProjectStatus
	DisabledIntegrations []string
}

func DefaultOperationalState() OperationalState {
	return OperationalState{ProjectStatus: ProjectActive, DisabledIntegrations: []string{}}
}

// ValidOperationalState requires a known status and a sorted, unique, bounded
// set of portable Integration keys. Unsorted or duplicated keys are
// contradictory state, never normalized on read.
func ValidOperationalState(s OperationalState) bool {
	if s.ProjectStatus != ProjectActive && s.ProjectStatus != ProjectArchived {
		return false
	}
	if s.DisabledIntegrations == nil || len(s.DisabledIntegrations) > MaxDisabledIntegrations {
		return false
	}
	for index, key := range s.DisabledIntegrations {
		if !ValidIntegrationKey(key) || index > 0 && s.DisabledIntegrations[index-1] >= key {
			return false
		}
	}
	return true
}

// ValidIntegrationKey accepts exactly the portable token grammar a manifest
// can declare for an Integration key.
func ValidIntegrationKey(key string) bool { return portableconfig.SafeValue(key, "token") }

func (s OperationalState) Archived() bool { return s.ProjectStatus == ProjectArchived }

func (s OperationalState) IntegrationDisabled(key string) bool {
	return slices.Contains(s.DisabledIntegrations, key)
}

func (s OperationalState) Equal(other OperationalState) bool {
	return s.ProjectStatus == other.ProjectStatus && slices.Equal(s.DisabledIntegrations, other.DisabledIntegrations)
}

func (s OperationalState) clone() OperationalState {
	s.DisabledIntegrations = append([]string{}, s.DisabledIntegrations...)
	return s
}

// OperationalObservation is one exact protected observation. Revision is
// OperationalRevisionAbsent for a missing record or the lowercase hex SHA-256
// of the exact record wire.
type OperationalObservation struct {
	Exists   bool
	Revision string
	State    OperationalState
}

// OperationalStore is consumer-owned. Inspect never collapses malformed,
// unknown-version, unsafe or interrupted state into absence (ErrUnsafe,
// ErrRecoveryRequired); a Project that is not installed is ErrNotFound.
// Commit replaces the complete record only while the current exact revision
// equals expected (ErrConflict otherwise) and reports a committed-but-
// unconfirmed outcome through an error implementing EffectCommitted.
type OperationalStore interface {
	InspectOperational(ctx context.Context, projectID string) (OperationalObservation, error)
	CommitOperational(ctx context.Context, projectID, expectedRevision string, next OperationalState) error
}

type OperationalOperation string

const (
	ArchiveProject     OperationalOperation = "archive"
	ReactivateProject  OperationalOperation = "reactivate"
	DisableIntegration OperationalOperation = "disable"
	EnableIntegration  OperationalOperation = "enable"
)

// OperationalRequest names one exact local transition. ProjectID is the
// resolved canonical identity; Integration is required for disable/enable
// only. Callers validate Integration declaration against portable intent;
// enable stays valid for an undeclared stale key so it can be cleared.
type OperationalRequest struct {
	ProjectID   string
	Operation   OperationalOperation
	Integration string
}

// Stable categories for the operational-state boundary.
const (
	OperationalInvalidInput     = "invalid_input"
	OperationalNotInstalled     = "project_not_installed"
	OperationalStateInvalid     = "operational_state_invalid"
	OperationalRecoveryRequired = "recovery_required"
	OperationalConflict         = "operational_state_conflict"
	OperationalCancelled        = "cancelled"
	OperationalStorageFailure   = "storage_failure"
	OperationalAuthorityDenied  = "local_authority_denied"
	OperationalPreviewReady     = "operational_preview_ready"
	OperationalApplied          = "operational_state_updated"
	OperationalAppliedRecovery  = "operational_state_committed_recovery_required"
)

// Deterministic no-op categories: an equivalent request writes nothing and
// needs no authority.
const (
	ProjectAlreadyArchived     = "project_already_archived"
	ProjectAlreadyActive       = "project_already_active"
	IntegrationAlreadyDisabled = "integration_already_disabled"
	IntegrationAlreadyEnabled  = "integration_already_enabled"
)

type OperationalView struct {
	ProjectStatus        ProjectStatus `json:"projectStatus"`
	DisabledIntegrations []string      `json:"disabledIntegrations"`
}

func ViewOperational(s OperationalState) OperationalView {
	return OperationalView{ProjectStatus: s.ProjectStatus, DisabledIntegrations: append([]string{}, s.DisabledIntegrations...)}
}

// OperationalBoundary is constant disclosure of what the operation can never
// change; it is part of the reviewed preview and its digest.
type OperationalBoundary struct {
	Portable    string `json:"portable"`
	Provider    string `json:"provider"`
	Credentials string `json:"credentials"`
}

var operationalBoundary = OperationalBoundary{Portable: "unchanged", Provider: "none", Credentials: "unchanged"}

// OperationalPreview is the complete reviewed local effect of one request.
// No effect means no publication.
type OperationalPreview struct {
	ProjectID   string               `json:"projectId"`
	Operation   OperationalOperation `json:"operation"`
	Integration string               `json:"integration,omitempty"`
	Current     OperationalView      `json:"current"`
	Result      OperationalView      `json:"result"`
	Revision    string               `json:"revision"`
	Effects     []EditEffect         `json:"effects"`
	Boundary    OperationalBoundary  `json:"boundary"`
	Digest      string               `json:"digest"`
}

type OperationalStatus string

const (
	OperationalPreviewed OperationalStatus = "preview"
	OperationalUnchanged OperationalStatus = "unchanged"
	OperationalCommitted OperationalStatus = "applied"
	OperationalDenied    OperationalStatus = "denied"
	OperationalFailed    OperationalStatus = "failed"
)

type OperationalResult struct {
	Status   OperationalStatus
	Category string
	Preview  *OperationalPreview
	// Committed is true when the store confirmed the commit point even if a
	// later protocol step requires recovery.
	Committed bool
}

// ProposeOperational applies one transition to the complete current state.
// It returns the complete next state and the no-op category when equivalent.
func ProposeOperational(current OperationalState, request OperationalRequest) (OperationalState, string, string) {
	if !ValidOperationalState(current) {
		return OperationalState{}, "", OperationalStateInvalid
	}
	next := current.clone()
	switch request.Operation {
	case ArchiveProject, ReactivateProject:
		if request.Integration != "" {
			return OperationalState{}, "", OperationalInvalidInput
		}
		target, unchanged := ProjectArchived, ProjectAlreadyArchived
		if request.Operation == ReactivateProject {
			target, unchanged = ProjectActive, ProjectAlreadyActive
		}
		if current.ProjectStatus == target {
			return next, unchanged, ""
		}
		next.ProjectStatus = target
	case DisableIntegration:
		if !ValidIntegrationKey(request.Integration) {
			return OperationalState{}, "", OperationalInvalidInput
		}
		if current.IntegrationDisabled(request.Integration) {
			return next, IntegrationAlreadyDisabled, ""
		}
		if len(next.DisabledIntegrations) >= MaxDisabledIntegrations {
			return OperationalState{}, "", OperationalInvalidInput
		}
		next.DisabledIntegrations = append(next.DisabledIntegrations, request.Integration)
		sort.Strings(next.DisabledIntegrations)
	case EnableIntegration:
		if !ValidIntegrationKey(request.Integration) {
			return OperationalState{}, "", OperationalInvalidInput
		}
		if !current.IntegrationDisabled(request.Integration) {
			return next, IntegrationAlreadyEnabled, ""
		}
		next.DisabledIntegrations = slices.DeleteFunc(next.DisabledIntegrations, func(key string) bool { return key == request.Integration })
	default:
		return OperationalState{}, "", OperationalInvalidInput
	}
	return next, "", ""
}

func validOperationalRequest(request OperationalRequest) bool {
	return len(project.ValidateIdentity(request.ProjectID, "operational")) == 0
}

// InspectOperational reads the current state and maps store failures to the
// stable categories. A missing record is the default state, never an error.
func InspectOperational(ctx context.Context, store OperationalStore, projectID string) (OperationalObservation, string) {
	if store == nil {
		return OperationalObservation{}, OperationalStorageFailure
	}
	if len(project.ValidateIdentity(projectID, "operational")) != 0 {
		return OperationalObservation{}, OperationalInvalidInput
	}
	if ctx.Err() != nil {
		return OperationalObservation{}, OperationalCancelled
	}
	observation, err := store.InspectOperational(ctx, projectID)
	if err != nil {
		return OperationalObservation{}, operationalCategory(ctx, err)
	}
	if !ValidOperationalState(observation.State) || observation.Revision == "" || observation.Exists == (observation.Revision == OperationalRevisionAbsent) {
		return OperationalObservation{}, OperationalStateInvalid
	}
	return observation, ""
}

// PreviewOperational observes fresh state and returns the reviewed effect.
// It never writes.
func PreviewOperational(ctx context.Context, store OperationalStore, request OperationalRequest) (OperationalPreview, OperationalState, string, string) {
	if !validOperationalRequest(request) {
		return OperationalPreview{}, OperationalState{}, "", OperationalInvalidInput
	}
	observation, category := InspectOperational(ctx, store, request.ProjectID)
	if category != "" {
		return OperationalPreview{}, OperationalState{}, "", category
	}
	next, unchanged, category := ProposeOperational(observation.State, request)
	if category != "" {
		return OperationalPreview{}, OperationalState{}, "", category
	}
	preview := OperationalPreview{
		ProjectID: request.ProjectID, Operation: request.Operation, Integration: request.Integration,
		Current: ViewOperational(observation.State), Result: ViewOperational(next),
		Revision: observation.Revision, Effects: []EditEffect{}, Boundary: operationalBoundary,
	}
	if unchanged == "" {
		preview.Effects = append(preview.Effects, operationalEffect(request))
	}
	preview.Digest = operationalDigest(preview)
	return preview, next, unchanged, ""
}

func operationalEffect(request OperationalRequest) EditEffect {
	switch request.Operation {
	case ArchiveProject:
		return EditEffect{Scope: LocalScope, Code: "archive_project"}
	case ReactivateProject:
		return EditEffect{Scope: LocalScope, Code: "reactivate_project"}
	case DisableIntegration:
		return EditEffect{Scope: LocalScope, Code: "disable_integration", Key: request.Integration}
	}
	return EditEffect{Scope: LocalScope, Code: "enable_integration", Key: request.Integration}
}

func operationalDigest(preview OperationalPreview) string {
	preview.Digest = ""
	wire, _ := json.Marshal(struct {
		Domain  string             `json:"domain"`
		Preview OperationalPreview `json:"preview"`
	}{"axiom-operational-state-v1", preview})
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:])
}

// ApplyOperational is the single reviewed mutation path for Project
// archive/reactivate and Integration disable/enable. An equivalent request is
// a deterministic no-op without authority. Otherwise the caller must supply
// explicit local authority bound to the exact digest of a fresh preview; the
// store then commits with the observed revision as its CAS precondition.
func ApplyOperational(ctx context.Context, store OperationalStore, request OperationalRequest, reviewedDigest string, authorizeLocal bool) OperationalResult {
	preview, next, unchanged, category := PreviewOperational(ctx, store, request)
	if category != "" {
		return OperationalResult{Status: OperationalFailed, Category: category}
	}
	if unchanged != "" {
		return OperationalResult{Status: OperationalUnchanged, Category: unchanged, Preview: &preview}
	}
	if !authorizeLocal {
		if reviewedDigest != "" {
			return OperationalResult{Status: OperationalDenied, Category: OperationalAuthorityDenied, Preview: &preview}
		}
		return OperationalResult{Status: OperationalPreviewed, Category: OperationalPreviewReady, Preview: &preview}
	}
	if reviewedDigest != preview.Digest {
		return OperationalResult{Status: OperationalDenied, Category: OperationalAuthorityDenied, Preview: &preview}
	}
	if ctx.Err() != nil {
		return OperationalResult{Status: OperationalFailed, Category: OperationalCancelled, Preview: &preview}
	}
	err := store.CommitOperational(ctx, request.ProjectID, preview.Revision, next)
	if err == nil {
		return OperationalResult{Status: OperationalCommitted, Category: OperationalApplied, Preview: &preview, Committed: true}
	}
	var committed interface{ EffectCommitted() bool }
	if errors.As(err, &committed) && committed.EffectCommitted() {
		return OperationalResult{Status: OperationalCommitted, Category: OperationalAppliedRecovery, Preview: &preview, Committed: true}
	}
	return OperationalResult{Status: OperationalFailed, Category: operationalCategory(ctx, err), Preview: &preview}
}

func operationalCategory(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded), ctx.Err() != nil:
		return OperationalCancelled
	case errors.Is(err, ErrRecoveryRequired):
		return OperationalRecoveryRequired
	case errors.Is(err, ErrNotFound):
		return OperationalNotInstalled
	case errors.Is(err, ErrConflict):
		return OperationalConflict
	case errors.Is(err, ErrUnsafe):
		return OperationalStateInvalid
	}
	return OperationalStorageFailure
}
