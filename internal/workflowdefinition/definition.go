// Package workflowdefinition owns portable, immutable workflow intent (Spec 007).
// It never resolves executables, dispatches agents, or grants authority.
package workflowdefinition

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const MaxBytes = 1 << 20

//go:embed workflow.schema.json
var schemaWire []byte

//go:embed default-sdd-r1.json
var defaultWire []byte
var ErrInvalid = errors.New("invalid_workflow_definition")
var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func ValidKey(s string) bool { return keyPattern.MatchString(s) }
func ValidDigest(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && hex.EncodeToString(b) == s
}

type Ref struct {
	WorkflowID string `json:"workflowId" yaml:"workflowId"`
	Revision   int    `json:"revision" yaml:"revision"`
	Digest     string `json:"digest" yaml:"digest"`
	Source     string `json:"source" yaml:"source"`
}

func (r Ref) Valid() bool {
	return ValidKey(r.WorkflowID) && r.Revision > 0 && r.Revision <= 2147483647 && ValidDigest(r.Digest) && (r.Source == "builtin" || r.Source == "project")
}

type Definition struct {
	SchemaVersion int     `json:"schemaVersion"`
	WorkflowID    string  `json:"workflowId"`
	Revision      int     `json:"revision"`
	Name          string  `json:"name"`
	Stages        []Stage `json:"stages"`
}
type Stage struct {
	ID                 string        `json:"id"`
	Purpose            string        `json:"purpose"`
	Instructions       string        `json:"instructions"`
	Phase              string        `json:"phase"`
	Checkpoint         bool          `json:"checkpoint"`
	Mode               string        `json:"mode"`
	Concurrency        int           `json:"concurrency"`
	Inputs             []Input       `json:"inputs"`
	Outputs            []Output      `json:"outputs"`
	CompletionCriteria []Criterion   `json:"completionCriteria"`
	Validators         []Validator   `json:"validators"`
	HumanGates         []Gate        `json:"humanGates"`
	FailurePolicy      FailurePolicy `json:"failurePolicy"`
	Agents             []Agent       `json:"agents"`
}
type Input struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Source   string `json:"source"`
	Required bool   `json:"required"`
}
type Output struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
}
type Criterion struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	OutputRef    string `json:"outputRef"`
	ValidatorRef string `json:"validatorRef"`
}
type Validator struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	PolicyRef string `json:"policyRef"`
}
type Gate struct {
	Kind   string `json:"kind"`
	Timing string `json:"timing"`
}
type FailurePolicy struct {
	OnFailure string `json:"onFailure"`
	OnUnknown string `json:"onUnknown"`
	Retry     string `json:"retry"`
}
type Effort struct {
	Mode  string `json:"mode"`
	Value string `json:"value"`
}
type Agent struct {
	ID                 string   `json:"id"`
	Role               string   `json:"role"`
	Responsibilities   string   `json:"responsibilities"`
	Complexity         string   `json:"complexity"`
	DependsOn          []string `json:"dependsOn"`
	Inputs             []string `json:"inputs"`
	Outputs            []string `json:"outputs"`
	Capabilities       []string `json:"capabilities"`
	RuntimeConstraints []string `json:"runtimeConstraints"`
	ProfileRef         string   `json:"profileRef"`
	Effort             Effort   `json:"effort"`
	TimeoutSeconds     int      `json:"timeoutSeconds"`
	MaximumAttempts    int      `json:"maximumAttempts"`
	EffectCeilings     []string `json:"effectCeilings"`
	IntegrationOwner   bool     `json:"integrationOwner"`
	ValidationOwner    bool     `json:"validationOwner"`
}
type Diagnostic struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}
type Document struct {
	Definition Definition
	Canonical  []byte
	Digest     string
}

func (d Document) Ref(source string) Ref {
	return Ref{d.Definition.WorkflowID, d.Definition.Revision, d.Digest, source}
}
func Builtin() Document {
	d, issues := Decode(defaultWire)
	if len(issues) != 0 {
		panic("invalid embedded default")
	}
	return d
}
func Encode(d Definition) (Document, []Diagnostic) {
	b, e := json.Marshal(d)
	if e != nil {
		return Document{}, []Diagnostic{{"definition", "invalid_json"}}
	}
	return Decode(b)
}

