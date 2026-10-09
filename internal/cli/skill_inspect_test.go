package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/codexruntime"
	"github.com/rgomids/axiom/internal/completion"
)

type noInspectionInput struct{}

func (noInspectionInput) Read([]byte) (int, error) { panic("discovery read stdin") }

type noInspectionService struct{ Service }

func discoverForTest(t *testing.T, name string) skillInspection {
	t.Helper()
	var output, prompts bytes.Buffer
	code := RunInteractive(context.Background(), []string{"--json", "skill", "inspect", name}, noInspectionService{}, completionProvenance(t), noInspectionInput{}, &output, &prompts)
	var got struct {
		completionEvent
		Skill skillInspection `json:"skill"`
	}
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if code != ExitSuccess || got.Status != completion.Success || got.Skill.Name != name || prompts.Len() != 0 {
		t.Fatalf("code=%d output=%s prompts=%s", code, output.String(), prompts.String())
	}
	return got.Skill
}

func TestSkillDiscoveryInventoryAndNoExecution(t *testing.T) {
	manifest, err := codexruntime.CurrentManifest()
	if err != nil {
		t.Fatal(err)
	}
	for _, skill := range manifest.Skills {
		t.Run(skill.Name, func(t *testing.T) {
			source, err := os.ReadFile("../codexruntime/skills/" + skill.Name + "/SKILL.md")
			if err != nil || !strings.Contains(string(source), "`axiom skill inspect "+skill.Name+"`") {
				t.Fatalf("missing thin inspection pointer: %s: %v", skill.Name, err)
			}
			got := discoverForTest(t, skill.Name)
			if len(got.Commands) == 0 {
				t.Fatal("missing command contract")
			}
			var output bytes.Buffer
			if code := Run(context.Background(), []string{"skill", "inspect", skill.Name}, nil, completionProvenance(t), &output); code != ExitSuccess {
				t.Fatalf("inspection needs application service: %s", output.String())
			}
		})
	}
}

func findArgument(t *testing.T, cmd skillCommand, name string) skillArgument {
	t.Helper()
	for _, arg := range cmd.Arguments {
		if arg.Name == name {
			return arg
		}
	}
	t.Fatalf("%s missing %s", cmd.Command, name)
	return skillArgument{}
}

// Pin the public command inventory to the workflows exposed by the thin skills.
// Do not derive expectations from skillOperations: an extra mapping is a regression.
func TestSkillDiscoveryExactCommands(t *testing.T) {
	expected := map[string][]string{
		"axiom-project":   {"axiom project configure", "axiom project list", "axiom project show", "axiom project validate", "axiom project archive", "axiom project reactivate", "axiom integration list", "axiom integration show", "axiom integration validate", "axiom integration disable", "axiom integration enable", "axiom integration remove"},
		"axiom-work-item": {"axiom work-item create", "axiom work-item select", "axiom workflow start", "axiom workflow advance", "axiom workflow fact", "axiom workflow resume", "axiom workflow reconcile", "axiom workflow status", "axiom workflow evidence", "axiom workflow list", "axiom work-item list", "axiom work-item show", "axiom work-item update", "axiom work-item comment", "axiom work-item close", "axiom work-item reopen"},
	}
	manifest, err := codexruntime.CurrentManifest()
	if err != nil {
		t.Fatal(err)
	}
	if len(expected) != len(manifest.Skills) {
		t.Fatal("skill inventory changed; reconcile exact command expectations")
	}
	for _, skill := range manifest.Skills {
		t.Run(skill.Name, func(t *testing.T) {
			want, exists := expected[skill.Name]
			if !exists {
				t.Fatalf("missing command expectation for %s", skill.Name)
			}
			got := discoverForTest(t, skill.Name)
			commands := make([]string, 0, len(got.Commands))
			for _, command := range got.Commands {
				commands = append(commands, command.Command)
			}
			if !reflect.DeepEqual(commands, want) {
				t.Fatalf("commands=%v, want=%v", commands, want)
			}
		})
	}
}

