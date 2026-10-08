package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #230 I230-T03: authorized Project EDIT publication across the
// portable and machine-local roots. The two objects cannot publish as one
// canonical object, so ADR-0007 is applied across that non-atomic boundary
// without claiming a cross-root transaction:
//
//  1. under exclusive installation-directory locks, both exact observed
//     generations are re-verified;
//  2. owned, versioned, bounded recovery state naming both objects, their
//     prior/new revisions and the complete next local wire is made durable in
//     the Project's installation directory;
//  3. the portable candidate is published with the exact observed manifest
//     as CAS precondition (ADR-0007 file protocol);
//  4. the complete local record is replaced with its exact observed wire as
//     CAS precondition (ADR-0007 file protocol);
//  5. the recovery state is retired.
//
// While the recovery state exists every installation reader fails closed
// with recovery_required, and recovery inspect classifies it from observed
// generations only. A confirmed portable commit is never rolled back.

const (
	editRecoveryPrefix        = ".axiom-edit-"
	editRecoveryFormatVersion = 1
	editRecoveryProtocol      = "project_edit_portable_then_local"
	// Base64 of a maximum local record plus bounded metadata.
	maxEditRecoveryBytes = 2*MaxRecordBytes + 16<<10
	installationRecord   = "installation.json"
)

type editRecoveryState struct {
	FormatVersion         int    `json:"formatVersion"`
	Protocol              string `json:"protocol"`
	OperationID           string `json:"operationId"`
	ProjectID             string `json:"projectId"`
	Slug                  string `json:"slug"`
	PortableSource        string `json:"portableSource"`
	PortableObject        string `json:"portableObject"`
	LocalObject           string `json:"localObject"`
	PriorPortableRevision string `json:"priorPortableRevision"`
	NewPortableRevision   string `json:"newPortableRevision"`
	PriorLocalRevision    string `json:"priorLocalRevision"`
	NewLocalRevision      string `json:"newLocalRevision"`
	NextLocalRecord       []byte `json:"nextLocalRecord"`
}

