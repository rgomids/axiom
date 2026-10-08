package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #231: presentation of the canonical readiness report and of the
// operation preflight projection. No readiness rule lives here.

type readinessCompletionEvent struct {
	completionEvent
	Readiness *projectapp.ReadinessReport    `json:"readiness,omitempty"`
	Preflight *projectapp.OperationReadiness `json:"preflight,omitempty"`
}

func emitReadinessCompletion(writer io.Writer, mode outputMode, result completion.Result, response Result) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	var content []byte
	if mode == humanOutput {
		content = renderCompletionHuman(result)
		var extra bytes.Buffer
		if report := response.Readiness; report != nil {
			fmt.Fprintf(&extra, "readiness: %s structure=%s\n", report.Effective, report.Structure)
			for _, operation := range report.Operations {
				renderOperation(&extra, operation)
			}
			for _, repository := range report.Repositories {
				fmt.Fprintf(&extra, "repository: %s binding=%s availability=%s remote=%s\n", repository.Key, repository.Binding, repository.Availability, repository.Remote)
			}
			for _, capability := range report.Capabilities {
				fmt.Fprintf(&extra, "capability: %s provider=%s status=%s credential=%s\n", capability.Capability, capability.Provider, capability.Readiness, capability.Credential)
			}
			fmt.Fprintf(&extra, "runtime: %s %s\n", report.Runtime.Status, report.Runtime.Detail)
			for _, source := range report.Documentation {
				fmt.Fprintf(&extra, "documentation: %s kind=%s status=%s\n", source.Key, source.Kind, source.Status)
			}
			for _, warning := range report.Warnings {
				fmt.Fprintf(&extra, "warning: %s %s\n", warning.Code, warning.Subject)
			}
		}
		if response.Preflight != nil {
			renderOperation(&extra, *response.Preflight)
		}
		content = append(content, extra.Bytes()...)
	} else {
		base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
		wire, err := json.Marshal(readinessCompletionEvent{completionEvent: base, Readiness: response.Readiness, Preflight: response.Preflight})
		if err != nil {
			return ExitFailure
		}
		content = append(wire, '\n')
	}
	if len(content) > MaxCompletionOutputBytes {
		return ExitFailure
	}
	if written, err := writer.Write(content); err != nil || written != len(content) {
		return ExitFailure
	}
	return completionExitCode(result.Status())
}

func renderOperation(output *bytes.Buffer, operation projectapp.OperationReadiness) {
	fmt.Fprintf(output, "operation: %s %s\n", operation.Operation, operation.Status)
	for _, blocker := range operation.Blockers {
		fmt.Fprintf(output, "blocker: %s %s %s\n", blocker.Code, blocker.Subject, blocker.Detail)
	}
}
