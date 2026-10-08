package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/project"
)

// Recovery classification of the cross-store Project edit state (I230-T03).
// The plan is selected only from generations positively identified by the
// owned recovery state and freshly observed:
//
//	prior portable + prior local -> restore_prior (retire the state)
//	new portable   + prior local -> finalize_committed (CAS-publish the
//	                                recorded next local wire, retire the state)
//	new portable   + new local   -> finalize_committed (retire the state)
//	anything else                -> preserved_review
//
// A confirmed portable commit is never rolled back and absence is never
// treated as proof of rollback.

func installationDirectoryName(directory string) bool {
	parts := splitDirectory(directory)
	return len(parts) == 2 && parts[0] == "projects" && len(project.ValidateIdentity(parts[1], "recovery")) == 0
}

const (
	revisionInvalid = "invalid"
	revisionPending = "pending"
)

func planEditRecovery(target *os.Root, directory, name, projectsRoot string) (RecoveryPlan, bool, error) {
	plan := RecoveryPlan{Scope: RecoveryScopeState, Directory: directory, Protocol: protocolProjectEdit, Marker: name, Action: PreservedReview}
	finish := func(explanation string) (RecoveryPlan, bool, error) {
		plan.Explanation = explanation
		plan.Digest = recoveryDigest(plan)
		return plan, true, nil
	}
	state, err := readEditRecoveryState(target, name)
	if err != nil || directory != "projects/"+state.ProjectID {
		plan.Objects = []RecoveryObject{{Role: "unrecognized", Name: name, Present: true}}
		return finish("project edit recovery state is malformed, unsupported, or names another Project; preserved for operator review")
	}
	portable, err := observeEditPortable(projectsRoot, state)
	if err != nil {
		return RecoveryPlan{}, false, err
	}
	local := observeRecoveryFile(target, "local", installationRecord)
	plan.Objects = []RecoveryObject{{Role: "marker", Name: name, Present: true}, portable, local}
	switch portable.Revision {
	case revisionPending:
		return finish("the portable publication has its own interrupted protocol state; recover that plan first")
	case revisionInvalid:
		return finish("the recorded portable source is outside the owned projects root, incomplete, or unsafe")
	}
	if local.Revision == revisionInvalid {
		return finish("the local installation record is present but incomplete or unsafe")
	}
	priorPortable := portable.Present && portable.Revision == state.PriorPortableRevision
	newPortable := portable.Present && portable.Revision == state.NewPortableRevision
	priorLocal := local.Present && local.Revision == state.PriorLocalRevision
	newLocal := local.Present && local.Revision == state.NewLocalRevision
	switch {
	case priorPortable && priorLocal:
		plan.Action = RestorePrior
		return finish("project edit stopped before its portable commit; prior portable and local generations remain canonical")
	case newPortable && priorLocal:
		plan.Action = FinalizeCommitted
		return finish("portable commit is confirmed; finalize publishes the recorded local generation over the exact prior local record")
	case newPortable && newLocal:
		plan.Action = FinalizeCommitted
		return finish("portable and local commits are confirmed; finalize retires the recovery state")
	}
	return finish("generation facts are ambiguous or contradict the recorded project edit")
}

// observeEditPortable reads only the Lingo-owned portable location recorded
// by the state, under a shared portable-root lock.
func observeEditPortable(projectsRoot string, state editRecoveryState) (RecoveryObject, error) {
	object := RecoveryObject{Role: "portable", Name: state.Slug + "/" + manifestName}
	invalid := func() (RecoveryObject, error) {
		object.Present, object.Revision = true, revisionInvalid
		return object, nil
	}
	if projectsRoot == "" || !filepath.IsAbs(projectsRoot) {
		return invalid()
	}
	canonical, err := trustedCanonical(projectsRoot)
	if err != nil {
		return invalid()
	}
	if source, err := trustedCanonical(state.PortableSource); err != nil || source != filepath.Join(canonical, state.Slug) {
		return invalid()
	}
	root, err := existingPrivateRoot(canonical)
	if errors.Is(err, ErrNotFound) {
		return object, nil
	}
	if err != nil {
		return invalid()
	}
	defer root.Close()
	lock, err := lockDirectory(root, false)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return RecoveryObject{}, ErrConflict
		}
		return invalid()
	}
	defer lock.Close()
	projectRoot, err := openManifestProject(root, state.Slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return object, nil
	case errors.Is(err, ErrRecoveryRequired):
		object.Present, object.Revision = true, revisionPending
		return object, nil
	case err != nil:
		return invalid()
	}
	defer projectRoot.Close()
	wire, err := readPrivateFile(projectRoot, manifestName)
	if err != nil {
		return invalid()
	}
	object.Present, object.Revision = true, hexDigest(wire)
	return object, nil
}

// applyEditRecovery runs under the exclusive installation chain locks after
// the fresh plan was proven identical to the authorized one.
func applyEditRecovery(ctx context.Context, chain []*os.Root, plan RecoveryPlan) (RecoveryResult, error) {
	result := RecoveryResult{Action: plan.Action, Removed: []string{}}
	target := chain[len(chain)-1]
	state, err := readEditRecoveryState(target, plan.Marker)
	if err != nil {
		return result, ErrConflict
	}
	if plan.Action == FinalizeCommitted {
		local := recoveryObject(plan.Objects, "local")
		if local.Revision == state.PriorLocalRevision {
			current, err := readPrivateFile(target, installationRecord)
			if err != nil || hexDigest(current) != state.PriorLocalRevision {
				return result, ErrConflict
			}
			if err := publishFile(ctx, target, installationRecord, current, state.NextLocalRecord, false, publicationHooks{}); err != nil {
				var publication *PublicationError
				if errors.As(err, &publication) && publication.EffectCommitted() {
					result.Published = append(result.Published, installationRecord)
				}
				if errors.Is(err, ErrConflict) && len(result.Published) == 0 {
					return result, ErrConflict
				}
				return result, ErrRecoveryRequired
			}
			result.Published = append(result.Published, installationRecord)
		}
	}
	if err := removeProtocolState(target, plan.Marker, publicationHooks{}); err != nil {
		return result, ErrRecoveryRequired
	}
	result.Removed = append(result.Removed, plan.Marker)
	return result, nil
}

func recoveryObject(objects []RecoveryObject, role string) RecoveryObject {
	for _, object := range objects {
		if object.Role == role {
			return object
		}
	}
	return RecoveryObject{}
}
