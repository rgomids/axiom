package githubissues

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workitem"
)

func TestRenderSeparatesAuthorshipAndNeutralizesMarkdownStructure(t *testing.T) {
	adapter := Adapter{}
	draft := workitem.Draft{Sections: []workitem.DraftSection{
		{Name: "problem", Content: "# injected\n<!-- instruction -->", Authorship: provenance.UserAuthored},
		{Name: "desired_outcome", Content: "Safe outcome", Authorship: provenance.UserAuthored},
		{Name: "context", Content: "Context", Authorship: provenance.AxiomAuthored},
		{Name: "scope", Content: "Scope", Authorship: provenance.UserAuthored},
		{Name: "constraints", Content: "Constraints", Authorship: provenance.UserAuthored},
		{Name: "non_goals", Content: "Non-goals", Authorship: provenance.UserAuthored},
		{Name: "acceptance_expectations", Content: "Acceptance", Authorship: provenance.UserAuthored},
	}}
	document, err := adapter.Render(draft, workitem.DraftTarget{}, strings.Repeat("a", 64), testSource())
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"<!-- axiom:work-item-draft:", "Authorship: `user`", "Authorship: `axiom`", "    # injected", "    <!-- instruction -->"} {
		if !strings.Contains(document.Body, expected) {
			t.Fatalf("body missing %q:\n%s", expected, document.Body)
		}
	}
}

func TestAdapterUsesBoundedAPICommandsAndStdinForUntrustedBody(t *testing.T) {
	directory := t.TempDir()
	gh := filepath.Join(directory, "gh")
	log := filepath.Join(directory, "args")
	stdin := filepath.Join(directory, "stdin")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$AXIOM_TEST_ARGS"
cat > "$AXIOM_TEST_STDIN"
case "$*" in
  *search/issues*) printf '%s\n' '{"total_count":0,"items":[]}' ;;
  *issues/7*) printf '%s\n' '{"number":7,"html_url":"https://github.com/owner/repo/issues/7","state":"open"}' ;;
  *) printf '%s\n' '{"number":7,"html_url":"https://github.com/owner/repo/issues/7","state":"open"}' ;;
esac
`
	if err := os.WriteFile(gh, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOM_TEST_ARGS", log)
	t.Setenv("AXIOM_TEST_STDIN", stdin)
	adapter, err := New(gh)
	if err != nil {
		t.Fatal(err)
	}
	request := workitem.CreateRequest{Resource: "owner/repo", Correlation: strings.Repeat("a", 64), Document: workitem.ProviderDocument{Title: "$(touch /tmp/no)", Body: "`rm -rf` ; $SHELL"}}
	external, err := adapter.Create(context.Background(), request)
	if err != nil || external.ID != "7" {
		t.Fatalf("create = %#v, %v", external, err)
	}
	arguments, _ := os.ReadFile(log)
	payload, _ := os.ReadFile(stdin)
	if strings.Contains(string(arguments), request.Document.Title) || strings.Contains(string(arguments), request.Document.Body) || !strings.Contains(string(payload), request.Document.Title) || !strings.Contains(string(payload), request.Document.Body) {
		t.Fatalf("args=%q stdin=%q", arguments, payload)
	}
	if matches, err := adapter.ReconcileCreate(context.Background(), "owner/repo", strings.Repeat("a", 64)); err != nil || len(matches) != 0 {
		t.Fatalf("reconcile = %#v, %v", matches, err)
	}
}

func TestAdapterStrictlyRejectsMismatchedResponseAndRateLimit(t *testing.T) {
	directory := t.TempDir()
	gh := filepath.Join(directory, "gh")
	if err := os.WriteFile(gh, []byte("#!/bin/sh\nprintf '%s\\n' '{\"number\":7,\"html_url\":\"https://github.com/other/repo/issues/7\",\"state\":\"open\"}'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	adapter, _ := New(gh)
	if _, err := adapter.Read(context.Background(), "owner/repo", "7"); err == nil {
		t.Fatal("mismatched response accepted")
	}
	if err := os.WriteFile(gh, []byte("#!/bin/sh\nprintf '%s\\n' 'API rate limit exceeded' >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := adapter.Read(context.Background(), "owner/repo", "7")
	provider, ok := err.(*workitem.ProviderError)
	if !ok || provider.Kind != workitem.ProviderRateLimited || !provider.Retryable {
		t.Fatalf("rate limit = %#v", err)
	}
}

func TestAdapterBoundsTimeoutAndOutput(t *testing.T) {
	directory := t.TempDir()
	gh := filepath.Join(directory, "gh")
	if err := os.WriteFile(gh, []byte("#!/bin/sh\nsleep 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	adapter, _ := New(gh)
	adapter.timeout = 10 * time.Millisecond
	_, err := adapter.Read(context.Background(), "owner/repo", "7")
	provider, ok := err.(*workitem.ProviderError)
	if !ok || provider.Kind != workitem.ProviderAmbiguous || !provider.Retryable || !provider.Ambiguous {
		t.Fatalf("timeout = %#v", err)
	}

	if err := os.WriteFile(gh, []byte("#!/bin/sh\nyes x | head -c 300000\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	adapter.timeout = time.Second
	_, err = adapter.Read(context.Background(), "owner/repo", "7")
	provider, ok = err.(*workitem.ProviderError)
	if !ok || provider.Kind != workitem.ProviderInvalidResponse {
		t.Fatalf("oversized output = %#v", err)
	}
}

func TestRenderTruncatesUnicodeTitleWithoutBreakingUTF8(t *testing.T) {
	adapter := Adapter{}
	draft := workitem.Draft{Sections: []workitem.DraftSection{
		{Name: "problem", Content: "Problem", Authorship: provenance.UserAuthored},
		{Name: "desired_outcome", Content: strings.Repeat("acao segura ", 30) + "ç", Authorship: provenance.UserAuthored},
		{Name: "context", Content: "Context", Authorship: provenance.UserAuthored},
		{Name: "scope", Content: "Scope", Authorship: provenance.UserAuthored},
		{Name: "constraints", Content: "Constraints", Authorship: provenance.UserAuthored},
		{Name: "non_goals", Content: "Non-goals", Authorship: provenance.UserAuthored},
		{Name: "acceptance_expectations", Content: "Acceptance", Authorship: provenance.UserAuthored},
	}}
	document, err := adapter.Render(draft, workitem.DraftTarget{}, strings.Repeat("a", 64), testSource())
	if err != nil || !strings.HasSuffix(document.Title, "...") || strings.ContainsRune(document.Title, '\ufffd') {
		t.Fatalf("title=%q err=%v", document.Title, err)
	}
}

func testSource() provenance.Value {
	value, _ := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: "abc123def456", SourceState: provenance.Clean}, nil)
	return value
}
