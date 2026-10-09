package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
)

// Issue #229: the canonical domain skills route an operation to existing Lingo
// commands. Semantic intent resolution is Runtime skill behavior and is not
// re-implemented here; these tests pin the deterministic boundary it hands
// off to: the routing table each canonical skill declares, the inspection
// metadata the binary derives, the parser that alone grants authority, and
// the retired names that must no longer resolve.

var canonicalDomainSkills = []string{"axiom-project", "axiom-work-item"}

// compatibilityOperation maps each operation-specific skill to the canonical
// domain operation whose capabilities remain available after retirement.
var compatibilityOperation = map[string][2]string{
	"axiom-project-configure": {"axiom-project", "configure"},
	"axiom-project-list":      {"axiom-project", "list"},
	"axiom-project-show":      {"axiom-project", "show"},
	"axiom-work-item-create":  {"axiom-work-item", "create"},
	"axiom-work-item-run":     {"axiom-work-item", "run"},
	"axiom-work-item-status":  {"axiom-work-item", "status"},
}

type routingRow struct {
	operation, mode, selector, effect, authority, semantic string
	commands                                               []string
}

func skillSource(t *testing.T, name string) string {
	t.Helper()
	wire, err := os.ReadFile("../codexruntime/skills/" + name + "/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(wire)
}

// routingTable parses the "## Operation routing" Markdown table of a canonical
// skill into the executable command, selector, effect and semantic policy of
// each row.
func routingTable(t *testing.T, name string) []routingRow {
	t.Helper()
	source := skillSource(t, name)
	_, section, ok := strings.Cut(source, "## Operation routing\n")
	if !ok {
		t.Fatalf("%s has no operation routing table", name)
	}
	rows := []routingRow{}
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "## ") {
			break
		}
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) != 6 {
			t.Fatalf("%s routing row has %d cells: %s", name, len(cells), line)
		}
		for index := range cells {
			cells[index] = strings.TrimSpace(cells[index])
		}
		row := routingRow{operation: strings.Trim(cells[0], "`"), mode: cells[1], authority: cells[4]}
		if row.mode == "-" {
			row.mode = "default"
		}
		for _, item := range strings.Split(cells[2], ",") {
			invocation := strings.Trim(strings.TrimSpace(item), "`")
			rest, ok := strings.CutPrefix(invocation, "axiom ")
			if !ok || strings.HasPrefix(rest, "--json") {
				t.Fatalf("%s routes outside human-first axiom: %s", name, invocation)
			}
			command, selector, _ := strings.Cut(rest, " --")
			if selector != "" {
				row.selector = "--" + strings.Fields(selector)[0]
			}
			row.commands = append(row.commands, "axiom "+command)
		}
		row.effect = strings.ReplaceAll(cells[3], " ", "-")
		switch {
		case cells[5] == "allowed":
			row.semantic = "allowed"
		case strings.HasPrefix(cells[5], "only for unambiguous "):
			row.semantic = "unambiguous-only"
		default:
			t.Fatalf("%s unknown semantic policy %q", name, cells[5])
		}
		rows = append(rows, row)
	}
	return rows
}

func modeByName(t *testing.T, skill skillInspection, operation, mode string) skillOperationMode {
	t.Helper()
	for _, item := range skill.Operations {
		if item.Name != operation {
			continue
		}
		for _, candidate := range item.Modes {
			if candidate.Name == mode {
				return candidate
			}
		}
	}
	t.Fatalf("%s has no %s/%s mode", skill.Name, operation, mode)
	return skillOperationMode{}
}

