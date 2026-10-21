package install

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/operationaldocs"
)

const (
	docsParentName      = ".axiom-docs"
	docsRootName        = "axiom"
	docsLockName        = "lock"
	docsGenerationsName = "generations"
	docsBindingsName    = "bindings"
	docsStagePrefix     = ".stage-"
)

// docsFault is the test seam for fault boundaries between publication steps.
var docsFault = func(string) error { return nil }

type docsStore struct {
	root, generations, bindings local.AnchoredDirectory
	path                        string
	locked                      bool
}

func docsFailure(cause error) error {
	return &Error{Category: "docs_publication_failed", cause: cause}
}

func (s *docsStore) Close() {
	if s.locked {
		_ = s.root.Remove(docsLockName)
	}
	s.bindings.Close()
	s.generations.Close()
	s.root.Close()
}

func openChild(parent local.AnchoredDirectory, parentPath, name string, create bool) (local.AnchoredDirectory, error) {
	if _, err := parent.Lstat(name); errors.Is(err, os.ErrNotExist) && create {
		if err := parent.Mkdir(name); err != nil && !errors.Is(err, os.ErrExist) {
			return local.AnchoredDirectory{}, err
		}
	} else if err != nil {
		return local.AnchoredDirectory{}, err
	}
	return local.OpenOwnedDirectory(filepath.Join(parentPath, name))
}

// openDocsStore opens the installation's private documentation namespace next
// to the binary. With create, missing private components are created; existing
// components are never repaired.
func openDocsStore(binaries local.AnchoredDirectory, binaryDir string, create bool) (docsStore, error) {
	var store docsStore
	parentPath := filepath.Join(binaryDir, docsParentName)
	parent, err := openChild(binaries, binaryDir, docsParentName, create)
	if err != nil {
		return store, err
	}
	defer parent.Close()
	store.path = filepath.Join(parentPath, docsRootName)
	if store.root, err = openChild(parent, parentPath, docsRootName, create); err != nil {
		return store, err
	}
	if store.generations, err = openChild(store.root, store.path, docsGenerationsName, create); err != nil {
		store.root.Close()
		return store, err
	}
	if store.bindings, err = openChild(store.root, store.path, docsBindingsName, create); err != nil {
		store.generations.Close()
		store.root.Close()
		return store, err
	}
	return store, nil
}

// lock serializes publishers of one installation, independent of receipt roots.
func (s *docsStore) lock() error {
	if err := s.root.Mkdir(docsLockName); err != nil {
		return &Error{Category: "installation_busy_or_interrupted", cause: err}
	}
	s.locked = true
	return nil
}

func (s docsStore) generationPath(generation string) string {
	return filepath.Join(s.path, docsGenerationsName, generation)
}

// generationState reports whether the exact bundle is published, absent, or
// conflicts with an existing different tree (which is preserved).
func (s docsStore) generationState(bundle *operationaldocs.Bundle) (string, error) {
	if _, err := s.generations.Lstat(bundle.Generation); errors.Is(err, os.ErrNotExist) {
		return "absent", nil
	} else if err != nil {
		return "", err
	}
	files, err := operationaldocs.ReadGeneration(s.generationPath(bundle.Generation))
	if err != nil {
		return "", err
	}
	if len(files) != len(bundle.Files) {
		return "conflict", nil
	}
	for name, wire := range bundle.Files {
		if !bytes.Equal(files[name], wire) {
			return "conflict", nil
		}
	}
	return "present", nil
}

