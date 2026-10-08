package manifest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
)

var minimalV2 = strings.Replace(minimal, "schemaVersion: 1", "schemaVersion: 2", 1)

const policyV2 = "runtimes: [{id: codex}, {id: claude}]\nmodelProfiles: [{key: worker, runtimeRef: codex, model: Approved/Model}, {key: careful, runtimeRef: claude, model: Other Model}, {key: later, state: unconfigured}]\n"

func TestV2VersionBoundary(t *testing.T) {
	decode(t, minimalV2)
	for _, field := range []string{"runtimes: []", "runtimes: unconfigured", "runtimePreferences: []", "runtimePreferences: unconfigured"} {
		reject(t, minimal+field)
	}
	for _, field := range []string{"runtime: {id: codex}", "runtime: unconfigured", "runtime: null"} {
		reject(t, minimalV2+field)
	}
	for _, version := range []string{`"2"`, "2.0", "02", "+2", "0x2", "2_0"} {
		reject(t, strings.Replace(minimalV2, "schemaVersion: 2", "schemaVersion: "+version, 1))
	}
}

func TestV2RejectsMalformedPolicy(t *testing.T) {
	for name, source := range map[string]string{
		"runtime null": "runtimes: null", "runtime mapping": "runtimes: {id: codex}", "runtime string": "runtimes: [codex]", "runtime missing id": "runtimes: [{}]", "runtime extra": "runtimes: [{id: codex, executable: synthetic}]", "runtime duplicate mapping key": "runtimes: [{id: codex, id: claude}]", "runtime scalar type": "runtimes: [{id: true}]", "runtime blank": "runtimes: [{id: ' '}]", "runtime duplicate id": "runtimes: [{id: codex}, {id: codex}]",
		"profile runtime absent": "modelProfiles: [{key: p, runtimeRef: codex, model: m}]", "profile runtime undeclared": "runtimes: [{id: codex}]\nmodelProfiles: [{key: p, runtimeRef: claude, model: m}]", "profile duplicate": "runtimes: [{id: codex}]\nmodelProfiles: [{key: p, runtimeRef: codex, model: m}, {key: p, state: unconfigured}]", "profile unresolved fields": "modelProfiles: [{key: p, state: unconfigured, runtimeRef: codex}]", "profile extra": "modelProfiles: [{key: p, state: unconfigured, capabilities: []}]",
		"preference null": "runtimePreferences: null", "preference mapping": "runtimePreferences: {}", "preference scalar": "runtimePreferences: [unconfigured]", "preference missing role": policyV2 + "runtimePreferences: [{complexity: high, modelProfileRef: worker}]", "preference missing complexity": policyV2 + "runtimePreferences: [{role: review, modelProfileRef: worker}]", "preference missing ref": policyV2 + "runtimePreferences: [{role: review, complexity: high}]", "preference extra": policyV2 + "runtimePreferences: [{role: review, complexity: high, modelProfileRef: worker, stage: implementation}]", "preference type": policyV2 + "runtimePreferences: [{role: true, complexity: high, modelProfileRef: worker}]", "preference missing profile": policyV2 + "runtimePreferences: [{role: review, complexity: high, modelProfileRef: missing}]", "preference unconfigured profile": policyV2 + "runtimePreferences: [{role: review, complexity: high, modelProfileRef: later}]", "preference duplicate pair": policyV2 + "runtimePreferences: [{role: review, complexity: high, modelProfileRef: careful}, {role: review, complexity: high, modelProfileRef: worker}]", "preference blank": policyV2 + "runtimePreferences: [{role: '', complexity: high, modelProfileRef: worker}]",
	} {
		t.Run(name, func(t *testing.T) { reject(t, minimalV2+source) })
	}
}

func TestV2SecurityAndLogicalTokens(t *testing.T) {
	for _, value := range []string{"/machine/path", "~/path", "C:/path", "file:synthetic", "%252Fmachine/path", "token=synthetic", "https://user:synthetic@example.com/path", "https://example.com?token=synthetic", "line\ncontrol"} {
		// Quote through fmt rather than YAML string concatenation so newline cases
		// represent decoded values and cannot change the surrounding closed mapping.
		reject(t, minimalV2+fmt.Sprintf("runtimes: [{id: %q}]", value))
	}
	for _, value := range []string{"", "High", "*", " review", "review ", "review/role", "role:secret", "role\\name", strings.Repeat("a", 257)} {
		for _, field := range []string{"role", "complexity"} {
			pref := "runtimePreferences: [{role: review, complexity: high, modelProfileRef: worker}]"
			old := field + ": review"
			if field == "complexity" {
				old = field + ": high"
			}
			reject(t, minimalV2+policyV2+strings.Replace(pref, old, fmt.Sprintf("%s: %q", field, value), 1))
		}
	}
	for _, value := range []string{"/machine/path", "%252Fmachine/path", "credential=synthetic", "file:synthetic", "namespace/../path"} {
		reject(t, minimalV2+policyV2+fmt.Sprintf("runtimePreferences: [{role: review, complexity: high, modelProfileRef: %q}]", value))
	}
	// Role/complexity use tokens; profile references preserve opaque logical keys.
	decode(t, minimalV2+"runtimes: [{id: Opaque/Runtime}]\nmodelProfiles: [{key: Namespace/Profile, runtimeRef: Opaque/Runtime, model: Approved/Model}]\nruntimePreferences: [{role: role_1.part-2, complexity: high_2, modelProfileRef: Namespace/Profile}]")
	decode(t, minimalV2+policyV2+"runtimePreferences: [{role: "+strings.Repeat("a", 256)+", complexity: high, modelProfileRef: worker}]")
}

