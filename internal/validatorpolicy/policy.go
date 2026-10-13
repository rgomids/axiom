// Package validatorpolicy owns reviewed repository commands, separate from
// Workflow definitions, agent records and Plan documents.
package validatorpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/strictjson"
)

const MaxPolicyBytes = 64 << 10

var ErrInvalid = errors.New("invalid validator policy")
var ErrAuthority = errors.New("validator policy authority denied")
var ErrDigest = errors.New("validator policy digest drift")

type Command struct {
	Reference string   `json:"reference"`
	Argv      []string `json:"argv"`
	Env       []string `json:"env"`
	OutputMax int      `json:"outputMax"`
}
type Definition struct {
	FormatVersion int       `json:"formatVersion"`
	ProjectID     string    `json:"projectId"`
	Commands      []Command `json:"commands"`
}
type Review struct {
	PreviewDigest string    `json:"previewDigest"`
	ActorKind     string    `json:"actorKind"`
	Actor         string    `json:"actor"`
	ReviewedAt    time.Time `json:"reviewedAt"`
	AuthorityRef  string    `json:"authorityRef"`
}
type Policy struct {
	Definition
	Review Review `json:"review"`
}

func ValidDigest(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && s == strings.ToLower(s)
}
func token(s string) bool {
	if s == "" || len(s) > 512 || s == "." || s == ".." {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.' || c == ':') {
			return false
		}
	}
	return true
}
func ValidateDefinition(d Definition) error {
	if d.FormatVersion != 1 || !token(d.ProjectID) || len(d.Commands) == 0 || len(d.Commands) > 64 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, c := range d.Commands {
		if !gitworkspace.ValidValidationCommand(toCommand(c)) || seen[c.Reference] {
			return ErrInvalid
		}
		seen[c.Reference] = true
		for _, arg := range append(append([]string(nil), c.Argv...), c.Env...) {
			if len(arg) > 4096 || strings.ContainsAny(arg, "\r\n") {
				return ErrInvalid
			}
		}
	}
	return nil
}
func PreviewDigest(d Definition) (string, error) {
	if ValidateDefinition(d) != nil {
		return "", ErrInvalid
	}
	b, e := json.Marshal(d)
	if e != nil || len(b)+1 > MaxPolicyBytes {
		return "", ErrInvalid
	}
	return digest(append(b, '\n')), nil
}
func Validate(p Policy) error {
	expected, e := PreviewDigest(p.Definition)
	_, offset := p.Review.ReviewedAt.Zone()
	if e != nil || expected != p.Review.PreviewDigest || p.Review.ActorKind != "human-operator" || !token(p.Review.Actor) || p.Review.ReviewedAt.IsZero() || offset != 0 || !ValidDigest(p.Review.AuthorityRef) {
		return ErrInvalid
	}
	return nil
}
func Encode(p Policy) ([]byte, error) {
	if Validate(p) != nil {
		return nil, ErrInvalid
	}
	b, e := json.Marshal(p)
	if e != nil || len(b)+1 > MaxPolicyBytes {
		return nil, ErrInvalid
	}
	return append(b, '\n'), nil
}
func Decode(b []byte) (Policy, error) {
	var p Policy
	if strictjson.Decode(b, &p, MaxPolicyBytes) != nil || Validate(p) != nil {
		return Policy{}, ErrInvalid
	}
	return p, nil
}
func digest(b []byte) string { d := sha256.Sum256(b); return hex.EncodeToString(d[:]) }
func Digest(p Policy) (string, error) {
	b, e := Encode(p)
	if e != nil {
		return "", e
	}
	return digest(b), nil
}
func toCommand(c Command) gitworkspace.ValidationCommand {
	return gitworkspace.ValidationCommand{Reference: c.Reference, Argv: append([]string(nil), c.Argv...), Env: append([]string(nil), c.Env...), OutputMax: c.OutputMax}
}

// Commands binds the exact reviewed artifact referenced by a dispatch preview.
func Commands(p Policy, expected string) ([]gitworkspace.ValidationCommand, error) {
	actual, e := Digest(p)
	if e != nil {
		return nil, e
	}
	if !ValidDigest(expected) || actual != expected {
		return nil, ErrDigest
	}
	result := make([]gitworkspace.ValidationCommand, 0, len(p.Commands))
	for _, c := range p.Commands {
		result = append(result, toCommand(c))
	}
	return result, nil
}
