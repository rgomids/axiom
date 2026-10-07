package portableconfig

import (
	"strings"
	"testing"
)

func TestContainsSecretBearingValue(t *testing.T) {
	for _, tc := range []struct {
		input  string
		want   bool
		reason string
	}{
		{"password=synthetic", true, "plain assignment"},
		{"See token=synthetic", true, "assignment inside prose"},
		{"password = synthetic", true, "whitespace assignment"},
		{"token : synthetic", true, "colon assignment"},
		{"password:", true, "operator without payload retains fail-closed policy"},
		{"PASSWORD=synthetic", true, "case folding"},
		{"Access_Token=synthetic", true, "underscore normalization"},
		{"Api-Key=synthetic", true, "dash normalization"},
		{"https://example.com,password=synthetic", true, "URL authority"},
		{"https://example.com!password=synthetic", true, "authority punctuation"},
		{"ssh://+token=synthetic@example.com/repo", true, "userinfo assignment"},
		{"https://example.com/token=synthetic", true, "URL path"},
		{"https://example.com/path/api_key=synthetic/more", true, "nested URL path"},
		{"https://example.com/docs?token=synthetic", true, "URL query"},
		{"https://example.com#password=synthetic", true, "URL fragment"},
		{"[docs](https://example.com#client_secret=synthetic)", true, "Markdown URL"},
		{"[docs](https://example.com!password)=synthetic", true, "operator after Markdown wrapper"},
		{"\"https://example.com/token=synthetic\"", true, "quoted URL"},
		{"**password** = synthetic", true, "Markdown key wrapper"},
		{"{\"authorization\": \"synthetic\"}", true, "quoted key"},
		{"https://example.com/%74oken%3Dsynthetic", true, "percent encoding"},
		{"https://example.com/%2574oken%253Dsynthetic", true, "nested percent encoding"},
		{"https://example.com/%25%37%34oken%25%33%44synthetic", true, "escape assembled across layers"},
		{"view=full; foo=bar&password=synthetic; lang=en", true, "multiple assignments"},
		{"password ** = synthetic", true, "prose wrapper between key and operator"},
		{"https://example.com/token/#:~:text=foo", false, "reference separators do not connect name to fragment operator"},
		{"https://example.com/token?=synthetic", false, "reference separator ends key/operator association"},
		{"%252525252525252525252525252574oken=synthetic", true, "normalization budget exhaustion"},
		{"The password policy protects users.", false, "ordinary prose"},
		{"The token identifies a request.", false, "name without operator"},
		{"Authentication tokens expire after 15 minutes.", false, "ordinary security discussion"},
		{"password", false, "name alone"},
		{"See https://example.com/password-policy", false, "ordinary URL path"},
		{"https://example.com/docs#token-authentication", false, "ordinary URL fragment"},
		{"https://example.com/docs?lang=en&next=/orders", false, "non-sensitive query"},
		{"https://example.com!password", false, "authority name without operator"},
		{"https://example.com=password", false, "name is assignment value"},
		{"notpassword=synthetic", false, "whole name comparison"},
		{"my-token=synthetic", false, "normalization does not match suffixes"},
		{"20% complete, 100% useful", false, "literal percent signs"},
		{"https://user:synthetic@example.com/docs", false, "userinfo passwords belong to URL validation"},
		{"file:/private/document", false, "file references belong to structural validation"},
		{"/home/user/private", false, "paths belong to structural validation"},
	} {
		t.Run(tc.reason, func(t *testing.T) {
			if got := ContainsSecretBearingValue(tc.input); got != tc.want {
				t.Fatalf("ContainsSecretBearingValue(%q) = %v, want %v: %s", tc.input, got, tc.want, tc.reason)
			}
		})
	}
}

func TestSecretPolicyAcrossReferenceWrappers(t *testing.T) {
	// One policy covers names across components; URL grammar does not choose
	// the scanner's input. Include every family in the existing taxonomy.
	names := []string{"token", "access_token", "refresh-token", "id_token", "auth_token", "oauth_token", "password", "passwd", "pwd", "api_key", "key", "secret", "client_secret", "signature", "sig", "credential", "authorization", "auth", "x-amz-signature", "x-amz-credential", "x-amz-security-token", "x-goog-signature", "x-goog-credential"}
	wrappers := []string{"%s", "See %s", "https://example.com!%s", "https://example.com/%s", "https://example.com?%s", "https://example.com#%s", "[docs](https://example.com/%s)", "\"https://example.com/%s\"", "ssh://%s@example.com/repo"}
	for _, name := range names {
		for _, wrapper := range wrappers {
			input := strings.ReplaceAll(wrapper, "%s", name+"=synthetic")
			if !ContainsSecretBearingValue(input) || SafeProse(input) {
				t.Fatalf("secret policy/composition accepted %q", input)
			}
		}
	}
}

func TestProsePolicyComposition(t *testing.T) {
	for _, input := range []string{"https://example.com/token=synthetic", "https://example.com#password=synthetic", "https://example.com!password=synthetic"} {
		if !safeProseStructure(input) || !ContainsSecretBearingValue(input) || SafeProse(input) {
			t.Fatalf("structural and secret responsibilities mixed for %q", input)
		}
	}
	for _, input := range []string{"/home/user/private", `C:\Users\user\private`, "file:/private/document", "https://user:synthetic@example.com/docs", "text\x00more", "\xff"} {
		if safeProseStructure(input) || SafeProse(input) {
			t.Fatalf("structural validation accepted %q", input)
		}
	}
}

func TestHistoricalScalarURLPolicy(t *testing.T) {
	for _, input := range []string{"https://example.com/token=synthetic", "https://example.com#password=synthetic", "https://example.com!password=synthetic", "ssh://git@example.com/repo"} {
		if !SafeValue(input, "remote") {
			t.Fatalf("historical scalar URL changed for %q", input)
		}
	}
	for _, input := range []string{"https://user:synthetic@example.com/docs", "https://example.com?token=synthetic", "file:/private/document"} {
		if SafeValue(input, "remote") {
			t.Fatalf("historical unsafe scalar URL accepted %q", input)
		}
	}
}
