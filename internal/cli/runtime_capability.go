package cli

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"strings"

	"github.com/rgomids/axiom/internal/provenance"
)

type RuntimeCapabilityInput struct {
	RuntimeID, Operation, ProfileID, Variant string
	RepositoryRoot, WorkspaceRoot            string
	PreviewDigest                            string
	Process, AuthorizeLocal                  bool
	ProbeID, ManifestDigest                  string
	AuthorizePrepare, AuthorizeExternal      bool
}

type RuntimeCapabilityResponse struct {
	Result  Result
	Payload any
}

type RuntimeCapabilityService interface {
	RuntimeCapability(context.Context, RuntimeCapabilityInput) RuntimeCapabilityResponse
}

func capabilityCommands(runtime string) commandDefinition {
	return group("capability", "Inspect, probe and invalidate exact-binding capability Evidence",
		leaf("show", "Inspect fresh Evidence; unknown never means proven", action("capability_"+runtime+"_show")),
		leaf("invalidate", "Preview or invalidate Evidence with local authority", action("capability_"+runtime+"_invalidate")),
		leaf("prepare", "Preview or prepare an envelope under separate local authority", action("capability_"+runtime+"_prepare")),
		leaf("probe", "Preview or run a bounded probe with local and external authority", action("capability_"+runtime+"_probe")))
}

func capabilityOperation(op action) bool { return strings.HasPrefix(string(op), "capability_") }

func capabilityFlagSet(op action, in *RuntimeCapabilityInput) *flag.FlagSet {
	set := flag.NewFlagSet(string(op), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	parts := strings.Split(string(op), "_")
	if len(parts) != 3 {
		return set
	}
	in.RuntimeID, in.Operation = parts[1], parts[2]
	set.StringVar(&in.ProfileID, "profile", "", "Exact canonical v3 Profile identity; `<token>`.")
	set.StringVar(&in.Variant, "variant", "", "Exact invocation variant; `<read-only|repository-write>`.")
	set.BoolVar(&in.Process, "process", false, "Inspect the variant with process effects enabled.")
	set.StringVar(&in.RepositoryRoot, "repository-root", "", "Repository for effort-source inspection; `<absolute-path>`.")
	set.StringVar(&in.WorkspaceRoot, "workspace-root", "", "Stage workspace namespace for binding observation; `<absolute-path>`.")
	set.BoolVar(&in.AuthorizeLocal, "authorize-local", false, "Consent to effort-source inspection and the exact previewed local effects; never external authority or envelope preparation.")
	if in.Operation != "show" {
		set.StringVar(&in.PreviewDigest, "preview-digest", "", "Exact operation preview digest; absent means preview only; `<digest>`.")
	}
	if in.Operation == "prepare" || in.Operation == "probe" {
		set.StringVar(&in.ProbeID, "probe-id", "", "Unique Axiom-owned envelope identity; `<token>`.")
	}
	if in.Operation == "prepare" {
		set.BoolVar(&in.AuthorizePrepare, "authorize-prepare", false, "Authorize only the exact local envelope preparation; never a Runtime invocation.")
	}
	if in.Operation == "probe" {
		set.StringVar(&in.ManifestDigest, "manifest-digest", "", "Exact digest from the independently prepared ready envelope; `<digest>`.")
		set.BoolVar(&in.AuthorizeExternal, "authorize-external", false, "Allow the exact previewed subscription-auth preflight and one bounded inference invocation.")
	}
	return set
}

func runRuntimeCapability(ctx context.Context, mode outputMode, args []string, service Service, source provenance.Value, out io.Writer) (bool, int) {
	command, _, rest, ok := resolveCommand(args)
	if !ok || !capabilityOperation(command.operation) {
		return false, 0
	}
	var in RuntimeCapabilityInput
	set := capabilityFlagSet(command.operation, &in)
	if invalidFlagSyntax(set, rest, nil) || set.Parse(rest) != nil || set.NArg() != 0 || duplicateFlag(rest) || !previewToken(in.ProfileID) || (in.Variant != "read-only" && in.Variant != "repository-write") {
		return true, emitParserFailure(out, mode, command.operation, "invalid_input", source)
	}
	if (in.Operation == "probe" || in.Operation == "prepare") && !previewToken(in.ProbeID) || in.Operation == "probe" && !validPreviewDigest(in.ManifestDigest) {
		return true, emitParserFailure(out, mode, command.operation, "invalid_input", source)
	}
	app, ok := service.(RuntimeCapabilityService)
	if !ok {
		return true, emit(out, mode, event{Operation: command.operation, Status: Failed, Category: "application_unavailable"})
	}
	r := app.RuntimeCapability(ctx, in)
	if r.Result.Completion == nil {
		return true, emitResponse(out, mode, command.operation, r.Result)
	}
	c := *r.Result.Completion
	v := struct {
		completionEvent
		Category   string `json:"category"`
		Capability any    `json:"capability,omitempty"`
	}{completionEvent{Status: c.Status(), Result: c.Result().String(), References: c.References(), Next: c.Next().String(), Details: c.Details(), Provenance: provenanceEvent{Product: c.Provenance().Product(), Version: c.Provenance().Version(), Revision: c.Provenance().Revision(), SourceState: c.Provenance().SourceState()}}, r.Result.Category, r.Payload}
	wire, err := json.Marshal(v)
	if err != nil {
		return true, ExitFailure
	}
	return true, presentEvent(out, mode, c.Status(), append(wire, '\n'), 256<<10)
}