func hexDigest(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func validDigestText(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == value
}

// validOperationID accepts exactly the 16-byte hex identity temporaryName
// produces.
func validOperationID(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 16 && hex.EncodeToString(decoded) == value
}

func validEditRecoveryState(state editRecoveryState) bool {
	if state.FormatVersion != editRecoveryFormatVersion || state.Protocol != editRecoveryProtocol || !validOperationID(state.OperationID) ||
		len(project.ValidateIdentity(state.ProjectID, "edit")) != 0 || !project.ValidSlug(state.Slug) ||
		!filepath.IsAbs(state.PortableSource) || filepath.Clean(state.PortableSource) != state.PortableSource || filepath.Base(state.PortableSource) != state.Slug ||
		state.PortableObject != manifestName || state.LocalObject != installationRecord {
		return false
	}
	for _, revision := range []string{state.PriorPortableRevision, state.NewPortableRevision, state.PriorLocalRevision, state.NewLocalRevision} {
		if !validDigestText(revision) {
			return false
		}
	}
	if state.PriorPortableRevision == state.NewPortableRevision || state.PriorLocalRevision == state.NewLocalRevision || hexDigest(state.NextLocalRecord) != state.NewLocalRevision {
		return false
	}
	record, _, issues := DecodeObservedRecord(state.NextLocalRecord, true)
	if len(issues) != 0 {
		return false
	}
	recorded := record.State()
	return recorded.ProjectID == state.ProjectID && recorded.ObservedSlug == state.Slug && recorded.SourceLocation == state.PortableSource
}

func encodeEditRecoveryState(state editRecoveryState) ([]byte, error) {
	if !validEditRecoveryState(state) {
		return nil, ErrUnsafe
	}
	wire, err := json.Marshal(state)
	if err != nil {
		return nil, ErrUnsafe
	}
	wire = append(wire, '\n')
	if len(wire) > maxEditRecoveryBytes {
		return nil, ErrUnsafe
	}
	return wire, nil
}

// decodeEditRecoveryState accepts exactly the canonical format-1 encoding.
func decodeEditRecoveryState(wire []byte) (editRecoveryState, error) {
	if len(wire) == 0 || len(wire) > maxEditRecoveryBytes {
		return editRecoveryState{}, ErrRecoveryRequired
	}
	decoder := json.NewDecoder(bytes.NewReader(wire))
	decoder.DisallowUnknownFields()
	var state editRecoveryState
	if err := decoder.Decode(&state); err != nil || decoder.More() || !validEditRecoveryState(state) {
		return editRecoveryState{}, ErrRecoveryRequired
	}
	canonical, err := json.Marshal(state)
	if err != nil || !bytes.Equal(append(canonical, '\n'), wire) {
		return editRecoveryState{}, ErrRecoveryRequired
	}
	return state, nil
}

func editRecoveryName(operationID string) string { return editRecoveryPrefix + operationID }

func readEditRecoveryState(root *os.Root, name string) (editRecoveryState, error) {
	wire, err := readPrivateFileBounded(root, name, maxEditRecoveryBytes)
	if err != nil {
		return editRecoveryState{}, ErrRecoveryRequired
	}
	state, err := decodeEditRecoveryState(wire)
	if err != nil || editRecoveryName(state.OperationID) != name {
		return editRecoveryState{}, ErrRecoveryRequired
	}
	return state, nil
}

// writeEditRecoveryState makes the owned recovery state durable before any
// cross-store effect. A torn write is never decodable and is preserved for
// operator review instead of being interpreted.
func writeEditRecoveryState(root *os.Root, state editRecoveryState, hooks publicationHooks) (string, error) {
	wire, err := encodeEditRecoveryState(state)
	if err != nil {
		return "", err
	}
	name := editRecoveryName(state.OperationID)
	if err := hooks.writeFile(root, name, wire); err != nil {
		if _, statErr := root.Lstat(name); os.IsNotExist(statErr) {
			return "", err
		}
		if hooks.removeName(root, name) != nil {
			return name, ErrRecoveryRequired
		}
		return "", err
	}
	if err := hooks.syncRoot(root); err != nil {
		return name, ErrRecoveryRequired
	}
	if observed, err := readEditRecoveryState(root, name); err != nil || observed.OperationID != state.OperationID {
		return name, ErrRecoveryRequired
	}
	return name, nil
}

// EditStage names a deterministic fault point of the cross-store publication.
type EditStage string

const (
	// EditStageRecorded: recovery state is durable; nothing is published.
	EditStageRecorded EditStage = "recorded"
	// EditStagePortableCommitted: the portable publication is confirmed; the
	// local record is not yet published.
	EditStagePortableCommitted EditStage = "portable_committed"
)

// ProjectEditPublisher implements projectapp.EditPublisher over the portable
// and installation stores.
type ProjectEditPublisher struct {
	Installation InstallationStore
	Portable     PortableStore
	// Fault is a deterministic test hook. ErrSimulatedInterruption abandons
	// the operation at that stage exactly as a process crash would; any other
	// error is an ordinary failure at that stage.
	Fault func(EditStage) error
}

func (p ProjectEditPublisher) fault(stage EditStage) error {
	if p.Fault == nil {
		return nil
	}
	return p.Fault(stage)
}

func editResult(status projectapp.EditPublicationStatus, category string) projectapp.EditPublicationResult {
	return projectapp.EditPublicationResult{Status: status, Category: category}
}

func (p ProjectEditPublisher) PublishEdit(ctx context.Context, request projectapp.EditPublicationRequest) projectapp.EditPublicationResult {
	if err := ctx.Err(); err != nil {
		return editResult(projectapp.EditPublicationFailed, "cancelled")
	}
	if len(project.ValidateIdentity(request.ProjectID, "edit")) != 0 || !project.ValidSlug(request.Slug) || len(request.PortableExpected) == 0 ||
		len(request.LocalExpected) == 0 || len(request.LocalNext) == 0 || len(request.LocalNext) > MaxRecordBytes || bytes.Equal(request.LocalExpected, request.LocalNext) {
		return editResult(projectapp.EditPublicationFailed, "invalid_publication_request")
	}
	// Source confinement: publication writes only the Lingo-owned portable
	// location of this slug. A record that names any other source fails
	// closed; it is never written through.
	source := request.PortableDestination
	if !filepath.IsAbs(source) || filepath.Clean(source) != source {
		return editResult(projectapp.EditPublicationFailed, "unsupported_portable_source")
	}
	if canonical, err := trustedCanonical(source); err != nil || canonical != filepath.Join(p.Portable.root, request.Slug) {
		return editResult(projectapp.EditPublicationFailed, "unsupported_portable_source")
	}
	if !filepath.IsAbs(request.LocalDestination) {
		return editResult(projectapp.EditPublicationFailed, "invalid_publication_request")
	}
	if canonical, err := trustedCanonical(request.LocalDestination); err != nil || canonical != filepath.Join(p.Installation.root, "projects", request.ProjectID) {
		return editResult(projectapp.EditPublicationFailed, "invalid_publication_request")
	}
	if record, _, issues := DecodeObservedRecord(request.LocalNext, true); len(issues) != 0 || record.State().ProjectID != request.ProjectID || record.State().SourceLocation != source || record.State().ObservedSlug != request.Slug {
		return editResult(projectapp.EditPublicationFailed, "invalid_publication_request")
	}
	root, projects, target, err := openInstallationChain(p.Installation.root, request.ProjectID)
	if err != nil {
		return editResult(projectapp.EditPublicationFailed, "invalid_existing_local_state")
	}
	defer root.Close()
	defer projects.Close()
	defer target.Close()
	locks, err := lockRoots(true, root, projects, target)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return editResult(projectapp.EditPublicationConflict, "local_state_locked")
		}
		return editResult(projectapp.EditPublicationFailed, "storage_failure")
	}
	defer closeFiles(locks)
	switch installationDirectoryIssue(target) {
	case "":
	case "recovery_required":
		return editResult(projectapp.EditPublicationRecoveryRequired, "recovery_required")
	default:
		return editResult(projectapp.EditPublicationFailed, "invalid_existing_local_state")
	}
	current, err := readPrivateFile(target, installationRecord)
	if err != nil {
		return editResult(projectapp.EditPublicationFailed, "invalid_existing_local_state")
	}
	if !bytes.Equal(current, request.LocalExpected) {
		return editResult(projectapp.EditPublicationConflict, "local_state_changed")
	}
	portable, err := p.Portable.Read(ctx, request.Slug)
	switch {
	case errors.Is(err, ErrRecoveryRequired):
		return editResult(projectapp.EditPublicationRecoveryRequired, "recovery_required")
	case errors.Is(err, ErrConflict):
		return editResult(projectapp.EditPublicationConflict, "portable_state_locked")
	case err != nil:
		return editResult(projectapp.EditPublicationFailed, "portable_state_unavailable")
	case !bytes.Equal(portable, request.PortableExpected):
		return editResult(projectapp.EditPublicationConflict, "portable_state_changed")
	}
	hooks := p.Installation.chainHooks(root, projects, target, request.ProjectID)
	if len(request.PortableNext) == 0 {
		// Local-only edit: one ADR-0007 file publication suffices.
		return localOnlyEditResult(replaceRecordLocked(ctx, target, request.LocalExpected, request.LocalNext, hooks))
	}
	if len(request.PortableNext) > MaxRecordBytes || bytes.Equal(request.PortableNext, request.PortableExpected) {
		return editResult(projectapp.EditPublicationFailed, "invalid_publication_request")
	}
	operation, err := temporaryName("")
	if err != nil {
		return editResult(projectapp.EditPublicationFailed, "storage_failure")
	}
	state := editRecoveryState{
		FormatVersion: editRecoveryFormatVersion, Protocol: editRecoveryProtocol, OperationID: operation,
		ProjectID: request.ProjectID, Slug: request.Slug, PortableSource: source,
		PortableObject: manifestName, LocalObject: installationRecord,
		PriorPortableRevision: hexDigest(request.PortableExpected), NewPortableRevision: hexDigest(request.PortableNext),
		PriorLocalRevision: hexDigest(request.LocalExpected), NewLocalRevision: hexDigest(request.LocalNext),
		NextLocalRecord: append([]byte(nil), request.LocalNext...),
	}
	name, err := writeEditRecoveryState(target, state, hooks)
	if err != nil {
		if name != "" {
			return editResult(projectapp.EditPublicationRecoveryRequired, "recovery_required")
		}
		return editResult(projectapp.EditPublicationFailed, "storage_failure")
	}
	retire := func(status projectapp.EditPublicationStatus, category string) projectapp.EditPublicationResult {
		if removeProtocolState(target, name, hooks) != nil {
			return editResult(projectapp.EditPublicationRecoveryRequired, "recovery_required")
		}
		return editResult(status, category)
	}
	if err := p.fault(EditStageRecorded); err != nil {
		if errors.Is(err, ErrSimulatedInterruption) {
			return editResult(projectapp.EditPublicationRecoveryRequired, "interrupted")
		}
		return retire(projectapp.EditPublicationFailed, "storage_failure")
	}
	if err := p.Portable.Update(ctx, request.Slug, request.PortableExpected, request.PortableNext); err != nil {
		var publication *PublicationError
		switch {
		case errors.As(err, &publication) && publication.EffectCommitted():
			// The portable commit is confirmed but its own protocol state
			// remains; recovery resolves it before finalizing local state.
			return editResult(projectapp.EditPublicationPartial, "portable_recovery_required")
		case errors.Is(err, ErrSimulatedInterruption), errors.Is(err, ErrRecoveryRequired):
			return editResult(projectapp.EditPublicationRecoveryRequired, "recovery_required")
		case errors.Is(err, ErrConflict):
			return retire(projectapp.EditPublicationConflict, "portable_state_changed")
		default:
			return retire(projectapp.EditPublicationFailed, "portable_publication_failed")
		}
	}
	if err := p.fault(EditStagePortableCommitted); err != nil {
		return editResult(projectapp.EditPublicationPartial, "local_publication_incomplete")
	}
	// The portable commit is confirmed: complete the local publication even
	// if the caller is cancelled now, so no avoidable partial state remains.
	if err := replaceRecordLocked(context.WithoutCancel(ctx), target, request.LocalExpected, request.LocalNext, hooks); err != nil {
		var publication *PublicationError
		result := editResult(projectapp.EditPublicationPartial, "local_publication_incomplete")
		if errors.As(err, &publication) && publication.EffectCommitted() {
			result.LocalCommitted, result.Category = true, "recovery_state_retained"
		}
		return result
	}
	if removeProtocolState(target, name, hooks) != nil {
		return projectapp.EditPublicationResult{Status: projectapp.EditPublicationPartial, Category: "recovery_state_retained", LocalCommitted: true}
	}
	return editResult(projectapp.EditPublicationApplied, "published")
}

