package manifest_test

import (
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func TestExplicitWorkflowSelectionUpgradeRetainsLegacyPresence(t *testing.T) {
	ref := workflowdefinition.Builtin().Ref("builtin")
	for _, version := range []string{"1", "2", "3"} {
		t.Run(version, func(t *testing.T) {
			wire := strings.Replace(minimal, "schemaVersion: 1", "schemaVersion: "+version, 1) + "businessContext: {text: unconfigured, documents: []}\nproviders: unconfigured\n"
			if version == "1" {
				wire += "runtime: {id: codex}\n"
			} else {
				wire += "runtimes: [{id: codex}]\n"
			}
			p, issues := manifest.Decode([]byte(wire))
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			if p.State().WorkflowSelection.Form() != project.Absent {
				t.Fatal("legacy implicitly selected")
			}
			next, issues := p.SelectWorkflow(project.WorkflowSelection{WorkflowID: ref.WorkflowID, Revision: ref.Revision, Digest: ref.Digest, Source: ref.Source})
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			result, issues := manifest.Encode(next)
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			round, issues := manifest.Decode(result)
			if len(issues) > 0 || !round.Equivalent(next) {
				t.Fatalf("round trip %v", issues)
			}
			if next.State().Providers.Form() != p.State().Providers.Form() || next.State().BusinessContext.Form() != p.State().BusinessContext.Form() {
				t.Fatal("presence changed")
			}
			if _, issues := manifest.Decode([]byte(strings.Replace(string(result), "schemaVersion: 4", "schemaVersion: 3", 1))); len(issues) == 0 {
				t.Fatal("downgrade silently accepted schema 4 fields")
			}
			if p.State().SchemaVersion == 4 {
				t.Fatal("original mutated")
			}
		})
	}
}

func TestWorkflowReferenceClosedShape(t *testing.T) {
	base := strings.Replace(minimal, "schemaVersion: 1", "schemaVersion: 4", 1)
	for _, selection := range []string{
		"unconfigured", "null", "{}",
		"{workflowId: default-sdd, revision: 0, digest: abc, source: builtin}",
		"{workflowId: default-sdd, revision: 1, digest: abc, source: latest}",
		"{workflowId: default-sdd, revision: 1, digest: abc, source: builtin, path: /tmp/tool}",
	} {
		if _, issues := manifest.Decode([]byte(base + "workflowSelection: " + selection + "\n")); len(issues) == 0 {
			t.Fatal("invalid selection accepted")
		}
	}
}
