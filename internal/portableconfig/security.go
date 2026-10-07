// Package portableconfig defines pure deterministic safety for portable Project values.
// It has no codec, filesystem, provider or runtime dependency.
package portableconfig

import (
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SafeValue preserves the portable scalar policy used by every manifest version.
func SafeValue(value, kind string) bool {
	if kind == "prose" {
		return SafeProse(value)
	}
	if kind == "token" {
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
	if kind == "remote" {
		return safeURL(value)
	}
	// Inspect URI escapes without changing stored identifiers. Bound total bytes
	// inspected across nested escaping; malformed/excessive escaping fails closed.
	// YAML escapes have already been resolved by the node parser.
	budget := 4 * len(value)
	for {
		if len(value) > budget || !safeReferenceSyntax(value, kind) {
			return false
		}
		budget -= len(value)
		if !strings.Contains(value, "%") {
			return true
		}
		decoded, err := url.PathUnescape(value)
		if err != nil {
			return false
		}
		value = decoded
	}
}

func safeReferenceSyntax(value, kind string) bool {
	if machinePath(strings.TrimSpace(value)) {
		return false
	}
	if kind == "logical" && strings.ContainsAny(value, "=\\$`?#") {
		return false
	}
	if kind == "logical" && strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return false
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return false
	}
	if referencePayload(value) {
		return false
	}
	if kind == "logical" && (strings.Contains(value, "://") || logicalPath(value)) {
		return false
	}
	return safeURL(value)
}

// Slash-separated namespaces remain valid logical names. Dot path components
// and rooted suffixes do not, even after a namespace. Drive-relative forms fail too.
func logicalPath(value string) bool {
	if driveReference(value) {
		return true
	}
	for _, suffix := range strings.Split(value, ":") {
		if machinePath(suffix) {
			return true
		}
		for _, part := range strings.Split(suffix, "/") {
			if part == "." || part == ".." {
				return true
			}
		}
	}
	return false
}

func machinePath(v string) bool {
	if strings.HasPrefix(v, "/") || strings.HasPrefix(v, "\\") || strings.HasPrefix(v, "~") {
		return true
	}
	if fileReference(v) {
		return true
	}
	return len(v) >= 3 && driveReference(v) && (v[2] == '/' || v[2] == '\\')
}

func driveReference(v string) bool {
	return len(v) >= 2 && ((v[0] >= 'A' && v[0] <= 'Z') || (v[0] >= 'a' && v[0] <= 'z')) && v[1] == ':'
}

func fileReference(value string) bool {
	scheme, _, found := strings.Cut(value, ":")
	return found && strings.EqualFold(scheme, "file")
}

func safeURL(raw string) bool {
	if fileReference(raw) {
		return false
	}
	if !strings.Contains(raw, "://") {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.User != nil {
		_, password := u.User.Password()
		if password || !strings.EqualFold(u.Scheme, "ssh") || u.User.Username() == "" {
			return false
		}
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return false
	}
	for key := range query {
		if sensitiveParameterV1(key) {
			return false
		}
	}
	return true
}

// safeProseURL adds v3 machine-reference checks without changing the legacy
// scalar URL contract. Secret inspection receives the full input separately.
func safeProseURL(raw string) bool {
	if !safeURL(raw) {
		return false
	}
	if !strings.Contains(raw, "://") {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	authority := u.Host
	if u.User != nil {
		authority = u.User.Username() + "@" + authority
	}
	parts := strings.FieldsFunc(authority, authorityProseDelimiter)
	if !safeProseParts(parts) {
		return false
	}
	// A drive prefix can remain in the authority while its rooted suffix is
	// parsed as a URL path. Inspect assignment suffixes without treating IPv6
	// colons or an ordinary host/path join as machine-local prose.
	if len(parts) > 0 && u.Path != "" {
		for suffix := parts[len(parts)-1]; ; {
			candidate := strings.Trim(suffix, "_")
			if driveReference(candidate) && machinePath(candidate+u.Path) {
				return false
			}
			separator := strings.IndexAny(suffix, ":=")
			if separator < 0 {
				break
			}
			suffix = suffix[separator+1:]
		}
	}
	// A delimiter at the end of the authority may swallow adjacent rooted
	// prose as the parsed URL path. Ordinary URL paths have no such delimiter.
	authorityEnd := strings.TrimRight(u.Host, "\"`{}[]<>_")
	if len(authorityEnd) > 0 && authoritySubDelimiter(rune(authorityEnd[len(authorityEnd)-1])) {
		if machinePath(u.Path) {
			return false
		}
	}
	return true
}

// authoritySubDelimiter is the RFC 3986 sub-delims grammar. A terminal
// sub-delimiter before a rooted path can mark adjacent prose, so it cannot
// authorize consumption of that path as part of a v3 prose URL.
func authoritySubDelimiter(r rune) bool {
	return strings.ContainsRune("!$&'()*+,;=", r)
}

// authorityProseDelimiter adds authority boundaries to prose punctuation and
// the userinfo separator. Keep '=' and ':' inside parts so machine-reference
// suffixes remain inspectable.
// '=' is a sub-delimiter for path transitions, but a suffix boundary here.
func authorityProseDelimiter(r rune) bool {
	return proseDelimiter(r) || r == '@' || r != '=' && authoritySubDelimiter(r)
}

func unsafeProseReference(value string) bool {
	for {
		if machinePath(value) {
			return true
		}
		separator := strings.IndexAny(value, ":=")
		if separator < 0 {
			return false
		}
		value = value[separator+1:]
	}
}

// SafeProse composes independent secret and structural policies for v3 prose.
// Every escape layer is inspected in full before URL/reference parsing can
// consume any portion. Callers retain ownership of bounds and line formatting.
func SafeProse(value string) bool {
	return inspectProse(value, func(layer string) bool {
		return !secretBearingAssignment(layer) && safeProseStructure(layer)
	})
}

// inspectProse shares bounded normalization between the independent policies.
// Literal percent signs remain prose; only complete URI escapes decode. Budget
// exhaustion fails closed, including when the secret policy is used in isolation.
func inspectProse(value string, safeLayer func(string) bool) bool {
	budget := 4 * len(value)
	for {
		if len(value) > budget || !safeLayer(value) {
			return false
		}
		budget -= len(value)
		decoded, changed := decodeProseEscapes(value)
		if !changed {
			return true
		}
		value = decoded
	}
}

func safeProseStructure(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	if strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) && r != '\n' }) >= 0 {
		return false
	}
	// Check full URLs before punctuation can split their query parameters.
	// Secrets have already been inspected independently across the full layer.
	// Only structural URL/reference validation may consume references here.
	parts := []string{}
	for _, word := range strings.Fields(value) {
		for {
			marker := strings.Index(word, "://")
			if marker < 0 {
				break
			}
			start := marker
			for start > 0 && schemeRune(word[start-1]) {
				start--
			}
			end := proseURLEnd(word, start)
			reference := strings.TrimRight(word[start:end], "\"'`*{}[](),;<>.")
			if !safeProseURL(reference) {
				return false
			}
			word = word[:start] + " " + word[end:]
		}
		parts = append(parts, strings.FieldsFunc(word, func(r rune) bool {
			return proseDelimiter(r)
		})...)
	}
	return safeProseParts(parts)
}

