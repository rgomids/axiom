package local

import (
	"os"
	"path/filepath"
	"strings"
)

// Probe leftovers are recovery state, never canonical evidence or authority.
// Missing manifests from interrupted preparation are recognized but preserved.
func walkProbeEnvelopes(w *inventoryWalk, root *os.Root) error {
	parent, err := existingPrivateChild(root, "capability-probes")
	if err != nil {
		return w.add("capability-probes", InventoryUnsafe, nil)
	}
	defer parent.Close()
	return eachName(w, parent, "capability-probes", func(id string) error {
		relative := "capability-probes/" + id
		if strings.HasPrefix(id, ".probe-cleanup-") && strings.HasSuffix(id, ".json") {
			probeID := strings.TrimSuffix(strings.TrimPrefix(id, ".probe-cleanup-"), ".json")
			if !probeName(probeID) {
				return w.add(relative, InventoryUnknown, nil)
			}
			intent, err := readEnvelopeIntent(root.Name(), probeID)
			if err != nil || intent == nil {
				return w.add(relative, InventoryUnsafe, nil)
			}
			return w.add(relative, InventoryRecovery, nil)
		}
		if !probeName(id) {
			return w.add(relative, InventoryUnknown, nil)
		}
		probe, err := existingPrivateChild(parent, id)
		if err != nil {
			return w.add(relative, InventoryUnsafe, nil)
		}
		defer probe.Close()
		if err := w.add(relative, InventoryRecovery, nil); err != nil {
			return err
		}
		return eachName(w, probe, relative, func(name string) error {
			entry := relative + "/" + name
			if name == "envelope.json" {
				wire, err := readPrivateFileBounded(probe, name, 16384)
				if err != nil {
					return w.add(entry, InventoryUnsafe, nil)
				}
				var m EnvelopeManifest
				if strictEnvelopeJSON(wire, &m) != nil || !validEnvelopeManifest(m, id) {
					return w.add(entry, InventoryMalformed, wire)
				}
				for owned, identity := range m.Identities {
					if owned != "." && owned != "workspace" && owned != "inbox" && owned != "sentinel" {
						return w.add(entry, InventoryMalformed, wire)
					}
					observed, err := envelopeIdentity(filepath.Join(probe.Name(), owned))
					if os.IsNotExist(err) && owned != "." {
						intent, intentErr := readEnvelopeIntent(root.Name(), id)
						if intentErr == nil && intent != nil && intent.Digest == digestEnvelope(wire) {
							continue
						}
					}
					if err != nil || observed != identity {
						return w.add(entry, InventoryUnsafe, nil)
					}
				}
				return w.add(entry, InventoryRecovery, wire)
			}
			if protocolName(name) {
				return w.add(entry, InventoryRecovery, nil)
			}
			if name != "workspace" && name != "inbox" && name != "sentinel" {
				return w.add(entry, InventoryUnknown, nil)
			}
			child, err := existingPrivateChild(probe, name)
			if err != nil {
				return w.add(entry, InventoryUnsafe, nil)
			}
			defer child.Close()
			return walkProbeObjects(w, child, entry)
		})
	})
}

func walkProbeObjects(w *inventoryWalk, root *os.Root, relative string) error {
	if err := w.add(relative, InventoryRecovery, nil); err != nil {
		return err
	}
	return eachName(w, root, relative, func(name string) error {
		entry := relative + "/" + name
		info, err := root.Lstat(name)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return w.add(entry, InventoryUnsafe, nil)
		}
		if info.IsDir() {
			child, err := existingPrivateChild(root, name)
			if err != nil {
				return w.add(entry, InventoryUnsafe, nil)
			}
			defer child.Close()
			return walkProbeObjects(w, child, entry)
		}
		if !info.Mode().IsRegular() || !ownedByUser(info) || forbiddenPermissions(info, 0077) {
			return w.add(entry, InventoryUnsafe, nil)
		}
		file, err := root.OpenFile(name, os.O_RDONLY|noFollow, 0)
		if err != nil {
			return w.add(entry, InventoryUnsafe, nil)
		}
		observed, statErr := file.Stat()
		safe := statErr == nil && os.SameFile(info, observed) && singleLink(file, observed) && checkPrivateACL(file) == nil
		file.Close()
		if !safe {
			return w.add(entry, InventoryUnsafe, nil)
		}
		return w.add(entry, InventoryRecovery, nil)
	})
}
