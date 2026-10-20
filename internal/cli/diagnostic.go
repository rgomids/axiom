package cli

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/executiongraph"
)

// Diagnostic is optional output guidance, never persisted state or authority.
// Mappers supply bounded, application-owned observations, never raw errors.
type Diagnostic struct {
	Attempt              *executiongraph.AttemptDiagnostic `json:"attempt,omitempty"`
	SchemaVersion        int                               `json:"schemaVersion"`
	Operation            string                            `json:"operation"`
	Category             string                            `json:"category"`
	Target               map[string]string                 `json:"target"`
	ObservedRevision     string                            `json:"observedRevision"`
	Observed             []string                          `json:"observed"`
	NotObserved          []string                          `json:"notObserved"`
	Hypotheses           []string                          `json:"hypotheses"`
	MissingPrerequisites []string                          `json:"missingPrerequisites"`
	Capability           string                            `json:"capability"`
	Authority            string                            `json:"authority"`
	Confinement          string                            `json:"confinement"`
	ConfirmedEffects     []string                          `json:"confirmedEffects"`
	UncertainEffects     []string                          `json:"uncertainEffects"`
	Options              []DiagnosticOption                `json:"options"`
	Verification         DiagnosticVerification            `json:"verification"`
}

type DiagnosticOption struct {
	ID                string                 `json:"id"`
	Availability      string                 `json:"availability"`
	Prerequisites     []string               `json:"prerequisites"`
	Effects           []string               `json:"effects"`
	RequiredAuthority []string               `json:"requiredAuthority"`
	Operation         string                 `json:"operation,omitempty"`
	Arguments         map[string]string      `json:"arguments"`
	MissingArguments  []string               `json:"missingArguments"`
	Purpose           string                 `json:"purpose"`
	Verification      DiagnosticVerification `json:"verification"`
}

type DiagnosticVerification struct {
	Operation    string `json:"operation"`
	ExpectedFact string `json:"expectedFact"`
}

// boundedDiagnostic omits invalid enrichment while retaining the canonical
// refusal. This boundary never truncates a result or turns it into success.
func boundedDiagnostic(d *Diagnostic) *Diagnostic {
	if d == nil || d.SchemaVersion != 1 || len(d.Options) > 3 || d.Attempt != nil && executiongraph.ValidateAttemptDiagnostic(*d.Attempt) != nil {
		return nil
	}
	for _, o := range d.Options {
		if o.Availability != "available" && o.Availability != "conditional" && o.Availability != "unavailable" {
			return nil
		}
		if o.Availability == "unavailable" && o.Operation != "" {
			return nil
		}
	}
	wire, err := json.Marshal(d)
	if err != nil || len(wire) > 16*1024 {
		return nil
	}
	return d
}

// Append only when enrichment fits; the original refusal is always retained.
func presentDiagnosticEvent(out io.Writer, mode outputMode, status completion.Status, wire []byte, limit int, d *Diagnostic) int {
	if d = boundedDiagnostic(d); d != nil {
		raw, err := json.Marshal(d)
		base := bytes.TrimSpace(wire)
		if err == nil && len(base) > 0 && base[len(base)-1] == '}' && len(base)+len(raw)+15 <= limit {
			enriched := make([]byte, 0, len(base)+len(raw)+15)
			enriched = append(enriched, base[:len(base)-1]...)
			enriched = append(enriched, `,"diagnostic":`...)
			enriched = append(enriched, raw...)
			wire = append(enriched, '}', '\n')
		}
	}
	return presentEvent(out, mode, status, wire, limit)
}
