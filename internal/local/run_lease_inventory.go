package local

import "os"

// Epoch publication uses the shared ADR-0007 marker/staging vocabulary. The
// inventory preserves interrupted state for guided recovery, never resets it.
func walkRunLeasesV1(w *inventoryWalk, version *os.Root, relative string) error {
	return eachName(w, version, relative, func(projectID string) error {
		if !validGraphToken(projectID) {
			return w.add(relative+"/"+projectID, InventoryUnknown, nil)
		}
		project, err := existingPrivateChild(version, projectID)
		if err != nil {
			return w.add(relative+"/"+projectID, InventoryUnsafe, nil)
		}
		defer project.Close()
		return eachName(w, project, relative+"/"+projectID, func(name string) error {
			rel := relative + "/" + projectID + "/" + name
			if !hexDigestName.MatchString(name) {
				return w.add(rel, InventoryUnknown, nil)
			}
			return eachFile(w, project, name, rel, func(file, _ string) (InventoryKind, int) {
				if protocolName(file) {
					return InventoryRecovery, 0
				}
				if file != "epoch.json" {
					return InventoryUnknown, 0
				}
				return "", MaxRecordBytes
			}, func(_ string, wire []byte) InventoryKind {
				r, err := decodeRunLease(wire)
				if err != nil {
					return versionedFailure(wire)
				}
				if r.ProjectID != projectID || graphName(r.ParentID) != name {
					return InventoryMalformed
				}
				return InventoryRunLease
			})
		})
	})
}
