package manifest_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/rgomids/axiom/internal/project"
)

func pairwise(t *testing.T, variants []string) {
	t.Helper()
	projects := make([]project.Project, len(variants))
	canonical := make([][]byte, len(variants))
	for i, variant := range variants {
		projects[i] = decode(t, minimal+variant)
		canonical[i] = encode(t, projects[i])
		if !projects[i].Equivalent(decode(t, string(canonical[i]))) {
			t.Fatal("presence lost on round trip")
		}
	}
	for i, p := range projects {
		for j, q := range projects {
			if p.Equivalent(q) != (i == j) || bytes.Equal(canonical[i], canonical[j]) != (i == j) {
				t.Fatal("distinct intent collapsed")
			}
		}
	}
}
func TestDeclarationStatesPairwise(t *testing.T) {
	pairwise(t, []string{"", "repositories: []", "repositories: [{key: local}]"})
	for _, field := range []string{"providers", "integrations", "modelProfiles", "credentialReferences", "policies"} {
		t.Run(field, func(t *testing.T) { pairwise(t, []string{"", field + ": unconfigured", field + ": []"}) })
	}
	pairwise(t, []string{"", "runtime: unconfigured", "runtime: {id: unconfigured}"})
	pairwise(t, []string{"", "businessContext: unconfigured", "businessContext: {}", "businessContext: {text: ''}", "businessContext: {documents: []}", "businessContext: {text: '', documents: []}", "businessContext: {text: unconfigured}"})
}
func TestNestedPresencePairwise(t *testing.T) {
	pairwise(t, []string{"repositories: [{key: r}]", "repositories: [{key: r, remote: 'https://example.com/Repo'}]"})
	pairwise(t, []string{"integrations: [{key: i}]", "integrations: [{key: i, capabilities: []}]", "integrations: [{key: i, capabilities: [unconfigured]}]"})
	pairwise(t, []string{"credentialReferences: [{key: c}]", "credentialReferences: [{key: c, sourceHint: environment}]", "credentialReferences: [{key: c, sourceHint: unconfigured}]"})
	common := "providers: [{key: p, id: p}]\ncredentialReferences: [{key: c}]\n"
	pairwise(t, []string{common + "integrations: [{key: i}]", common + "integrations: [{key: i, providerRef: p}]", common + "integrations: [{key: i, credentialRef: c}]", common + "integrations: [{key: i, providerRef: p, transport: {id: t}}]", common + "integrations: [{key: i, providerRef: p, transport: {id: t, reference: unconfigured}}]"})
	runtime := "runtime: {id: r}\n"
	pairwise(t, []string{runtime + "modelProfiles: [{key: m, state: unconfigured}]", runtime + "modelProfiles: [{key: m, runtimeRef: r, model: unconfigured}]"})
}
func TestEquivalentStatesCanonicalBytes(t *testing.T) {
	input, err := os.ReadFile("testdata/configured.yaml")
	if err != nil {
		t.Fatal(err)
	}
	p := decode(t, string(input))
	s := p.State()
	repositories, _ := s.Repositories.Value()
	repositories[0], repositories[2] = repositories[2], repositories[0]
	repositories[2].Remote = project.Configured("HTTPS://EXAMPLE.COM/Org/API.git")
	providers, _ := s.Providers.Value()
	providers[0], providers[1] = providers[1], providers[0]
	integrations, _ := s.Integrations.Value()
	integrations[0], integrations[1] = integrations[1], integrations[0]
	profiles, _ := s.ModelProfiles.Value()
	profiles[0], profiles[1] = profiles[1], profiles[0]
	credentials, _ := s.CredentialReferences.Value()
	credentials[0], credentials[1] = credentials[1], credentials[0]
	q, issues := project.New(s)
	if len(issues) != 0 || !p.Equivalent(q) || !bytes.Equal(encode(t, p), encode(t, q)) {
		t.Fatal("equivalent state changed canonical bytes")
	}
	a := decode(t, minimal+"providers: []").State()
	a.Providers = project.Configured[[]project.Provider](nil)
	r, issues := project.New(a)
	if len(issues) != 0 || !bytes.Equal(encode(t, r), encode(t, decode(t, minimal+"providers: []"))) {
		t.Fatal("nil/empty distinction invented")
	}
}
func TestOrderedListsAndOpaqueContent(t *testing.T) {
	for _, variants := range [][]string{
		{"policies: [a.md, b.md]", "policies: [b.md, a.md]"},
		{"businessContext: {documents: [a.md, b.md]}", "businessContext: {documents: [b.md, a.md]}"},
		{"integrations: [{key: i, capabilities: [a, b]}]", "integrations: [{key: i, capabilities: [b, a]}]"},
		{"repositories: [{key: r, remote: 'https://example.com/Repo'}]", "repositories: [{key: r, remote: 'https://example.com/repo'}]", "repositories: [{key: r, remote: 'git@example.com:Repo'}]"},
		{"providers: [{key: p, id: Opaque/Provider}]", "providers: [{key: p, id: opaque/provider}]"},
	} {
		pairwise(t, variants)
	}
	for _, text := range []string{`"1"`, `"true"`, `"null"`, `"2026-09-15"`, `"unconfigured"`, `""`, `"Unicode: ação 世界 😀"`, `"line\nline"`, `"a\x00b"`} {
		p := decode(t, minimal+"businessContext: {text: "+text+"}")
		if !p.Equivalent(decode(t, string(encode(t, p)))) {
			t.Fatal("text changed")
		}
	}
}
