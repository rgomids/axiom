package cli

import (
	"encoding/json"
	"io"

	"github.com/rgomids/axiom/internal/completion"
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
	content, err := json.Marshal(value)
	if err != nil {
		return ExitFailure
	}
	if mode == humanOutput {
		extra, err := json.Marshal(value.RuntimeResolution)
		if err != nil {
			return ExitFailure
		}
		content = renderCompletionHuman(result)
		if value.ExecutionTarget != nil {
			target, err := json.Marshal(value.ExecutionTarget)
			if err != nil {
				return ExitFailure
			}
			content = append(content, "executionTarget: "...)
			content = append(content, target...)
			content = append(content, '\n')
		}
		content = append(content, "runtimeResolution: "...)
		content = append(content, extra...)
		content = append(content, "\npreviewDigest: "...)
		content = append(content, value.PreviewDigest...)
	}
	content = append(content, '\n')
	if len(content) > MaxCompletionOutputBytes {
		return ExitFailure
	}
	written, err := writer.Write(content)
	if err != nil || written != len(content) {
		return ExitFailure
	}
	return completionExitCode(result.Status())
}