func TestV2CollectionBounds(t *testing.T) {
	for name, limit := range map[string]int{"runtimes": 8, "modelProfiles": 32, "runtimePreferences": 32} {
		t.Run(name, func(t *testing.T) {
			for _, size := range []int{limit, limit + 1} {
				source := minimalV2
				if name == "runtimePreferences" {
					source += policyV2
				}
				source += name + ": ["
				for i := 0; i < size; i++ {
					if i > 0 {
						source += ", "
					}
					switch name {
					case "runtimes":
						source += fmt.Sprintf("{id: runtime-%d}", i)
					case "modelProfiles":
						source += fmt.Sprintf("{key: profile-%d, state: unconfigured}", i)
					case "runtimePreferences":
						source += fmt.Sprintf("{role: role-%d, complexity: high, modelProfileRef: worker}", i)
					}
				}
				source += "]"
				if size == limit {
					decode(t, source)
				} else {
					reject(t, source)
					_, issues := manifest.Decode([]byte(source))
					if issues[0].Code != "collection_limit" {
						t.Fatal(issues)
					}
				}
			}
		})
	}
}

func TestV2DeclarationStatesIndependentRoundTrip(t *testing.T) {
	for _, runtimes := range []string{"", "runtimes: unconfigured\n", "runtimes: []\n"} {
		for _, profiles := range []string{"", "modelProfiles: unconfigured\n", "modelProfiles: []\n"} {
			for _, preferences := range []string{"", "runtimePreferences: unconfigured\n", "runtimePreferences: []\n"} {
				p := decode(t, minimalV2+runtimes+profiles+preferences)
				q := decode(t, string(encode(t, p)))
				if !p.Equivalent(q) {
					t.Fatal("independent declaration forms lost")
				}
			}
		}
	}
	// Programmatic unresolved forms remain identical after canonical output.
	s := decode(t, minimalV2).State()
	s.Runtimes = project.Unconfigured[[]project.Runtime]()
	s.RuntimePreferences = project.Unconfigured[[]project.RuntimePreference]()
	p, issues := project.New(s)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if !p.Equivalent(decode(t, string(encode(t, p)))) {
		t.Fatal("domain declarations lost")
	}
}

func TestV2EncoderRejectsUnsafeDomainPolicy(t *testing.T) {
	for _, runtime := range []string{"/machine/path", "token=synthetic", "file:synthetic"} {
		s := decode(t, minimalV2).State()
		s.Runtimes = project.Configured([]project.Runtime{{ID: runtime}})
		p, issues := project.New(s)
		if len(issues) != 0 {
			t.Fatal(issues)
		}
		if output, issues := manifest.Encode(p); output != nil || len(issues) == 0 {
			t.Fatal("unsafe runtime escaped encoder")
		}
	}
	s := decode(t, minimalV2).State()
	s.Runtimes = project.Configured([]project.Runtime{{ID: "codex"}})
	s.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "token=synthetic", RuntimeRef: project.Configured("codex"), Model: project.Configured("safe")}})
	s.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "review", Complexity: "high", ModelProfileRef: "token=synthetic"}})
	p, issues := project.New(s)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if output, issues := manifest.Encode(p); output != nil || len(issues) == 0 {
		t.Fatal("unsafe profile escaped encoder")
	}
}

func TestV1ProfileBoundsRemainUnchanged(t *testing.T) {
	source := minimal + "modelProfiles: ["
	for i := 0; i < 33; i++ {
		if i > 0 {
			source += ", "
		}
		source += fmt.Sprintf("{key: p-%d, state: unconfigured}", i)
	}
	source += "]"
	p := decode(t, source)
	if !p.Equivalent(decode(t, string(encode(t, p)))) {
		t.Fatal("v1 profile capacity changed")
	}
}
