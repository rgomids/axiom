package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"

	"github.com/rgomids/axiom/internal/executiongraph"
)

// CommitRetry publishes one immutable, content-addressed retry receipt beside
// the successor graph it consumes. The graph is not written: the receipt binds
// the exact graph storage revision observed under the same exclusive locks,
// so the retry attempt's own save (or any other write) makes it unusable.
func (s GraphStore) CommitRetry(ctx context.Context, g executiongraph.SuccessorGraph, a executiongraph.RetryAuthority) (string, error) {
	if _, err := executiongraph.CheckRetryDispatch(g, a); err != nil {
		return "", err
	}
	wire, err := executiongraph.EncodeRetryAuthority(a)
	if err != nil {
		return "", err
	}
	return s.commitGraphReceipt(ctx, g, a.ProjectID, a.ParentID, wire)
}

// CommitIntegrationReceipt publishes the single-use local authority of one
// integration attempt under the same protocol as retry receipts.
func (s GraphStore) CommitIntegrationReceipt(ctx context.Context, g executiongraph.SuccessorGraph, a executiongraph.IntegrationReceipt) (string, error) {
	if err := executiongraph.CheckIntegrationDispatch(g, a); err != nil {
		return "", err
	}
	wire, err := executiongraph.EncodeIntegrationReceipt(a)
	if err != nil {
		return "", err
	}
	return s.commitGraphReceipt(ctx, g, a.ProjectID, a.ParentID, wire)
}

// CommitDispatchInputs publishes the inert reviewed preview before the
// dispatch commit point. It never writes the graph or grants authority; an
// orphan left by an interrupted commit is recognized and inert.
func (s GraphStore) CommitDispatchInputs(ctx context.Context, d executiongraph.DispatchInputs) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	wire, err := executiongraph.EncodeDispatchInputs(d)
	if err != nil {
		return "", err
	}
	root, graphs, version, project, err := s.openProject(d.ProjectID, true)
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
	return commitInertInputs(ctx, project, wire, s.hooks)
}

func commitInertInputs(ctx context.Context, project *os.Root, wire []byte, base publicationHooks) (string, error) {
	ref := hexDigest(wire)
	existing, err := readPrivateFileBounded(project, ref+".json", executiongraph.MaxDispatchInputsBytes)
	switch {
	case err == nil:
		if !bytes.Equal(existing, wire) {
			return "", ErrUnsafe
		}
		return ref, nil
	case !errors.Is(err, ErrNotFound) && !errors.Is(err, os.ErrNotExist):
		return "", err
	}
	hooks := base
	hooks.contentLimit = executiongraph.MaxDispatchInputsBytes
	if err := publishFile(ctx, project, ref+".json", nil, wire, true, hooks); err != nil {
		return "", err
	}
	return ref, nil
}

// LoadDispatchInputs finds the inputs record of one graph and preview digest.
// Callers must still verify the preview's own digest and the committed
// authority before using any content.
func (s GraphStore) LoadDispatchInputs(ctx context.Context, projectID, parentID, previewDigest string) (executiongraph.DispatchInputs, error) {
	if err := ctx.Err(); err != nil {
		return executiongraph.DispatchInputs{}, err
	}
	if !validGraphAddress(projectID, parentID) {
		return executiongraph.DispatchInputs{}, ErrUnsafe
	}
	root, graphs, version, project, err := s.openProject(projectID, false)
	if err != nil {
		return executiongraph.DispatchInputs{}, err
	}
	defer root.Close()
	defer graphs.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(false, root, graphs, version, project)
	if err != nil {
		return executiongraph.DispatchInputs{}, err
	}
	defer closeFiles(locks)
	return findDispatchInputs(project, projectID, parentID, previewDigest)
}

func findDispatchInputs(project *os.Root, projectID, parentID, previewDigest string) (executiongraph.DispatchInputs, error) {
	pending, err := protocolStatePresent(project)
	if err != nil {
		return executiongraph.DispatchInputs{}, err
	}
	if pending {
		return executiongraph.DispatchInputs{}, ErrRecoveryRequired
	}
	names, err := readDirectoryNamesBounded(project, maxLocalDirectoryEntries)
	if err != nil {
		return executiongraph.DispatchInputs{}, err
	}
	for _, name := range names {
		if !hexDigestName.MatchString(name) {
			continue
		}
		wire, err := readPrivateFileBounded(project, name, executiongraph.MaxDispatchInputsBytes)
		if err != nil {
			return executiongraph.DispatchInputs{}, err
		}
		d, err := executiongraph.DecodeDispatchInputs(wire)
		if err != nil || d.ParentID != parentID || d.PreviewDigest != previewDigest {
			continue
		}
		if hexDigest(wire)+".json" != name || d.ProjectID != projectID {
			return executiongraph.DispatchInputs{}, ErrUnsafe
		}
		return d, nil
	}
	return executiongraph.DispatchInputs{}, ErrNotFound
}

