package runtimeprofile

import "strings"

// Violation names one declaration field and the rule it breaks. It explains a
// refusal; Validate remains the only acceptance authority.
type Violation struct {
	Field    string // id, runtimeId, model, effort.mode, effort.value, capabilities, complexities
	Required bool   // true when the field is absent rather than malformed
	Rule     string
}

// ProfileViolations reports why a current-format Model Profile declaration is
// refused by Validate, field by field. Values are never echoed.
func ProfileViolations(p ModelProfile) []Violation {
	out := []Violation{}
	add := func(field string, required bool, rule string) {
		out = append(out, Violation{Field: field, Required: required, Rule: rule})
	}
	switch {
	case p.ID == "":
		add("id", true, "a Profile identity is required")
	case !validToken(p.ID) || !SafeIdentifier(p.ID):
		add("id", false, "Profile identity must be a lowercase token of a-z, 0-9, '-', '_' or '.'")
	}
	if p.RuntimeID != "" && !SafeIdentifier(p.RuntimeID) {
		add("runtimeId", false, "Runtime identity must be a supported token")
	}
	switch {
	case p.Model == "":
		add("model", true, "an exact model intent is required")
	case !validText(p.Model) || !SafeIdentifier(p.Model):
		add("model", false, "model intent must be printable text without whitespace edges, shell metacharacters or secret-bearing forms")
	}
	switch {
	case p.Effort.Mode == "":
		add("effort.mode", true, "effort mode is required: explicit or runtime-default")
	case p.Effort.Mode != EffortExplicit && p.Effort.Mode != EffortRuntimeDefault:
		add("effort.mode", false, "effort mode must be explicit or runtime-default")
	case p.Effort.Value == "":
		add("effort.value", true, "effort value is required: an adapter token for explicit, or default for runtime-default")
	case p.Effort.Mode == EffortRuntimeDefault && p.Effort.Value != "default":
		add("effort.value", false, "runtime-default requires the value default")
	case p.Effort.Mode == EffortExplicit && !SafeIdentifier(p.Effort.Value):
		add("effort.value", false, "explicit effort must be a safe adapter token")
	}
	if len(p.Capabilities) == 0 {
		add("capabilities", true, "at least one declared capability is required")
	} else if rule := tokenListRule(p.Capabilities, true); rule != "" {
		add("capabilities", false, rule)
	}
	if rule := tokenListRule(p.Complexities, false); rule != "" {
		add("complexities", false, rule)
	}
	return out
}

func tokenListRule(values []string, capability bool) string {
	seen := map[string]bool{}
	for _, v := range values {
		switch {
		case !validToken(v) || !SafeIdentifier(v):
			return "each entry must be a lowercase token of a-z, 0-9, '-', '_' or '.'"
		case seen[v]:
			return "entries must not repeat"
		case capability && strings.HasPrefix(v, "reasoning-effort-"):
			return "reasoning-effort-* is reserved; declare effort with --effort-mode/--effort"
		}
		seen[v] = true
	}
	return ""
}
