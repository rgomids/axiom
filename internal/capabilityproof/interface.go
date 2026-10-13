package capabilityproof

import "strings"

// InterfaceObservation is refutation only. Vendor prose is never proof.
type InterfaceObservation struct {
	ExecutableDigest string
	Version          string
	Capabilities     map[string]Result
}

// ParseInterface accepts only the pinned, recognizable option listing. Unknown
// versions or malformed output fail to unknown rather than inventing absence.
func ParseInterface(runtimeID, variant, version, help string) InterfaceObservation {
	o := InterfaceObservation{Capabilities: map[string]Result{}}
	for _, c := range []string{Read, RepositoryWrite, Process, Noninteractive, AxiomCoordination} {
		o.Capabilities[c] = Result{State: Unknown}
	}
	expected, prefix := "", ""
	switch runtimeID {
	case "codex":
		expected, prefix = "0.159.1", "codex "
	case "claude":
		expected, prefix = "2.1.296", ""
	default:
		return o
	}
	v := strings.TrimSpace(version)
	if runtimeID == "claude" {
		v = strings.TrimSuffix(v, " (Claude Code)")
	} else {
		v = strings.TrimPrefix(v, prefix)
	}
	if v != expected {
		return o
	}
	o.Version = v
	if variant != ReadOnlyVariant && variant != RepositoryWriteVariant || len(help) > 65536 || !strings.Contains(help, "Usage:") || !strings.Contains(help, "Options:") || strings.ContainsRune(help, '\x00') {
		return o
	}
	options := map[string]bool{}
	optionLines := map[string]string{}
	inOptions := false
	for _, line := range strings.Split(help, "\n") {
		if strings.TrimSpace(line) == "Options:" {
			inOptions = true
			continue
		}
		if !inOptions || !strings.HasPrefix(line, " ") {
			continue
		}
		fields := strings.Fields(line)
		for _, f := range fields {
			if strings.HasPrefix(f, "--") {
				option := strings.TrimRight(f, ",")
				options[option] = true
				optionLines[option] = line
			} else if !strings.HasPrefix(f, "-") {
				break
			}
		}
	}
	if len(options) == 0 {
		return o
	}
	required := []string{"--print", "--model", "--input-format", "--permission-prompts", "--permission-mode", "--allowedTools", "--add-dir"}
	if runtimeID == "codex" {
		required = []string{"--model", "--ignore-user-config", "--ephemeral", "--sandbox"}
		if variant == RepositoryWriteVariant {
			required = append(required, "--add-dir")
		}
	}
	for _, option := range required {
		if !options[option] {
			for c := range o.Capabilities {
				o.Capabilities[c] = Result{State: Unsupported, Facts: ProbeFacts{InterfaceAbsent: true}}
			}
			return o
		}
	}
	modes := map[string][]string{"--permission-mode": {"dontAsk"}, "--permission-prompts": {"none"}, "--input-format": {"text"}}
	if runtimeID == "codex" {
		modes = map[string][]string{"--sandbox": {"workspace-write"}}
	}
	for option, values := range modes {
		for _, value := range values {
			if !strings.Contains(optionLines[option], value) {
				for c := range o.Capabilities {
					o.Capabilities[c] = Result{State: Unknown}
				}
				return o
			}
		}
	}
	return o
}