// The canonical skill text and `skill inspect` describe one routing contract:
// same operations, modes, commands, selectors, effects, authority inputs and
// semantic policy.
func TestCanonicalSkillRoutingTableConvergesWithInspection(t *testing.T) {
	for _, name := range canonicalDomainSkills {
		t.Run(name, func(t *testing.T) {
			skill := discoverForTest(t, name)
			rows := routingTable(t, name)
			modes := 0
			operations := []string{}
			for _, operation := range skill.Operations {
				operations = append(operations, "`"+operation.Name+"`")
				modes += len(operation.Modes)
			}
			if len(rows) != modes {
				t.Fatalf("routing rows=%d inspection modes=%d", len(rows), modes)
			}
			declared := "Supported domain operations are " + strings.Join(operations[:len(operations)-1], ", ") + ", and " + operations[len(operations)-1] + "."
			if !strings.Contains(skillSource(t, name), declared) {
				t.Fatalf("skill does not declare exactly %s", declared)
			}
			for _, row := range rows {
				mode := modeByName(t, skill, row.operation, row.mode)
				if !reflect.DeepEqual(row.commands, mode.Commands) || row.selector != mode.Selector || row.effect != mode.Effect || row.semantic != mode.Semantic {
					t.Fatalf("%s/%s skill row %+v != inspection %+v", row.operation, row.mode, row, mode)
				}
				for _, input := range append(append([]string{}, mode.AuthorityInputs...), mode.RejectedInputs...) {
					if !strings.Contains(row.authority, "`"+input+"`") {
						t.Fatalf("%s/%s skill authority %q omits %s", row.operation, row.mode, row.authority, input)
					}
				}
			}
		})
	}
}

// Metadata is derived from, and checked against, the executable parser: every
// selector, authority input, rejected input and example flag is a real flag
// of the mode's command, and every operation's commands are covered by its
// modes exactly once.
func TestSkillOperationMetadataMatchesExecutableParser(t *testing.T) {
	for _, name := range append(append([]string{}, canonicalDomainSkills...), mapKeys(compatibilityOperation)...) {
		for _, spec := range skillOperationSpecs(name) {
			covered := []action{}
			for _, mode := range spec.modes {
				covered = append(covered, mode.actions...)
				flags := map[string]bool{}
				for _, operation := range mode.actions {
					var values requestInput
					skillFlagSet(operation, &values).VisitAll(func(f *flag.Flag) { flags["--"+f.Name] = true })
				}
				inputs := append(append([]string{}, mode.authorityInputs...), mode.rejectedInputs...)
				if mode.selector != "" {
					inputs = append(inputs, mode.selector)
				}
				fields := strings.Fields(mode.example)
				if len(fields) < 3 || fields[0] != "axiom" || !slices.Contains(commandNames(mode.actions), "axiom "+fields[1]+" "+fields[2]) {
					t.Fatalf("%s/%s/%s example %q is not one of its commands", name, spec.name, mode.name, mode.example)
				}
				for _, field := range fields {
					if strings.HasPrefix(field, "--") {
						inputs = append(inputs, field)
					}
				}
				for _, input := range inputs {
					if !flags[input] {
						t.Fatalf("%s/%s/%s names %s, which its commands do not accept", name, spec.name, mode.name, input)
					}
				}
				if (mode.effect == effectReadOnly || mode.effect == effectPreviewOnly) != (len(mode.authorityInputs) == 0) && mode.name != "transition" {
					t.Fatalf("%s/%s/%s effect %s with authority inputs %v", name, spec.name, mode.name, mode.effect, mode.authorityInputs)
				}
			}
			if !sameActions(covered, spec.actions) {
				t.Fatalf("%s/%s modes cover %v, operation lists %v", name, spec.name, covered, spec.actions)
			}
		}
	}
}

func commandNames(actions []action) []string {
	names := []string{}
	for _, operation := range actions {
		names = append(names, skillCommandName(operation))
	}
	return names
}

// sameActions compares action sets; modes of one operation may share a
// command, told apart by an explicit selector.
func sameActions(left, right []action) bool {
	a, b := append([]action{}, left...), append([]action{}, right...)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(slices.Compact(a), slices.Compact(b))
}

