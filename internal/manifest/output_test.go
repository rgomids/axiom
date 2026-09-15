package manifest

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/project"
)

func TestBoundedOutputWriter(t *testing.T) {
	for _, chunkSize := range []int{1, 997, MaxBytes - 1, MaxBytes} {
		t.Run(strconv.Itoa(chunkSize), func(t *testing.T) {
			var output boundedOutput
			for len(output.data) < MaxBytes {
				chunk := bytes.Repeat([]byte{'x'}, min(chunkSize, MaxBytes-len(output.data)))
				if n, err := output.Write(chunk); n != len(chunk) || err != nil {
					t.Fatal("within-bound write failed")
				}
				if cap(output.data) > MaxBytes {
					t.Fatal("output capacity exceeded bound")
				}
			}
			if n, err := output.Write(nil); n != 0 || err != nil {
				t.Fatal("empty write at limit failed")
			}
			before := bytes.Clone(output.data)
			if n, err := output.Write([]byte{'x'}); n != 0 || err != io.ErrShortBuffer || !output.exceeded {
				t.Fatal("limit+1 did not stop writer")
			}
			if n, err := output.Write(nil); n != 0 || err != io.ErrShortBuffer || !bytes.Equal(before, output.data) {
				t.Fatal("failure not sticky or buffer changed")
			}
		})
	}
	var output boundedOutput
	if n, err := output.Write(make([]byte, MaxBytes*4)); n != 0 || err != io.ErrShortBuffer || len(output.data) != 0 || cap(output.data) != 0 {
		t.Fatal("oversized first write allocated output")
	}
	var crossing boundedOutput
	if _, err := crossing.Write([]byte("prefix")); err != nil {
		t.Fatal(err)
	}
	if n, err := crossing.Write(make([]byte, MaxBytes)); n != 0 || err != io.ErrShortBuffer || string(crossing.data) != "prefix" {
		t.Fatal("crossing write partially copied")
	}
	if n, err := crossing.Write([]byte("small retry")); n != 0 || err != io.ErrShortBuffer || string(crossing.data) != "prefix" {
		t.Fatal("write succeeded after overflow")
	}
}

func TestEncodeStopsAtWriterBound(t *testing.T) {
	for _, payload := range []string{strings.Repeat("x", MaxBytes*4), strings.Repeat("\x01", MaxBytes/2)} {
		s := reviewProject(t, reviewMinimal).State()
		s.BusinessContext = project.Configured(project.BusinessContext{Text: project.Configured(payload)})
		p, issues := project.New(s)
		if len(issues) != 0 {
			t.Fatal(issues)
		}
		var output boundedOutput
		result, issues := encodeWithOutput(p, &output)
		if result != nil || len(issues) != 1 || issues[0].Code != "byte_limit" {
			t.Fatalf("expected byte_limit without partial output: %v", issues)
		}
		// Observe the writer used by Encode, not allocation counts or dependency
		// chunk sizes. Decode(output.data) alone cannot produce byte_limit here.
		if !output.exceeded || len(output.data) > MaxBytes || cap(output.data) > MaxBytes {
			t.Fatal("Encode did not enforce the writer bound")
		}
		if b, publicIssues := Encode(p); b != nil || len(publicIssues) != 1 || publicIssues[0].Code != "byte_limit" {
			t.Fatal("public Encode did not preserve bound failure")
		}
	}
}

func TestEncodeOutputByteBoundary(t *testing.T) {
	s := reviewProject(t, reviewMinimal).State()
	s.Name = "x"
	p, _ := project.New(s)
	base, issues := Encode(p)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	for _, size := range []int{MaxBytes - 1, MaxBytes, MaxBytes + 1} {
		s.Name = strings.Repeat("x", size-len(base)+1)
		p, _ := project.New(s)
		b, issues := Encode(p)
		if size > MaxBytes {
			if b != nil || len(issues) != 1 || issues[0].Code != "byte_limit" {
				t.Fatal("over-limit output accepted")
			}
			continue
		}
		if len(issues) != 0 || len(b) != size || b[len(b)-1] != '\n' {
			t.Fatalf("exact boundary or final newline lost: %v", issues)
		}
		q, issues := Decode(b)
		if len(issues) != 0 || !p.Equivalent(q) {
			t.Fatal("boundary round trip lost intent")
		}
	}
}
