package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workitem"
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
	if writer == nil || !result.Valid() || (format != CompletionHuman && format != CompletionJSON) {
		return ExitFailure
	}
	wire, err := renderCompletionJSON(result)
	if err != nil {
		return ExitFailure
	}
	mode := humanOutput
	if format == CompletionJSON {
		mode = jsonOutput
	}
	return presentEvent(writer, mode, result.Status(), wire, MaxCompletionOutputBytes)
}

func emitCompletion(writer io.Writer, mode outputMode, result completion.Result) int {
	if mode == jsonOutput {
		return WriteCompletion(writer, CompletionJSON, result)
	}
	return WriteCompletion(writer, CompletionHuman, result)
}

type setupCompletionEvent struct {
	completionEvent
	Setup projectapp.SetupPreview `json:"setup"`
}

type projectCompletionEvent struct {
	completionEvent
	Project ProjectView `json:"project"`
}

type projectListCompletionEvent struct {
	completionEvent
	Projects []ProjectListView `json:"projects"`
}

type runtimeCompletionEvent struct {
	completionEvent
	Runtime RuntimeView `json:"runtime"`
}

type workItemCompletionEvent struct {
	completionEvent
	Draft     *workitem.DraftPreview     `json:"draft,omitempty"`
	Selection *workitem.SelectionPreview `json:"selection,omitempty"`
	WorkItem  *WorkItemView              `json:"workItem,omitempty"`
	Questions []workitem.Question        `json:"questions,omitempty"`
}

type workflowCompletionEvent struct {
	completionEvent
	Workflow   *WorkflowView               `json:"workflow,omitempty"`
	Projection *workflow.ProjectionPreview `json:"projection,omitempty"`
}

const maxWorkItemPreviewOutputBytes = 128 * 1024

func emitWorkItemCompletion(writer io.Writer, mode outputMode, result completion.Result, response Result) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	value := workItemCompletionEvent{completionEvent: base, Draft: response.Draft, Selection: response.Selection, WorkItem: response.WorkItem}
	if len(response.Questions) != 0 {
		value.Questions = response.Questions
	}
	content, err := marshalWorkItemValue(value, false)
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), content, maxWorkItemPreviewOutputBytes)
}

func emitWorkflowCompletion(writer io.Writer, mode outputMode, result completion.Result, response Result) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	value := workflowCompletionEvent{completionEvent: base, Workflow: response.Workflow, Projection: response.Projection}
	content, err := json.Marshal(value)
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), append(content, '\n'), maxWorkItemPreviewOutputBytes)
}

func marshalWorkItemValue(value any, indented bool) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if indented {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	if output.Len() > maxWorkItemPreviewOutputBytes {
		return nil, errors.New("work item preview exceeds output limit")
	}
	return output.Bytes(), nil
}

func emitRuntimeCompletion(writer io.Writer, mode outputMode, result completion.Result, runtime RuntimeView) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	content, err := json.Marshal(runtimeCompletionEvent{completionEvent: base, Runtime: runtime})
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), append(content, '\n'), MaxCompletionOutputBytes)
}

type bootstrapCompletionEvent struct {
	completionEvent
	FirstRun BootstrapView `json:"firstRun"`
}

func emitBootstrapCompletion(writer io.Writer, mode outputMode, result completion.Result, bootstrap BootstrapView) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	encoded, err := json.Marshal(bootstrapCompletionEvent{completionEvent: base, FirstRun: bootstrap})
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), append(encoded, '\n'), MaxCompletionOutputBytes)
}

func emitRuntimeHuman(writer io.Writer, runtime RuntimeView) {
	fmt.Fprintf(writer, "skill-set: %s binary-compatibility=%s\n", runtime.SkillSetVersion, runtime.BinaryCompatibility)
	for _, skill := range runtime.Skills {
		fmt.Fprintf(writer, "skill: %s sha256=%s state=%s\n", skill.Name, skill.SHA256, skill.State)
	}
	if runtime.Receipt != "" {
		fmt.Fprintf(writer, "receipt: state=%s\n", runtime.Receipt)
	}
	for _, conflict := range runtime.Conflicts {
		io.WriteString(writer, conflictLine(conflict))
	}
}

// conflictLine renders one preserved artifact relative to the Runtime skill
// root; Axiom never overwrites it.
func conflictLine(conflict RuntimeConflictView) string {
	line := "conflict: artifact=" + conflict.Artifact + " state=" + conflict.State
	if conflict.SHA256 != "" {
		line += " sha256=" + conflict.SHA256
	}
	return line + " preserved=true\n"
}

func emitSetupCompletion(writer io.Writer, mode outputMode, result completion.Result, setup projectapp.SetupPreview) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	content, err := json.Marshal(setupCompletionEvent{completionEvent: base, Setup: setup})
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), append(content, '\n'), MaxCompletionOutputBytes)
}

type editCompletionEvent struct {
	completionEvent
	Edit projectapp.EditPreview `json:"edit"`
}

// An EDIT preview carries the complete portable manifest, so it uses the
// larger bounded preview budget rather than the summary-only limit.
func emitEditCompletion(writer io.Writer, mode outputMode, result completion.Result, edit projectapp.EditPreview) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	encoded, err := marshalWorkItemValue(editCompletionEvent{completionEvent: base, Edit: edit}, false)
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), encoded, maxWorkItemPreviewOutputBytes)
}

func emitProjectCompletion(writer io.Writer, mode outputMode, result completion.Result, project ProjectView) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	content, err := json.Marshal(projectCompletionEvent{completionEvent: base, Project: project})
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), append(content, '\n'), MaxCompletionOutputBytes)
}

const maxProjectListOutputBytes = 2 * 1024 * 1024

func emitProjectListCompletion(writer io.Writer, mode outputMode, result completion.Result, projects []ProjectListView) int {
	if writer == nil || !result.Valid() || projects == nil {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	encoded, err := json.Marshal(projectListCompletionEvent{completionEvent: base, Projects: projects})
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), append(encoded, '\n'), maxProjectListOutputBytes)
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

func completionExitCode(status completion.Status) int {
	if status == completion.Success {
		return ExitSuccess
	}
	if status == completion.Interrupted {
		return ExitCancelled
	}
	return ExitFailure
}