func TestSkillDiscoveryRequiredOptionalDescriptionForms(t *testing.T) {
	show := discoveryCommand(t, "axiom-project", "axiom project show")
	selector := findArgument(t, show, "--selector")
	if selector.Required || selector.RequiredWhen != "no effective Project context is available" || !strings.Contains(selector.Description, "Project identity") || !reflect.DeepEqual(selector.AcceptedForms, []string{"--selector <uuid-or-slug>", "--selector=<uuid-or-slug>"}) {
		t.Fatalf("selector=%+v", selector)
	}
	if arg := findArgument(t, show, "--slug"); arg.Required || arg.RequiredWhen != "" {
		t.Fatalf("optional=%+v", arg)
	}
	configure := discoveryCommand(t, "axiom-project", "axiom project configure")
	repository := findArgument(t, configure, "--repository")
	if repository.Required || repository.RequiredWhen != "creating (no --project)" || !repository.Repeatable {
		t.Fatalf("conditional=%+v", repository)
	}
	boolean := findArgument(t, configure, "--authorize-local")
	if !reflect.DeepEqual(boolean.AcceptedForms, []string{"--authorize-local", "--authorize-local=<boolean>"}) {
		t.Fatalf("boolean=%+v", boolean)
	}
	start := discoveryCommand(t, "axiom-work-item", "axiom workflow start")
	for _, name := range []string{"role", "complexity", "capabilities"} {
		arg := findArgument(t, start, "--"+name)
		if !arg.Required || arg.RequiredWhen != "" {
			t.Fatalf("policy requirement=%+v", arg)
		}
	}
	reviewed := findArgument(t, start, "--runtime-preview")
	if reviewed.Required || reviewed.RequiredWhen != "creating Execution after review" {
		t.Fatalf("reviewed preview requirement=%+v", reviewed)
	}
	list := discoveryCommand(t, "axiom-project", "axiom project list")
	if archived := findArgument(t, list, "--include-archived"); len(list.Arguments) != 1 || archived.Required || archived.RequiredWhen != "" {
		t.Fatalf("list inputs=%+v", list)
	}
}

func parseSkillFlags(operation action, args []string) (requestInput, bool) {
	// The #230 lifecycle commands have their own parsers; argument metadata
	// must be accepted by exactly those parsers.
	switch {
	case operation == projectArchiveAction || operation == projectReactivateAction:
		_, issue := projectLifecycleFlags(operation, args)
		return requestInput{}, issue != "invalid_input"
	case operation == listAction || operation == validateAction:
		var slug string
		var lifecycle ProjectLifecycleInput
		var list ProjectListInput
		set := projectListFlagSet(&list)
		if operation == validateAction {
			set = projectValidateFlagSet(&slug, &lifecycle)
		}
		return requestInput{}, !invalidFlagSyntax(set, args, nil) && set.Parse(args) == nil && set.NArg() == 0
	case operation == workflowListAction:
		_, ok := executionListFlags(withSelectors(args, "--project", "alpha"))
		return requestInput{}, ok
	case integrationOperation(operation):
		args = withSelectors(args, "--project", "alpha")
		if operation != integrationListAction && operation != integrationValidateAction {
			args = withSelectors(args, "--integration", "work-items")
		}
		// The parser checks the digest shape; the advertised form uses a
		// placeholder value.
		for index := range args {
			if args[index] == "example" && index > 0 && args[index-1] == "--preview-digest" {
				args[index] = strings.Repeat("a", 64)
			}
			if args[index] == "--preview-digest=example" {
				args[index] = "--preview-digest=" + strings.Repeat("a", 64)
			}
		}
		_, ok := integrationFlags(operation, args)
		return requestInput{}, ok
	}
	if knownWorkItem(operation) {
		return workItemFlags(operation, args)
	}
	if knownWorkflow(operation) {
		return workflowFlags(operation, args)
	}
	return flags(operation, args)
}

