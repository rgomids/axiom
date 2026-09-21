package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/provenance"
)

const MaxCompletionOutputBytes = 16 * 1024

type CompletionFormat string

const (
	CompletionHuman CompletionFormat = "human"
	CompletionJSON  CompletionFormat = "json"
)

type completionEvent struct {
	Status     completion.Status `json:"status"`
	Result     string            `json:"result"`
	References []string          `json:"references,omitempty"`
	Next       string            `json:"next,omitempty"`
	Details    string            `json:"details,omitempty"`
	Provenance provenanceEvent   `json:"provenance"`
}

type provenanceEvent struct {
	Product     string                 `json:"product"`
	Version     string                 `json:"version"`
	Revision    string                 `json:"revision"`
	SourceState provenance.SourceState `json:"sourceState"`
}

func WriteCompletion(writer io.Writer, format CompletionFormat, result completion.Result) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	content, err := renderCompletion(format, result)
	if err != nil || len(content) > MaxCompletionOutputBytes {
		return ExitFailure
	}
	written, err := writer.Write(content)
	if err != nil || written != len(content) {
		return ExitFailure
	}
	return completionExitCode(result.Status())
}

func emitCompletion(writer io.Writer, mode outputMode, result completion.Result) int {
	if mode == jsonOutput {
		return WriteCompletion(writer, CompletionJSON, result)
	}
	return WriteCompletion(writer, CompletionHuman, result)
}

func renderCompletion(format CompletionFormat, result completion.Result) ([]byte, error) {
	if format == CompletionJSON {
		return renderCompletionJSON(result)
	}
	if format == CompletionHuman {
		return renderCompletionHuman(result), nil
	}
	return nil, errors.New("unsupported completion format")
}

func renderCompletionJSON(result completion.Result) ([]byte, error) {
	value := completionEvent{
		Status:     result.Status(),
		Result:     result.Result().String(),
		References: result.References(),
		Next:       result.Next().String(),
		Details:    result.Details(),
		Provenance: provenanceEvent{
			Product:     result.Provenance().Product(),
			Version:     result.Provenance().Version(),
			Revision:    result.Provenance().Revision(),
			SourceState: result.Provenance().SourceState(),
		},
	}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func renderCompletionHuman(result completion.Result) []byte {
	var output bytes.Buffer
	fmt.Fprintf(&output, "status: %s\n", result.Status())
	fmt.Fprintf(&output, "result: %s\n", result.Result().String())
	for _, reference := range result.References() {
		fmt.Fprintf(&output, "reference: %s\n", reference)
	}
	if !result.Next().Empty() {
		fmt.Fprintf(&output, "next: %s\n", result.Next().String())
	}
	if result.Details() != "" {
		fmt.Fprintf(&output, "details: %s\n", result.Details())
	}
	fmt.Fprintf(&output, "provenance: %s %s revision=%s source=%s\n", result.Provenance().Product(), result.Provenance().Version(), result.Provenance().Revision(), result.Provenance().SourceState())
	return output.Bytes()
}

func completionExitCode(status completion.Status) int {
	if status == completion.Success {
		return ExitSuccess
	}
	if status == completion.Interrupted {
		return ExitCancelled
	}
	return ExitFailure
}
