package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"os"

	"github.com/rgomids/axiom/internal/executiongraph"
)

// CommitRetry publishes one immutable, content-addressed retry receipt beside
// the successor graph it consumes. The graph is not written: the receipt binds
// the exact graph storage revision observed under the same exclusive locks,
// so the retry attempt's own save (or any other write) makes it unusable.
func (s GraphStore) CommitRetry(ctx context.Context, g executiongraph.SuccessorGraph, a executiongraph.RetryAuthority) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if _, err := executiongraph.CheckRetryDispatch(g, a); err != nil {
		return "", err
	}
	wire, err := executiongraph.EncodeRetryAuthority(a)
	if err != nil {
		return "", err
	}
	ref := hexDigest(wire)
	if ref == g.DispatchAuthorityRef {
		return "", executiongraph.ErrInvalidGraph
	}
	root, graphs, version, project, err := s.openProject(a.ProjectID, false)
	if err != nil {
		return "", err
	}
	defer root.Close()
	defer graphs.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(true, root, graphs, version, project)
	if err != nil {
		return "", err
	}
	defer closeFiles(locks)
	current, err := readSuccessorPublishedFile(project, graphName(a.ParentID))
	if err != nil {
		return "", err
	}
	if sha256.Sum256(current) != g.StorageRevision {
		return "", ErrConflict
	}
	existing, err := readPublishedFile(project, ref+".json")
	switch {
	case err == nil:
		// Content addressing makes a repeated identical commit a read-back.
		if !bytes.Equal(existing, wire) {
			return "", ErrUnsafe
		}
		return ref, nil
	case !errors.Is(err, ErrNotFound) && !errors.Is(err, os.ErrNotExist):
		return "", err
	}
	hooks := s.hooks
	hooks.contentLimit = executiongraph.MaxRetryAuthorityBytes
	if err := publishFile(ctx, project, ref+".json", nil, wire, true, hooks); err != nil {
		return "", err
	}
	confirmed, err := readPrivateFileBounded(project, ref+".json", executiongraph.MaxRetryAuthorityBytes)
	if err != nil || !bytes.Equal(confirmed, wire) {
		return "", &PublicationError{Stage: FaultF8, Committed: true, Err: ErrRecoveryRequired}
	}
	return ref, nil
}

// LoadRetry resolves a retry receipt only by its exact digest and only for
// the named graph. Callers must still apply CheckRetryDispatch under the lease.
func (s GraphStore) LoadRetry(ctx context.Context, projectID, parentID, ref string) (executiongraph.RetryAuthority, error) {
	if err := ctx.Err(); err != nil {
		return executiongraph.RetryAuthority{}, err
	}
	if !validGraphAddress(projectID, parentID) || !hexDigestName.MatchString(ref+".json") {
		return executiongraph.RetryAuthority{}, ErrUnsafe
	}
	root, graphs, version, project, err := s.openProject(projectID, false)
	if err != nil {
		return executiongraph.RetryAuthority{}, err
	}
	defer root.Close()
	defer graphs.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(false, root, graphs, version, project)
	if err != nil {
		return executiongraph.RetryAuthority{}, err
	}
	defer closeFiles(locks)
	pending, err := protocolStatePresent(project)
	if err != nil {
		return executiongraph.RetryAuthority{}, err
	}
	if pending {
		return executiongraph.RetryAuthority{}, ErrRecoveryRequired
	}
	wire, err := readPublishedFile(project, ref+".json")
	if err != nil {
		return executiongraph.RetryAuthority{}, err
	}
	a, err := executiongraph.DecodeRetryAuthority(wire)
	if err != nil || hexDigest(wire) != ref || a.ProjectID != projectID || a.ParentID != parentID {
		return executiongraph.RetryAuthority{}, ErrUnsafe
	}
	return a, nil
}

func classifyRetryAuthority(projectID, file string, wire []byte) InventoryKind {
	a, err := executiongraph.DecodeRetryAuthority(wire)
	if err != nil {
		return ""
	}
	if a.ProjectID != projectID || hexDigest(wire)+".json" != file {
		return InventoryMalformed
	}
	return InventoryRetryAuthority
}
