package manifest_test

import (
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
)

func TestClosedMappingsAtEveryShape(t *testing.T) {
	for name, template := range map[string]string{
		"project":     "project: {id: 12345678-1234-4abc-8def-123456789abc, slug: sample, name: Sample%s}",
		"repository":  "repositories: [{key: r%s}]",
		"runtime":     "runtime: {id: r%s}",
		"provider":    "providers: [{key: p, id: p%s}]",
		"integration": "integrations: [{key: i%s}]",
		"transport":   "providers: [{key: p, id: p}]\nintegrations: [{key: i, providerRef: p, transport: {id: t%s}}]",
		"profile":     "modelProfiles: [{key: m, state: unconfigured%s}]",
		"context":     "businessContext: {text: data%s}",
		"credential":  "credentialReferences: [{key: c%s}]",
	} {
		t.Run(name, func(t *testing.T) {
			prefix := minimal
			if name == "project" {
				prefix = "schemaVersion: 1\n"
			}
			valid := strings.Replace(template, "%s", "", 1)
			decode(t, prefix+valid)
			for _, extra := range []string{", " + sentinel + ": data", ", 3: data", ", '<<': {}"} {
				reject(t, prefix+strings.Replace(template, "%s", extra, 1))
			}
			// Duplicate an existing safe key, including quoted spelling.
			key := "key"
			if name == "project" {
				key = "name"
			}
			if name == "runtime" || name == "transport" {
				key = "id"
			}
			if name == "context" {
				key = "text"
			}
			reject(t, prefix+strings.Replace(template, "%s", ", '"+key+"': "+sentinel, 1))
		})
	}
}
func TestStrictScalarTypesAndShapes(t *testing.T) {
	for _, template := range []string{
		"runtime: {id: VALUE}", "providers: [{key: p, id: VALUE}]",
		"repositories: [{key: VALUE}]", "repositories: [{key: r, remote: VALUE}]",
		"integrations: [{key: i, capabilities: [VALUE]}]",
		"credentialReferences: [{key: c, sourceHint: VALUE}]",
		"businessContext: {text: VALUE}", "businessContext: {documents: [VALUE]}",
		"policies: [VALUE]",
	} {
		for _, value := range []string{"null", "true", "1", "1.5", "2026-09-15", "{}", "[]"} {
			reject(t, minimal+strings.Replace(template, "VALUE", value, 1))
		}
	}
	// YAML 1.2 string values remain strings, never bool coercions.
	for _, value := range []string{"yes", "on", "off", "no", `"1"`, `!!str 1`} {
		decode(t, minimal+"runtime: {id: "+value+"}")
	}
	for _, extra := range []string{
		"repositories: unconfigured", "runtime: []", "runtime: {}", "providers: {}", "providers: [null]", "providers: [{}]",
		"integrations: [{key: i, state: unconfigured}]", "integrations: [{key: i, capabilities: unconfigured}]",
		"integrations: [{key: i, transport: unconfigured}]", "integrations: [{key: i, transport: {}}]",
		"businessContext: []", "businessContext: {documents: unconfigured}", "credentialReferences: [unconfigured]",
		"modelProfiles: [{key: m}]", "modelProfiles: [{key: m, state: configured}]",
		"modelProfiles: [{key: m, state: unconfigured, model: m}]", "modelProfiles: [{key: m, state: null}]",
	} {
		reject(t, minimal+extra)
	}
	for _, field := range []string{"repositories", "runtime", "providers", "integrations", "modelProfiles", "businessContext", "credentialReferences", "policies"} {
		reject(t, minimal+field+": null")
	}
}
func TestDomainValidationAndNormalization(t *testing.T) {
	for _, source := range []string{
		strings.Replace(minimal, "4abc", "1abc", 1), strings.Replace(minimal, "8def", "cdef", 1),
		strings.Replace(minimal, "slug: sample", "slug: ../sample", 1), strings.Replace(minimal, "name: Sample", "name: ''", 1),
		minimal + "repositories: [{key: r}, {key: r}]",
		minimal + "repositories: [{key: a, remote: 'https://EXAMPLE.com/Repo'}, {key: b, remote: 'https://example.com/Repo'}]",
		minimal + "integrations: [{key: i, providerRef: missing}]", minimal + "integrations: [{key: i, credentialRef: missing}]",
		minimal + "integrations: [{key: i, transport: {id: t}}]",
		minimal + "runtime: {id: r}\nmodelProfiles: [{key: m, runtimeRef: other, model: m}]",
		minimal + "policies: ['../outside']", minimal + "businessContext: {documents: ['/machine/path']}",
	} {
		reject(t, source)
	}
	for _, locator := range []string{"HTTPS://EXAMPLE.COM:443/Team/%52epo.git/", "git@EXAMPLE.COM:Team/Repo.git", "ssh://git@EXAMPLE.COM:22/Team/Repo", "https://[2001:DB8::1]/Repo"} {
		p := decode(t, minimal+"repositories: [{key: r, remote: '"+locator+"'}]")
		repos, _ := p.State().Repositories.Value()
		remote, _ := repos[0].Remote.Value()
		want, _ := project.NormalizeLocator(locator)
		if remote != want {
			t.Fatal("T01 normalization changed")
		}
		if !p.Equivalent(decode(t, string(encode(t, p)))) {
			t.Fatal("locator round trip changed")
		}
	}
}
func TestByteDepthAndNodeLimits(t *testing.T) {
	at := minimal + "#" + strings.Repeat("x", manifest.MaxBytes-len(minimal)-1)
	decode(t, at)
	for _, tc := range []struct{ source, code string }{
		{at + "x", "byte_limit"},
		{strings.Repeat("[", manifest.MaxDepth) + "x" + strings.Repeat("]", manifest.MaxDepth), "depth_limit"},
		{"[" + strings.Repeat("x,", manifest.MaxNodes) + "]", "node_limit"},
	} {
		_, issues := manifest.Decode([]byte(tc.source))
		if len(issues) != 1 || issues[0].Code != tc.code {
			t.Fatalf("expected %s: %v", tc.code, issues)
		}
		reject(t, tc.source)
	}
	// Valid schema reaches exactly MaxNodes: base is 11 nodes, policies adds 2.
	valid := minimal + "policies: [" + strings.Repeat("a,", manifest.MaxNodes-13) + "]"
	decode(t, valid)
	reject(t, strings.Replace(valid, "]", "a,]", 1))
	reject(t, strings.Repeat("[", 10001)+"x"+strings.Repeat("]", 10001))
}
func TestEncoderRejectsInvalidUnsafeOrOversizedState(t *testing.T) {
	if b, issues := manifest.Encode(project.Project{}); b != nil || len(issues) == 0 {
		t.Fatal("zero Project encoded")
	}
	for _, name := range []string{strings.Repeat("x", manifest.MaxBytes), string([]byte{0xff})} {
		state := decode(t, minimal).State()
		state.Name = name
		p, issues := project.New(state)
		if len(issues) != 0 {
			t.Fatal(issues)
		}
		if b, issues := manifest.Encode(p); b != nil || len(issues) == 0 {
			t.Fatal("unsafe encoding returned bytes")
		}
	}
}

