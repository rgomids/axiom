package codexruntime

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkflowSkillCoversSchemaAndAuthorityBoundaries(t *testing.T) {
	wire, err := fs.ReadFile(skillFiles, "skills/axiom-workflow/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(wire)
	schemaWire, err := os.ReadFile("../../docs/specifications/007-configurable-workflows/workflow.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(schemaWire, &schema); err != nil {
		t.Fatal(err)
	}
	var visit func(map[string]any)
	visit = func(node map[string]any) {
		if props, ok := node["properties"].(map[string]any); ok {
			for name, property := range props {
				if !strings.Contains(text, "`"+name+"`") {
					t.Errorf("missing schema field %s", name)
				}
				visit(property.(map[string]any))
			}
		}
		if items, ok := node["items"].(map[string]any); ok {
			visit(items)
		}
	}
	visit(schema)
	for _, required := range []string{"regular file", "outside Project, Repository and Axiom state roots", "Configuration Valid", "Runtime Ready", "Authentication Ready", "never blocks", "point-in-time", "#305", "#306", "#231", "#307", "#276", "#277", "policy-default", "separate `axiom-project`", "never grants authority"} {
		if !strings.Contains(text, required) {
			t.Errorf("missing boundary %q", required)
		}
	}
	project, _ := fs.ReadFile(skillFiles, "skills/axiom-project/SKILL.md")
	for _, required := range []string{"`activeWorkflow`", "`none`", "`unresolvable`", "separate", "never create a change preview"} {
		if !strings.Contains(string(project), required) {
			t.Errorf("Project missing %q", required)
		}
	}
	item, _ := fs.ReadFile(skillFiles, "skills/axiom-work-item/SKILL.md")
	for _, required := range []string{"## acceptance (stage-plan, R-1)", "provisional `approved: true`", "`planDocumentDigest`", "`planRevision`", "`planDigest`", "`stageId`", "`workflowRef`", "`plan.digest`", "no Axiom fact", "#278"} {
		if !strings.Contains(string(item), required) {
			t.Errorf("Work Item missing %q", required)
		}
	}
}

func TestInstalledWorkflowSkillOwnershipAfterInstallAndReinstall(t *testing.T) {
	for _, runtime := range []integration{codexIntegration, claudeIntegration} {
		t.Run(runtime.runtime, func(t *testing.T) {
			root := privateSkillRoot(t)
			service := Service{root: root, integration: runtime, binaryCompatibility: BinaryCompatibility}
			for _, expected := range []Status{Applied, Unchanged} {
				got := service.Install(context.Background())
				if got.Status != expected {
					t.Fatalf("install=%+v", got)
				}
				for _, name := range skillNames {
					installed, err := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
					if err != nil {
						t.Fatal(err)
					}
					embedded, _ := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
					if string(installed) != string(embedded) {
						t.Fatalf("%s bytes differ", name)
					}
					if name == "axiom-project" {
						for _, old := range []string{"list", "show", "create", "edit", "validate", "remove", "recover"} {
							if strings.Contains(string(installed), "workflow."+old) {
								t.Fatalf("stale Project route %s", old)
							}
						}
					}
				}
			}
		})
	}
}
