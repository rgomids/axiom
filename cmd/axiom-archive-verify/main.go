// axiom-archive-verify is a separately authenticated bootstrap tool. It never
// executes archive contents or writes installation state. Only the Go standard
// library is linked into this program.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

var identity = "development"

const maxCompressed = 256 << 20
const maxExpanded = 512 << 20

func decode(wire []byte) ([]byte, error) {
	if len(wire) == 0 || len(wire) > maxCompressed {
		return nil, errors.New("compressed size limit or empty archive")
	}
	source := bytes.NewReader(wire)
	var output bytes.Buffer
	members := 0
	for source.Len() > 0 {
		reader, err := gzip.NewReader(source)
		if err != nil {
			return nil, fmt.Errorf("gzip member %d: %w", members+1, err)
		}
		reader.Multistream(false)
		_, err = io.Copy(&output, io.LimitReader(reader, int64(maxExpanded-output.Len()+1)))
		closeErr := reader.Close()
		if err != nil || closeErr != nil || output.Len() >= maxExpanded {
			return nil, errors.New("gzip CRC, framing, truncation or expansion limit")
		}
		members++
	}
	raw := output.Bytes()
	sourceTar := bytes.NewReader(raw)
	reader := tar.NewReader(sourceTar)
	seen := map[string]bool{}
	root := ""
	version2 := false
	entries := 0
	for {
		before := sourceTar.Len()
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			// archive/tar also returns EOF for a physically missing terminator.
			// Require two complete zero blocks, then only zero padding.
			consumed := before - sourceTar.Len()
			offset := len(raw) - sourceTar.Len()
			if entries == 0 || consumed < 1024 || offset < 1024 || !allZero(raw[offset-1024:]) {
				return nil, errors.New("missing tar EOF or nonzero tar trailer")
			}
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar framing: %w", err)
		}
		entries++
		name := strings.TrimSuffix(header.Name, "/")
		if entries > 192 || name == "" || strings.Contains(name, "\\") || path.Clean(name) != name || strings.HasPrefix(name, "/") || name == ".." || strings.HasPrefix(name, "../") || seen[name] {
			return nil, errors.New("tar count, duplicate or unsafe path")
		}
		seen[name] = true
		first, rest, _ := strings.Cut(name, "/")
		if root == "" {
			root = first
		}
		if first != root || !strings.HasPrefix(root, "axiom-") {
			return nil, errors.New("tar root mismatch")
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
			return nil, errors.New("tar link or special file")
		}
		limit := int64(1 << 20)
		if rest == "axiom" {
			limit = maxCompressed
		}
		if header.Size < 0 || header.Size > limit {
			return nil, errors.New("tar file size limit")
		}
		if rest == "release-metadata.txt" {
			metadata, err := io.ReadAll(reader)
			if err != nil {
				return nil, err
			}
			version2 = bytes.HasPrefix(metadata, []byte("formatVersion=2\n"))
		}
		// Consume payload here so Next's EOF accounting cannot mistake unread
		// zero-filled file data for the two required tar terminator blocks.
		if _, err := io.Copy(io.Discard, reader); err != nil {
			return nil, fmt.Errorf("tar payload: %w", err)
		}
	}
	if version2 && (members != 1 || len(raw) > 288<<20) || !version2 && entries > 64 {
		return nil, errors.New("archive generation member/count/size limit")
	}
	return raw, nil
}

func allZero(p []byte) bool {
	for _, b := range p {
		if b != 0 {
			return false
		}
	}
	return true
}

func run(args []string) error {
	if len(args) == 1 && args[0] == "identity" {
		fmt.Println(identity)
		return nil
	}
	if len(args) != 2 {
		return errors.New("usage: axiom-archive-verify archive output-tar | identity")
	}
	info, err := os.Lstat(args[0])
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxCompressed {
		return errors.New("archive unavailable, unsafe or oversized")
	}
	file, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer file.Close()
	wire, err := io.ReadAll(io.LimitReader(file, maxCompressed+1))
	if err != nil {
		return err
	}
	raw, err := decode(wire)
	if err != nil {
		return err
	}
	// Caller supplies a private staging path. Never overwrite any existing leaf.
	out, err := os.OpenFile(args[1], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := out.Write(raw)
	closeErr := out.Close()
	if writeErr != nil || closeErr != nil {
		os.Remove(args[1])
		return errors.New("cannot write verified tar staging file")
	}
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "archive_verify_error:", err)
		os.Exit(1)
	}
}