func localOnlyEditResult(err error) projectapp.EditPublicationResult {
	var publication *PublicationError
	switch {
	case err == nil:
		return editResult(projectapp.EditPublicationApplied, "published")
	case errors.As(err, &publication) && publication.EffectCommitted():
		return projectapp.EditPublicationResult{Status: projectapp.EditPublicationRecoveryRequired, Category: "recovery_state_retained", LocalCommitted: true}
	case errors.Is(err, ErrConflict):
		return editResult(projectapp.EditPublicationConflict, "local_state_changed")
	case errors.Is(err, ErrRecoveryRequired), errors.Is(err, ErrSimulatedInterruption):
		return editResult(projectapp.EditPublicationRecoveryRequired, "recovery_required")
	default:
		return editResult(projectapp.EditPublicationFailed, "local_publication_failed")
	}
}

// ReplaceRecord replaces the complete installation record only while its
// exact current wire equals expected, through the ADR-0007 file protocol so
// an interruption is classified by recovery inspect. It never creates an
// installation and refuses any pending protocol or recovery state.
func (s InstallationStore) ReplaceRecord(ctx context.Context, projectID string, expected, next []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	root, projects, target, err := openInstallationChain(s.root, projectID)
	if err != nil {
		return err
	}
	defer root.Close()
	defer projects.Close()
	defer target.Close()
	locks, err := lockRoots(true, root, projects, target)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	switch installationDirectoryIssue(target) {
	case "":
	case "recovery_required":
		return ErrRecoveryRequired
	default:
		return ErrUnsafe
	}
	if record, _, issues := DecodeObservedRecord(next, true); len(issues) != 0 || record.State().ProjectID != projectID {
		return ErrUnsafe
	}
	return replaceRecordLocked(ctx, target, expected, next, s.chainHooks(root, projects, target, projectID))
}