func (s docsStore) publishGeneration(bundle *operationaldocs.Bundle) error {
	state, err := s.generationState(bundle)
	if err != nil {
		return docsFailure(err)
	}
	switch state {
	case "present":
		return nil
	case "conflict":
		return &Error{Category: "docs_generation_conflict"}
	}
	stageName, err := localStageName(s.generations)
	if err != nil {
		return docsFailure(err)
	}
	stagePath := s.generationPath(stageName)
	committed := false
	defer func() {
		if !committed {
			removeTree(stagePath)
		}
	}()
	names := make([]string, 0, len(bundle.Files))
	for name := range bundle.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	dirs := map[string]bool{}
	for _, name := range names {
		for dir := filepath.Dir(name); dir != "."; dir = filepath.Dir(dir) {
			dirs[dir] = true
		}
	}
	ordered := make([]string, 0, len(dirs))
	for dir := range dirs {
		ordered = append(ordered, dir)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return len(ordered[i]) < len(ordered[j]) || len(ordered[i]) == len(ordered[j]) && ordered[i] < ordered[j]
	})
	stage, err := local.OpenOwnedDirectory(stagePath)
	if err != nil {
		return docsFailure(err)
	}
	stage.Close()
	for _, dir := range ordered {
		parent, err := local.OpenOwnedDirectory(filepath.Join(stagePath, filepath.Dir(dir)))
		if err != nil {
			return docsFailure(err)
		}
		err = parent.Mkdir(filepath.Base(dir))
		parent.Close()
		if err != nil {
			return docsFailure(err)
		}
	}
	for _, name := range names {
		dir, err := local.OpenOwnedDirectory(filepath.Join(stagePath, filepath.Dir(name)))
		if err != nil {
			return docsFailure(err)
		}
		err = dir.CreateExclusive(filepath.Base(name), bundle.Files[name], 0o400)
		if err == nil {
			err = dir.Sync()
		}
		dir.Close()
		if err != nil {
			return docsFailure(err)
		}
	}
	for i := len(ordered) - 1; i >= 0; i-- {
		if err := os.Chmod(filepath.Join(stagePath, ordered[i]), 0o500); err != nil {
			return docsFailure(err)
		}
	}
	if err := os.Chmod(stagePath, 0o500); err != nil {
		return docsFailure(err)
	}
	if err := docsFault("generation_staged"); err != nil {
		return docsFailure(err)
	}
	if err := s.generations.PublishNew(stageName, bundle.Generation); err != nil {
		if errors.Is(err, os.ErrExist) {
			if state, checkErr := s.generationState(bundle); checkErr == nil && state == "present" {
				return nil
			}
			return &Error{Category: "docs_generation_conflict"}
		}
		return docsFailure(err)
	}
	committed = true
	if err := s.generations.Sync(); err != nil {
		return docsFailure(err)
	}
	return docsFault("generation_committed")
}

func localStageName(dir local.AnchoredDirectory) (string, error) {
	for attempt := 0; attempt < 8; attempt++ {
		name := fmt.Sprintf("%s%d-%d", docsStagePrefix, os.Getpid(), attempt)
		if err := dir.Mkdir(name); err == nil {
			return name, nil
		} else if !errors.Is(err, os.ErrExist) {
			return "", err
		}
	}
	return "", os.ErrExist
}

func removeTree(path string) {
	_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			_ = os.Chmod(p, 0o700)
		}
		return nil
	})
	_ = os.RemoveAll(path)
}

// docsBinding builds the immutable binding for the installed binary identity.
func docsBinding(candidate Candidate, binaryPath string, parent, binary os.FileInfo) (operationaldocs.Binding, []byte, string, error) {
	pdev, pino := fileIdentity(parent)
	bdev, bino := fileIdentity(binary)
	b := operationaldocs.Binding{SchemaVersion: 1, BinaryPath: binaryPath, ParentDevice: pdev, ParentInode: pino, BinaryDevice: bdev, BinaryInode: bino, BinarySHA256: digest(candidate.Binary), SourceCommit: candidate.DocsMetadata.SourceCommit, Version: candidate.Version, ArchiveSHA256: candidate.ArchiveSHA256, MetadataSHA256: candidate.MetadataSHA256, Generation: candidate.Docs.Generation, ProvenanceSHA256: candidate.Docs.ProvenanceSHA256, ManMapSHA256: candidate.Docs.ManMapSHA256, SkillManifestSHA256: candidate.SkillManifestSHA256}
	wire, err := b.Wire()
	if err != nil {
		return b, nil, "", err
	}
	if _, err := operationaldocs.ParseBinding(wire); err != nil {
		return b, nil, "", err
	}
	return b, wire, digest(wire), nil
}

