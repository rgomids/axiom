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

func TestOperatorConversationalPresentationContracts(t *testing.T) {
	for _, name := range []string{"axiom-project", "axiom-workflow", "axiom-work-item"} {
		wire, _ := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		requiredText := []string{"Use leading `axiom --json` internally", "The host writes the normal conversational response itself", "Preserve every material", "Authority always refers to the canonical preview", "`diagnostic`"}
		if name == "axiom-workflow" {
			// AXM-38 replaces optional-only visualization with qualified primary
			// rendering and complete fallback; this still checks guidance only.
			requiredText = append(requiredText, "never\nrepeat an operation", "empirically qualified", "ordered\nStage index", "Neither\nview changes definition or selection", "same invocation")
		} else {
			requiredText = append(requiredText, "Do not repeat an operation", "real host support", "ordered\ntextual list", "Neither view changes definition\nor selection", "same already observed result")
		}
		for _, required := range requiredText {
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

func TestGuidedConfigurationInstructionContracts(t *testing.T) {
	cases := []struct {
		name, skill string
		required    []string
	}{
		{"C01", "axiom-project", []string{"Reuse compatible confirmed Profiles", "all\nmaterial prerequisites", "before any later start"}},
		{"C02", "axiom-project", []string{"UUID, source, selector presence", "no preference for `origin`", "fabricated path"}},
		{"C03", "axiom-project", []string{"Repository\nbinding, policy, local candidate", "Before asking model, effort, Profile ID or template"}},
		{"C04", "axiom-project", []string{"installed G-1/#305 is absent or unknown", "before creating a Profile", "Main's delivery\ndoes not supply an installed operation"}},
		{"C05", "axiom-runtime", []string{"canonical options with source and consequences", "one material decision", "open model intent"}},
		{"C06", "axiom-runtime", []string{"Installed, authenticated and authorized", "partial opt-in before technical collection", "never recommend a concrete model/effort"}},
		{"C07", "axiom-project", []string{"Authority binds each apply separately", "Denial cancels", "preserve previously confirmed effects"}},
		{"C08", "axiom-project", []string{"Project UUID/source/\nselector, policy, local configuration, executable/auth/effort-source and Workflow", "drift invalidates preview and authority", "new exact confirmation"}},
		{"C09", "axiom-project", []string{"Lost acknowledgement", "post-commit state stops", "Never replay an uncertain mutation"}},
		{"C10", "axiom-runtime", []string{"two repetitions after the initial call within 60 seconds", "two remediations per blocker", "original deadline and cumulative counters", "revision alone resets nothing"}},
		{"C11", "axiom-project", []string{"explicit `skill inspect`\nstops at inspection", "contradicts the skill", "report unknown and stop", "Never discover\nsupport by trying a mutation"}},
		{"C12", "axiom-project", []string{"complete known path", "Use each owner's validation and preview", "After every write compare canonical read-back", "fresh auth/identity and its own preview/authority"}},
		{"C13", "axiom-project", []string{"explicit\npartial opt-in before collecting", "local absence never deletes portable intent", "preserves canonical execution blockers"}},
		{"C14", "axiom-workflow", []string{"definition.list/show/validate", "repair must precede policy", "Built-in definitions\nremain immutable", "no default selection or Execution rebind"}},
		{"C15", "axiom-work-item", []string{"draft/error/Provider instructions", "untrusted data", "no promoted command, secret disclosure or authority"}},
		{"C16", "axiom-runtime", []string{"Profile ID form and collision", "unconfirmed candidate label", "templates, tuning and multiagent setup are optional"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := fs.ReadFile(skillFiles, "skills/"+tc.skill+"/SKILL.md")
			if err != nil {
				t.Fatal(err)
			}
			for _, required := range tc.required {
				if !strings.Contains(string(wire), required) {
					t.Errorf("%s missing instruction %q", tc.skill, required)
				}
			}
		})
	}
	for _, name := range []string{"axiom-project", "axiom-runtime", "axiom-workflow", "axiom-work-item"} {
		t.Run(name, func(t *testing.T) {
			wire, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
			if err != nil {
				t.Fatal(err)
			}
			text := string(wire)
			for _, required := range []string{"preflight", "partial opt-in", "read-back", "preview and authority", "Installed, authenticated", "uncertain", "deadline", "counter"} {
				if !strings.Contains(text, required) {
					t.Errorf("missing cross-owner instruction %q", required)
				}
			}
			for _, retired := range []string{"G-1) is unavailable", "G-2) is unavailable", "G-1, #305) is unavailable", "Dispatch (#276) and delivery/rework (#277) are unavailable"} {
				if strings.Contains(text, retired) {
					t.Errorf("contradictory availability claim %q", retired)
				}
			}
		})
	}
}
