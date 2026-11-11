package projectapp

import (
	"errors"
	"testing"
)

type ancestorStorageRefusal struct{}

func (ancestorStorageRefusal) Error() string { return "bounded refusal" }
func (ancestorStorageRefusal) StorageCategory() string {
	return "untrusted_ancestor_owner:ancestor:uid=65534:uid_mapping=unmapped"
}
func TestAncestorOwnerStorageCategory(t *testing.T) {
	err := errors.Join(ErrUnsafe, ancestorStorageRefusal{})
	want := ancestorStorageRefusal{}.StorageCategory()
	if got := storageResult(err); got.Category != want || got.Status != LifecycleFailed {
		t.Fatalf("lifecycle: %+v", got)
	}
	if got := projectListError(t.Context(), err); got.Category != want || got.Status != ProjectListFailed {
		t.Fatalf("catalog: %+v", got)
	}
}