// FindStageGraph resolves the committed successor graph of one Execution
// stage from inert dispatch-inputs records whose graph was committed. Orphan
// inputs of an interrupted commit are ignored; more than one committed graph
// for the same Execution stage is ambiguous and fails closed.
func (s GraphStore) FindStageGraph(ctx context.Context, projectID, executionID, stageID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	root, graphs, version, project, err := s.openProject(projectID, false)
	if errors.Is(err, ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	defer root.Close()
	defer graphs.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(false, root, graphs, version, project)
	if err != nil {
		return "", err
	}
	defer closeFiles(locks)
	names, err := readDirectoryNamesBounded(project, maxLocalDirectoryEntries)
	if err != nil {
		return "", err
	}
	present := map[string]bool{}
	for _, name := range names {
		present[name] = true
	}
	found := map[string]bool{}
	for _, name := range names {
		if !hexDigestName.MatchString(name) {
			continue
		}
		wire, err := readPrivateFileBounded(project, name, executiongraph.MaxDispatchInputsBytes)
		if err != nil {
			return "", err
		}
		d, err := executiongraph.DecodeDispatchInputs(wire)
		if err != nil || d.ProjectID != projectID || !present[graphName(d.ParentID)] {
			continue
		}
		var identity struct {
			Plan struct {
				Execution struct {
					ExecutionID string `json:"executionId"`
				} `json:"executionRef"`
				StageID string `json:"stageId"`
			} `json:"stagePlan"`
		}
		if json.Unmarshal(d.Preview, &identity) != nil {
			continue
		}
		if identity.Plan.Execution.ExecutionID == executionID && identity.Plan.StageID == stageID {
			found[d.ParentID] = true
		}
	}
	if len(found) == 0 {
		return "", ErrNotFound
	}
	if len(found) > 1 {
		return "", ErrRecoveryRequired
	}
	for parent := range found {
		return parent, nil
	}
	return "", ErrNotFound
}

func classifyDispatchInputs(projectID, file string, wire []byte) InventoryKind {
	d, err := executiongraph.DecodeDispatchInputs(wire)
	if err != nil {
		return ""
	}
	if d.ProjectID != projectID || hexDigest(wire)+".json" != file {
		return InventoryMalformed
	}
	return InventoryDispatchInputs
}

func (s GraphStore) commitGraphReceipt(ctx context.Context, g executiongraph.SuccessorGraph, projectID, parentID string, wire []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ref := hexDigest(wire)
	if ref == g.DispatchAuthorityRef {
		return "", executiongraph.ErrInvalidGraph
	}
	root, graphs, version, project, err := s.openProject(projectID, false)
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
	current, err := readSuccessorPublishedFile(project, graphName(parentID))
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
	wire, err := s.loadGraphReceipt(ctx, projectID, parentID, ref)
	if err != nil {
		return executiongraph.RetryAuthority{}, err
	}
	a, err := executiongraph.DecodeRetryAuthority(wire)
	if err != nil || a.ProjectID != projectID || a.ParentID != parentID {
		return executiongraph.RetryAuthority{}, ErrUnsafe
	}
	return a, nil
}

// LoadIntegrationReceipt resolves an integration receipt by its exact digest.
func (s GraphStore) LoadIntegrationReceipt(ctx context.Context, projectID, parentID, ref string) (executiongraph.IntegrationReceipt, error) {
	wire, err := s.loadGraphReceipt(ctx, projectID, parentID, ref)
	if err != nil {
		return executiongraph.IntegrationReceipt{}, err
	}
	a, err := executiongraph.DecodeIntegrationReceipt(wire)
	if err != nil || a.ProjectID != projectID || a.ParentID != parentID {
		return executiongraph.IntegrationReceipt{}, ErrUnsafe
	}
	return a, nil
}

func (s GraphStore) loadGraphReceipt(ctx context.Context, projectID, parentID, ref string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !validGraphAddress(projectID, parentID) || !hexDigestName.MatchString(ref+".json") {
		return nil, ErrUnsafe
	}
	root, graphs, version, project, err := s.openProject(projectID, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	defer graphs.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(false, root, graphs, version, project)
	if err != nil {
		return nil, err
	}
	defer closeFiles(locks)
	wire, err := readPublishedFile(project, ref+".json")
	if err != nil {
		return nil, err
	}
	if hexDigest(wire) != ref {
		return nil, ErrUnsafe
	}
	return wire, nil
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

func classifyIntegrationReceipt(projectID, file string, wire []byte) InventoryKind {
	a, err := executiongraph.DecodeIntegrationReceipt(wire)
	if err != nil {
		return ""
	}
	if a.ProjectID != projectID || hexDigest(wire)+".json" != file {
		return InventoryMalformed
	}
	return InventoryIntegrationAuthority
}
