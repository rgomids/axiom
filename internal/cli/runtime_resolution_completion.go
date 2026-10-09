package cli

import (
	"encoding/json"
	"io"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
)

type runtimeResolutionCompletionEvent struct {
	completionEvent
	RuntimeResolution runtimeapplication.Preview `json:"runtimeResolution"`
	ExecutionTarget   *ExecutionTargetView       `json:"executionTarget,omitempty"`
	PreviewDigest     string                     `json:"previewDigest"`
}

func emitRuntimeResolutionCompletion(writer io.Writer, mode outputMode, result completion.Result, response Result) int {
	if writer == nil || !result.Valid() || response.RuntimeResolution == nil {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	value := runtimeResolutionCompletionEvent{completionEvent: base, RuntimeResolution: *response.RuntimeResolution, PreviewDigest: response.PreviewDigest, ExecutionTarget: response.ExecutionTarget}
	wire, err := json.Marshal(value)
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), append(wire, '\n'), MaxCompletionOutputBytes)
}

type runtimeAuthCompletionEvent struct {
	completionEvent
	RuntimeAuth runtimeadapter.AuthReport `json:"runtimeAuth"`
}

// emitRuntimeAuthCompletion presents the sanitized preflight report; human
// and JSON modes carry the same fields.
func emitRuntimeAuthCompletion(writer io.Writer, mode outputMode, result completion.Result, report runtimeadapter.AuthReport) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	wire, err := json.Marshal(runtimeAuthCompletionEvent{completionEvent: base, RuntimeAuth: report})
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), append(wire, '\n'), MaxCompletionOutputBytes)
}
