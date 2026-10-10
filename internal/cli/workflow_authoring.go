package cli

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"strings"

	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/provenance"
)

type ProjectWorkflowInput struct {
	projectapp.WorkflowRequest
	File string
}
type ProjectWorkflowService interface {
	ProjectWorkflow(context.Context, ProjectWorkflowInput) Result
}

func workflowAuthoringFlagSet(op action, in *ProjectWorkflowInput) *flag.FlagSet {
	set := flag.NewFlagSet(string(op), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(&in.Project, "project", "", "Exact installed Project; `<uuid-or-slug>`.")
	operation := strings.TrimPrefix(string(op), "project_workflow_")
	in.Operation = operation
	switch operation {
	case "create", "edit", "validate", "recover":
		set.StringVar(&in.File, "file", "", "Complete UTF-8 JSON definition; `<path>`.")
	}
	if operation == "create" {
		set.BoolVar(&in.FromDefault, "from-default", false, "Copy the ready-to-use SDD definition.")
		set.StringVar(&in.WorkflowID, "workflow", "", "New logical workflow key; `<key>`.")
	}
	if operation == "show" || operation == "select" || operation == "remove" {
		set.StringVar(&in.Ref.WorkflowID, "workflow", "", "Exact workflow key; `<key>`.")
		set.IntVar(&in.Ref.Revision, "revision", 0, "Exact immutable revision; `<positive-integer>`.")
		set.StringVar(&in.Ref.Digest, "digest", "", "Exact content SHA-256; `<digest>`.")
		set.StringVar(&in.Ref.Source, "source", "project", "Reference source; `builtin|project`.")
	}
	if operation == "edit" {
		in.Prior.Source = "project"
		set.StringVar(&in.Prior.WorkflowID, "workflow", "", "Existing workflow key; `<key>`.")
		set.IntVar(&in.Prior.Revision, "prior-revision", 0, "Reviewed prior revision; `<positive-integer>`.")
		set.StringVar(&in.Prior.Digest, "prior-digest", "", "Reviewed prior SHA-256; `<digest>`.")
	}
	if operation != "list" && operation != "show" && operation != "validate" {
		set.StringVar(&in.ExpectedRevision, "expected-revision", "", "Expected Project revision returned by preview; `<digest>`.")
		set.StringVar(&in.PreviewDigest, "preview-digest", "", "Exact reviewed preview; `<digest>`.")
		set.BoolVar(&in.AuthorizeLocal, "authorize-local", false, "Confirm only the exact reviewed local effects.")
	}
	return set
}
func runProjectWorkflow(ctx context.Context, mode outputMode, args []string, service Service, source provenance.Value, out io.Writer) (bool, int) {
	command, _, rest, ok := resolveCommand(args)
	if !ok || !strings.HasPrefix(string(command.operation), "project_workflow_") {
		return false, 0
	}
	in := ProjectWorkflowInput{}
	set := workflowAuthoringFlagSet(command.operation, &in)
	if set.Parse(rest) != nil || set.NArg() != 0 || duplicateFlag(rest) || in.Project == "" {
		return true, emitParserFailure(out, mode, command.operation, "invalid_input", source)
	}
	adapter, ok := service.(ProjectWorkflowService)
	if !ok {
		return true, emit(out, mode, event{Operation: command.operation, Status: Failed, Category: "application_unavailable"})
	}
	response := adapter.ProjectWorkflow(ctx, in)
	if response.Completion == nil {
		return true, emitResponse(out, mode, command.operation, response)
	}
	c := *response.Completion
	base := completionEvent{Status: c.Status(), Result: c.Result().String(), References: c.References(), Next: c.Next().String(), Details: c.Details(), Provenance: provenanceEvent{Product: c.Provenance().Product(), Version: c.Provenance().Version(), Revision: c.Provenance().Revision(), SourceState: c.Provenance().SourceState()}}
	value := struct {
		completionEvent
		Category string                     `json:"category"`
		Workflow *projectapp.WorkflowReport `json:"workflowAuthoring,omitempty"`
	}{base, response.Category, response.WorkflowAuthoring}
	wire, e := json.Marshal(value)
	if e != nil {
		return true, ExitFailure
	}
	return true, presentEvent(out, mode, c.Status(), append(wire, '\n'), 2<<20)
}
func duplicateFlag(args []string) bool {
	seen := map[string]bool{}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			key := strings.SplitN(strings.TrimLeft(a, "-"), "=", 2)[0]
			if seen[key] {
				return true
			}
			seen[key] = true
		}
	}
	return false
}

func workflowAuthoringCommands() commandDefinition {
	var children []commandDefinition
	for _, op := range []string{"list", "show", "create", "edit", "validate", "select", "remove", "recover"} {
		children = append(children, leaf(op, "Project workflow "+op+" (immutable revisions)", action("project_workflow_"+op)))
	}
	return group("workflow", "Author and select Project workflow definitions", children...)
}
func workflowAuthoringSkillSpecs() []skillOperationSpec {
	var specs []skillOperationSpec
	for _, op := range []string{"list", "show", "create", "edit", "validate", "select", "remove", "recover"} {
		act := action("project_workflow_" + op)
		effect, authority := effectReadOnly, "none"
		inputs := []string{}
		if op != "list" && op != "show" && op != "validate" {
			effect = effectLocalMutation
			authority = "preview first; exact --expected-revision and --preview-digest plus --authorize-local"
			inputs = []string{"--expected-revision", "--preview-digest", "--authorize-local"}
		}
		mode := skillModeSpec{name: "default", selector: "--project", effect: effect, authority: authority, actions: []action{act}, authorityInputs: inputs, rejectedInputs: []string{}, example: "axiom project workflow " + op + " --project <uuid-or-slug>"}
		specs = append(specs, skillOperationSpec{name: "workflow." + op, actions: []action{act}, modes: []skillModeSpec{mode}})
	}
	return specs
}
