package local

import (
	"os"
	"strconv"
	"strings"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

const (
	InventoryWorkflowDefinition InventoryKind = "workflow_definition"
	InventoryWorkflowIndex      InventoryKind = "workflow_revision_index"
)

func walkPortableProject(w *inventoryWalk, parent *os.Root, name string) error {
	target, e := existingPrivateChild(parent, name)
	if e != nil {
		return w.add(name, InventoryUnsafe, nil)
	}
	defer target.Close()
	return eachName(w, target, name, func(file string) error {
		relative := name + "/" + file
		if protocolName(file) || strings.HasPrefix(file, ".lingo-") {
			return w.add(relative, InventoryRecovery, nil)
		}
		if file == "workflows" {
			return walkWorkflowDefinitions(w, target, name)
		}
		if file != manifestName {
			return w.add(relative, InventoryUnknown, nil)
		}
		wire, e := readPrivateFileBounded(target, file, MaxRecordBytes)
		if e != nil {
			return w.add(relative, InventoryUnsafe, nil)
		}
		kind := InventoryMalformed
		if _, issues := manifest.Decode(wire); len(issues) == 0 {
			kind = InventoryPortableManifest
		} else if m := schemaVersionRe.FindSubmatch(wire); m != nil {
			version, _ := strconv.Atoi(string(m[1]))
			if version > 4 {
				kind = InventoryNewer
			} else if version < 1 {
				kind = InventoryOlder
			}
		}
		return w.add(relative, kind, wire)
	})
}

func walkWorkflowDefinitions(w *inventoryWalk, projectRoot *os.Root, prefix string) error {
	relative := prefix + "/workflows"
	catalog, e := readWorkflowCatalog(projectRoot)
	if e != nil || len(catalog.Unindexed) > 0 {
		kind := InventoryMalformed
		if e == ErrRecoveryRequired || len(catalog.Unindexed) > 0 {
			kind = InventoryRecovery
		}
		return w.add(relative, kind, nil)
	}
	if len(catalog.Wire) == 0 {
		return w.add(relative, InventoryMalformed, nil)
	}
	if e := w.add(relative+"/index.json", InventoryWorkflowIndex, catalog.Wire); e != nil {
		return e
	}
	for _, entry := range catalog.Index.Revisions {
		key := workflowName(entry.WorkflowID, entry.Revision)
		if _, ok := catalog.Documents[key]; !ok {
			continue
		}
		wire, e := ReadOwnedFile(projectRoot.Name(), "workflows/"+key, workflowdefinition.MaxBytes)
		if e != nil {
			return w.add(relative+"/"+key, InventoryUnsafe, nil)
		}
		if e = w.add(relative+"/"+key, InventoryWorkflowDefinition, wire); e != nil {
			return e
		}
	}
	return nil
}