func proseDelimiter(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("\"'`*{}[](),;<>", r)
}

func safeProseParts(parts []string) bool {
	for _, part := range parts {
		part = strings.Trim(part, "_")
		if unsafeProseReference(part) || !safeURL(part) {
			return false
		}
	}
	return true
}

func decodeProseEscapes(value string) (string, bool) {
	var out strings.Builder
	changed := false
	for i := 0; i < len(value); i++ {
		if value[i] == '%' && i+2 < len(value) {
			decoded, err := url.PathUnescape(value[i : i+3])
			if err == nil {
				out.WriteString(decoded)
				i += 2
				changed = true
				continue
			}
		}
		out.WriteByte(value[i])
	}
	return out.String(), changed
}

// URI schemes use ASCII letters, digits and +.-; surrounding Markdown is not
// part of a URL. The existing URL policy still owns parsing and query safety.
func schemeRune(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.'
}

// A surrounding Markdown/quote delimiter bounds a URL. Preserve everything
// after it for prose inspection; commas and semicolons inside bare URLs retain
// their URI meaning. Balanced parentheses may occur inside a linked URL.
func proseURLEnd(word string, start int) int {
	if start == 0 {
		return len(word)
	}
	opener := word[start-1]
	closer := opener
	switch opener {
	case '(':
		depth := 1
		for i := start; i < len(word); i++ {
			if word[i] == '(' {
				depth++
			}
			if word[i] == ')' {
				depth--
				if depth == 0 {
					return i
				}
			}
		}
		return len(word)
	case '<':
		closer = '>'
	case '\'', '"', '`', '*', '_':
	default:
		return len(word)
	}
	if offset := strings.IndexByte(word[start:], closer); offset >= 0 {
		return start + offset
	}
	return len(word)
}
