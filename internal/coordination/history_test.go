package coordination

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Compatibility recognition may survive; an operational writer/encoder may not.
func TestHistoryExportsAreReadOnly(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "coordination.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.IsExported() {
			switch fn.Name.Name {
			case "DecodeStream", "ValidRecord", "ValidStream":
			default:
				t.Errorf("operational API survived: %s", fn.Name.Name)
			}
		}
	}
}

func TestFrozenHistoryRejectsTampering(t *testing.T) {
	paths, err := filepath.Glob("../compatibility/testdata/stable-v1/snapshots/*/state/coordination/v1/*/*.json")
	if err != nil || len(paths) == 0 {
		t.Fatalf("fixtures: %v", err)
	}
	for _, p := range paths {
		wire, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		stream, err := DecodeStream(wire)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		altered := bytes.Replace(wire, []byte(stream.Records[0].Digest), bytes.Repeat([]byte("0"), 64), 1)
		if _, err := DecodeStream(altered); !errors.Is(err, ErrInvalidRecord) {
			t.Fatalf("tampered history accepted: %s", p)
		}
	}
}

// Resealing distinguishes semantic refusal from a mere digest mismatch.
func TestFrozenHistorySemanticValidation(t *testing.T) {
	paths, err := filepath.Glob("../compatibility/testdata/stable-v1/snapshots/*/state/coordination/v1/*/*.json")
	if err != nil || len(paths) == 0 {
		t.Fatalf("fixtures: %v", err)
	}
	wire, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	type historyCase struct {
		name   string
		mutate func(*Record)
		valid  bool
	}
	tests := []historyCase{
		{"empty", func(r *Record) { r.Fields[0].Value = "" }, false},
		{"oversized", func(r *Record) { r.Fields[0].Value = strings.Repeat("x", 4097) }, false},
		{"nul", func(r *Record) { r.Fields[0].Value = "a\x00b" }, false},
		{"newline", func(r *Record) { r.Fields[0].Value = "a\nb" }, false},
		{"carriage-return", func(r *Record) { r.Fields[0].Value = "a\rb" }, false},
		{"assignment", func(r *Record) { r.Fields[0].Value = "password=placeholder" }, false},
		{"bearer", func(r *Record) { r.Fields[0].Value = "Authorization: Bearer placeholder" }, false},
		{"key-marker", func(r *Record) { r.Fields[0].Value = "-----BEGIN " + "PRIVATE KEY-----" }, false},
		{"known-token", func(r *Record) { r.Fields[0].Value = "ghp_" + strings.Repeat("x", 20) }, false},
		{"assistant-prompt", func(r *Record) { r.Fields[0].Value = "<|assistant|>" }, false},
		{"user-prompt", func(r *Record) { r.Fields[0].Value = "<|user|>" }, false},
		{"system-prompt", func(r *Record) { r.Fields[0].Value = "<|system|>" }, false},
		{"keywords-are-prose", func(r *Record) { r.Fields[0].Value = "token budget and password policy; system prompt reviewed" }, true},
		{"maximum-value", func(r *Record) { r.Fields[0].Value = strings.Repeat("x", 4096) }, true},
	}
	for _, outcome := range []string{"succeeded", "failed", "", "success", "Succeeded", " succeeded", "failed "} {
		outcome := outcome
		tests = append(tests, historyCase{"outcome-" + outcome, func(r *Record) { r.Kind = Result; r.Fields = []Field{{Name: "outcome", Value: outcome}} }, outcome == "succeeded" || outcome == "failed"})
	}
	tests = append(tests, historyCase{"missing-outcome", func(r *Record) { r.Kind = Result; r.Fields = []Field{{Name: "summary", Value: "done"}} }, false})
	usages := []struct {
		name  string
		usage UsageObservation
		valid bool
	}{
		{"measured", UsageObservation{"runtime", "tokens", "42", UsageMeasured}, true},
		{"partial", UsageObservation{"runtime", "tokens", "unknown", UsagePartial}, true},
		{"unavailable", UsageObservation{Source: "runtime", Status: UsageUnavailable}, true},
		{"source-empty", UsageObservation{"", "tokens", "42", UsageMeasured}, false},
		{"source-shape", UsageObservation{"bad source", "tokens", "42", UsageMeasured}, false},
		{"source-range", UsageObservation{strings.Repeat("x", 257), "tokens", "42", UsageMeasured}, false},
		{"unit-shape", UsageObservation{"runtime", "bad unit", "42", UsageMeasured}, false},
		{"unit-empty", UsageObservation{"runtime", "", "42", UsageMeasured}, false},
		{"value-empty", UsageObservation{"runtime", "tokens", "", UsageMeasured}, false},
		{"value-range", UsageObservation{"runtime", "tokens", strings.Repeat("1", 129), UsagePartial}, false},
		{"value-max", UsageObservation{"runtime", "tokens", strings.Repeat("1", 128), UsagePartial}, true},
		{"value-nul", UsageObservation{"runtime", "tokens", "4\x002", UsageMeasured}, false},
		{"value-newline", UsageObservation{"runtime", "tokens", "4\n2", UsageMeasured}, false},
		{"value-cr", UsageObservation{"runtime", "tokens", "4\r2", UsageMeasured}, false},
		{"unavailable-unit", UsageObservation{"runtime", "tokens", "", UsageUnavailable}, false},
		{"unavailable-value", UsageObservation{"runtime", "", "42", UsageUnavailable}, false},
		{"status-enum", UsageObservation{"runtime", "tokens", "42", "Measured"}, false},
	}
	for _, u := range usages {
		u := u
		tests = append(tests, historyCase{"usage-" + u.name, func(r *Record) { r.Usage = &u.usage }, u.valid})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream, err := DecodeStream(wire)
			if err != nil {
				t.Fatal(err)
			}
			stream.Records = stream.Records[:1]
			r := &stream.Records[0]
			tt.mutate(r)
			r.Digest = ""
			content, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(content)
			r.Digest = hex.EncodeToString(digest[:])
			if ValidRecord(*r) != tt.valid || ValidStream(stream) != tt.valid {
				t.Fatalf("semantic validity: want %v", tt.valid)
			}
			encoded, err := json.Marshal(stream)
			if err != nil {
				t.Fatal(err)
			}
			_, err = DecodeStream(encoded)
			if tt.valid && err != nil || !tt.valid && !errors.Is(err, ErrInvalidRecord) {
				t.Fatalf("decode classification: %v", err)
			}
		})
	}
	// Every released fixture remains readable and byte-identical on disk.
	for _, path := range paths {
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = DecodeStream(original); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(original, after) {
			t.Fatalf("history changed: %s", path)
		}
	}
}

