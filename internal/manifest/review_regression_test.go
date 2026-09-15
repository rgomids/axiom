package manifest

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/project"
	"go.yaml.in/yaml/v3"
)

const reviewMinimal = "schemaVersion: 1\nproject: {id: 12345678-1234-4abc-8def-123456789abc, slug: sample, name: Sample}\n"
const reviewSentinel = "SYNTHETIC_DIAGNOSTIC_SENTINEL"

// Deliberately bypass adapter security only in test setup: these are structurally
// valid DTOs accepted by the unchanged domain. Both public codec paths must guard them.
func reviewProject(t *testing.T, source string) project.Project {
	t.Helper()
	var dto manifestDTO
	if err := yaml.Unmarshal([]byte(source), &dto); err != nil {
		t.Fatal(err)
	}
	p, issues := project.New(toDomain(dto))
	if len(issues) != 0 {
		t.Fatalf("regression fixture violates domain: %v", issues)
	}
	return p
}

func checkReviewValue(t *testing.T, template, value string, allowed bool) {
	t.Helper()
	source := reviewMinimal + strings.ReplaceAll(template, "VALUE", strconv.Quote(value))
	p := reviewProject(t, source)
	t.Run("decode", func(t *testing.T) {
		input := []byte(source)
		q, issues := Decode(input)
		if !bytes.Equal(input, []byte(source)) {
			t.Fatal("input changed")
		}
		if allowed {
			if len(issues) != 0 || !p.Equivalent(q) {
				t.Fatalf("legitimate value rejected or changed: %v", issues)
			}
			return
		}
		checkReviewRejection(t, issues)
		if q.Equivalent(q) {
			t.Fatal("rejection returned usable Project")
		}
	})
	t.Run("encode", func(t *testing.T) {
		output, issues := Encode(p)
		if allowed {
			q, decodingIssues := Decode(output)
			if len(issues) != 0 || len(decodingIssues) != 0 || !p.Equivalent(q) || !bytes.HasSuffix(output, []byte("\n")) {
				t.Fatalf("legitimate value lost on round trip: %v / %v", issues, decodingIssues)
			}
			return
		}
		checkReviewRejection(t, issues)
		if output != nil {
			t.Fatal("rejection returned partial output")
		}
	})
}

func checkReviewRejection(t *testing.T, issues []project.Issue) {
	t.Helper()
	if len(issues) != 1 || issues[0].Code != "forbidden_value" || strings.Contains(fmt.Sprint(issues), reviewSentinel) {
		t.Fatalf("missing sanitized structural rejection: %v", issues)
	}
}

func TestReviewLogicalPaths(t *testing.T) {
	const template = "credentialReferences: [{key: aws, sourceHint: VALUE}]"
	for _, value := range []string{
		"../../.aws/credentials", "./credentials", "../secret", ".", "..", "store/../secret", "store/./entry",
		"file:/Users/example/.aws/credentials", "FILE:relative", "file:///synthetic", "file:../secret",
		"/synthetic", "~/credentials", "C:/credentials", "C:credentials", `..\secret`, `\\host\share`,
		"keychain:../secret", "store:/local", "store:~/local", "store:file:/local",
		"%2e%2e%2fsecret", ".%2e/secret", "%2Fsynthetic", "%7e/secret", "file%3A/synthetic", "C%3Asecret",
		"store/%2e%2e/secret", "%2e%2e%5csecret", "env:%24%7BSYNTHETIC_ENV%7D",
		"%252e%252e%252fsecret", "file%253A/synthetic", "store:%252e%252e/secret", "env:%GG",
		"%252525252525252525252e%252525252525252525252e%252525252525252525252fsecret",
	} {
		t.Run(value, func(t *testing.T) { checkReviewValue(t, template, value, false) })
	}
	for _, value := range []string{
		"AWS_PROFILE", "AWS_ACCESS_KEY_ID", "environment", "env:AWS_PROFILE", "keychain:com.example.aws",
		"secret-service:team/aws", "libsecret:project/aws", "windows-credential-manager:example",
		"runtime-managed:aws", "vault:team/aws", "team/aws", "arn:aws:secretsmanager:region:account:secret:team/aws",
		"token", "password", "unconfigured", "com.example.credential", "store:entry..name", "store:Team%2FEntry",
	} {
		t.Run(value, func(t *testing.T) { checkReviewValue(t, template, value, true) })
	}
}

var reviewIdentifierFields = map[string]string{
	"runtime.id":                "runtime: {id: VALUE}",
	"repositories.key":          "repositories: [{key: VALUE}]",
	"providers.key":             "providers: [{key: VALUE, id: provider}]",
	"providers.id":              "providers: [{key: p, id: VALUE}]",
	"integrations.key":          "integrations: [{key: VALUE}]",
	"integrations.capabilities": "integrations: [{key: i, capabilities: [VALUE]}]",
	"transport.id":              "providers: [{key: p, id: provider}]\nintegrations: [{key: i, providerRef: p, transport: {id: VALUE}}]",
	"modelProfiles.key":         "modelProfiles: [{key: VALUE, state: unconfigured}]",
	"modelProfiles.model":       "runtime: {id: r}\nmodelProfiles: [{key: m, runtimeRef: r, model: VALUE}]",
}