func mapKeys(values map[string][2]string) []string {
	keys := []string{}
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// Explicit operation dispatch is a fixed table lookup: inspection is
// deterministic, each operation name is unique, and the skill text forbids
// semantic classification of an explicit operation.
func TestExplicitOperationRoutingIsDeterministic(t *testing.T) {
	explicitRule := map[string]string{
		"axiom-project":   "Do not classify or reinterpret an explicit\noperation semantically.",
		"axiom-work-item": "Do not perform semantic classification for an\nexplicit operation.",
	}
	for _, name := range canonicalDomainSkills {
		first, second := discoverForTest(t, name), discoverForTest(t, name)
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("%s inspection is not deterministic", name)
		}
		seen := map[string]bool{}
		for _, operation := range first.Operations {
			if seen[operation.Name] || len(operation.Modes) == 0 {
				t.Fatalf("%s operation %q is duplicated or unroutable", name, operation.Name)
			}
			seen[operation.Name] = true
			// Several modes of one operation are told apart only by explicit
			// inputs: a distinct command or an explicit selector.
			for i, left := range operation.Modes {
				for _, right := range operation.Modes[i+1:] {
					if reflect.DeepEqual(left.Commands, right.Commands) && left.Selector == right.Selector {
						t.Fatalf("%s/%s modes %s and %s are not separated by explicit input", name, operation.Name, left.Name, right.Name)
					}
				}
			}
		}
		if !strings.Contains(skillSource(t, name), explicitRule[name]) {
			t.Fatalf("%s lacks the explicit-operation rule", name)
		}
	}
}

// Ambiguous intent never resolves to a mutating path, and resolution never
// supplies authority.
func TestSemanticResolutionIsBoundedAndNeverMutatesOnAmbiguity(t *testing.T) {
	rules := map[string][]string{
		"axiom-project": {
			"interpret clear domain intent across all\nsupported operations and modes",
			"Never turn ambiguous intent into a mutating `configure`\noperation, archive, reactivation, detach, disable, enable, or remove.",
			"The user need not name\nan English CLI token or another skill",
			"it never grants\nauthority, supplies `--authorize-local`, invents selectors, or replaces Lingo\nvalidation.",
		},
		"axiom-work-item": {
			"interpret clear domain intent across all\nsupported operations and modes",
			"Never silently resolve ambiguous intent to `create`, `run`, or\nanother mutating path.",
			"need not name an English CLI token or another skill",
			"never grants external/local authority, supplies `--authorize-external` or\n`--authorize-local`, invents selectors, or changes workflow state.",
		},
	}
	for _, name := range canonicalDomainSkills {
		source := skillSource(t, name)
		for _, rule := range rules[name] {
			if !strings.Contains(source, rule) {
				t.Fatalf("%s lacks bounded semantic rule %q", name, rule)
			}
		}
		for _, operation := range discoverForTest(t, name).Operations {
			for _, mode := range operation.Modes {
				if (mode.Effect == effectReadOnly) != (mode.Semantic == "allowed") {
					t.Fatalf("%s/%s/%s effect %s allows semantic %s", name, operation.Name, mode.Name, mode.Effect, mode.Semantic)
				}
			}
		}
	}
}

// authorityProbe records the authority each application request carries.
type authorityProbe struct {
	recordingService
	calls []string
	local []bool
	ext   []bool
}

func (p *authorityProbe) record(call string, local, external bool) Result {
	p.calls, p.local, p.ext = append(p.calls, call), append(p.local, local), append(p.ext, external)
	return Result{Status: Succeeded, Category: "applied"}
}
func (p *authorityProbe) Configure(_ context.Context, input ConfigureInput) Result {
	return p.record("configure", input.AuthorizeLocal, false)
}
func (p *authorityProbe) WorkItemCreate(_ context.Context, input WorkItemInput) Result {
	return p.record("work-item create", input.AuthorizeLocal, input.AuthorizeExternal)
}
func (p *authorityProbe) WorkItemSelect(_ context.Context, input WorkItemInput) Result {
	return p.record("work-item select", input.AuthorizeLocal, input.AuthorizeExternal)
}
func (p *authorityProbe) WorkflowStart(_ context.Context, input WorkflowInput) Result {
	return p.record("workflow start", input.AuthorizeLocal, input.AuthorizeExternal)
}
func (p *authorityProbe) WorkflowAdvance(_ context.Context, input WorkflowInput) Result {
	return p.record("workflow advance", input.AuthorizeLocal, input.AuthorizeExternal)
}
func (p *authorityProbe) WorkflowFact(_ context.Context, input WorkflowInput) Result {
	return p.record("workflow fact", input.AuthorizeLocal, input.AuthorizeExternal)
}
func (p *authorityProbe) WorkflowResume(_ context.Context, input WorkflowInput) Result {
	return p.record("workflow resume", input.AuthorizeLocal, input.AuthorizeExternal)
}
func (p *authorityProbe) WorkflowReconcile(_ context.Context, input WorkflowInput) Result {
	return p.record("workflow reconcile", input.AuthorizeLocal, input.AuthorizeExternal)
}
func (p *authorityProbe) WorkItemComment(_ context.Context, input WorkItemInput) Result {
	return p.record("work-item comment", input.AuthorizeLocal, input.AuthorizeExternal)
}