// replaceRecordLocked runs under the caller's exclusive installation chain
// locks. Owned edit recovery state in the directory does not block it.
func replaceRecordLocked(ctx context.Context, target *os.Root, expected, next []byte, hooks publicationHooks) error {
	if len(expected) == 0 || len(next) == 0 || len(next) > MaxRecordBytes {
		return ErrUnsafe
	}
	return publishFile(ctx, target, installationRecord, expected, next, false, hooks)
}

// chainHooks binds the store's test hooks and a pre-commit check that the
// opened installation chain is still the one visible at its canonical path.
func (s InstallationStore) chainHooks(root, projects, target *os.Root, projectID string) publicationHooks {
	hooks := publicationHooks{remove: s.removeAttempt, sync: s.syncDirectory, write: s.writeFile}
	hooks.beforeCommit = func() error {
		for _, check := range []struct {
			root *os.Root
			path string
		}{
			{root, s.root},
			{projects, filepath.Join(s.root, "projects")},
			{target, filepath.Join(s.root, "projects", projectID)},
		} {
			if err := stillAtPath(check.root, check.path); err != nil {
				return err
			}
		}
		return nil
	}
	return hooks
}

func openInstallationChain(path, projectID string) (*os.Root, *os.Root, *os.Root, error) {
	if len(project.ValidateIdentity(projectID, "installation")) != 0 {
		return nil, nil, nil, ErrUnsafe
	}
	root, err := existingPrivateRoot(path)
	if err != nil {
		return nil, nil, nil, err
	}
	projects, err := existingPrivateChild(root, "projects")
	if err != nil {
		root.Close()
		return nil, nil, nil, err
	}
	target, err := existingPrivateChild(projects, projectID)
	if err != nil {
		projects.Close()
		root.Close()
		return nil, nil, nil, err
	}
	return root, projects, target, nil
}

func editRecoveryStateName(name string) bool { return strings.HasPrefix(name, editRecoveryPrefix) }
