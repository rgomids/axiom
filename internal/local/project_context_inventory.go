package local

import (
	"os"
	"regexp"
)

var contextSessionName = regexp.MustCompile(`^session-[0-9a-f]{64}\.json$`)

func walkProjectContextV1(w *inventoryWalk, version *os.Root, relative string) error {
	return eachFile(w, version, ".", relative, func(file, _ string) (InventoryKind, int) {
		if protocolName(file) {
			return InventoryRecovery, 0
		}
		if file != "default.json" && !contextSessionName.MatchString(file) {
			return InventoryUnknown, 0
		}
		return "", MaxRecordBytes
	}, func(_ string, wire []byte) InventoryKind {
		if _, err := decodeProjectContext(wire); err == nil {
			return InventoryProjectContext
		}
		return versionedFailure(wire)
	})
}
