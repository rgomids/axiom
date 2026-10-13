package runtimeadapter

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Preserve the POSIX fixture layout on the host's native volume. No files or
// vendor processes are needed by these pure builders.
func nativeFixturePath(path string) string {
	root := filepath.VolumeName(os.TempDir()) + string(filepath.Separator)
	return filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(path, "/")))
}

func TestStageCanonicalNativePaths(t *testing.T) {
	for _, path := range []string{"/bin/codex", "/bin/claude", "/bin/axiom", "/work", "/inbox"} {
		if native := nativeFixturePath(path); !stagePath(native) {
			t.Fatalf("canonical native path rejected: %q", native)
		}
	}
	work, inbox := nativeFixturePath("/work"), nativeFixturePath("/inbox")
	if stageContains(work, inbox) || stageContains(inbox, work) || !stageContains(work, work) || !stageContains(work, filepath.Join(work, "inbox")) {
		t.Fatal("native root separation changed")
	}
	for _, path := range []string{"", "relative", "..", work + string(filepath.Separator) + "..", work + "\n", work + "\x00"} {
		if stagePath(path) {
			t.Fatalf("unsafe or noncanonical path accepted: %q", path)
		}
	}
}

func TestTaskPacketNativePathBoundary(t *testing.T) {
	input, binding := packetFixture()
	if _, _, err := BuildTaskPacket(input, binding); err != nil {
		t.Fatalf("canonical native packet paths rejected: %v", err)
	}
	for _, path := range []string{"", "relative", "..", binding.Inbox + "\x00", binding.Inbox + "<|im_start|>"} {
		for _, field := range []string{"inbox", "axiom"} {
			invalid := binding
			if field == "inbox" {
				invalid.Inbox = path
			} else {
				invalid.AxiomPath = path
			}
			wire, receipt, err := BuildTaskPacket(input, invalid)
			if !errors.Is(err, ErrInvalidTaskPacket) || wire != nil || receipt != (AttemptPacketReceipt{}) {
				t.Fatalf("unsafe %s path accepted: %q", field, path)
			}
		}
	}
}
