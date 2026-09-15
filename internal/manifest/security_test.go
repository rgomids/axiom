package manifest_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

func TestStructuralSecretAndLocalStateExclusion(t *testing.T) {
	for _, extra := range []string{
		"secrets: {value: " + sentinel + "}", "formatVersion: 1", "installation: {path: /synthetic}",
		"repositories: [{key: r, path: /synthetic}]", "runtime: {id: r, executable: /synthetic}",
		"runtime: {id: r, pid: 123}", "runtime: {id: r, availability: available}",
		"credentialReferences: [{key: c, value: " + sentinel + "}]",
		"credentialReferences: [{key: c, password: " + sentinel + "}]",
		"credentialReferences: [{key: c, sourceHint: {value: " + sentinel + "}}]",
		"integrations: [{key: i, credentialRef: {token: " + sentinel + "}}]",
		"credentialReferences: [{key: 'token=" + sentinel + "'}]",
		"credentialReferences: [{key: c, sourceHint: 'token=" + sentinel + "'}]",
		"credentialReferences: [{key: c, sourceHint: 'password:" + sentinel + "'}]",
		"repositories: [{key: r, remote: 'file://machine/synthetic/Repo'}]",
		"credentialReferences: [{key: c, sourceHint: '/synthetic/local'}]",
		"credentialReferences: [{key: c, sourceHint: 'C:\\synthetic\\local'}]",
		"credentialReferences: [{key: c, sourceHint: '${SYNTHETIC_ENV}'}]",
		"credentialReferences: [{key: c, sourceHint: 'https://example.com/store'}]",
		"runtime: {id: /synthetic/executable}", "runtime: {id: 'C:/synthetic/executable'}",
		"providers: [{key: p, id: '~/synthetic'}]",
		"policies: ['${SYNTHETIC_ENV}/policy.md']",
	} {
		reject(t, minimal+extra)
	}
	// Reference keys with secret-related names are ordinary identifiers. No value
	// scanner or ambient source lookup exists; free text remains untrusted data.
	decode(t, minimal+"credentialReferences: [{key: token, sourceHint: environment}]\nbusinessContext: {text: '"+sentinel+" ${SYNTHETIC_ENV}'}")
}
func TestURLSensitiveParameterPolicyV1(t *testing.T) {
	names := []string{"token", "access_token", "refresh-token", "id_token", "auth_token", "oauth_token", "password", "passwd", "pwd", "api-key", "key", "secret", "client_secret", "signature", "sig", "credential", "authorization", "auth", "X-Amz-Signature", "X-Amz-Credential", "X-Amz-Security-Token", "X-Goog-Signature", "X-Goog-Credential", "%74oken", "API_KEY"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			source := minimal + "repositories: [{key: r, remote: 'https://example.com/Repo?" + name + "=" + sentinel + "'}]"
			reject(t, source)
			state := decode(t, minimal).State()
			state.Repositories = project.Configured([]project.Repository{{Key: "r", Remote: project.Configured("https://example.com/Repo?" + name + "=" + sentinel)}})
			p, issues := project.New(state)
			if len(issues) != 0 {
				t.Fatal(issues)
			}
			b, encodingIssues := manifest.Encode(p)
			if b != nil || len(encodingIssues) == 0 || strings.Contains(fmt.Sprint(encodingIssues), sentinel) {
				t.Fatal("encoder security bypass")
			}
		})
	}
	for _, url := range []string{
		"https://user:" + sentinel + "@example.com/Repo", "https://" + sentinel + "@example.com/Repo",
		"ssh://git:" + sentinel + "@example.com/Repo", "https://example.com/Repo?bad=%GG",
		"https://example.com/Repo?ok=one;token=" + sentinel,
	} {
		reject(t, minimal+"repositories: [{key: r, remote: '"+url+"'}]")
	}
	for _, url := range []string{"ssh://git@example.com/Repo", "git@example.com:Repo", "https://example.com/Repo?ref=Main%2fTree#ReadMe", "https://example.com/Repo?tokenizer=synthetic"} {
		decode(t, minimal+"repositories: [{key: r, remote: '"+url+"'}]")
	}
}
func TestT02PortAndReadSnapshotFailureHasNoEffects(t *testing.T) {
	var codec projectapp.ManifestCodec = manifest.Codec{}
	input := []byte(minimal)
	original := bytes.Clone(input)
	p, issues := codec.Decode(input)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	b, issues := codec.Encode(p)
	if len(issues) != 0 || len(b) == 0 {
		t.Fatal(issues)
	}
	snapshot, issues := projectapp.ReadSnapshot(codec, input, nil)
	if len(issues) != 0 || !snapshot.Project().Equivalent(p) {
		t.Fatal("snapshot integration failed")
	}
	if !bytes.Equal(input, original) {
		t.Fatal("source bytes rewritten")
	}
	for _, bad := range []string{minimal + "secret: " + sentinel, minimal + "runtime: *" + sentinel} {
		source := []byte(bad)
		before := bytes.Clone(source)
		snapshot, issues := projectapp.ReadSnapshot(codec, source, nil)
		if len(issues) == 0 || snapshot.Project().Equivalent(snapshot.Project()) {
			t.Fatal("failure escaped as usable snapshot")
		}
		if !bytes.Equal(source, before) || strings.Contains(fmt.Sprint(issues), sentinel) {
			t.Fatal("failure mutated source or leaked")
		}
		if issues[0].Category() != "validation" || issues[0].Remedy() != "correct_input" {
			t.Fatal("port classification changed")
		}
	}
	if b, issues := codec.Encode(project.Project{}); b != nil || len(issues) == 0 {
		t.Fatal("invalid project bypassed port")
	}
	// No writer, observer, clock, entropy, process or credential-read seam is
	// accepted by Codec. Boundary tests additionally inspect production imports.
}
func FuzzDecodeSafeRoundTrip(f *testing.F) {
	for _, s := range []string{minimal, "", minimal + "runtime: *missing", minimal + "runtime: &a [*a]", minimal + "businessContext: {text: '世界'}"} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		before := bytes.Clone(input)
		p, issues := manifest.Decode(input)
		if !bytes.Equal(input, before) {
			t.Fatal("input changed")
		}
		if len(issues) != 0 {
			if p.Equivalent(p) {
				t.Fatal("invalid project returned")
			}
			for _, issue := range issues {
				if len(issue.Field) > 256 || len(issue.Code) > 64 || strings.Contains(issue.Field+issue.Code, sentinel) {
					t.Fatal("unsafe diagnostic")
				}
			}
			return
		}
		b, issues := manifest.Encode(p)
		// Canonical quoting/indentation can exceed the byte bound even when compact
		// source fits. Controlled size rejection is allowed, never partial output.
		if len(issues) != 0 {
			if b != nil || issues[0].Code != "byte_limit" {
				t.Fatalf("valid decode failed encoding: %v", issues)
			}
			return
		}
		q, issues := manifest.Decode(b)
		if len(issues) != 0 || !p.Equivalent(q) {
			t.Fatal("semantic round trip failed")
		}
		c, issues := manifest.Encode(q)
		if len(issues) != 0 || !bytes.Equal(b, c) {
			t.Fatal("canonical output changed")
		}
	})
}