func TestReviewIdentifierPayloads(t *testing.T) {
	for field, template := range reviewIdentifierFields {
		t.Run(field, func(t *testing.T) {
			for _, family := range []string{"token", "access_token", "refresh-token", "idToken", "authToken", "oauthToken", "password", "passwd", "pwd", "apikey", "API_KEY", "secret", "client-secret", "credential", "authorization", "auth", "key", "signature", "sig", "X-Amz-Signature", "X-Amz-Credential", "X-Amz-Security-Token", "X-Goog-Signature", "X-Goog-Credential"} {
				for _, delimiter := range []string{":", "="} {
					t.Run(family+delimiter, func(t *testing.T) { checkReviewValue(t, template, family+delimiter+reviewSentinel, false) })
				}
			}
			for _, value := range []string{"token", "password", "auth", "Opaque/Model", "vendor:model-v1", "v:model", "tokenizer:opaque", "edition=enterprise", "store:password-entry", "Opaque%2FModel"} {
				t.Run(value, func(t *testing.T) { checkReviewValue(t, template, value, true) })
			}
		})
	}
}

func TestReviewLogicalFieldVariants(t *testing.T) {
	for field, template := range map[string]string{
		"credential.key":            "credentialReferences: [{key: VALUE}]",
		"credential.sourceHint":     "credentialReferences: [{key: c, sourceHint: VALUE}]",
		"integration.providerRef":   "providers: [{key: VALUE, id: provider}]\nintegrations: [{key: i, providerRef: VALUE}]",
		"integration.credentialRef": "credentialReferences: [{key: VALUE}]\nintegrations: [{key: i, credentialRef: VALUE}]",
		"transport.reference":       "providers: [{key: p, id: provider}]\nintegrations: [{key: i, providerRef: p, transport: {id: t, reference: VALUE}}]",
		"profile.runtimeRef":        "runtime: {id: VALUE}\nmodelProfiles: [{key: m, runtimeRef: VALUE, model: model}]",
	} {
		t.Run(field, func(t *testing.T) {
			for _, value := range []string{"../secret", "file:/synthetic", "password:" + reviewSentinel, "auth=" + reviewSentinel} {
				t.Run(value, func(t *testing.T) { checkReviewValue(t, template, value, false) })
			}
			checkReviewValue(t, template, "logical:entry", true)
		})
	}
}

func TestReviewEscapedSyntaxAndFreeText(t *testing.T) {
	for _, value := range []string{"%70assword:" + reviewSentinel, "token%3D" + reviewSentinel, "%2570assword%253A" + reviewSentinel, " password :" + reviewSentinel, "file:/synthetic"} {
		t.Run(value, func(t *testing.T) { checkReviewValue(t, "runtime: {id: VALUE}", value, false) })
	}
	// YAML escapes are resolved before structural validation.
	for _, extra := range []string{`credentialReferences: [{key: c, sourceHint: "\u002e\x2e\x2fsecret"}]`, `runtime: {id: "pass\u0077ord\x3aSYNTHETIC_DIAGNOSTIC_SENTINEL"}`} {
		_, issues := Decode([]byte(reviewMinimal + extra))
		checkReviewRejection(t, issues)
	}
	for _, value := range []string{"password:" + reviewSentinel, "token=" + reviewSentinel, "../../.aws/credentials", "file:/synthetic", "${SYNTHETIC_ENV}"} {
		t.Run("text/"+value, func(t *testing.T) { checkReviewValue(t, "businessContext: {text: VALUE}", value, true) })
		p := reviewProject(t, strings.Replace(reviewMinimal, "name: Sample", "name: "+strconv.Quote(value), 1))
		if _, issues := Encode(p); len(issues) != 0 {
			t.Fatal("human name treated as reference")
		}
	}
}

func TestReviewLocalURIRemotes(t *testing.T) {
	for _, value := range []string{"file:/synthetic", "FILE:relative", "file:../synthetic", "file://machine/synthetic"} {
		t.Run(value, func(t *testing.T) { checkReviewValue(t, "repositories: [{key: r, remote: VALUE}]", value, false) })
	}
	for _, value := range []string{"git@example.com:Team/Repo", "ssh://git@example.com/Repo", "https://example.com/Repo?ref=Main%2FTree", "https://example.com/Repo?ref=Main%252FTree"} {
		t.Run(value, func(t *testing.T) { checkReviewValue(t, "repositories: [{key: r, remote: VALUE}]", value, true) })
	}
}