// Issue #230 lifecycle entrypoints (optional services).
func (p *authorityProbe) ProjectArchive(_ context.Context, input ProjectLifecycleInput) Result {
	return p.record("project archive", input.AuthorizeLocal, false)
}
func (p *authorityProbe) ProjectReactivate(_ context.Context, input ProjectLifecycleInput) Result {
	return p.record("project reactivate", input.AuthorizeLocal, false)
}
func (p *authorityProbe) ProjectListFiltered(context.Context, ProjectListInput) Result {
	return p.record("project list", false, false)
}
func (p *authorityProbe) ProjectValidateInstalled(context.Context, ProjectLifecycleInput) Result {
	return p.record("project validate", false, false)
}
func (p *authorityProbe) IntegrationList(context.Context, IntegrationInput) Result {
	return p.record("integration list", false, false)
}
func (p *authorityProbe) IntegrationShow(context.Context, IntegrationInput) Result {
	return p.record("integration show", false, false)
}
func (p *authorityProbe) IntegrationValidate(context.Context, IntegrationInput) Result {
	return p.record("integration validate", false, false)
}
func (p *authorityProbe) IntegrationDisable(_ context.Context, input IntegrationInput) Result {
	return p.record("integration disable", input.AuthorizeLocal, false)
}
func (p *authorityProbe) IntegrationEnable(_ context.Context, input IntegrationInput) Result {
	return p.record("integration enable", input.AuthorizeLocal, false)
}
func (p *authorityProbe) IntegrationRemove(_ context.Context, input IntegrationInput) Result {
	return p.record("integration remove", input.AuthorizeLocal, false)
}
func (p *authorityProbe) WorkItemList(context.Context, WorkItemInput) Result {
	return p.record("work-item list", false, false)
}
func (p *authorityProbe) WorkItemUpdate(_ context.Context, input WorkItemInput) Result {
	return p.record("work-item update", input.AuthorizeLocal, input.AuthorizeExternal)
}
func (p *authorityProbe) WorkItemClose(_ context.Context, input WorkItemInput) Result {
	return p.record("work-item close", input.AuthorizeLocal, input.AuthorizeExternal)
}
func (p *authorityProbe) WorkItemReopen(_ context.Context, input WorkItemInput) Result {
	return p.record("work-item reopen", input.AuthorizeLocal, input.AuthorizeExternal)
}

