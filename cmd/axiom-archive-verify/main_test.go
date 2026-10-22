package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func packed(t *testing.T, data []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestStrictMembersAndNoOutputOnRefusal(t *testing.T) {
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	if err := w.WriteHeader(&tar.Header{Name: "axiom-test/release-metadata.txt", Mode: 0600, Size: 16}); err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("formatVersion=1\n"))
	w.Close()
	raw := b.Bytes()
	valid := packed(t, raw)
	var zeros bytes.Buffer
	zw := tar.NewWriter(&zeros)
	if err := zw.WriteHeader(&tar.Header{Name: "axiom-test/zero-file", Mode: 0600, Size: 2048}); err != nil {
		t.Fatal(err)
	}
	if _, err := zw.Write(make([]byte, 2048)); err != nil {
		t.Fatal(err)
	}
	// Deliberately omit tar.Writer.Close and its terminator blocks.
	missingEOFWithZeroPayload := packed(t, zeros.Bytes())
	crc := append([]byte{}, valid...)
	crc[len(crc)-8] ^= 1
	for _, tc := range []struct {
		name string
		wire []byte
		ok   bool
	}{
		{"valid", valid, true},
		{"split-valid", append(packed(t, raw[:17]), packed(t, raw[17:])...), true},
		{"empty-prefix-valid-v1", append(packed(t, nil), valid...), true},
		{"empty-suffix-valid-v1", append(append([]byte{}, valid...), packed(t, nil)...), true},
		{"empty-archive", packed(t, nil), false},
		{"zero-trailing", append(append([]byte{}, valid...), make([]byte, 16)...), false},
		{"garbage-trailing", append(append([]byte{}, valid...), []byte("garbage")...), false},
		{"crc", crc, false},
		{"truncation", valid[:len(valid)-3], false},
		{"split-invalid", append(packed(t, raw[:17]), packed(t, raw[18:])...), false},
		{"tar-eof-absent", packed(t, raw[:1024]), false},
		{"tar-eof-absent-zero-payload", missingEOFWithZeroPayload, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			archive, out := filepath.Join(dir, "archive"), filepath.Join(dir, "verified.tar")
			if err := os.WriteFile(archive, tc.wire, 0600); err != nil {
				t.Fatal(err)
			}
			err := run([]string{archive, out})
			if (err == nil) != tc.ok {
				t.Fatalf("accept=%t error=%v", tc.ok, err)
			}
			_, statErr := os.Lstat(out)
			if !tc.ok && !os.IsNotExist(statErr) {
				t.Fatal("refusal persisted output")
			}
		})
	}
}
