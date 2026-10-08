package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

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
	var content []byte
	if mode == humanOutput {
		content = renderCompletionHuman(result)
		var extra bytes.Buffer
		fmt.Fprintf(&extra, "category: %s\n", response.Category)
		if preview := response.Operational; preview != nil {
			fmt.Fprintf(&extra, "operation: %s", preview.Operation)
			if preview.Integration != "" {
				fmt.Fprintf(&extra, " integration=%s", preview.Integration)
			}
			fmt.Fprintf(&extra, "\nproject: %s revision=%s\n", preview.ProjectID, preview.Revision)
			fmt.Fprintf(&extra, "current: %s disabled=[%s]\n", preview.Current.ProjectStatus, strings.Join(preview.Current.DisabledIntegrations, ","))
			fmt.Fprintf(&extra, "result: %s disabled=[%s]\n", preview.Result.ProjectStatus, strings.Join(preview.Result.DisabledIntegrations, ","))
			for _, effect := range preview.Effects {
				fmt.Fprintf(&extra, "effect: %s %s", effect.Scope, effect.Code)
				if effect.Key != "" {
					fmt.Fprintf(&extra, " key=%s", effect.Key)
				}
				extra.WriteString("\n")
			}
			fmt.Fprintf(&extra, "boundary: portable=%s provider=%s credentials=%s\n", preview.Boundary.Portable, preview.Boundary.Provider, preview.Boundary.Credentials)
			fmt.Fprintf(&extra, "preview-digest: %s\n", preview.Digest)
		}
		content = append(content, extra.Bytes()...)
	} else {
		base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
		wire, err := json.Marshal(operationalCompletionEvent{completionEvent: base, Category: response.Category, Operational: response.Operational})
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