// mutatingRequests are the selector-complete requests a Runtime forms for
// each mutating command after operation resolution, before any authority.
var mutatingRequests = map[string][]string{
	"configure/create":    {"project", "configure", "--slug", "alpha", "--name", "Alpha", "--repository", "main=/work/main", "--work-item-provider", "github"},
	"configure/edit":      {"project", "configure", "--project", "alpha", "--name", "Renamed"},
	"work-item create":    {"work-item", "create", "--project", "alpha", "--repository", "main", "--provider-repository", "owner/repo", "--intent", "Fix it"},
	"work-item select":    {"work-item", "select", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7"},
	"workflow start":      {"workflow", "start", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--role", "implementation", "--complexity", "high", "--capabilities", "axiom-skills", "--runtime", "claude"},
	"workflow advance":    {"workflow", "advance", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", "e-1", "--expected-revision", "3", "--gate", "specification", "--outcome", "pass", "--reference", "evidence:spec.md:abc"},
	"workflow resume":     {"workflow", "resume", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", "e-1", "--expected-revision", "3"},
	"workflow fact":       {"workflow", "fact", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", "e-1", "--expected-revision", "3", "--fact", "review_started", "--active", "--reference", "evidence:review.md:abc"},
	"workflow reconcile":  {"workflow", "reconcile", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", "e-1", "--expected-revision", "3"},
	"project archive":     {"project", "archive", "--project", "alpha"},
	"project reactivate":  {"project", "reactivate", "--project", "alpha"},
	"integration disable": {"integration", "disable", "--project", "alpha", "--integration", "work-items"},
	"integration enable":  {"integration", "enable", "--project", "alpha", "--integration", "work-items"},
	"integration remove":  {"integration", "remove", "--project", "alpha", "--integration", "work-items"},
	"work-item update":    {"work-item", "update", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--title", "Renamed"},
	"work-item comment":   {"work-item", "comment", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--message", "Note"},
	"work-item close":     {"work-item", "close", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7"},
	"work-item reopen":    {"work-item", "reopen", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7"},
}

// authorityValues are explicit, well-formed values for authority inputs that
// take a value.
var authorityValues = map[string]string{"--preview-digest": strings.Repeat("a", 64), "--project-id": "123e4567-e89b-42d3-a456-426614174000"}

func routedRequest(t *testing.T, operation string, mode skillOperationMode, command string) []string {
	t.Helper()
	if args, ok := mutatingRequests[operation+"/"+mode.Name]; ok {
		return args
	}
	args, ok := mutatingRequests[strings.TrimPrefix(command, "axiom ")]
	if !ok {
		t.Fatalf("no routed request for %s/%s %s", operation, mode.Name, command)
	}
	return args
}

// A request formed by routing alone reaches the application with no authority;
// only the mode's advertised authority inputs, supplied explicitly, set it.
func TestRoutingAloneNeverGrantsAuthority(t *testing.T) {
	for _, name := range canonicalDomainSkills {
		for _, operation := range discoverForTest(t, name).Operations {
			for _, mode := range operation.Modes {
				if mode.Effect == effectReadOnly {
					continue
				}
				for _, command := range mode.Commands {
					args := routedRequest(t, operation.Name, mode, command)
					probe := &authorityProbe{}
					var output bytes.Buffer
					Run(context.Background(), args, probe, completionProvenance(t), &output)
					if len(probe.calls) != 1 || probe.calls[0] != strings.TrimPrefix(command, "axiom ") && probe.calls[0] != "configure" || probe.local[0] || probe.ext[0] {
						t.Fatalf("%s routed without authority reached %v local=%v external=%v: %s", command, probe.calls, probe.local, probe.ext, output.String())
					}
					wantLocal, wantExternal := slices.Contains(mode.AuthorityInputs, "--authorize-local"), slices.Contains(mode.AuthorityInputs, "--authorize-external")
					if !wantLocal && !wantExternal {
						continue
					}
					authorized := append([]string{}, args...)
					for _, input := range mode.AuthorityInputs {
						authorized = append(authorized, input)
						if value, ok := authorityValues[input]; ok {
							authorized = append(authorized, value)
						}
					}
					probe = &authorityProbe{}
					output.Reset()
					Run(context.Background(), authorized, probe, completionProvenance(t), &output)
					if len(probe.calls) != 1 || probe.local[0] != wantLocal || probe.ext[0] != wantExternal {
						t.Fatalf("%s with %v reached %v local=%v external=%v: %s", command, mode.AuthorityInputs, probe.calls, probe.local, probe.ext, output.String())
					}
				}
			}
		}
	}
}

// CR-002 as amended by #230: Project edit publication requires the complete
// replay tuple. The full tuple reaches the application; each advertised
// replay input supplied alone is a partial tuple and fails before it.
func TestProjectEditPublicationRequiresTheCompleteReplayTuple(t *testing.T) {
	edit := modeByName(t, discoverForTest(t, "axiom-project"), "configure", "edit")
	create := modeByName(t, discoverForTest(t, "axiom-project"), "configure", "create")
	if edit.Effect != effectLocalMutation || !reflect.DeepEqual(edit.AuthorityInputs, []string{"--project-id", "--preview-digest", "--authorize-local"}) || len(edit.RejectedInputs) != 0 {
		t.Fatalf("edit mode=%+v", edit)
	}
	if create.Effect != effectLocalMutation || !reflect.DeepEqual(create.AuthorityInputs, []string{"--preview-digest", "--authorize-local"}) || len(create.RejectedInputs) != 0 {
		t.Fatalf("create mode=%+v", create)
	}
	for _, input := range edit.AuthorityInputs {
		args := []string{"project", "configure", "--project", "sample", "--name", "Renamed", input}
		if value, ok := authorityValues[input]; ok {
			args = append(args, value)
		}
		service := newEditRecordingService(t)
		var output bytes.Buffer
		code := Run(context.Background(), args, service, completionProvenance(t), &output)
		var event completionEvent
		if err := json.Unmarshal(output.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		if code != ExitFailure || len(service.inputs) != 0 || event.Status != completion.ValidationFailure || event.Result != "Project edit authority is incomplete" {
			t.Fatalf("%s: code=%d inputs=%d output=%s", input, code, len(service.inputs), output.String())
		}
	}
	full := []string{"project", "configure", "--project", "sample", "--name", "Renamed", "--project-id", authorityValues["--project-id"], "--preview-digest", authorityValues["--preview-digest"], "--authorize-local"}
	service := newEditRecordingService(t)
	var output bytes.Buffer
	Run(context.Background(), full, service, completionProvenance(t), &output)
	if len(service.inputs) != 1 || !service.inputs[0].AuthorizeLocal || service.inputs[0].ProjectID != authorityValues["--project-id"] {
		t.Fatalf("complete tuple did not reach the application: %s", output.String())
	}
	source := skillSource(t, "axiom-project")
	_, editSection, _ := strings.Cut(source, "### Edit an existing Project")
	editSection, _, _ = strings.Cut(editSection, "\nUse guided")
	for _, required := range []string{"Publish only after the user approves that exact `edit` preview", "`--project-id`, `--preview-digest`,\nand `--authorize-local`", "never\ndeletes the working copy or a remote Repository"} {
		if !strings.Contains(editSection, required) {
			t.Fatalf("axiom-project edit contract missing %q", required)
		}
	}
}

// Operations outside the declared set are not routed: Lingo has no such
// command and fails before any application call.
func TestUnsupportedDomainOperationsFailSafely(t *testing.T) {
	for _, args := range [][]string{
		{"project", "delete", "--selector", "alpha"},
		{"project", "remove", "--selector", "alpha"},
		{"work-item", "delete", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7"},
		{"workflow", "cancel", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", "e-1"},
		{"integration", "revoke", "--project", "alpha", "--integration", "work-items"},
		{"integration", "delete", "--project", "alpha", "--integration", "work-items"},
		{"repository", "detach", "--project", "alpha", "--repository", "main"},
	} {
		probe := &authorityProbe{}
		var output bytes.Buffer
		if code := Run(context.Background(), args, probe, completionProvenance(t), &output); code == ExitSuccess || len(probe.calls) != 0 || probe.call != "" {
			t.Fatalf("%v: code=%d calls=%v output=%s", args, code, probe.calls, output.String())
		}
	}
	for _, name := range canonicalDomainSkills {
		for _, operation := range discoverForTest(t, name).Operations {
			// #230 adds resource-specific operations; generic deletion,
			// cancellation and credential cleanup stay unsupported.
			if slices.Contains([]string{"delete", "cancel", "uninstall", "revoke", "logout"}, operation.Name) {
				t.Fatalf("%s exposes unsupported operation %s", name, operation.Name)
			}
		}
	}
}

// Malformed, duplicate and conflicting selectors stay deterministic Lingo
// validation on the canonical routes; the skills forward them unchanged.
func TestInvalidSelectorsRemainLingoValidation(t *testing.T) {
	for _, args := range [][]string{
		{"project", "show", "--selector", "alpha", "--selector", "beta"},
		{"project", "show", "--selector", "alpha", "--unknown"},
		{"work-item", "select", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo"},
		{"work-item", "select", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--number", "7"},
		{"workflow", "status", "--project", "alpha", "--repository", "main", "--work-item", "not-a-selector", "--execution", "e-1"},
	} {
		probe := &authorityProbe{}
		var output bytes.Buffer
		code := Run(context.Background(), args, probe, completionProvenance(t), &output)
		var event completionEvent
		_ = json.Unmarshal(output.Bytes(), &event)
		if code == ExitSuccess || len(probe.calls) != 0 || event.Status != completion.ValidationFailure {
			t.Fatalf("%v: code=%d calls=%v output=%s", args, code, probe.calls, output.String())
		}
	}
	for _, name := range canonicalDomainSkills {
		source := skillSource(t, name)
		if !strings.Contains(source, "conflicting inputs") || !strings.Contains(source, "Lingo") {
			t.Fatalf("%s does not forward invalid inputs to Lingo validation", name)
		}
	}
}

// Retired entrypoints fail closed, while their original domain operation remains.
func TestRetiredSkillsPreserveCanonicalCapabilities(t *testing.T) {
	for retired, target := range compatibilityOperation {
		if _, ok := inspectSkill(retired); ok {
			t.Fatalf("retired skill %s resolves", retired)
		}
		canonical := discoverForTest(t, target[0])
		if !slices.ContainsFunc(canonical.Operations, func(op skillOperation) bool { return op.Name == target[1] }) {
			t.Fatalf("retired skill %s lost capability", retired)
		}
	}
}

func (p *authorityProbe) List(context.Context) Result {
	return p.record("project list", false, false)
}

func (p *authorityProbe) Show(context.Context, ResolveInput) Result {
	return p.record("project show", false, false)
}
func (p *authorityProbe) WorkItemShow(context.Context, WorkItemInput) Result {
	return p.record("work-item show", false, false)
}
func (p *authorityProbe) WorkflowStatus(context.Context, WorkflowInput) Result {
	return p.record("workflow status", false, false)
}
func (p *authorityProbe) WorkflowEvidence(context.Context, WorkflowInput) Result {
	return p.record("workflow evidence", false, false)
}
func (p *authorityProbe) WorkflowList(context.Context, ExecutionListInput) ExecutionListResponse {
	return ExecutionListResponse{Result: p.record("workflow list", false, false)}
}

// Exercise every advertised read-only path against operation-shaped service
// dispatch, including optional lifecycle interfaces. Together with
// TestRoutingAloneNeverGrantsAuthority this covers every catalog command's
// application handoff, rather than merely asserting command spelling.
func TestCanonicalReadOnlyCatalogDispatch(t *testing.T) {
	requests := map[string][]string{
		"axiom project list":         {"project", "list"},
		"axiom project show":         {"project", "show", "--selector", "alpha"},
		"axiom project validate":     {"project", "validate", "--project", "alpha"},
		"axiom integration list":     {"integration", "list", "--project", "alpha"},
		"axiom integration show":     {"integration", "show", "--project", "alpha", "--integration", "work-items"},
		"axiom integration validate": {"integration", "validate", "--project", "alpha"},
		"axiom work-item list":       {"work-item", "list", "--project", "alpha"},
		"axiom work-item show":       {"work-item", "show", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7"},
		"axiom workflow list":        {"workflow", "list", "--project", "alpha"},
		"axiom workflow status":      {"workflow", "status", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", "e-1"},
		"axiom workflow evidence":    {"workflow", "evidence", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", "e-1"},
	}
	for _, name := range canonicalDomainSkills {
		for _, operation := range discoverForTest(t, name).Operations {
			for _, mode := range operation.Modes {
				if mode.Effect != effectReadOnly {
					continue
				}
				for _, command := range mode.Commands {
					args, ok := requests[command]
					if !ok {
						t.Fatalf("unverified application path: %s", command)
					}
					probe := &authorityProbe{}
					var output bytes.Buffer
					Run(context.Background(), args, probe, completionProvenance(t), &output)
					if !reflect.DeepEqual(probe.calls, []string{strings.TrimPrefix(command, "axiom ")}) || probe.local[0] || probe.ext[0] {
						t.Fatalf("%s dispatched %v: %s", command, probe.calls, &output)
					}
				}
			}
		}
	}
}
