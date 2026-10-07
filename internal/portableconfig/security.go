// Package portableconfig defines pure structural safety for portable Project values.
// It has no codec, filesystem, provider or runtime dependency.
package portableconfig

import (
	"net/url"
	"strings"
	"unicode"
)

// sensitiveParameterV1 is a fixed structural policy, not a secret-value scanner.
// Percent decoding is done once using URL query syntax; key comparison folds case
// and removes '-'/'_' to cover explicitly documented spelling families.
func sensitiveParameterV1(name string) bool {
	name = strings.ToLower(name)
	name = strings.NewReplacer("-", "", "_", "").Replace(name)
	switch name {
	case "token", "accesstoken", "refreshtoken", "idtoken", "authtoken", "oauthtoken",
		"password", "passwd", "pwd", "apikey", "key", "secret", "clientsecret",
		"signature", "sig", "credential", "authorization", "auth",
		"xamzsignature", "xamzcredential", "xamzsecuritytoken", "xgoogsignature", "xgoogcredential":
		return true
	}
	return false
}

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
func referencePayload(value string) bool {
	separator := strings.IndexAny(value, ":=")
	return separator >= 0 && sensitiveParameterV1(strings.TrimSpace(value[:separator]))
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

// safeProseURL adds v3 prose safety without changing the legacy scalar URL
// contract. Inspect authority sub-delimiters only after parsing: path/query
// commas and semicolons retain their URL meaning.
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
	parts := strings.FieldsFunc(authority, func(r rune) bool {
		return proseDelimiter(r) || r == '&' || r == '@'
	})
	if !safeProseParts(parts) {
		return false
	}
	// A delimiter at the end of the authority may swallow adjacent rooted
	// prose as the parsed URL path. Ordinary URL paths have no such delimiter.
	authorityEnd := strings.TrimRight(u.Host, "\"'`*{}[]()<>_")
	if strings.HasSuffix(authorityEnd, ",") || strings.HasSuffix(authorityEnd, ";") || strings.HasSuffix(authorityEnd, "=") {
		return !machinePath(u.Path)
	}
	return true
}

func unsafeProseReference(value string) bool {
	for {
		if machinePath(value) || referencePayload(value) {
			return true
		}
		separator := strings.IndexAny(value, ":=")
		if separator < 0 {
			return false
		}
		value = value[separator+1:]
	}
}

// SafeProse checks human text without imposing identifier syntax. Inspect each
// word/reference and adjacent assignment syntax, including references embedded
// in prose. Newlines are allowed here; callers own text bounds and formatting.
// URI escapes are inspected with the same bounded budget as scalar references.
// Literal percent signs remain valid prose; only complete URI escapes decode.
func SafeProse(value string) bool {
	budget := 4 * len(value)
	for {
		if len(value) > budget || !safeProseSyntax(value) {
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

func safeProseSyntax(value string) bool {
	if strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) && r != '\n' }) >= 0 {
		return false
	}
	// Check full URLs before punctuation can split their query parameters.
	// After inspection, tokenize only their surrounding prose, not URL syntax.
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
	for i, part := range parts {
		part = strings.Trim(part, "_")
		if unsafeProseReference(part) || !safeURL(part) {
			return false
		}
		// Whitespace/punctuation around a structural assignment does not hide it.
		name := strings.TrimRight(part, ":=")
		if sensitiveParameterV1(name) && (name != part || i+1 < len(parts) && strings.HasPrefix(parts[i+1], "=") || i+1 < len(parts) && strings.HasPrefix(parts[i+1], ":")) {
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
