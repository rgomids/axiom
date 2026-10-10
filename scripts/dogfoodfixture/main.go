// dogfoodfixture publishes synthetic typed outputs through the production
// artifact store. It never edits canonical Execution records.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/detailartifact"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "synthetic dogfood fixture failed:", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) == 2 && os.Args[1] == "definition" {
		return json.NewEncoder(os.Stdout).Encode(workflowdefinition.Builtin().Ref("builtin"))
	}
	if len(os.Args) != 3 || os.Args[1] != "result" {
		return fmt.Errorf("expected definition or result STATE_ROOT")
	}
	var event struct {
		Workflow   *cli.WorkflowView `json:"workflow"`
		Provenance provenance.Build  `json:"provenance"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&event); err != nil {
		return err
	}
	v := event.Workflow
	if v == nil || v.Binding == nil || v.StageContract == nil {
		return fmt.Errorf("missing configured stage")
	}
	store, err := local.NewArtifactStore(os.Args[2])
	if err != nil {
		return err
	}
	source, err := provenance.FromBuild(event.Provenance, nil)
	if err != nil {
		return err
	}
	result := workflow.StageResult{Inputs: v.StageInputs, Outputs: map[string]workflow.Reference{}}
	for _, output := range v.StageContract.Outputs {
		a, err := store.Create(context.Background(), detailartifact.Draft{ExecutionID: v.ExecutionID, Category: output.Kind, Outcome: "success", Retention: detailartifact.Evidence, Markdown: []byte("# Synthetic dogfood output\n"), References: []detailartifact.Reference{{Kind: "workflow-stage", Value: v.StageContract.ID}, {Kind: "workflow-definition", Value: v.Binding.Definition.Digest}, {Kind: "workflow-output", Value: output.ID}}, Provenance: source})
		if err != nil {
			return err
		}
		result.Outputs[output.ID] = workflow.Reference{Kind: "artifact", ID: a.ID, Digest: hex.EncodeToString(a.Digest[:])}
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
