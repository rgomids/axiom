package local

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// AncestorOwnerError is a trust refusal, not evidence of corrupt local state.
// PathClass deliberately excludes directory names and personal absolute paths.
type AncestorOwnerError struct {
	PathClass  string
	UID        uint32
	UIDMapping string
}

func (e *AncestorOwnerError) Error() string {
	return fmt.Sprintf("untrusted_ancestor_owner:%s:uid=%d:uid_mapping=%s", e.PathClass, e.UID, e.UIDMapping)
}
func (e *AncestorOwnerError) Unwrap() error           { return ErrUnsafe }
func (e *AncestorOwnerError) StorageCategory() string { return e.Error() }

// AncestorOwnerCategory carries only bounded observations through existing
// string-category application boundaries. Raw filesystem errors never cross it.
func AncestorOwnerCategory(err error) string {
	var refusal *AncestorOwnerError
	if errors.As(err, &refusal) {
		return refusal.StorageCategory()
	}
	return ""
}

func storageCategory(err error, fallback string) string {
	if category := AncestorOwnerCategory(err); category != "" {
		return category
	}
	return fallback
}

// ParseAncestorOwnerCategory validates the bounded internal category before
// presentation separates the stable category from its observed facts.
func ParseAncestorOwnerCategory(category string) (AncestorOwnerError, bool) {
	parts := strings.Split(category, ":")
	if len(parts) != 4 || parts[0] != "untrusted_ancestor_owner" || (parts[1] != "filesystem_root" && parts[1] != "ancestor") || !strings.HasPrefix(parts[2], "uid=") || !strings.HasPrefix(parts[3], "uid_mapping=") {
		return AncestorOwnerError{}, false
	}
	uid, err := strconv.ParseUint(strings.TrimPrefix(parts[2], "uid="), 10, 32)
	mapping := strings.TrimPrefix(parts[3], "uid_mapping=")
	if err != nil || (mapping != "unknown" && mapping != "mapped" && mapping != "unmapped") {
		return AncestorOwnerError{}, false
	}
	return AncestorOwnerError{PathClass: parts[1], UID: uint32(uid), UIDMapping: mapping}, true
}
