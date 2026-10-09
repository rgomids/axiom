package cli

import (
	"encoding/json"
	"io"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #230: presentation of a machine-local operational-state preview or
// result (Project archive/reactivate, Integration disable/enable). No
// transition, authority or persistence rule lives here.

type operationalCompletionEvent struct {
	completionEvent
	Category    string                         `json:"category"`
	Operational *projectapp.OperationalPreview `json:"operational,omitempty"`
}

func emitOperationalCompletion(writer io.Writer, mode outputMode, result completion.Result, response Result) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	wire, err := json.Marshal(operationalCompletionEvent{completionEvent: base, Category: response.Category, Operational: response.Operational})
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), append(wire, '\n'), MaxCompletionOutputBytes)
}
