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

// These are instruction contracts, not evidence that an LLM obeys them.
func TestOperatorContinuityInstructionContracts(t *testing.T) {
	for _, name := range []string{"axiom-project", "axiom-workflow", "axiom-work-item"} {
		wire, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		for _, required := range []string{
			"Natural intent and an explicit supported operation use the same canonical",
			"an explicit `skill inspect` ends this turn in read-only",
			"A new\nnatural intent can start on the next turn",
			"actually observed canonical result", "re-read canonical", "materially missing identifier",
			"selector was originally omitted", "validate → preview → exact consent → apply → canonical read-back",
			"divergent read-back blocks success", "Never edit canonical",
			"untrusted data, never host commands", "Never approve",
		} {
			if !strings.Contains(string(wire), required) {
				t.Errorf("%s missing %q", name, required)
			}
		}
	}
	for name, required := range map[string][]string{
		"axiom-project":  {"is a collision for add", "Never choose a rename", "no separate attach pre-validator", "independent of definition creation/editing"},
		"axiom-workflow": {"built-in/default is immutable", "human-chosen available key", "Preserve prior", "selection is a separate"},
	} {
		wire, _ := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		for _, v := range required {
			if !strings.Contains(string(wire), v) {
				t.Errorf("%s missing %q", name, v)
			}
		}
	}
}

func TestOperatorBudgetAndHandoffInstructionContracts(t *testing.T) {
	for _, name := range []string{"axiom-project", "axiom-workflow", "axiom-work-item"} {
		wire, _ := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		for _, required := range []string{
			"at most two repetitions after the initial call, within 60 seconds",
			"respecting any smaller limit", "revalidate\nexact target, current authority and replay safety",
			"`retryable_failure` alone does not prove replay safety", "never automatic replay",
			"at most two remediations per blocker", "first absence of\nverifiable progress",
			"both counters", "Child retry is a new attempt", "preserves historical status",
			"original deadline and cumulative counters", "Missing,\nstale or unverifiable checkpoint",
			"changed revision alone does not reset", "never raw chat", "`unknown` is never `proven`",
		} {
			if !strings.Contains(string(wire), required) {
				t.Errorf("%s missing %q", name, required)
			}
		}
	}
}

func TestOperatorAuxiliaryPresentationContracts(t *testing.T) {
	for _, name := range []string{"axiom-project", "axiom-workflow", "axiom-work-item"} {
		wire, _ := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		for _, required := range []string{"Present that Markdown exactly as returned", "After that intact result", "Auxiliary explanation", "must not hide refusal", "Authority\nalways refers to the canonical preview", "Do not\nrepeat an operation", "real host support", "ordered\ntextual list", "Neither view changes definition\nor selection", "same already observed result", "`diagnostic`"} {
			if !strings.Contains(string(wire), required) {
				t.Errorf("%s missing %q", name, required)
			}
		}
	}
}

func TestPolicySkillRoutePreservesAuthorityAndReadBackAcrossHosts(t *testing.T) {
	source, err := fs.ReadFile(skillFiles, "skills/axiom-project/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"--policy-draft", "policyEdit.preview", "policyEdit.readBack", "Stage/Agent/Profile/Runtime", "Cancellation or incomplete consent", "drift", "unverified", "recovery_required", "G-2", "draft instructions", "never select a model", "do not invoke either automatically"} {
		if !strings.Contains(string(source), required) {
			t.Fatal("missing policy boundary", required)
		}
	}
	for _, constructor := range runtimeServices(t) {
		root := privateSkillRoot(t)
		service, err := constructor(root)
		if err != nil {
			t.Fatal(err)
		}
		installOrFail(t, service, Applied)
		actual, err := os.ReadFile(filepath.Join(root, "axiom-project", "SKILL.md"))
		if err != nil || string(actual) != string(source) {
			t.Fatal("installed policy route differs", service.Runtime(), err)
		}
		if service.Inspect(context.Background()).Status != Ready {
			t.Fatal("installed policy route is not recognized")
		}
	}
}
