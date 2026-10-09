package cli

import (
	"encoding/json"
	"io"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #231: presentation of the canonical readiness report and of the
// operation preflight projection; Issue #230 adds the admission decision of
// the same pre-effect gate. No readiness or admission rule lives here.

type readinessCompletionEvent struct {
	completionEvent
	Readiness *projectapp.ReadinessReport    `json:"readiness,omitempty"`
	Preflight *projectapp.OperationReadiness `json:"preflight,omitempty"`
	Admission *projectapp.AdmissionDecision  `json:"admission,omitempty"`
	State     *ProjectStateView              `json:"state,omitempty"`
}

func emitReadinessCompletion(writer io.Writer, mode outputMode, result completion.Result, response Result) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	wire, err := json.Marshal(readinessCompletionEvent{completionEvent: base, Readiness: response.Readiness, Preflight: response.Preflight, Admission: response.Admission, State: response.ProjectState})
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), append(wire, '\n'), MaxCompletionOutputBytes)
}