func TestSkillDiscoveryMatchesParserAndAcceptedForms(t *testing.T) {
	manifest, err := codexruntime.CurrentManifest()
	if err != nil {
		t.Fatal(err)
	}
	for _, skill := range manifest.Skills {
		got := discoverForTest(t, skill.Name)
		for i, operation := range skillOperations(skill.Name) {
			command := got.Commands[i]
			var values requestInput
			set := skillFlagSet(operation, &values)
			count := 0
			set.VisitAll(func(*flag.Flag) { count++ })
			if len(command.Arguments) != count {
				t.Fatalf("%s inventory mismatch", command.Command)
			}
			for _, rule := range skillRequirements(operation, values) {
				if set.Lookup(rule.name) == nil {
					t.Fatalf("orphan requirement %s/%s", operation, rule.name)
				}
			}
			set.VisitAll(func(f *flag.Flag) {
				t.Run(string(operation)+"/"+f.Name, func(t *testing.T) {
					arg := findArgument(t, command, "--"+f.Name)
					if arg.Description == "" || len(arg.AcceptedForms) != 2 {
						t.Fatalf("incomplete metadata: %+v", arg)
					}
					value := "example"
					var typed any
					if getter, ok := f.Value.(flag.Getter); ok {
						typed = getter.Get()
					}
					switch typed.(type) {
					case bool:
						value = "true"
					case int, uint64:
						value = "7"
					}
					if f.Name == "runtime-preview" {
						value = strings.Repeat("a", 64)
					}
					if f.Name == "runtime" {
						value = "claude"
					}
					if f.Name == "elaborated-section" {
						value = "problem=Example"
					}
					if f.Name == "work-item" {
						value = "github:owner/repo#7"
					}
					forms := [][]string{{arg.Name, value}, {arg.Name + "=" + value}}
					if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
						forms[0] = []string{arg.Name}
					}
					for _, args := range forms {
						if _, ok := parseSkillFlags(operation, args); !ok {
							t.Fatalf("advertised form rejected: %v", args)
						}
					}
					second := forms[0]
					if f.Name == "elaborated-section" {
						second = []string{arg.Name, "scope=Example"}
					}
					_, duplicates := parseSkillFlags(operation, append(append([]string{}, forms[0]...), second...))
					if duplicates != arg.Repeatable {
						t.Fatalf("repeatability drift: %+v accepted=%v", arg, duplicates)
					}
				})
			})
			if _, ok := parseSkillFlags(operation, []string{"--unknown", "example"}); ok {
				t.Fatal("unknown flag accepted")
			}
		}
	}
}

func TestSkillDiscoveryRequirementsMatchRequests(t *testing.T) {
	cases := []struct {
		skill string
		index int
		args  []string
	}{
		{"axiom-project", 2, []string{"--selector", "alpha"}},
		{"axiom-project", 0, []string{"--slug", "alpha", "--name", "Alpha", "--repository", "main=/tmp/alpha"}},
		{"axiom-work-item", 0, []string{"--project", "alpha", "--repository", "main", "--provider-repository", "owner/repo"}},
		{"axiom-work-item", 2, []string{"--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--role", "implementation", "--complexity", "high", "--capabilities", "code"}},
		{"axiom-work-item", 3, []string{"--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", "execution", "--expected-revision", "1", "--gate", "review", "--outcome", "passed"}},
		{"axiom-work-item", 1, []string{"--project", "alpha", "--repository", "main", "--number", "7", "--provider-repository", "owner/repo"}},
		{"axiom-work-item", 4, []string{"--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", "execution", "--expected-revision", "1", "--fact", "review_started", "--reference", "docs/review.md"}},
		{"axiom-work-item", 5, []string{"--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", "execution", "--expected-revision", "1"}},
		{"axiom-work-item", 7, []string{"--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", "execution"}},
	}
	for _, tc := range cases {
		cmd := discoverForTest(t, tc.skill).Commands[tc.index]
		prefix := strings.Fields(strings.TrimPrefix(cmd.Command, "axiom "))
		if _, _, issue := request(append(append([]string{}, prefix...), tc.args...), &recordingService{}); issue != nil {
			t.Fatalf("baseline %s: %s", cmd.Command, *issue)
		}
		for i := 0; i < len(tc.args); i += 2 {
			arg := findArgument(t, cmd, tc.args[i])
			if !arg.Required && arg.RequiredWhen == "" {
				continue
			}
			without := append(append([]string{}, tc.args[:i]...), tc.args[i+2:]...)
			_, _, issue := request(append(append([]string{}, prefix...), without...), &recordingService{})
			if issue == nil || *issue != "missing_required_input" {
				t.Fatalf("%s removing required %s: %v", cmd.Command, arg.Name, issue)
			}
		}
	}
	// Edit has no CREATE requirements, and legacy numeric workflow calls remain valid.
	for _, args := range [][]string{{"project", "configure", "--project", "alpha"}, {"workflow", "status", "--project", "alpha", "--repository", "main", "--number", "7"}} {
		if _, _, issue := request(args, &recordingService{}); issue != nil {
			t.Fatalf("optional inputs became required: %v: %s", args, *issue)
		}
	}
}

