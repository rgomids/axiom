package runtimeadapter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// AdapterVersion identifies adapter-controlled invocation/confinement semantics,
// independently of vendor version, executable, Profile, model and effort. Bump
// the affected adapter's version when those semantics materially change.
func AdapterVersion(runtimeID string) (string, error) {
	switch runtimeID {
	case "codex":
		return "codex-adapter-v1", nil
	case "claude":
		return "claude-adapter-v1", nil
	default:
		return "", ErrInvalidAdapterConfiguration
	}
}

// modeIdentityV1 is a closed projection of BuildStageInvocation, not a process
// argv/environment hash. JSON field order and option order are part of v1.
type modeIdentityV1 struct {
	Encoding    string   `json:"encoding"`
	Runtime     string   `json:"runtime"`
	Adapter     string   `json:"adapter"`
	Variant     string   `json:"variant"`
	WorkingRoot string   `json:"workingRoot"`
	Environment string   `json:"environment"`
	Options     []string `json:"options"`
}

// ModeDigest is shared identity computation for probe preparation, Evidence
// binding creation and dispatch revalidation. It neither observes nor proves
// capabilities. Callers must supply the current effective request, never derive
// it from stored Evidence. Model/effort and binary identities have separate
// Binding/ModelBinding fields. Only the closed #276 stage mode is supported;
// historical configured invocations are deliberately outside this contract.
func ModeDigest(request StageInvocationRequest) (string, error) {
	wire, err := encodeModeIdentity(request)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:]), nil
}

func encodeModeIdentity(request StageInvocationRequest) ([]byte, error) {
	// Reuse the adapter's version gate, injection rejection and path separation.
	// This builder performs no process, filesystem, authentication or probe work.
	inv, err := BuildStageInvocation(request)
	if err != nil {
		return nil, err
	}
	adapter, err := AdapterVersion(inv.RuntimeID)
	if err != nil {
		return nil, err
	}
	identity := modeIdentityV1{
		Encoding: "axiom-runtime-mode-v1", Runtime: inv.RuntimeID, Adapter: adapter,
		Variant: "read-only", WorkingRoot: "inbox", Environment: "stage-allowlist-v1-exact",
		Options: []string{},
	}
	if request.RepositoryWrite {
		identity.Variant, identity.WorkingRoot = "repository-write", "workspace"
	}
	// Select only known mode-defining options from adapter-owned output. Path
	// roles preserve the distinction between the writable inbox and workspace;
	// the pinned Axiom executable's bytes are bound separately by AxiomDigest.
	for i := 1; i < len(inv.Argv); i++ {
		option := inv.Argv[i]
		switch option {
		case "--model":
			i++ // Profile identity is separate; model is never serialized here.
		case "exec", "--print", "--ignore-user-config", "--ephemeral", "-":
			identity.Options = append(identity.Options, option)
		case "--sandbox", "--input-format", "--permission-prompts", "--permission-mode":
			i++
			identity.Options = append(identity.Options, option, inv.Argv[i])
		case "-C":
			i++
			identity.Options = append(identity.Options, option, identity.WorkingRoot)
		case "--add-dir":
			i++
			identity.Options = append(identity.Options, option, "inbox")
		case "--allowedTools":
			i++
			tools := strings.ReplaceAll(inv.Argv[i], "Bash("+request.AxiomPath+":*)", "Bash(pinned-axiom:*)")
			identity.Options = append(identity.Options, option, tools)
		default:
			// A newly emitted option needs an explicit identity contract.
			return nil, ErrInvalidAdapterConfiguration
		}
	}
	return json.Marshal(identity)
}
