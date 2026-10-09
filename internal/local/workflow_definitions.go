package local

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type WorkflowEntry = workflowdefinition.Entry
type WorkflowIndex = workflowdefinition.Index
type WorkflowCatalog = workflowdefinition.Catalog

func workflowName(id string, revision int) string { return id + "/" + strconv.Itoa(revision) + ".json" }
func EmptyWorkflowCatalog() WorkflowCatalog {
	return WorkflowCatalog{Index: WorkflowIndex{SchemaVersion: 1, Revisions: []WorkflowEntry{}}, Documents: map[string]workflowdefinition.Document{}}
}
func DecodeWorkflowIndex(wire []byte) (WorkflowIndex, error) {
	var idx WorkflowIndex
	if workflowdefinition.StrictJSON(wire, &idx) != nil || idx.SchemaVersion != 1 || idx.Revisions == nil || len(idx.Revisions) > 1024 {
		return idx, ErrUnsafe
	}
	// encoding/json accepts case-insensitive struct aliases; the portable index
	// has an exact closed shape, so check its original member names as well.
	var shape map[string]json.RawMessage
	if err := json.Unmarshal(wire, &shape); err != nil || len(shape) != 2 || shape["schemaVersion"] == nil || shape["revisions"] == nil {
		return idx, ErrUnsafe
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(shape["revisions"], &entries); err != nil {
		return idx, ErrUnsafe
	}
	for _, entry := range entries {
		if len(entry) != 4 || entry["workflowId"] == nil || entry["revision"] == nil || entry["digest"] == nil || entry["state"] == nil {
			return idx, ErrUnsafe
		}
	}
	previous := WorkflowEntry{}
	for i, e := range idx.Revisions {
		if !e.Ref().Valid() || (e.State != "published" && e.State != "retired") || (i > 0 && (e.WorkflowID < previous.WorkflowID || e.WorkflowID == previous.WorkflowID && e.Revision <= previous.Revision)) {
			return idx, ErrUnsafe
		}
		previous = e
	}
	return idx, nil
}
func EncodeWorkflowIndex(idx WorkflowIndex) ([]byte, error) {
	sort.Slice(idx.Revisions, func(i, j int) bool {
		a, b := idx.Revisions[i], idx.Revisions[j]
		if a.WorkflowID != b.WorkflowID {
			return a.WorkflowID < b.WorkflowID
		}
		return a.Revision < b.Revision
	})
	b, e := json.Marshal(idx)
	if e != nil {
		return nil, e
	}
	if _, e = DecodeWorkflowIndex(b); e != nil {
		return nil, e
	}
	return append(b, '\n'), nil
}

// readWorkflowCatalog validates the complete bounded tree without following
// links. An orphan is reported explicitly, never selectable or silently adopted.
func readWorkflowCatalog(projectRoot *os.Root) (WorkflowCatalog, error) {
	c := EmptyWorkflowCatalog()
	wf, e := existingPrivateChild(projectRoot, "workflows")
	if errors.Is(e, ErrNotFound) || os.IsNotExist(e) {
		return c, nil
	}
	if e != nil {
		return c, e
	}
	defer wf.Close()
	if pending, e := protocolStatePresent(wf); e != nil || pending {
		return c, ErrRecoveryRequired
	}
	names, e := readDirectoryNamesBounded(wf, 1025)
	if e != nil {
		return c, e
	}
	c.Wire, e = readPrivateFileBounded(wf, "index.json", workflowdefinition.MaxBytes)
	if os.IsNotExist(e) {
		c.Wire = nil
	} else if e != nil {
		return c, e
	} else if c.Index, e = DecodeWorkflowIndex(c.Wire); e != nil {
		return c, e
	}
	indexed := map[string]WorkflowEntry{}
	for _, entry := range c.Index.Revisions {
		indexed[workflowName(entry.WorkflowID, entry.Revision)] = entry
	}
	count := 0
	for _, id := range names {
		if id == "index.json" {
			continue
		}
		if !workflowdefinition.ValidKey(id) {
			return c, ErrUnsafe
		}
		dir, e := existingPrivateChild(wf, id)
		if e != nil {
			return c, e
		}
		if pending, e := protocolStatePresent(dir); e != nil || pending {
			dir.Close()
			return c, ErrRecoveryRequired
		}
		files, e := readDirectoryNamesBounded(dir, 1024)
		if e != nil {
			dir.Close()
			return c, e
		}
		for _, name := range files {
			count++
			if count > 1024 {
				dir.Close()
				return c, ErrCapacity
			}
			n, e := strconv.Atoi(name[:max(0, len(name)-5)])
			if e != nil || name != strconv.Itoa(n)+".json" || n < 1 {
				dir.Close()
				return c, ErrUnsafe
			}
			b, e := readPrivateFileBounded(dir, name, workflowdefinition.MaxBytes)
			if e != nil {
				dir.Close()
				return c, e
			}
			doc, issues := workflowdefinition.Decode(b)
			if len(issues) != 0 || doc.Definition.WorkflowID != id || doc.Definition.Revision != n {
				dir.Close()
				return c, ErrUnsafe
			}
			key := workflowName(id, n)
			entry, ok := indexed[key]
			if ok && entry.Digest != doc.Digest {
				dir.Close()
				return c, ErrUnsafe
			}
			c.Documents[key] = doc
			if !ok {
				c.Unindexed = append(c.Unindexed, doc.Ref("project"))
			}
		}
		dir.Close()
	}
	for key, entry := range indexed {
		if _, ok := c.Documents[key]; !ok && entry.State == "published" {
			return c, ErrRecoveryRequired
		}
	}
	return c, nil
}

// ValidatePortableLayout extends the historical one-manifest layout only with
// the strict companion tree. Unknown files still fail closed.
func validatePortableLayout(root *os.Root) error {
	names, e := readDirectoryNamesBounded(root, 2)
	if e != nil {
		return e
	}
	found := false
	for _, name := range names {
		switch name {
		case manifestName:
			found = true
		case "workflows":
		default:
			return ErrUnsafe
		}
	}
	if !found {
		return ErrUnsafe
	}
	c, e := readWorkflowCatalog(root)
	if e != nil {
		return e
	}
	if len(c.Unindexed) > 0 {
		return ErrRecoveryRequired
	}
	wire, e := readPrivateFile(root, manifestName)
	if e != nil {
		return e
	}
	return validateWorkflowSelection(wire, c)
}

func validateWorkflowSelection(wire []byte, c WorkflowCatalog) error {
	p, issues := manifest.Decode(wire)
	// Preserve historical store behavior: generic byte CAS does not validate
	// manifests. Selection integrity applies only to decoded schema 4 intent.
	if len(issues) != 0 {
		return nil
	}
	selected, ok := p.State().WorkflowSelection.Value()
	if !ok {
		return nil
	}
	ref := workflowdefinition.Ref{WorkflowID: selected.WorkflowID, Revision: selected.Revision, Digest: selected.Digest, Source: selected.Source}
	if ref.Source == "builtin" {
		if ref != workflowdefinition.Builtin().Ref("builtin") {
			return ErrUnsafe
		}
		return nil
	}
	for _, entry := range c.Index.Revisions {
		if entry.Ref() == ref && entry.State == "published" {
			return nil
		}
	}
	return ErrUnsafe
}

func validateNextWorkflowSelection(root *os.Root, wire []byte) error {
	c, e := readWorkflowCatalog(root)
	if e != nil {
		return e
	}
	return validateWorkflowSelection(wire, c)
}

func (s PortableStore) WorkflowCatalog(ctx context.Context, slug string) (WorkflowCatalog, error) {
	c := EmptyWorkflowCatalog()
	e := s.withLock(slug, false, false, func(root *os.Root) error {
		target, e := existingPrivateChild(root, slug)
		if e != nil {
			return e
		}
		defer target.Close()
		if pending, e := protocolStatePresent(target); e != nil || pending {
			return ErrRecoveryRequired
		}
		c, e = readWorkflowCatalog(target)
		return e
	})
	return c, e
}

// InspectWorkflows permits explicit inspection/recovery of an unindexed
// publication while retaining the closed portable root and manifest checks.
func (s PortableStore) InspectWorkflows(ctx context.Context, slug string) (projectapp.ArtifactSnapshot, WorkflowCatalog, error) {
	var snapshot projectapp.ArtifactSnapshot
	catalog := EmptyWorkflowCatalog()
	e := s.withLock(slug, false, false, func(root *os.Root) error {
		target, e := existingPrivateChild(root, slug)
		if e != nil {
			return e
		}
		defer target.Close()
		names, e := readDirectoryNamesBounded(target, 2)
		if e != nil {
			return e
		}
		for _, name := range names {
			if name != manifestName && name != "workflows" {
				return ErrRecoveryRequired
			}
		}
		wire, e := readPrivateFile(target, manifestName)
		if e != nil {
			return e
		}
		var issues []projectapp.Issue
		snapshot, issues = projectapp.ReadSnapshot(manifest.Codec{}, wire, nil)
		if len(issues) != 0 {
			return ErrUnsafe
		}
		catalog, e = readWorkflowCatalog(target)
		return e
	})
	return snapshot, catalog, e
}

// PublishWorkflow uses the existing ADR-0007 file protocol for content and
// index, in that order, under the portable namespace lock. Each confirmed
// content assignment is immutable. A replay never overwrites content.
func (s PortableStore) PublishWorkflow(ctx context.Context, slug string, expectedManifest, expectedIndex []byte, doc *workflowdefinition.Document, nextIndex []byte) error {
	return s.withLock(slug, false, true, func(root *os.Root) error {
		target, e := existingPrivateChild(root, slug)
		if e != nil {
			return e
		}
		defer target.Close()
		current, e := readPrivateFile(target, manifestName)
		if e != nil {
			return e
		}
		if !bytes.Equal(current, expectedManifest) {
			return ErrConflict
		}
		catalog, e := readWorkflowCatalog(target)
		if e != nil {
			return e
		}
		if !bytes.Equal(catalog.Wire, expectedIndex) {
			return ErrConflict
		}
		idx, e := DecodeWorkflowIndex(nextIndex)
		if e != nil {
			return e
		}
		// Store boundary enforces append-only identity, even if a caller bypasses
		// the application. Retirement can never resurrect or reassign a revision.
		for _, old := range catalog.Index.Revisions {
			found := false
			for _, next := range idx.Revisions {
				if old.WorkflowID == next.WorkflowID && old.Revision == next.Revision {
					found = true
					if old.Digest != next.Digest || old.State == "retired" && next.State != "retired" {
						return ErrConflict
					}
				}
			}
			if !found {
				return ErrConflict
			}
		}
		for _, next := range idx.Revisions {
			known := false
			maximum := 0
			for _, old := range catalog.Index.Revisions {
				if old.WorkflowID == next.WorkflowID {
					if old.Revision > maximum {
						maximum = old.Revision
					}
					known = known || old.Revision == next.Revision
				}
			}
			if !known && (doc == nil || next.Ref() != doc.Ref("project") || next.Revision <= maximum || next.State != "published") {
				return ErrConflict
			}
		}
		// Recheck selection under the same namespace lock as every selector writer.
		candidate := catalog
		candidate.Index = idx
		if e = validateWorkflowSelection(current, candidate); e != nil {
			return ErrConflict
		}
		wf, e := privateChild(target, "workflows")
		if e != nil {
			return e
		}
		defer wf.Close()
		hooks := s.publicationHooks()
		hooks.contentLimit = workflowdefinition.MaxBytes
		hooks.fault = s.workflowFault
		hooks.stagePrefix = ".axiom-stage-file-"
		fence := func() error {
			if s.workflowBeforeCommit != nil {
				if e := s.workflowBeforeCommit(); e != nil {
					return e
				}
			}
			for _, check := range []struct {
				handle *os.Root
				path   string
			}{{root, s.root}, {target, filepath.Join(s.root, slug)}, {wf, filepath.Join(s.root, slug, "workflows")}} {
				if e := stillAtPath(check.handle, check.path); e != nil {
					return e
				}
			}
			return nil
		}
		hooks.beforeCommit = fence
		if doc != nil {
			verified, issues := workflowdefinition.Decode(doc.Canonical)
			if len(issues) != 0 || verified.Ref("project") != doc.Ref("project") {
				return ErrUnsafe
			}
			dir, e := privateChild(wf, doc.Definition.WorkflowID)
			if e != nil {
				return e
			}
			defer dir.Close()
			name := strconv.Itoa(doc.Definition.Revision) + ".json"
			previous, e := readPrivateFileBounded(dir, name, workflowdefinition.MaxBytes)
			contentHooks := hooks
			contentHooks.beforeCommit = func() error {
				if e := fence(); e != nil {
					return e
				}
				return stillAtPath(dir, filepath.Join(s.root, slug, "workflows", doc.Definition.WorkflowID))
			}
			if e == nil {
				old, issues := workflowdefinition.Decode(previous)
				if len(issues) != 0 || old.Digest != doc.Digest {
					return ErrConflict
				}
			} else if os.IsNotExist(e) {
				if e = publishFile(ctx, dir, name, nil, doc.Canonical, true, contentHooks); e != nil {
					return e
				}
			} else {
				return e
			}
		}
		if e = publishFile(ctx, wf, "index.json", expectedIndex, nextIndex, len(expectedIndex) == 0, hooks); e != nil {
			if doc != nil {
				return &PublicationError{Committed: true, Err: e}
			}
			return e
		}
		observed, e := readWorkflowCatalog(target)
		if e != nil || len(observed.Unindexed) != 0 {
			return ErrRecoveryRequired
		}
		return nil
	})
}

// WithWorkflowIndex binds a selection publication to the catalog reviewed in
// its preview. The comparison occurs under the portable writer lock.
func (s PortableStore) WithWorkflowIndex(expected []byte) PortableStore {
	wire := append([]byte(nil), expected...)
	s.workflowExpectedIndex = &wire
	return s
}

// PublishWorkflow holds the same local chain locks as Project edits before
// acquiring the portable lock, preventing a binding edit from racing authority.
func (s InstallationStore) PublishWorkflow(ctx context.Context, portable PortableStore, projectID, slug string, expectedLocal, expectedManifest, expectedIndex []byte, doc *workflowdefinition.Document, nextIndex []byte) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	record, _, issues := DecodeObservedRecord(expectedLocal, true)
	if len(issues) != 0 || record.State().ProjectID != projectID || record.State().ObservedSlug != slug || record.State().SourceLocation != filepath.Join(portable.root, slug) {
		return ErrUnsafe
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
	if category := installationDirectoryIssue(target); category != "" {
		return ErrRecoveryRequired
	}
	check := func() error {
		for _, entry := range []struct {
			handle *os.Root
			path   string
		}{{root, s.root}, {projects, filepath.Join(s.root, "projects")}, {target, filepath.Join(s.root, "projects", projectID)}} {
			if err := stillAtPath(entry.handle, entry.path); err != nil {
				return err
			}
		}
		current, err := readPrivateFile(target, installationRecord)
		if err != nil {
			return err
		}
		if !bytes.Equal(current, expectedLocal) {
			return ErrConflict
		}
		return nil
	}
	if err := check(); err != nil {
		return err
	}
	portable.workflowBeforeCommit = check
	return portable.PublishWorkflow(ctx, slug, expectedManifest, expectedIndex, doc, nextIndex)
}