func TestFrozenHistoryHeaderLineageAndDecoderRefusals(t *testing.T) {
	paths, err := filepath.Glob("../compatibility/testdata/stable-v1/snapshots/*/state/coordination/v1/*/*.json")
	if err != nil || len(paths) == 0 {
		t.Fatalf("fixtures: %v", err)
	}
	wire, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name   string
		change func(*Stream)
	}{
		{"stream-version", func(s *Stream) { s.FormatVersion++ }},
		{"stream-parent", func(s *Stream) { s.ParentID = "invalid" }},
		{"stream-child", func(s *Stream) { s.ChildID = "invalid" }},
		{"empty-records", func(s *Stream) { s.Records = nil }},
		{"record-version", func(s *Stream) { s.Records[0].FormatVersion++ }},
		{"unknown-kind", func(s *Stream) { s.Records[0].Kind = "unknown" }},
		{"id-length", func(s *Stream) { s.Records[0].RecordID = "short" }},
		{"id-separator", func(s *Stream) { s.Records[0].RecordID = "00000000_0000-4000-8000-000000000001" }},
		{"id-hex", func(s *Stream) { s.Records[0].RecordID = "g0000000-0000-4000-8000-000000000001" }},
		{"parent-lineage", func(s *Stream) { s.Records[0].ParentID = "00000000-0000-4000-8000-000000000091" }},
		{"child-lineage", func(s *Stream) { s.Records[0].ChildID = "00000000-0000-4000-8000-000000000092" }},
		{"revision-lineage", func(s *Stream) { s.Records[0].Revision = 2; s.Records[0].PreviousDigest = strings.Repeat("a", 64) }},
		{"digest-lineage", func(s *Stream) {
			r := s.Records[0]
			r.Revision = 2
			r.PreviousDigest = strings.Repeat("b", 64)
			s.Records = append(s.Records[:1], r)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, err := DecodeStream(wire)
			if err != nil {
				t.Fatal(err)
			}
			s.Records = s.Records[:1]
			tt.change(&s)
			for i := range s.Records {
				s.Records[i].Digest, err = recordDigest(s.Records[i])
				if err != nil {
					t.Fatal(err)
				}
			}
			encoded, err := json.Marshal(s)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = DecodeStream(encoded); !errors.Is(err, ErrInvalidRecord) {
				t.Fatalf("invalid lineage accepted: %v", err)
			}
		})
	}
	for _, bad := range [][]byte{nil, append(append([]byte{}, wire...), []byte("{}")...), bytes.Repeat([]byte(" "), MaxStreamBytes+1)} {
		if _, err := DecodeStream(bad); !errors.Is(err, ErrInvalidRecord) {
			t.Fatalf("invalid envelope accepted: %v", err)
		}
	}
}
