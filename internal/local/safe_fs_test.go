package local

import (
	"bytes"
	"errors"
	"io"
	"syscall"
	"testing"
)

type faultWriter struct {
	limit int
	fault error
	data  bytes.Buffer
}

func (w *faultWriter) Write(content []byte) (int, error) {
	if w.fault != nil {
		return 0, w.fault
	}
	if w.limit == 0 {
		return 0, nil
	}
	if len(content) > w.limit {
		content = content[:w.limit]
	}
	return w.data.Write(content)
}

func TestWriteCompleteHandlesShortWritesAndStorageFaults(t *testing.T) {
	content := []byte("complete manifest")
	partial := &faultWriter{limit: 2}
	if err := writeComplete(partial, content); err != nil || !bytes.Equal(partial.data.Bytes(), content) {
		t.Fatalf("short chunks: %q, %v", partial.data.Bytes(), err)
	}
	for _, test := range []struct {
		name   string
		writer *faultWriter
		want   error
	}{
		{"no progress", &faultWriter{}, io.ErrShortWrite},
		{"disk full", &faultWriter{fault: syscall.ENOSPC}, syscall.ENOSPC},
		{"quota", &faultWriter{fault: syscall.EDQUOT}, syscall.EDQUOT},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := writeComplete(test.writer, content); !errors.Is(err, test.want) {
				t.Fatalf("write error = %v, want %v", err, test.want)
			}
		})
	}
}