var compiledSchema, schemaError = compileSchema()

func compileSchema() (*jsonschema.Schema, error) {
	var resource any
	if e := json.Unmarshal(schemaWire, &resource); e != nil {
		return nil, e
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	if e := c.AddResource("urn:axiom:workflow-definition:1", resource); e != nil {
		return nil, e
	}
	return c.Compile("urn:axiom:workflow-definition:1")
}

// StrictJSON checks duplicate keys, UTF-8, Unicode surrogate validity and bounds
// before any lossy Go JSON decoding. Depth and node counts are bounded.
func StrictJSON(b []byte, target any) error {
	return StrictJSONBounded(b, target, MaxBytes)
}

// StrictJSONBounded applies the same closed decoding to a bounded local
// envelope that may retain a complete definition plus its execution ledger.
func StrictJSONBounded(b []byte, target any, limit int) error {
	if len(b) == 0 || len(b) > limit || !utf8.Valid(b) {
		return ErrInvalid
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	nodes := 0
	var scan func(int) error
	scan = func(depth int) error {
		nodes++
		if depth > 32 || nodes > 100000 {
			return ErrInvalid
		}
		t, e := dec.Token()
		if e != nil {
			return ErrInvalid
		}
		if delim, ok := t.(json.Delim); ok {
			switch delim {
			case '{':
				seen := map[string]bool{}
				for dec.More() {
					k, e := dec.Token()
					if e != nil {
						return ErrInvalid
					}
					key, ok := k.(string)
					if !ok || seen[key] {
						return ErrInvalid
					}
					seen[key] = true
					if e := scan(depth + 1); e != nil {
						return e
					}
				}
			case '[':
				for dec.More() {
					if e := scan(depth + 1); e != nil {
						return e
					}
				}
			default:
				return ErrInvalid
			}
			if _, e := dec.Token(); e != nil {
				return ErrInvalid
			}
		}
		return nil
	}
	if e := scan(0); e != nil {
		return e
	}
	if _, e := dec.Token(); e != io.EOF {
		return ErrInvalid
	}
	// Bound the tree before invoking JCS, which also rejects escaped lone
	// surrogates that encoding/json would otherwise replace silently.
	if _, e := jsoncanonicalizer.Transform(b); e != nil {
		return ErrInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if e := decoder.Decode(target); e != nil {
		return ErrInvalid
	}
	return nil
}
func Decode(b []byte) (Document, []Diagnostic) {
	bad := func(code string) (Document, []Diagnostic) { return Document{}, []Diagnostic{{"definition", code}} }
	var raw any
	if e := StrictJSON(b, &raw); e != nil {
		return bad("invalid_json")
	}
	if schemaError != nil {
		return bad("schema_unavailable")
	}
	if e := compiledSchema.Validate(raw); e != nil {
		return bad("invalid_schema")
	}
	// Integers only, no null, and byte limits supplement schema character limits.
	var bounded func(any) bool
	bounded = func(v any) bool {
		switch x := v.(type) {
		case nil:
			return false
		case json.Number:
			_, e := x.Int64()
			return e == nil
		case string:
			return len(x) <= 16384
		case []any:
			for _, i := range x {
				if !bounded(i) {
					return false
				}
			}
		case map[string]any:
			for _, i := range x {
				if !bounded(i) {
					return false
				}
			}
		}
		return true
	}
	if !bounded(raw) {
		return bad("invalid_value")
	}
	var d Definition
	if e := json.Unmarshal(b, &d); e != nil {
		return bad("invalid_schema")
	}
	if issues := validate(d); len(issues) != 0 {
		return Document{}, issues
	}
	canonical, e := jsoncanonicalizer.Transform(b)
	if e != nil || len(canonical) > MaxBytes {
		return bad("invalid_canonical_content")
	}
	digest := sha256.Sum256(canonical)
	return Document{d, canonical, hex.EncodeToString(digest[:])}, nil
}

func logicalSource(s string) bool {
	// Logical keys/artifact references only: never host paths, URIs or argv.
	if s == "" || len(s) > 256 || strings.ContainsAny(s, "\\:\x00\r\n\t ") || strings.HasPrefix(s, "/") {
		return false
	}
	for _, part := range strings.Split(s, "/") {
		if !ValidKey(part) {
			return false
		}
	}
	return true
}
