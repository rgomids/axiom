package cli

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"strings"

	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type RuntimeAuthoringService interface {
	RuntimeAuthoring(context.Context, runtimeapplication.AuthoringRequest) Result
}

func runtimeAuthoringFlagSet(op action, r *runtimeapplication.AuthoringRequest) *flag.FlagSet {
	set := flag.NewFlagSet(string(op), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	pieces := strings.Split(strings.TrimPrefix(string(op), "runtime_authoring_"), "_")
	if len(pieces) != 2 {
		return set
	}
	r.Kind, r.Operation = pieces[0], pieces[1]
	set.StringVar(&r.RuntimeID, "runtime", "", "Exact supported Runtime; `<codex|claude>`.")
	set.StringVar(&r.ProfileID, "profile", "", "Operator-defined Profile identity; `<token>`.")
	switch r.Operation {
	case "create", "edit", "validate", "validate-candidate":
		set.StringVar(&r.Model, "model", "", "Explicit model intent; `<model>`.")
		set.StringVar(&r.Effort.Mode, "effort-mode", "", "Operator choice; `<explicit|runtime-default>`.")
		set.StringVar(&r.Effort.Value, "effort", "", "Exact adapter token or explicit `default`.")
		set.Func("capabilities", "Declared capabilities; `<comma-list>`.", func(v string) error { r.Capabilities = strings.Split(v, ","); return nil })
		set.Func("complexities", "Independent complexity constraints; `<comma-list>`.", func(v string) error { r.Complexities = strings.Split(v, ","); return nil })
		set.Func("enabled", "Runtime eligibility; `<true|false>`.", func(v string) error {
			if v != "true" && v != "false" {
				return flag.ErrHelp
			}
			b := v == "true"
			r.Enabled = &b
			return nil
		})
	case "remove":
		set.BoolVar(&r.Cascade, "cascade", false, "Include reviewed compound Profile/Runtime removal.")
	case "migrate":
		r.MigrationEfforts = map[string]runtimeprofile.Effort{}
		set.Func("profile-effort", "Explicit mapping; repeat `<profile>:<mode>:<value>`.", func(v string) error {
			p := strings.Split(v, ":")
			if len(p) != 3 {
				return flag.ErrHelp
			}
			if _, ok := r.MigrationEfforts[p[0]]; ok {
				return flag.ErrHelp
			}
			r.MigrationEfforts[p[0]] = runtimeprofile.Effort{Mode: p[1], Value: p[2]}
			return nil
		})
		set.BoolVar(&r.ClearLegacyReferences, "clear-legacy-references", false, "Include explicit removal of unresolved legacy references.")
	}
	// Specification 007 BI-3.1: the only v3 authoring flags, on binding
	// migrate (explicit v2→v3) and binding edit (declaration authoring).
	if r.Kind == "binding" && r.Operation == "migrate" {
		set.IntVar(&r.TargetFormat, "target-format", 0, "Explicit local format transition; only `<3>`.")
	}
	if r.Kind == "binding" && (r.Operation == "migrate" || r.Operation == "edit") {
		set.StringVar(&r.EffortDeclaration, "effort-declaration", "", "One EffortDeclaration for --runtime; `<json-file>`.")
	}
	if r.Operation != "list" && r.Operation != "show" && r.Operation != "validate" && r.Operation != "validate-candidate" {
		set.StringVar(&r.ExpectedRevision, "expected-revision", "", "Exact opaque snapshot token from preview; `<digest>`.")
		set.StringVar(&r.PreviewDigest, "preview-digest", "", "Exact mutation preview digest; `<digest>`.")
		set.BoolVar(&r.AuthorizeLocal, "authorize-local", false, "Apply only reviewed machine-local effects.")
	}
	return set
}
func runtimeAuthoringCommands(kind string) []commandDefinition {
	out := []commandDefinition{}
	for _, op := range []string{"list", "show", "create", "edit", "enable", "disable", "remove", "migrate", "validate-candidate", "recover"} {
		out = append(out, leaf(op, "Machine-local "+kind+" "+op, action("runtime_authoring_"+kind+"_"+op)))
	}
	return out
}
func runRuntimeAuthoring(ctx context.Context, mode outputMode, args []string, service Service, source provenance.Value, out io.Writer) (bool, int) {
	command, _, rest, ok := resolveCommand(args)
	if !ok || !strings.HasPrefix(string(command.operation), "runtime_authoring_") {
		return false, 0
	}
	req := runtimeapplication.AuthoringRequest{}
	set := runtimeAuthoringFlagSet(command.operation, &req)
	// Repeated profile-effort mappings are supported; every other flag is unique.
	filtered := []string{}
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--profile-effort" {
			i++
			continue
		}
		if strings.HasPrefix(rest[i], "--profile-effort=") {
			continue
		}
		filtered = append(filtered, rest[i])
	}
	if invalidFlagSyntax(set, rest, map[string]bool{"profile-effort": true}) || set.Parse(rest) != nil || set.NArg() != 0 || duplicateFlag(filtered) {
		return true, emitParserFailure(out, mode, command.operation, "invalid_input", source)
	}
	adapter, ok := service.(RuntimeAuthoringService)
	if !ok {
		return true, emit(out, mode, event{Operation: command.operation, Status: Failed, Category: "application_unavailable"})
	}
	response := adapter.RuntimeAuthoring(ctx, req)
	if response.Completion == nil {
		return true, emitResponse(out, mode, command.operation, response)
	}
	c := *response.Completion
	value := struct {
		completionEvent
		Category  string                              `json:"category"`
		Authoring *runtimeapplication.AuthoringReport `json:"runtimeAuthoring,omitempty"`
	}{completionEvent{Status: c.Status(), Result: c.Result().String(), References: c.References(), Next: c.Next().String(), Details: c.Details(), Provenance: provenanceEvent{Product: c.Provenance().Product(), Version: c.Provenance().Version(), Revision: c.Provenance().Revision(), SourceState: c.Provenance().SourceState()}}, response.Category, response.RuntimeAuthoring}
	wire, err := json.Marshal(value)
	if err != nil {
		return true, ExitFailure
	}
	return true, presentEvent(out, mode, c.Status(), append(wire, '\n'), 256<<10)
}
func runtimeAuthoringSkillSpecs() []skillOperationSpec {
	out := []skillOperationSpec{}
	for _, kind := range []string{"binding", "profile"} {
		for _, op := range []string{"list", "show", "create", "edit", "enable", "disable", "remove", "migrate", "validate-candidate", "recover"} {
			act := action("runtime_authoring_" + kind + "_" + op)
			effect, authority := effectReadOnly, "none"
			inputs := []string{}
			if op != "list" && op != "show" && op != "validate-candidate" {
				effect = effectLocalMutation
				authority = "preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`)"
				inputs = []string{"--expected-revision", "--preview-digest", "--authorize-local"}
			}
			out = append(out, skillOperationSpec{name: kind + "." + op, actions: []action{act}, modes: []skillModeSpec{{name: "default", effect: effect, authority: authority, actions: []action{act}, authorityInputs: inputs, rejectedInputs: []string{}, example: "axiom runtime " + kind + " " + op}}})
		}
	}
	return out
}
