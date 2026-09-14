package project

import (
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

var dnsLabel = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// NormalizeLocator performs syntax-only matching, with no DNS or network I/O.
// Only URI scheme and DNS host case change. Original escaping, path, port,
// user, suffix, query and fragment bytes are retained. SSH shorthand stays SSH.
func NormalizeLocator(raw string) (string, []Issue) {
	if raw == "" || strings.ContainsRune(raw, '\\') || strings.IndexFunc(raw, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return invalidLocator()
	}
	if strings.Contains(raw, "://") {
		return normalizeURI(raw)
	}
	return normalizeSSH(raw)
}

func invalidLocator() (string, []Issue) {
	return "", []Issue{{Field: "remote", Code: "invalid_locator"}}
}

func normalizeURI(raw string) (string, []Issue) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" || u.Opaque != "" {
		return invalidLocator()
	}
	if u.User != nil && !safeSSHUser(u) {
		return invalidLocator()
	}
	host := u.Hostname()
	if !validHost(host) {
		return invalidLocator()
	}
	// Use the original authority to avoid URL serialization changing escaped bytes.
	schemeEnd := strings.Index(raw, "://")
	authorityStart := schemeEnd + 3
	authorityEnd := len(raw)
	if i := strings.IndexAny(raw[authorityStart:], "/?#"); i >= 0 {
		authorityEnd = authorityStart + i
	}
	authority := raw[authorityStart:authorityEnd]
	hostStart := strings.LastIndex(authority, "@") + 1
	if strings.HasPrefix(authority[hostStart:], "[") {
		return strings.ToLower(raw[:schemeEnd]) + raw[schemeEnd:], nil
	}
	normalizedAuthority := authority[:hostStart] + strings.ToLower(host) + authority[hostStart+len(host):]
	return strings.ToLower(raw[:schemeEnd]) + "://" + normalizedAuthority + raw[authorityEnd:], nil
}

func safeSSHUser(u *url.URL) bool {
	_, password := u.User.Password()
	return strings.EqualFold(u.Scheme, "ssh") && !password && u.User.Username() != ""
}

func normalizeSSH(raw string) (string, []Issue) {
	colon := strings.IndexByte(raw, ':')
	if colon <= 0 || colon == len(raw)-1 {
		return invalidLocator()
	}
	authority, remotePath := raw[:colon], raw[colon+1:]
	if strings.ContainsAny(authority, "/?#[]") || strings.Count(authority, "@") > 1 {
		return invalidLocator()
	}
	hostStart := strings.LastIndex(authority, "@") + 1
	if hostStart == 1 {
		return invalidLocator()
	}
	host := authority[hostStart:]
	// A one-letter host is ambiguous with a Windows drive path.
	if (len(host) == 1 && hostStart == 0) || !validHost(host) {
		return invalidLocator()
	}
	return authority[:hostStart] + strings.ToLower(host) + ":" + remotePath, nil
}

func validHost(host string) bool {
	if _, err := netip.ParseAddr(host); err == nil {
		return true
	}
	if len(host) > 253 || host == "" {
		return false
	}
	for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
		if len(label) > 63 || !dnsLabel.MatchString(label) {
			return false
		}
	}
	return true
}
