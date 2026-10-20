package runtimeadapter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

var ErrEffortOverrideConflict = runtimeprofile.ErrEffortOverrideConflict

type EffortSource struct {
	Scope          string
	Root, Relative string
	// Format is toml for Codex or json for Claude, never a login/session file.
	Format string
	// Directory explicitly consents to the entire bounded managed drop-in inventory.
	Directory bool
}
type effortSourceFact struct {
	Binding  string `json:"binding,omitempty"`
	Scope    string `json:"scope"`
	Key      string `json:"key"`
	Present  bool   `json:"present"`
	Selector bool   `json:"selector"`
}

type effortIntent struct {
	RuntimeID, InvocationMode, ExecutableDigest, Model string
	Effort                                             runtimeprofile.Effort
}

type effortRules struct {
	JSONKeys, EnvironmentKeys []string
	Sources                   []EffortSource
	NestedProfiles            bool
}

func observeEffortFacts(ctx context.Context, p effortIntent, d effortRules, inv executiongraph.Invocation) ([]effortSourceFact, error) {
	if ctx.Err() != nil || len(inv.Argv) < 4 || inv.RuntimeID != p.RuntimeID || len(d.Sources) > 32 || len(d.JSONKeys) > 16 || len(d.EnvironmentKeys) > 16 {
		return nil, ErrEffortConfigurationUnsafe
	}
	identity, e := ExecutableIdentity(inv.Argv[0])
	if e != nil || identity != p.ExecutableDigest {
		return nil, ErrEffortConfigurationUnsafe
	}
	mode := "exec"
	if p.RuntimeID == "claude" {
		mode = "--print"
	}
	if inv.Argv[1] != mode || inv.Argv[2] != "--model" || inv.Argv[3] != p.Model {
		return nil, ErrEffortConfigurationUnsafe
	}
	facts := []effortSourceFact{}
	args := inv.Argv[4:]
	// Only the Profile-derived canonical argv override is permitted.
	if p.Effort.Mode == runtimeprofile.EffortExplicit {
		switch p.RuntimeID {
		case "codex":
			if len(args) < 2 || args[len(args)-2] != "--config" || args[len(args)-1] != "model_reasoning_effort="+p.Effort.Value {
				return nil, ErrEffortConfigurationUnsafe
			}
			args = args[:len(args)-2]
			facts = append(facts, effortSourceFact{Scope: "argv", Key: "model_reasoning_effort", Present: true, Selector: true})
		case "claude": // Canonical adapter encodes effort as one explicit child env value.
			count := 0
			for _, v := range inv.Env {
				if strings.HasPrefix(v, "CLAUDE_CODE_EFFORT_LEVEL=") {
					count++
					if v != "CLAUDE_CODE_EFFORT_LEVEL="+p.Effort.Value {
						return nil, ErrEffortOverrideConflict
					}
				}
			}
			if count != 1 {
				return nil, ErrEffortConfigurationUnsafe
			}
			facts = append(facts, effortSourceFact{Scope: "canonical-environment", Key: "CLAUDE_CODE_EFFORT_LEVEL", Present: true, Selector: true})
		default:
			return nil, ErrEffortConfigurationUnsafe
		}
	} else {
		facts = append(facts, effortSourceFact{Scope: "argv", Key: "profile-no-override", Present: true})
	}

	for n := 0; n < len(args); n++ {
		a := args[n]
		if n+1 < len(args) && effortPathOperand(p.RuntimeID, a, args[n+1]) {
			n++
			continue
		}
		lower := strings.ToLower(a)
		if strings.Contains(lower, "effort") || a == "--config" || a == "-c" || a == "--profile" || a == "-p" {
			return nil, ErrEffortOverrideConflict
		}
	}
	env := executiongraph.EffectiveInvocationEnvironment(inv)
	keys := append([]string(nil), d.EnvironmentKeys...)
	if p.RuntimeID == "claude" && !containsKey(keys, "CLAUDE_CODE_EFFORT_LEVEL") {
		return nil, ErrEffortConfigurationUnsafe
	}
	seenKeys := map[string]bool{}
	for _, key := range keys {
		normalized := strings.ToUpper(key)
		if seenKeys[normalized] {
			return nil, ErrEffortConfigurationUnsafe
		}
		seenKeys[normalized] = true
		if !safeEffortKey(key) {
			return nil, ErrEffortConfigurationUnsafe
		}
		present := false
		count := 0
		for _, v := range env {
			name, value, ok := strings.Cut(v, "=")
			if ok && strings.EqualFold(name, key) {
				count++
				present = true
				if p.RuntimeID == "claude" && p.Effort.Mode == runtimeprofile.EffortExplicit && key == "CLAUDE_CODE_EFFORT_LEVEL" && value == p.Effort.Value && name == key {
					present = false
				}
			}
		}
		if p.RuntimeID == "claude" && p.Effort.Mode == runtimeprofile.EffortExplicit && key == "CLAUDE_CODE_EFFORT_LEVEL" && count != 1 {
			return nil, ErrEffortOverrideConflict
		}
		facts = append(facts, effortSourceFact{Scope: "environment", Key: key, Present: present, Selector: present})
		if present {
			return nil, ErrEffortOverrideConflict
		}
	}
	seen := map[string]bool{}
	for _, source := range d.Sources {
		if ctx.Err() != nil {
			return nil, ErrEffortConfigurationUnsafe
		}
		if !recognizedEffortSource(p.RuntimeID, source) || seen[source.Scope+"\x00"+source.Root+"\x00"+source.Relative] {
			return nil, ErrEffortConfigurationUnsafe
		}
		seen[source.Scope+"\x00"+source.Root+"\x00"+source.Relative] = true
		wires, present, binding, e := observeEffortSource(source)
		if e != nil {
			return nil, ErrEffortConfigurationUnsafe
		}
		selectors := []string{}
		for _, wire := range wires {
			var selected []string
			if source.Format == "toml" {
				selected, e = tomlEffortSelectorsBounded(wire, d.NestedProfiles)
			} else {
				selected, e = jsonEffortSelectors(wire, d.JSONKeys)
			}
			if e != nil {
				return nil, ErrEffortConfigurationInvalid
			}
			selectors = append(selectors, selected...)
		}
		facts = append(facts, effortSourceFact{Scope: source.Scope, Key: "source", Present: present, Binding: binding})
		for _, key := range selectors {
			facts = append(facts, effortSourceFact{Scope: source.Scope, Key: key, Present: true, Selector: true})
		}

		// Native no-selector semantics, not value merging or invented precedence.
		if len(selectors) != 0 {
			return nil, ErrEffortOverrideConflict
		}
	}
	// Require explicit complete scope declaration; absent files are facts, while
	// omitted scopes or managed drop-ins cannot be approved by an empty digest.
	required := []string{"user", "project", "managed"}
	if p.RuntimeID == "claude" {
		required = append(required, "project-local", "managed-drop-in")
	}
	for _, scope := range required {
		found := false
		for _, s := range d.Sources {
			found = found || s.Scope == scope
		}
		if !found {
			return nil, ErrEffortConfigurationUnsafe
		}
	}
	sort.Slice(facts, func(a, b int) bool {
		if facts[a].Scope != facts[b].Scope {
			return facts[a].Scope < facts[b].Scope
		}
		if facts[a].Key != facts[b].Key {
			return facts[a].Key < facts[b].Key
		}
		if facts[a].Binding != facts[b].Binding {
			return facts[a].Binding < facts[b].Binding
		}
		if facts[a].Present != facts[b].Present {
			return !facts[a].Present
		}
		return !facts[a].Selector && facts[b].Selector
	})
	return facts, nil
}

