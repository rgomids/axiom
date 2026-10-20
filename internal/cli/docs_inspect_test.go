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
	handled, code := InspectDocs(args, completionProvenance(t), &output, func() operationaldocs.Resolution { calls++; return resolution })
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
		})
		if !handled || code != 1 || strings.Contains(output.String(), `"docs":`) {
			t.Fatalf("%v: %v %d %s", args, handled, code, output.String())
		}
	}
}
