package cli

import (
	"context"
	"encoding/hex"
	"flag"
	"io"
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/workflow"
)

const runtimeProfilePreviewAction action = "runtime_profile_preview"

type RuntimeProfilePreviewInput struct {
	Project, Role, Complexity, Runtime, Observations string
	Capabilities                                     []string
}

type RuntimeProfilePreviewService interface {
	RuntimeProfilePreview(context.Context, RuntimeProfilePreviewInput) Result
}

func runtimeRequestFlags(set *flag.FlagSet, role, complexity, capabilities, observations *string) {
	set.StringVar(role, "role", "", "Logical role; `<token>`.")
	set.StringVar(complexity, "complexity", "", "Required complexity; `<token>`.")
	set.StringVar(capabilities, "capabilities", "", "Required proven capabilities; `<comma-list>`.")
	set.StringVar(observations, "observations", "", "Operator-supplied existing observation inventory; `<absolute-path>`.")
}

func runtimePreviewFlags(args []string) (RuntimeProfilePreviewInput, bool) {
	var input RuntimeProfilePreviewInput
	var capabilities string
	set := flag.NewFlagSet(string(runtimeProfilePreviewAction), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(&input.Project, "project", "", "")
	set.StringVar(&input.Runtime, "runtime", "", "")
	runtimeRequestFlags(set, &input.Role, &input.Complexity, &capabilities, &input.Observations)
	if invalidFlagSyntax(set, args, nil) {
		return RuntimeProfilePreviewInput{}, false
	}
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return RuntimeProfilePreviewInput{}, false
	}
	input.Capabilities = strings.Split(capabilities, ",")
	if flagSupplied(args, "--runtime") && !workflow.SupportedRuntime(input.Runtime) || !ValidRuntimePreviewInput(input) {
		return RuntimeProfilePreviewInput{}, false
	}
	return input, true
}

// ValidRuntimePreviewInput bounds input before it reaches the policy source.
func ValidRuntimePreviewInput(input RuntimeProfilePreviewInput) bool {
	if input.Project == "" || len(input.Project) > 256 || !filepath.IsAbs(input.Observations) || len(input.Observations) > 4096 || !previewToken(input.Role) || !previewToken(input.Complexity) || input.Runtime != "" && !workflow.SupportedRuntime(input.Runtime) || len(input.Capabilities) == 0 || len(input.Capabilities) > 32 {
		return false
	}
	seen := map[string]bool{}
	for _, capability := range input.Capabilities {
		if !previewToken(capability) || seen[capability] {
			return false
		}
		seen[capability] = true
	}
	return true
}

func previewToken(value string) bool {
	if len(value) == 0 || len(value) > 256 {
		return false
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
func validPreviewDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32 && strings.ToLower(value) == value
}

func (input RuntimeProfilePreviewInput) Request() runtimeapplication.Request {
	return runtimeapplication.Request{Role: input.Role, Complexity: input.Complexity, Capabilities: append([]string(nil), input.Capabilities...), RuntimeID: input.Runtime}
}