func TestRequiredFieldsAndNestedScalarTypes(t *testing.T) {
	reject(t, "schemaVersion: 1\n")
	for _, member := range []string{"id: 12345678-1234-4abc-8def-123456789abc, ", "slug: sample, ", ", name: Sample"} {
		reject(t, strings.Replace(minimal, member, "", 1))
	}
	for _, member := range []string{"id: 12345678-1234-4abc-8def-123456789abc", "slug: sample", "name: Sample"} {
		key := strings.Split(member, ":")[0]
		for _, value := range []string{"null", "1", "true", "[]", "{}"} {
			reject(t, strings.Replace(minimal, member, key+": "+value, 1))
		}
	}
	for _, source := range []string{
		"providers: [{id: p}]", "providers: [{key: p}]", "integrations: [{}]", "credentialReferences: [{}]",
		"providers: [{key: p, id: p}]\nintegrations: [{key: i, providerRef: true}]",
		"credentialReferences: [{key: c}]\nintegrations: [{key: i, credentialRef: true}]",
		"providers: [{key: p, id: p}]\nintegrations: [{key: i, providerRef: p, transport: {id: true}}]",
		"providers: [{key: p, id: p}]\nintegrations: [{key: i, providerRef: p, transport: {id: t, reference: true}}]",
		"runtime: {id: r}\nmodelProfiles: [{key: m, runtimeRef: true, model: m}]",
		"runtime: {id: r}\nmodelProfiles: [{key: m, runtimeRef: r, model: true}]",
	} {
		reject(t, minimal+source)
	}
}

func BenchmarkHostileBounds(b *testing.B) {
	for name, input := range map[string][]byte{
		"byte":  []byte(strings.Repeat("x", manifest.MaxBytes+1)),
		"deep":  []byte(strings.Repeat("[", 10001) + "x" + strings.Repeat("]", 10001)),
		"dense": []byte("[" + strings.Repeat("x,", (manifest.MaxBytes-2)/2) + "]"),
	} {
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				_, issues := manifest.Decode(input)
				if len(issues) == 0 {
					b.Fatal("hostile input accepted")
				}
			}
		})
	}
}
