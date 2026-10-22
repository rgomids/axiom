package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/operationaldocs"
)

func inspectDocsForTest(t *testing.T, resolution operationaldocs.Resolution, args ...string) (int, string) {
	t.Helper()
	var output bytes.Buffer
	calls := 0
	handled, code := InspectDocs(args, completionProvenance(t), &output, func() operationaldocs.Resolution { calls++; return resolution }, SkillExpectation{}, nil)
	if !handled {
		t.Fatal("not handled")
	}
	if calls > 1 {
		t.Fatal("resolver called repeatedly")
	}
	return code, output.String()
}

func TestDocsInspectStatesAndExits(t *testing.T) {
	cases := []struct {
		resolution     operationaldocs.Resolution
		status, reason string
		code           int
	}{
		{operationaldocs.Resolution{State: operationaldocs.StateAbsent, Reason: "binding_absent"}, "failure", "binding_absent", 1},
		{operationaldocs.Resolution{State: operationaldocs.StateUnknown, Reason: "permission_denied"}, "failure", "permission_denied", 1},
		{operationaldocs.Resolution{State: operationaldocs.StateInvalid, Reason: "hash_mismatch"}, "validation_failure", "hash_mismatch", 1},
	}
	for _, c := range cases {
		code, out := inspectDocsForTest(t, c.resolution, "--json", "docs", "inspect")
		var got struct {
			Status string     `json:"status"`
			Docs   docsReport `json:"docs"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil || code != c.code || got.Status != c.status || got.Docs.Reason != c.reason {
			t.Fatalf("%s: %v %d %s", c.reason, err, code, out)
		}
		if got.Docs.Paths.Root != nil || got.Docs.Paths.Index != nil || got.Docs.Man.Page != nil || len(got.Docs.Diagnostics) != 1 || got.Docs.Skills.Observed == nil {
			t.Fatalf("refusal exposed paths or omitted arrays: %s", out)
		}
	}
}

func TestDocsInspectVerifiedExposesPathsAndNoLoadedClaim(t *testing.T) {
	resolution := operationaldocs.Resolution{State: operationaldocs.StateVerified, Reason: "ok", Binary: "/opt/axiom", BinarySHA256: strings.Repeat("a", 64), BindingSHA256: strings.Repeat("b", 64), Root: "/opt/.axiom-docs/axiom/generations/g", Binding: operationaldocs.Binding{SourceCommit: strings.Repeat("c", 40), Version: "0.2.0", Generation: strings.Repeat("d", 64), SkillManifestSHA256: strings.Repeat("e", 64)}}
	code, out := inspectDocsForTest(t, resolution, "--json", "docs", "inspect")
	if code != 0 || !strings.Contains(out, `"index":"/opt/.axiom-docs/axiom/generations/g/docs/text/index.md"`) || !strings.Contains(out, `"readerSelection":"unknown"`) || strings.Contains(out, "loaded") {
		t.Fatalf("%d %s", code, out)
	}
	if code, human := inspectDocsForTest(t, resolution, "--human", "docs", "inspect"); code != 0 || !strings.Contains(human, "Installed documentation verified") {
		t.Fatalf("%d %s", code, human)
	}
}

func TestDocsInspectRejectsSuffixAndExtraInput(t *testing.T) {
	absent := operationaldocs.Resolution{State: operationaldocs.StateAbsent, Reason: "binding_absent"}
	for _, args := range [][]string{{"docs", "inspect", "--json"}, {"--json", "docs", "inspect", "x"}, {"--json", "docs"}, {"docs", "inspect", "--root", "/tmp"}} {
		var output bytes.Buffer
		handled, code := InspectDocs(args, completionProvenance(t), &output, func() operationaldocs.Resolution {
			t.Fatal("resolver ran for invalid input")
			return absent
		}, SkillExpectation{}, nil)
		if !handled || code != 1 || strings.Contains(output.String(), `"docs":`) {
			t.Fatalf("%v: %v %d %s", args, handled, code, output.String())
		}
	}
}

func TestDocsInspectReportsExpectedAndObservedSkillsWithoutLoadedClaim(t *testing.T) {
	manifest := strings.Repeat("e", 64)
	resolution := operationaldocs.Resolution{State: operationaldocs.StateVerified, Reason: "ok", Root: "/r", Binding: operationaldocs.Binding{SkillManifestSHA256: manifest}}
	var output bytes.Buffer
	expectation := SkillExpectation{Digests: map[string]string{"axiom-project": strings.Repeat("1", 64), "axiom-runtime": strings.Repeat("2", 64)}, Manifest: manifest}
	observe := func() []SkillObservation {
		return []SkillObservation{{"codex", "recognized", "match", manifest}, {"claude", "absent", "mismatch", ""}}
	}
	InspectDocs([]string{"--json", "docs", "inspect"}, completionProvenance(t), &output, func() operationaldocs.Resolution { return resolution }, expectation, observe)
	var got struct{ Docs docsReport }
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	skills := got.Docs.Skills
	if len(skills.Expected) != 2 || skills.Expected[0]["name"] != "axiom-project" || len(skills.Observed) != 2 {
		t.Fatalf("%s", output.String())
	}
	for _, entry := range skills.Observed {
		if entry["loaded"] != "unknown" {
			t.Fatal("loaded claimed")
		}
	}
	if skills.Observed[1]["manifestSha256"] != nil {
		t.Fatal("mismatching runtime reported a manifest")
	}

	output.Reset()
	expectation.Manifest = strings.Repeat("0", 64)
	InspectDocs([]string{"--json", "docs", "inspect"}, completionProvenance(t), &output, func() operationaldocs.Resolution { return resolution }, expectation, observe)
	if json.Unmarshal(output.Bytes(), &got) != nil || len(got.Docs.Skills.Expected) != 0 || len(got.Docs.Diagnostics) != 1 {
		t.Fatalf("unreproduced manifest must not list expected skills: %s", output.String())
	}
}
