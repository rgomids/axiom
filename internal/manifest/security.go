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
	if machinePath(value) {
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
	if kind == "logical" && referencePayload(value) {
		return false
	}
	if kind == "logical" && strings.Contains(value, "://") {
		return false
	}
	return safeURL(value)
}
func referencePayload(value string) bool {
	name, _, found := strings.Cut(value, ":")
	return found && sensitiveParameterV1(name)
}

func machinePath(v string) bool {
	if strings.HasPrefix(v, "/") || strings.HasPrefix(v, "\\") || strings.HasPrefix(v, "~") {
		return true
	}
	return len(v) >= 3 && ((v[0] >= 'A' && v[0] <= 'Z') || (v[0] >= 'a' && v[0] <= 'z')) && v[1] == ':' && (v[2] == '/' || v[2] == '\\')
}
func safeURL(raw string) bool {
	if !strings.Contains(raw, "://") {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if strings.EqualFold(u.Scheme, "file") {
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
