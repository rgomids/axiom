package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RetireHistoricalStateObject removes only the exact preserved generation of a
// POC workflow or Work Item link. The caller must first prove preservation and
// authorize this entry. Coordination uses the owning stores' broad-to-narrow
// state -> family -> Project locks, held through deletion and empty-container
// cleanup. A confirmed concurrent writer therefore invalidates a stale revision
// or conflicts before publication; its generation cannot be deleted here.
func RetireHistoricalStateObject(ctx context.Context, state string, entry InventoryEntry) error {
	return retireHistoricalStateObject(ctx, state, entry, publicationHooks{})
}

func retireHistoricalStateObject(ctx context.Context, state string, entry InventoryEntry, hooks publicationHooks) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	parts := strings.Split(entry.Relative, "/")
	if !filepath.IsAbs(state) || filepath.Clean(state) == string(filepath.Separator) || len(parts) != 3 || !validUUID(parts[1]) || !safeEntryName(parts[2]) || strings.HasPrefix(parts[2], ".") || !strings.HasSuffix(parts[2], ".json") || !validDigest(entry.Digest) || entry.Bytes <= 0 || entry.Bytes > MaxRecordBytes {
		return ErrUnsafe
	}
	if entry.Kind != InventoryWorkItem || parts[0] != "work-items" {
		if entry.Kind != InventoryPOCWorkflow || parts[0] != "workflows" {
			return ErrUnsafe
		}
	}
	directories := make([]AnchoredDirectory, 0, 3)
	defer func() {
		for index := len(directories) - 1; index >= 0; index-- {
			if directories[index].root != nil {
				_ = directories[index].Close()
			}
		}
	}()
	for _, path := range []string{state, filepath.Join(state, parts[0]), filepath.Join(state, parts[0], parts[1])} {
		directory, err := OpenOwnedDirectory(path)
		if err != nil {
			return err
		}
		directories = append(directories, directory)
	}
	locks, err := lockRoots(true, directories[0].root, directories[1].root, directories[2].root)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	for _, directory := range directories {
		if err := directory.StillAtPath(); err != nil {
			return err
		}
	}
	target := directories[2]
	name := parts[2]
	info, err := target.Lstat(name)
	if err != nil {
		return err
	}
	wire, err := readPublishedFile(target.root, name)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(wire)
	if hex.EncodeToString(digest[:]) != entry.Digest || int64(len(wire)) != entry.Bytes {
		return ErrConflict
	}
	if entry.Kind == InventoryWorkItem {
		link, err := decodeWorkItem(wire)
		if err != nil || link.ProjectID != parts[1] || name != legacyWorkItemName(link.RepositoryKey, link.ExternalID) && name != workItemName(link.RepositoryKey, link.Provider, link.Resource, link.ExternalID) {
			return ErrUnsafe
		}
	} else {
		var workflow pocWorkflowDTO
		if !validPOCWorkflow(wire) || json.Unmarshal(wire, &workflow) != nil || workflow.ProjectID != parts[1] || name != legacyWorkItemName(workflow.RepositoryKey, fmt.Sprint(workflow.WorkItem)) {
			return ErrUnsafe
		}
	}
	if hooks.beforeCommit != nil {
		if err := hooks.beforeCommit(); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := target.StillAtPath(); err != nil {
		return err
	}
	current, err := target.Lstat(name)
	if err != nil || !os.SameFile(info, current) {
		return ErrConflict
	}
	// Revalidate controlled non-cooperating changes at the commit boundary too.
	wire, err = readPublishedFile(target.root, name)
	if err != nil || sha256.Sum256(wire) != digest {
		return errors.Join(ErrConflict, err)
	}
	if err := target.Remove(name); err != nil {
		return err
	}
	if err := errors.Join(target.Sync(), target.StillAtPath()); err != nil {
		return errors.Join(ErrRecoveryRequired, err)
	}
	// Remove only empty, still-identical containers while coordination remains
	// held. Never remove the state root or another writer's populated directory.
	for index := 2; index > 0; index-- {
		child, parent := directories[index], directories[index-1]
		names, err := readDirectoryNamesBounded(child.root, maxLocalDirectoryEntries)
		if err != nil || len(names) != 0 {
			break
		}
		if err := child.StillAtPath(); err != nil {
			return errors.Join(ErrRecoveryRequired, err)
		}
		_ = child.Close()
		directories[index].root = nil
		if err := parent.Remove(parts[index-1]); err != nil {
			break
		}
		if err := errors.Join(parent.Sync(), parent.StillAtPath()); err != nil {
			return errors.Join(ErrRecoveryRequired, err)
		}
	}
	return nil
}