// bindingState reports whether this exact binding file is published.
func (s docsStore) bindingState(binarySHA256, hash string, wire []byte) (string, error) {
	if _, err := s.bindings.Lstat(binarySHA256); errors.Is(err, os.ErrNotExist) {
		return "absent", nil
	} else if err != nil {
		return "", err
	}
	dir, err := local.OpenOwnedDirectory(filepath.Join(s.path, docsBindingsName, binarySHA256))
	if err != nil {
		return "", err
	}
	defer dir.Close()
	entries, err := dir.ReadDir()
	if err != nil {
		return "", err
	}
	if len(entries) >= operationaldocs.MaxBindings {
		return "", &Error{Category: "quota_exceeded"}
	}
	if _, err := dir.Lstat(hash + ".json"); errors.Is(err, os.ErrNotExist) {
		return "absent", nil
	}
	existing, err := dir.ReadFile(hash+".json", operationaldocs.MaxBinding)
	if err != nil {
		return "", err
	}
	if !bytes.Equal(existing, wire) {
		return "conflict", nil
	}
	return "present", nil
}

func (s docsStore) publishBinding(binarySHA256, hash string, wire []byte) error {
	state, err := s.bindingState(binarySHA256, hash, wire)
	if err != nil {
		return docsFailure(err)
	}
	switch state {
	case "present":
		return nil
	case "conflict":
		return &Error{Category: "docs_binding_conflict"}
	}
	if err := s.bindings.Mkdir(binarySHA256); err != nil && !errors.Is(err, os.ErrExist) {
		return docsFailure(err)
	}
	dir, err := local.OpenOwnedDirectory(filepath.Join(s.path, docsBindingsName, binarySHA256))
	if err != nil {
		return docsFailure(err)
	}
	defer dir.Close()
	if err := dir.CreateExclusive(hash+".json", wire, 0o400); err != nil {
		return docsFailure(err)
	}
	if err := errors.Join(dir.Sync(), s.bindings.Sync()); err != nil {
		return docsFailure(err)
	}
	return docsFault("binding_published")
}

