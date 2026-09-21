// Package provenance defines Axiom's canonical informational build provenance.
package provenance

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	Product       = "Axiom"
	Development   = "development"
	Unavailable   = "unavailable"
	maxTextBytes  = 512
	maxBuildBytes = 128
)

type SourceState string

const (
	Clean   SourceState = "clean"
	Dirty   SourceState = "dirty"
	Unknown SourceState = "unknown"
)

type Build struct {
	Release     bool
	Version     string
	Revision    string
	SourceState SourceState
}

type Setting struct {
	Key   string
	Value string
}

type Value struct {
	product     string
	version     string
	revision    string
	sourceState SourceState
}

func (v Value) Product() string          { return v.product }
func (v Value) Version() string          { return v.version }
func (v Value) Revision() string         { return v.revision }
func (v Value) SourceState() SourceState { return v.sourceState }

func (v Value) Valid() bool {
	return v.product == Product && validBuildToken(v.version) && validBuildToken(v.revision) && v.sourceState.valid()
}

func FromBuild(input Build, settings []Setting) (Value, error) {
	input = observed(input, settings)
	if input.Release {
		return release(input)
	}
	return development(input)
}

func release(input Build) (Value, error) {
	if !validSemanticVersion(input.Version) || input.Revision == Unavailable || !validBuildToken(input.Revision) || input.SourceState != Clean {
		return Value{}, errors.New("invalid release provenance")
	}
	return Value{product: Product, version: input.Version, revision: input.Revision, sourceState: Clean}, nil
}

func development(input Build) (Value, error) {
	if input.Version == "" {
		input.Version = Development
	}
	if input.Revision == "" {
		input.Revision = Unavailable
	}
	if input.Version != Development || !validBuildToken(input.Revision) || !input.SourceState.valid() {
		return Value{}, errors.New("invalid development provenance")
	}
	return Value{product: Product, version: Development, revision: input.Revision, sourceState: input.SourceState}, nil
}

func observed(input Build, settings []Setting) Build {
	if input.Revision == "" {
		input.Revision = Unavailable
	}
	for _, setting := range settings {
		if setting.Key == "vcs.revision" && input.Revision == Unavailable && validBuildToken(setting.Value) {
			input.Revision = shortRevision(setting.Value)
		}
		if setting.Key == "vcs.modified" && input.SourceState == Unknown {
			input.SourceState = observedState(setting.Value)
		}
	}
	return input
}

func observedState(value string) SourceState {
	if value == "true" {
		return Dirty
	}
	if value == "false" {
		return Clean
	}
	return Unknown
}

func shortRevision(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}

func (s SourceState) valid() bool {
	return s == Clean || s == Dirty || s == Unknown
}

var semanticVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

func validSemanticVersion(value string) bool {
	if !semanticVersion.MatchString(value) {
		return false
	}
	withoutBuild := strings.SplitN(value, "+", 2)[0]
	parts := strings.SplitN(withoutBuild, "-", 2)
	if len(parts) == 1 {
		return true
	}
	for _, identifier := range strings.Split(parts[1], ".") {
		if len(identifier) > 1 && identifier[0] == '0' && numeric(identifier) {
			return false
		}
	}
	return true
}

func numeric(value string) bool {
	for _, current := range value {
		if current < '0' || current > '9' {
			return false
		}
	}
	return true
}

func validBuildToken(value string) bool {
	if value == "" || len(value) > maxBuildBytes || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, current := range value {
		if unicode.IsControl(current) || unicode.IsSpace(current) {
			return false
		}
	}
	return true
}

type Authorship string

const (
	AxiomAuthored Authorship = "axiom"
	UserAuthored  Authorship = "user"
)

type Text struct {
	value      string
	authorship Authorship
}

func NewText(value string, authorship Authorship) (Text, error) {
	if authorship != AxiomAuthored && authorship != UserAuthored {
		return Text{}, errors.New("invalid text authorship")
	}
	if !validText(value) {
		return Text{}, errors.New("invalid provenance text")
	}
	return Text{value: value, authorship: authorship}, nil
}

func RequireAxiomAuthored(value Text) (Text, error) {
	if value.authorship != AxiomAuthored || !validText(value.value) {
		return Text{}, errors.New("text is not Axiom-authored")
	}
	return value, nil
}

func (t Text) String() string         { return t.value }
func (t Text) Authorship() Authorship { return t.authorship }
func (t Text) Empty() bool            { return t.value == "" }

func validText(value string) bool {
	if value == "" || len(value) > maxTextBytes || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, current := range value {
		if unicode.IsControl(current) {
			return false
		}
	}
	return true
}