// effortPathOperand reports whether value is the filesystem operand of one of
// the adapter's own path options (BuildStageInvocation). A clean absolute path
// there names a location, not an effort selector, so its spelling is not
// scanned; the option name and every other token still are. It adds no fact.
func effortPathOperand(runtimeID, option, value string) bool {
	switch {
	case option == "--add-dir" || runtimeID == "codex" && option == "-C":
		return stagePath(value)
	case runtimeID == "claude" && option == "--allowedTools" && !strings.HasPrefix(value, "-"):
		for _, tool := range strings.Split(value, ",") {
			path, ok := strings.CutPrefix(tool, "Bash(")
			if ok {
				path, ok = strings.CutSuffix(path, ":*)")
			}
			if !(ok && stagePath(path)) && strings.Contains(strings.ToLower(tool), "effort") {
				return false
			}
		}
		return true
	}
	return false
}
func containsKey(keys []string, key string) bool {
	for _, v := range keys {
		if v == key {
			return true
		}
	}
	return false
}
func safeEffortKey(v string) bool {
	if v == "" || len(v) > 128 {
		return false
	}
	for _, r := range v {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}
func recognizedEffortSource(runtimeID string, s EffortSource) bool {
	if !filepath.IsAbs(s.Root) || filepath.Clean(s.Root) != s.Root || filepath.Clean(s.Relative) != s.Relative || !filepath.IsLocal(s.Relative) || strings.ContainsAny(s.Relative, "\x00\r\n") {
		return false
	}
	if s.Scope != "user" && s.Scope != "project" && s.Scope != "managed" && s.Scope != "project-local" && s.Scope != "managed-drop-in" {
		return false
	}
	if s.Directory {
		return runtimeID == "claude" && s.Scope == "managed-drop-in" && s.Format == "json"
	}
	if s.Scope == "managed-drop-in" {
		return false
	}
	base := filepath.Base(s.Relative)
	if runtimeID == "codex" {
		return s.Format == "toml" && (base == "config.toml" || base == "managed_config.toml")
	}
	return runtimeID == "claude" && s.Format == "json" && (base == "settings.json" || base == "settings.local.json" || base == "managed-settings.json" || s.Scope == "managed-drop-in" && strings.HasSuffix(base, ".json"))
}

// effortPathBinding includes exact canonical target, nearest absence boundary,
// and every existing ancestor/object identity. Symlinks and special files fail.
func effortPathBinding(target string) (string, bool, error) {
	if !filepath.IsAbs(target) || filepath.Clean(target) != target {
		return "", false, ErrEffortConfigurationUnsafe
	}
	canonical, e := effortCanonicalTarget(target)
	if e != nil {
		return "", false, ErrEffortConfigurationUnsafe
	}
	target = canonical
	facts := target
	current := filepath.VolumeName(target) + string(filepath.Separator)
	for _, part := range strings.Split(strings.TrimPrefix(target, current), string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, e := os.Lstat(current)
		if os.IsNotExist(e) {
			return effortHash([]byte(facts + "\x00absent:" + current)), false, nil
		}
		if e != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() && !info.Mode().IsRegular() {
			return "", false, ErrEffortConfigurationUnsafe
		}
		f, e := os.Open(current)
		if e != nil {
			return "", false, ErrEffortConfigurationUnsafe
		}
		opened, e := f.Stat()
		identity, e2 := effortObjectIdentity(f)
		f.Close()
		after, e3 := os.Lstat(current)
		if e != nil || e2 != nil || e3 != nil || after.Mode()&os.ModeSymlink != 0 || !os.SameFile(info, opened) || !os.SameFile(after, opened) {
			return "", false, ErrEffortConfigurationUnsafe
		}
		facts += "\x00" + identity
	}
	return effortHash([]byte(facts)), true, nil
}
func effortHash(w []byte) string { hash := sha256.Sum256(w); return hex.EncodeToString(hash[:]) }

func readEffortSource(s EffortSource) ([]byte, bool, string, error) {
	target := filepath.Join(s.Root, s.Relative)
	beforeBinding, present, e := effortPathBinding(target)
	if e != nil || !present {
		return nil, present, beforeBinding, e
	}
	root, e := os.OpenRoot(s.Root)
	if e != nil {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	defer root.Close()
	before, e := root.Lstat(s.Relative)
	if e != nil || !before.Mode().IsRegular() || before.Size() > maxConfigFileBytes {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	f, e := root.Open(s.Relative)
	if e != nil {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(before, opened) {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	wire, e := io.ReadAll(io.LimitReader(f, maxConfigFileBytes+1))
	afterBinding, afterPresent, e2 := effortPathBinding(target)
	after, e3 := f.Stat()
	if e != nil || e2 != nil || e3 != nil || !afterPresent || beforeBinding != afterBinding || len(wire) > maxConfigFileBytes || opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime()) {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	// Verify a second read against the same object; no in-place rewrite during
	// this declared observation boundary can produce a hybrid approved wire.
	if _, e = f.Seek(0, 0); e != nil {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	again, e := io.ReadAll(io.LimitReader(f, maxConfigFileBytes+1))
	if e != nil || !bytes.Equal(wire, again) {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	return wire, true, effortHash([]byte(beforeBinding + "\x00" + effortHash(wire))), nil
}

func observeEffortSource(s EffortSource) ([][]byte, bool, string, error) {
	if !s.Directory {
		wire, present, binding, e := readEffortSource(s)
		if !present || e != nil {
			return nil, present, binding, e
		}
		return [][]byte{wire}, true, binding, nil
	}
	target := filepath.Join(s.Root, s.Relative)
	before, present, e := effortPathBinding(target)
	if e != nil || !present {
		return nil, present, before, e
	}
	root, e := os.OpenRoot(target)
	if e != nil {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	defer root.Close()
	directory, e := root.Open(".")
	if e != nil {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	defer directory.Close()
	entries, e := directory.ReadDir(65)
	if e != nil && e != io.EOF || len(entries) > 64 {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".json") {
			return nil, false, "", ErrEffortConfigurationUnsafe
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	wires := [][]byte{}
	facts := before
	for _, name := range names {
		child := EffortSource{Root: target, Relative: name, Format: "json"}
		wire, present, binding, e := readEffortSource(child)
		if e != nil || !present {
			return nil, false, "", ErrEffortConfigurationUnsafe
		}
		facts += "\x00" + name + "\x00" + binding
		wires = append(wires, wire)
	}
	// Membership and ancestor identities are rechecked before digest admission.
	check, e := root.Open(".")
	if e != nil {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	afterEntries, e := check.ReadDir(65)
	check.Close()
	if e != nil && e != io.EOF || len(afterEntries) != len(entries) {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	afterNames := []string{}
	for _, entry := range afterEntries {
		afterNames = append(afterNames, entry.Name())
	}
	sort.Strings(afterNames)
	recheckedFacts := before
	for _, name := range names {
		_, childPresent, binding, e := readEffortSource(EffortSource{Root: target, Relative: name, Format: "json"})
		if e != nil || !childPresent {
			return nil, false, "", ErrEffortConfigurationUnsafe
		}
		recheckedFacts += "\x00" + name + "\x00" + binding
	}
	if recheckedFacts != facts {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	after, stillPresent, e := effortPathBinding(target)
	if e != nil || !stillPresent || before != after || strings.Join(names, "\x00") != strings.Join(afterNames, "\x00") {
		return nil, false, "", ErrEffortConfigurationUnsafe
	}
	return wires, true, effortHash([]byte(facts)), nil
}

var effortTOMLNumber = regexp.MustCompile(`^[+-]?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)

func validEffortTOMLScalar(value string) bool {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || strings.ContainsAny(value, "\x00\r\n") {
		return false
	}
	if value == "true" || value == "false" || effortTOMLNumber.MatchString(value) {
		return true
	}
	if len(value) < 2 {
		return false
	}
	if value[0] == '\'' {
		return value[len(value)-1] == '\'' && !strings.ContainsAny(value[1:len(value)-1], "'\x00\r\n")
	}
	if value[0] != '"' || strings.Contains(value, `\/`) {
		return false
	}
	var decoded string
	return json.Unmarshal([]byte(value), &decoded) == nil && !strings.ContainsAny(decoded, "\x00\r\n")
}

// Only TOML basic/literal/bare keys whose quoting is fully validated enter
// the narrow parser; Go-only escapes accepted by the historic helper refuse.
func effortTOMLKey(raw string) ([]string, bool) {
	parts, ok := tomlKey(raw)
	if !ok || !utf8.ValidString(raw) {
		return nil, false
	}
	for _, part := range parts {
		for _, r := range part {
			if r < 0x20 || r == 0x7f {
				return nil, false
			}
		}
	}
	quote := byte(0)
	start := 0
	for index := 0; index < len(raw); index++ {
		char := raw[index]
		if quote == '"' && char == '\\' {
			index++
			continue
		}
		if quote != 0 && char == quote {
			if quote == '"' {
				literal := raw[start : index+1]
				var decoded string
				if strings.Contains(literal, `\/`) || json.Unmarshal([]byte(literal), &decoded) != nil {
					return nil, false
				}
			}
			quote = 0
			continue
		}
		if quote == 0 && (char == '"' || char == '\'') {
			quote = char
			start = index
		}
	}
	return parts, quote == 0
}

func tomlEffortSelectorsBounded(w []byte, nestedProfiles bool) ([]string, error) {
	keys := []string{}
	table := []string{}
	seen := map[string]bool{}
	tables := map[string]bool{}
	identity := func(parts []string) string { wire, _ := json.Marshal(parts); return string(wire) }
	conflicts := func(parts []string) bool {
		for n := 1; n <= len(parts); n++ {
			if seen[identity(parts[:n])] {
				return true
			}
		}
		for key := range seen {
			var existing []string
			json.Unmarshal([]byte(key), &existing)
			if len(existing) > len(parts) && identity(existing[:len(parts)]) == identity(parts) {
				return true
			}
		}
		return false
	}
	if !utf8.Valid(w) {
		return nil, ErrEffortConfigurationUnsafe
	}
	for _, raw := range strings.Split(string(w), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		comment := indexOutsideQuotes(line, "#")
		if comment == -2 {
			return nil, ErrEffortConfigurationUnsafe
		}
		if comment >= 0 {
			line = strings.TrimSpace(line[:comment])
		}
		if header, ok := tomlHeader(line); ok {
			segments, valid := effortTOMLKey(header)
			if !valid || strings.HasPrefix(line, "[[") || !nestedProfiles && containsKey(segments, "profiles") || tables[identity(segments)] || conflicts(segments) {
				return nil, ErrEffortConfigurationUnsafe
			}
			table = segments
			tables[identity(segments)] = true
			continue
		}
		key, value, ok := tomlKeyValue(line)
		if !ok || !validEffortTOMLScalar(value) {
			return nil, ErrEffortConfigurationUnsafe
		}
		segments, ok := effortTOMLKey(key)
		if !ok {
			return nil, ErrEffortConfigurationUnsafe
		}
		full := append(append([]string(nil), table...), segments...)
		if conflicts(full) || tables[identity(full)] {
			return nil, ErrEffortConfigurationUnsafe
		}
		seen[identity(full)] = true
		for _, k := range full {
			if k == "model_reasoning_effort" {
				keys = append(keys, k)
			}
		}
	}
	return keys, nil
}
func jsonEffortSelectors(w []byte, declared []string) ([]string, error) {
	if trimmed := strings.TrimSpace(string(w)); len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, ErrEffortConfigurationUnsafe
	}
	if len(declared) == 0 {
		return nil, ErrEffortConfigurationUnsafe
	}
	for _, k := range declared {
		if !safeEffortKey(k) {
			return nil, ErrEffortConfigurationUnsafe
		}
	}
	// Strict duplicate rejection before scanning (JSON objects with duplicate
	// recognized keys have unknown effective semantics).
	var visit func(*json.Decoder, int) ([]string, error)
	visit = func(d *json.Decoder, depth int) ([]string, error) {
		if depth > 32 {
			return nil, ErrEffortConfigurationUnsafe
		}
		token, e := d.Token()
		if e != nil {
			return nil, e
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil, nil
		}
		keys := []string{}
		seen := map[string]bool{}
		for d.More() {
			if delimiter == '{' {
				token, e = d.Token()
				if e != nil {
					return nil, e
				}
				key, ok := token.(string)
				if !ok || seen[key] {
					return nil, ErrEffortConfigurationUnsafe
				}
				seen[key] = true
				if containsKey(declared, key) {
					keys = append(keys, key)
				}
			}
			nested, e := visit(d, depth+1)
			if e != nil {
				return nil, e
			}
			keys = append(keys, nested...)
		}
		_, e = d.Token()
		return keys, e
	}
	d := json.NewDecoder(strings.NewReader(string(w)))
	keys, e := visit(d, 0)
	if e != nil {
		return nil, ErrEffortConfigurationUnsafe
	}
	if _, e = d.Token(); e != io.EOF {
		return nil, ErrEffortConfigurationUnsafe
	}
	return keys, nil
}
