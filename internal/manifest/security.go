package manifest

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
func safeValue(value, kind string) bool {
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
