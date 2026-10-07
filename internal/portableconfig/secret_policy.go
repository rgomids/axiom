package portableconfig

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// sensitiveParameterV1 is a fixed structural policy, not a secret-value scanner.
// Legacy URL query decoding remains with safeURL; key comparison folds case
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

func referencePayload(value string) bool {
	separator := strings.IndexAny(value, ":=")
	return separator >= 0 && sensitiveParameterV1(strings.TrimSpace(value[:separator]))
}

// ContainsSecretBearingValue inspects the complete original v3 prose value,
// independent of URL parsing. It detects sensitive names followed by ':' or
// '=', including whitespace and punctuation wrappers, at every bounded percent
// escape layer. A name alone is allowed; normalization budget exhaustion is unsafe.
// This is a structural assignment policy, not a general credential recognizer.
func ContainsSecretBearingValue(value string) bool {
	return !inspectProse(value, func(layer string) bool {
		return !secretBearingAssignment(layer)
	})
}

func secretBearingAssignment(value string) bool {
	for i := 0; i < len(value); {
		r, width := utf8.DecodeRuneInString(value[i:])
		if !secretNameRune(r) {
			i += width
			continue
		}
		start := i
		for i < len(value) {
			r, width = utf8.DecodeRuneInString(value[i:])
			if !secretNameRune(r) {
				break
			}
			i += width
		}
		if !sensitiveParameterV1(value[start:i]) {
			continue
		}
		// Prose/Markdown wrappers may surround a key/operator. Reference
		// separators terminate the association: never cross components to find
		// an operator, or skip another name. No URL parsing defines this rule.
		for i < len(value) {
			r, width = utf8.DecodeRuneInString(value[i:])
			if r == ':' || r == '=' {
				return true
			}
			if !unicode.IsSpace(r) && !strings.ContainsRune("\"'`*{}_[](),;<>", r) {
				break
			}
			i += width
		}
	}
	return false
}

func secretNameRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r) || r == '-' || r == '_'
}
