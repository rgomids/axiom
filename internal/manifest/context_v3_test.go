package manifest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
)

var minimalV3 = strings.Replace(minimal, "schemaVersion: 1", "schemaVersion: 3", 1)

const repositoryV3 = "repositories: [{key: core}]\n"

func TestV3VersionBoundary(t *testing.T) {
	decode(t, minimalV3)
	decode(t, minimalV3+policyV2)
	reject(t, minimalV3+"runtime: {id: codex}")
	for _, field := range []string{"technologyContext: []", "documentationSources: []", "businessContext: {sourceRefs: []}", "businessContext: {glossary: []}"} {
		reject(t, minimal+field)
		reject(t, minimalV2+field)
		decode(t, minimalV3+field)
	}
	for _, version := range []string{`"3"`, "3.0", "03", "+3", "0x3"} {
		reject(t, strings.Replace(minimalV3, "schemaVersion: 3", "schemaVersion: "+version, 1))
	}
}

func TestV3RejectsMalformedContext(t *testing.T) {
	for name, source := range map[string]string{
		"technology null":        "technologyContext: null",
		"technology mapping":     "technologyContext: {key: a, value: b}",
		"technology extra":       "technologyContext: [{key: a, value: b, source: detected}]",
		"technology missing":     "technologyContext: [{key: a}]",
		"technology bad key":     "technologyContext: [{key: Language, value: go}]",
		"technology duplicate":   "technologyContext: [{key: a, value: b}, {key: a, value: c}]",
		"technology path":        "technologyContext: [{key: a, value: /usr/local/bin/go}]",
		"technology secret":      "technologyContext: [{key: a, value: 'token=synthetic'}]",
		"technology file":        "technologyContext: [{key: a, value: 'file:synthetic'}]",
		"technology type":        "technologyContext: [{key: a, value: true}]",
		"source unknown kind":    "documentationSources: [{key: d, kind: notion}]",
		"source extra":           "documentationSources: [{key: d, kind: local-file, location: /tmp/x}]",
		"source local path":      "documentationSources: [{key: d, kind: local-file, path: notes.md}]",
		"source absolute":        repositoryV3 + "documentationSources: [{key: d, kind: repository, repositoryRef: core, path: /etc/passwd}]",
		"source traversal":       repositoryV3 + "documentationSources: [{key: d, kind: repository, repositoryRef: core, path: ../x}]",
		"source dangling repo":   "documentationSources: [{key: d, kind: repository, repositoryRef: core, path: docs}]",
		"source missing path":    repositoryV3 + "documentationSources: [{key: d, kind: repository, repositoryRef: core}]",
		"source secret path":     repositoryV3 + "documentationSources: [{key: d, kind: repository, repositoryRef: core, path: 'token=x'}]",
		"sourceRef dangling":     "businessContext: {sourceRefs: [missing]}",
		"sourceRef unconfigured": "businessContext: {sourceRefs: unconfigured}",
		"glossary extra":         "businessContext: {glossary: [{key: k, term: T, definition: D, source: x}]}",
		"glossary missing":       "businessContext: {glossary: [{key: k, term: T}]}",
		"glossary blank":         "businessContext: {glossary: [{key: k, term: ' ', definition: D}]}",
		"glossary unconfigured":  "businessContext: {glossary: unconfigured}",
	} {
		t.Run(name, func(t *testing.T) { reject(t, minimalV3+source) })
	}
}

func TestV3CollectionBounds(t *testing.T) {
	for name, limit := range map[string]int{"technologyContext": 64, "documentationSources": 64, "glossary": 128} {
		for _, size := range []int{limit, limit + 1} {
			items := make([]string, size)
			for i := range items {
				switch name {
				case "technologyContext":
					items[i] = fmt.Sprintf("{key: k%d, value: v}", i)
				case "documentationSources":
					items[i] = fmt.Sprintf("{key: k%d, kind: local-file}", i)
				default:
					items[i] = fmt.Sprintf("{key: k%d, term: T, definition: D}", i)
				}
			}
			source := minimalV3 + name + ": [" + strings.Join(items, ", ") + "]"
			if name == "glossary" {
				source = minimalV3 + "businessContext: {glossary: [" + strings.Join(items, ", ") + "]}"
			}
			if size == limit {
				decode(t, source)
				continue
			}
			_, issues := manifest.Decode([]byte(source))
			if len(issues) == 0 || issues[0].Code != "collection_limit" {
				t.Fatalf("%s %d: %v", name, size, issues)
			}
		}
	}
}

func TestV3DeclarationStatesRoundTrip(t *testing.T) {
	for _, technology := range []string{"", "technologyContext: unconfigured\n", "technologyContext: []\n"} {
		for _, sources := range []string{"", "documentationSources: unconfigured\n", "documentationSources: []\n"} {
			for _, context := range []string{"", "businessContext: {}\n", "businessContext: {sourceRefs: [], glossary: []}\n"} {
				p := decode(t, minimalV3+technology+sources+context)
				if !p.Equivalent(decode(t, string(encode(t, p)))) {
					t.Fatal("v3 declaration form lost")
				}
			}
		}
	}
}

func TestV3EncoderRejectsUnsafeDomainContext(t *testing.T) {
	s := decode(t, minimalV3).State()
	s.TechnologyContext = project.Configured([]project.TechnologyFact{{Key: "a", Value: "/machine/path"}})
	p, issues := project.New(s)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if output, issues := manifest.Encode(p); output != nil || len(issues) == 0 {
		t.Fatal("machine path escaped encoder")
	}
}

func TestV1AndV2BytesUnchangedByV3(t *testing.T) {
	for _, source := range []string{minimal, minimalV2 + policyV2} {
		p := decode(t, source)
		if strings.Contains(string(encode(t, p)), "technologyContext") || p.State().SchemaVersion == 3 {
			t.Fatal("older version gained v3 output")
		}
	}
}