func expectedReceiptV2(destination string, candidate Candidate, installedAt, bindingSHA256 string) []byte {
	v := candidate.Values
	meta := candidate.DocsMetadata
	lines := []string{
		"formatVersion=2", "destination=" + destination, "sha256=" + digest(candidate.Binary), "product=Axiom",
		"version=" + candidate.Version, "revision=" + v["revision"], "sourceState=" + v["sourceState"], "release=" + v["release"],
		"platform=" + v["platform"], "goos=" + v["goos"], "architecture=" + v["architecture"], "skillSetVersion=1",
		"archiveSha256=" + candidate.ArchiveSHA256, "skillManifestSha256=" + candidate.SkillManifestSHA256, "installedAt=" + installedAt,
		"sourceCommit=" + meta.SourceCommit, "docsGeneration=" + candidate.Docs.Generation, "docsBindingSha256=" + bindingSHA256,
		"docsInventorySha256=" + candidate.Docs.InventorySHA256, "docsProvenanceSha256=" + candidate.Docs.ProvenanceSHA256, "docsManMapSha256=" + candidate.Docs.ManMapSHA256,
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func fileIdentity(info os.FileInfo) (uint64, uint64) {
	stat := info.Sys().(*syscall.Stat_t)
	return uint64(stat.Dev), uint64(stat.Ino)
}

const pendingBinding = "at_binary_commit"

func hasDocsEffect(effects []Effect) bool {
	for _, effect := range effects {
		if strings.HasPrefix(effect.Kind, "docs_") || effect.Kind == "binary" {
			return true
		}
	}
	return false
}

func installedBinding(candidate Candidate, target Target, binaryDir local.AnchoredDirectory) (operationaldocs.Binding, []byte, string, error) {
	binary, err := binaryDir.Lstat(binaryName)
	if err != nil {
		return operationaldocs.Binding{}, nil, "", err
	}
	parent, err := os.Lstat(target.BinaryDir)
	if err != nil {
		return operationaldocs.Binding{}, nil, "", err
	}
	return docsBinding(candidate, filepath.Join(target.BinaryDir, binaryName), parent, binary)
}

func publishInstalledBinding(store docsStore, candidate Candidate, target Target, binaryDir local.AnchoredDirectory) error {
	_, wire, hash, err := installedBinding(candidate, target, binaryDir)
	if err != nil {
		return &Error{Category: "unsafe_binary"}
	}
	return store.publishBinding(digest(candidate.Binary), hash, wire)
}

// planDocs plans the documentation effects before the binary effect: a missing
// generation, and a missing binding when the installed binary stays.
func planDocs(target Target, binaryDir local.AnchoredDirectory, candidate Candidate, destination string, binaryUnchanged bool) ([]Effect, int64, error) {
	var effects []Effect
	var needed int64
	store, openErr := openDocsStore(binaryDir, target.BinaryDir, false)
	namespace := openErr == nil
	if openErr != nil && !errors.Is(openErr, os.ErrNotExist) {
		return nil, 0, &Error{Category: "docs_namespace_unsafe", cause: openErr}
	}
	if namespace {
		defer store.Close()
	}
	generationTarget := filepath.Join(target.BinaryDir, docsParentName, docsRootName, docsGenerationsName, candidate.Docs.Generation)
	generation := "absent"
	if namespace {
		var err error
		if generation, err = store.generationState(candidate.Docs); err != nil {
			return nil, 0, &Error{Category: "docs_namespace_unsafe", cause: err}
		}
	}
	switch generation {
	case "conflict":
		return nil, 0, &Error{Category: "docs_generation_conflict"}
	case "absent":
		effects = append(effects, Effect{Kind: "docs_generation", Target: generationTarget, Expected: absentRevision, Next: candidate.Docs.Generation})
		needed += int64(candidate.Docs.Bytes)
	}
	bindingTarget := filepath.Join(target.BinaryDir, docsParentName, docsRootName, docsBindingsName, digest(candidate.Binary))
	if !binaryUnchanged {
		return append(effects, Effect{Kind: "docs_binding", Target: bindingTarget, Expected: absentRevision, Next: pendingBinding}), needed, nil
	}
	_, wire, hash, err := installedBinding(candidate, target, binaryDir)
	if err != nil {
		return nil, 0, &Error{Category: "unsafe_binary"}
	}
	state := "absent"
	if namespace {
		if state, err = store.bindingState(digest(candidate.Binary), hash, wire); err != nil {
			return nil, 0, &Error{Category: "docs_namespace_unsafe", cause: err}
		}
	}
	switch state {
	case "conflict":
		return nil, 0, &Error{Category: "docs_binding_conflict"}
	case "absent":
		effects = append(effects, Effect{Kind: "docs_binding", Target: bindingTarget, Expected: absentRevision, Next: hash})
	}
	return effects, needed, nil
}

// verifyDocsInstallation confirms the complete v2 outcome from disk.
func verifyDocsInstallation(store docsStore, candidate Candidate, target Target, binaryDir, receiptDir local.AnchoredDirectory) error {
	_, wire, hash, err := installedBinding(candidate, target, binaryDir)
	if err != nil {
		return err
	}
	if state, err := store.generationState(candidate.Docs); err != nil || state != "present" {
		return errors.Join(err, errDocsDrift)
	}
	if state, err := store.bindingState(digest(candidate.Binary), hash, wire); err != nil || state != "present" {
		return errors.Join(err, errDocsDrift)
	}
	receipt, err := receiptDir.ReadFile(receiptName, maxReceiptBytes)
	if err != nil {
		return err
	}
	values, err := parseReceipt(receipt)
	if err != nil || !bytes.Equal(receipt, expectedReceiptV2(filepath.Join(target.BinaryDir, binaryName), candidate, values["installedAt"], hash)) {
		return errors.Join(err, errDocsDrift)
	}
	return nil
}

var errDocsDrift = errors.New("documentation installation drift")

func receiptValuesV2(wire []byte) (map[string]string, error) {
	if _, err := operationaldocs.ParseReceipt(wire); err != nil {
		return nil, errors.New("receipt schema")
	}
	values := map[string]string{}
	for _, line := range strings.Split(strings.TrimSuffix(string(wire), "\n"), "\n") {
		key, value, _ := strings.Cut(line, "=")
		values[key] = value
	}
	return values, nil
}

// verifyDocsInstallationReadOnly decides whether a reinstall of the same v2
// candidate is a true no-op: receipt, binding and generation all exact.
func verifyDocsInstallationReadOnly(binaries, receipts local.AnchoredDirectory, target Target, candidate Candidate) error {
	store, err := openDocsStore(binaries, target.BinaryDir, false)
	if err != nil {
		return err
	}
	defer store.Close()
	return verifyDocsInstallation(store, candidate, target, binaries, receipts)
}
