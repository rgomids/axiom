package projectdiscovery

import "strings"

// config is the subset of a Git config that bootstrap needs: remote url values
// plus a flag for constructs that make the remote set unprovable.
type config struct {
	urls       []urlEntry
	incomplete bool
}

// parseConfig scans a Git config for [remote "<name>"] url values. ok=false
// means the text is malformed. Unknown keys are ignored; include sections,
// insteadOf rewrites, line continuations and exceeded bounds mark the result
// incomplete.
func parseConfig(data string) (cfg config, ok bool) {
	var section, sub string
	var inSection bool
	remotes := map[string]struct{}{}
	lines := strings.Split(data, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(strings.TrimSuffix(lines[i], "\r"))
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		if line[0] == '[' {
			var rest string
			if section, sub, rest, ok = parseHeader(line); !ok {
				return config{}, false
			}
			inSection = true
			if section == "include" || section == "includeif" {
				cfg.incomplete = true
			}
			if section == "remote" && sub != "" {
				remotes[sub] = struct{}{}
				if len(remotes) > maxRemotes {
					cfg.incomplete = true
				}
			}
			if line = strings.TrimSpace(rest); line == "" || line[0] == '#' || line[0] == ';' {
				continue
			}
		}
		if !inSection {
			return config{}, false
		}
		key, value, cont, ok := parseVariable(line)
		if !ok {
			return config{}, false
		}
		if cont {
			cfg.incomplete = true
			for strings.HasSuffix(strings.TrimSuffix(lines[i], "\r"), "\\") && i+1 < len(lines) {
				i++
			}
			continue
		}
		switch {
		case section == "url" && (key == "insteadof" || key == "pushinsteadof"):
			cfg.incomplete = true
		case section == "remote" && sub != "" && key == "url":
			if len(cfg.urls) >= maxURLValues || len(remotes) > maxRemotes {
				cfg.incomplete = true
				continue
			}
			cfg.urls = append(cfg.urls, urlEntry{name: sub, raw: value})
		}
	}
	return cfg, true
}

// parseHeader parses `[section]`, `[section.legacy]` and `[section "sub"]`
// and returns any text after the closing bracket.
func parseHeader(s string) (section, sub, rest string, ok bool) {
	i := 1
	for i < len(s) && (isAlnum(s[i]) || s[i] == '-' || s[i] == '.') {
		i++
	}
	name := s[1:i]
	if name == "" {
		return "", "", "", false
	}
	if i < len(s) && s[i] == ']' {
		section, legacy, _ := strings.Cut(name, ".")
		return strings.ToLower(section), strings.ToLower(legacy), s[i+1:], section != ""
	}
	if strings.Contains(name, ".") {
		return "", "", "", false
	}
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	if i >= len(s) || s[i] != '"' {
		return "", "", "", false
	}
	var b strings.Builder
	for i++; i < len(s); i++ {
		switch s[i] {
		case '\\':
			if i+1 >= len(s) {
				return "", "", "", false
			}
			i++
			b.WriteByte(s[i])
		case '"':
			if i+1 >= len(s) || s[i+1] != ']' {
				return "", "", "", false
			}
			if b.Len() == 0 {
				return "", "", "", false
			}
			return strings.ToLower(name), b.String(), s[i+2:], true
		default:
			b.WriteByte(s[i])
		}
	}
	return "", "", "", false
}

// parseVariable parses `key`, `key = value`. cont reports a trailing
// backslash continuation.
func parseVariable(s string) (key, value string, cont, ok bool) {
	i := 0
	for i < len(s) && (isAlnum(s[i]) || s[i] == '-') {
		i++
	}
	if i == 0 || !isAlpha(s[0]) {
		return "", "", false, false
	}
	key = strings.ToLower(s[:i])
	rest := strings.TrimLeft(s[i:], " \t")
	if rest == "" || rest[0] == '#' || rest[0] == ';' {
		return key, "", false, true
	}
	if rest[0] != '=' {
		return "", "", false, false
	}
	value, cont, ok = parseValue(rest[1:])
	return key, value, cont, ok
}

// parseValue handles quotes, \" \\ \n \t \b escapes and inline comments.
func parseValue(s string) (value string, cont, ok bool) {
	var b strings.Builder
	keep, inQuote := 0, false
loop:
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\':
			if i+1 >= len(s) {
				return "", true, true
			}
			i++
			switch s[i] {
			case '"', '\\':
				b.WriteByte(s[i])
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'b':
				b.WriteByte('\b')
			default:
				return "", false, false
			}
			keep = b.Len()
		case c == '"':
			inQuote = !inQuote
			keep = b.Len()
		case !inQuote && (c == '#' || c == ';'):
			break loop
		case !inQuote && (c == ' ' || c == '\t'):
			if b.Len() > 0 {
				b.WriteByte(c)
			}
		default:
			b.WriteByte(c)
			keep = b.Len()
		}
	}
	if inQuote {
		return "", false, false
	}
	return b.String()[:keep], false, true
}

func isAlpha(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isAlnum(c byte) bool { return isAlpha(c) || c >= '0' && c <= '9' }