func TestSkillDiscoveryRejectsInvalidRequestsAndWorkflowConflicts(t *testing.T) {
	for _, args := range [][]string{{"skill"}, {"skill", "inspect"}, {"skill", "inspect", "unknown"}, {"skill", "inspect", "../axiom-project-show"}, {"skill", "inspect", "axiom-project-show", "--selector", "alpha"}, {"skill", "inspect", "axiom-project-show", "--unknown"}, {"skill", "run", "axiom-project-show"}} {
		var output bytes.Buffer
		code := Run(context.Background(), args, nil, completionProvenance(t), &output)
		var event completionEvent
		if err := json.Unmarshal(output.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		if code != ExitFailure || event.Status != completion.ValidationFailure || event.Result != "Skill inspection input is invalid" {
			t.Fatalf("%v: code=%d output=%s", args, code, output.String())
		}
	}
	discoverForTest(t, "axiom-work-item")
	for _, args := range [][]string{
		{"workflow", "start", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--number", "7"},
		{"work-item", "select", "--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7", "--provider-repository", "owner/repo"},
		{"project", "configure", "--project", "alpha", "--work-item-provider", "github", "--remove-work-item-provider"},
		{"project", "show", "--unknown", "example"},
	} {
		assertStrictParserFailure(t, args)
	}
}

func TestSkillDiscoveryPreservesFullyGuidedInvocation(t *testing.T) {
	discoverForTest(t, "axiom-project")
	service := &recordingService{}
	var output, prompts bytes.Buffer
	input := strings.NewReader("alpha\nAlpha\nmain=/tmp/alpha\n\nnone\n")
	code := RunInteractive(context.Background(), []string{"--json", "project", "configure"}, service, completionProvenance(t), input, &output, &prompts)
	if code != ExitSuccess || service.call == "" || !strings.Contains(prompts.String(), "Project slug") || !strings.Contains(prompts.String(), "Work Item provider") {
		t.Fatalf("guided regression: %d %s %s call=%s", code, output.String(), prompts.String(), service.call)
	}
}

func TestSkillDiscoveryHumanAndWriterFailure(t *testing.T) {
	var output bytes.Buffer
	handled, code := InspectSkill([]string{"skill", "inspect", "axiom-project"}, completionProvenance(t), &output)
	if !handled || code != ExitSuccess || !strings.Contains(output.String(), "#### skill") || !strings.Contains(output.String(), "`--selector`") || !strings.Contains(output.String(), "`--slug`") {
		t.Fatalf("human output=%s", output.String())
	}
	for _, writer := range []io.Writer{nil, shortInspectionWriter{}} {
		_, code := InspectSkill([]string{"skill", "inspect", "axiom-project"}, completionProvenance(t), writer)
		if code != ExitFailure {
			t.Fatal("output failure reported success")
		}
	}
}

type shortInspectionWriter struct{}

func (shortInspectionWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

// withSelectors adds a required selector only when the form under test does
// not already supply it, so duplicate detection stays observable.
func withSelectors(args []string, name, value string) []string {
	if flagSupplied(args, name) {
		return append([]string{}, args...)
	}
	return append([]string{name, value}, args...)
}

func discoveryCommand(t *testing.T, skill, command string) skillCommand {
	t.Helper()
	for _, candidate := range discoverForTest(t, skill).Commands {
		if candidate.Command == command {
			return candidate
		}
	}
	t.Fatalf("missing command %s", command)
	return skillCommand{}
}

func TestRetiredSkillsFailClosed(t *testing.T) {
	for _, name := range []string{"axiom-project-configure", "axiom-project-list", "axiom-project-show", "axiom-work-item-create", "axiom-work-item-run", "axiom-work-item-status"} {
		var output bytes.Buffer
		_, code := InspectSkill([]string{"--json", "skill", "inspect", name}, completionProvenance(t), &output)
		if code != ExitFailure {
			t.Fatalf("retired skill accepted: %s", name)
		}
	}
}
