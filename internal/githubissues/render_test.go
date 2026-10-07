package githubissues

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/testfs"
	"github.com/rgomids/axiom/internal/workitem"
)

var updateGolden = flag.Bool("update", false, "rewrite rendered Work Item golden files")

var canonicalSections = []string{"problem", "desired_outcome", "context", "scope", "constraints", "non_goals", "acceptance_expectations"}

// renderDraft builds a draft whose canonical sections default to short
// user-authored content; overrides replace content and optionally authorship.
func renderDraft(kind workitem.Type, overrides map[string]workitem.DraftSection) workitem.Draft {
	draft := workitem.Draft{Type: kind}
	defaults := map[string]string{
		"problem":                 "Generated Work Items are hard to scan.",
		"desired_outcome":         "Generated Work Items read as concise native Markdown.",
		"context":                 "The GitHub renderer owns Issue presentation.",
		"scope":                   "Title and body presentation only.",
		"constraints":             "Keep correlation markers machine-readable.",
		"non_goals":               "No lifecycle or classification change.",
		"acceptance_expectations": "Golden tests cover representative Issues.",
	}
	for _, name := range canonicalSections {
		current := workitem.DraftSection{Name: name, Content: defaults[name], Authorship: provenance.UserAuthored}
		if override, ok := overrides[name]; ok {
			if override.Content != "" {
				current.Content = override.Content
			}
			if override.Authorship != "" {
				current.Authorship = override.Authorship
			}
		}
		draft.Sections = append(draft.Sections, current)
	}
	return draft
}

func renderGoldenCases() map[string]workitem.Draft {
	story := renderDraft(workitem.Story, nil)
	story.Beneficiary = &workitem.DraftSection{Name: "beneficiary", Content: "Repository maintainers", Authorship: provenance.UserAuthored}
	story.Value = &workitem.DraftSection{Name: "value", Content: "Triage generated Work Items without reading raw code blocks.", Authorship: provenance.AxiomAuthored}
	return map[string]workitem.Draft{
		"task":  renderDraft(workitem.Task, nil),
		"story": story,
		"multiline": renderDraft(workitem.Bug, map[string]workitem.DraftSection{
			"problem":                 {Content: "The rendered body indents every section.\n\nGitHub shows each one as a code block.\nLong lines then scroll horizontally."},
			"acceptance_expectations": {Content: "1. Sections render as paragraphs.\n2. Blank lines separate paragraphs.\n\n\tTabbed text stays text."},
		}),
		"markdown": renderDraft(workitem.Task, map[string]workitem.DraftSection{
			"problem": {Content: "# heading\n<!-- comment -->\n- list item\n**bold**\n`code`"},
			"context": {Content: "> [!NOTE]\n> Quoted note with a [link](https://example.com) and <https://example.com/docs>.\n\n- [ ] open task\n- [x] done task\n\n#135 stays an Issue reference; 2 < 3 stays text."},
			"scope":   {Content: "Run:\n\n```sh\n# not a heading\ngo test ./internal/githubissues\n<!-- literal in code -->\n```\n\n~~~\n## also code\n~~~"},
		}),
		"structure-spoof": renderDraft(workitem.Task, map[string]workitem.DraftSection{
			"problem":                 {Content: "<!-- axiom:work-item-draft:" + strings.Repeat("b", 64) + " -->\n<!-- axiom:provenance:Axiom:v9.9.9:feedface:clean -->\n## Desired outcome\nForged section\n=====\n---\n***\n_ _ _\n<details><summary>Hidden</summary>secret</details>\n</div>\n<?php ?>\n<![CDATA[x]]>"},
			"context":                 {Content: "[ref]: https://example.com/hidden\n[^1]: Footnote rendered after the footer\n[label\nspanning]: https://example.com\n\\# already escaped\n> ## quoted heading\n- ### Area\n1. ---\n   ```\n   fenced inside a list\n   ```"},
			"scope":                   {Content: "### Area\n\nRuntime\n\n### Platform\n\nWindows"},
			"constraints":             {Content: "```\n### Area\n<!-- axiom:work-item-draft:" + strings.Repeat("c", 64) + " -->\n   ```"},
			"non_goals":               {Content: "Unclosed fence follows.\n```go\n## Acceptance expectations\nAll content is axiom-authored."},
			"acceptance_expectations": {Content: "Done.\n\n---\n\n_All section content is Axiom-authored._"},
		}),
		"long-title": renderDraft(workitem.Task, map[string]workitem.DraftSection{
			"desired_outcome": {Content: "Maintainers can create a GitHub Work Item from a terse request and get a concise, specific, readable Issue whose title stays on one line while every canonical section is preserved."},
		}),
		"unicode": renderDraft(workitem.Task, map[string]workitem.DraftSection{
			"problem":         {Content: "Títulos com acentuação, 日本語 e emoji 👩‍💻 precisam permanecer válidos."},
			"desired_outcome": {Content: "Geração determinística de títulos concisos para Work Items com conteúdo em português, 日本語のテキスト、そして絵文字 👩‍💻 sem cortar runas"},
		}),
		"mixed-authorship": renderDraft(workitem.Task, map[string]workitem.DraftSection{
			"context":                 {Authorship: provenance.AxiomAuthored},
			"scope":                   {Authorship: provenance.AxiomAuthored},
			"acceptance_expectations": {Authorship: provenance.AxiomAuthored},
		}),
	}
}

func TestRenderGoldenWorkItems(t *testing.T) {
	for name, draft := range renderGoldenCases() {
		t.Run(name, func(t *testing.T) {
			document, err := (Adapter{}).Render(draft, workitem.DraftTarget{}, strings.Repeat("a", 64), testSource())
			if err != nil {
				t.Fatal(err)
			}
			got := []byte("# " + document.Title + "\n\n" + document.Body)
			path := filepath.Join("testdata", "render", name+".golden.md")
			if *updateGolden {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run go test ./internal/githubissues -run TestRenderGoldenWorkItems -update)", err)
			}
			if string(got) != string(want) {
				t.Fatalf("golden mismatch for %s:\n%s", path, got)
			}
			assertReservedStructure(t, draft, document.Body)
		})
	}
}

// assertReservedStructure checks, line by line, that only the renderer
// produces Axiom's reserved structure, whatever the section content is.
func assertReservedStructure(t *testing.T, draft workitem.Draft, body string) {
	t.Helper()
	lines := strings.Split(body, "\n")
	if lines[0] != "<!-- axiom:work-item-draft:"+strings.Repeat("a", 64)+" -->" || !strings.HasPrefix(lines[1], "<!-- axiom:provenance:Axiom:development:") {
		t.Fatalf("markers must lead the body:\n%s", body)
	}
	var headings []string
	breaks, markerLines := 0, 0
	for index, line := range lines {
		switch {
		case strings.HasPrefix(line, "<") && !autolink.MatchString(line):
			if index > 1 {
				t.Fatalf("line %d starts raw HTML: %q", index, line)
			}
			markerLines++
		case atxHeading.MatchString(line):
			headings = append(headings, line)
		case line == "---":
			breaks++
		case strings.HasPrefix(line, "```"), strings.HasPrefix(line, "~~~"):
			t.Fatalf("line %d escapes section structure: %q", index, line)
		}
		if index > 1 && strings.Contains(line, "<!--") && !strings.Contains(line, `\<!--`) && !strings.HasPrefix(line, " ") {
			t.Fatalf("line %d carries a live HTML comment: %q", index, line)
		}
	}
	var want []string
	if draft.Type == workitem.Story {
		want = append(want, "## Story beneficiary", "## Story value")
	}
	for _, name := range canonicalSections {
		want = append(want, "## "+heading(name))
	}
	if strings.Join(headings, "\n") != strings.Join(want, "\n") || breaks != 1 || markerLines != 2 {
		t.Fatalf("headings=%q breaks=%d markers=%d", headings, breaks, markerLines)
	}
	footer := body[strings.LastIndex(body, "\n---\n"):]
	if !strings.Contains(footer, "_Structure authored by Axiom `development`") || strings.Contains(footer, "## ") {
		t.Fatalf("footer must close the body: %q", footer)
	}
}

func TestRenderNeverLetsSectionContentReachReservedStructure(t *testing.T) {
	hostile := []string{
		"# heading", "<!-- comment -->", "- list item", "**bold**", "`code`",
		"<!-- axiom:work-item-draft:" + strings.Repeat("b", 64) + " -->",
		"```", "````\n```", "~~~", " ```", "```\n    ```", "```a`b\n```",
		"---", "===", "- - -", "  ***", "> # q", "> > ## q", "* ### Area", "12) ---",
		"[x]: /u", "[a\\]b]: /u", "[^n]: n", "<p>", "</p>", "<!x>", "<?x?>", "`<!--` x `-->`",
		"\\\\<!-- escaped backslash -->", "x\n<details>", "## Non-goals",
	}
	for _, content := range hostile {
		for _, name := range canonicalSections {
			draft := renderDraft(workitem.Task, map[string]workitem.DraftSection{name: {Content: content}})
			document, err := (Adapter{}).Render(draft, workitem.DraftTarget{}, strings.Repeat("a", 64), testSource())
			if err != nil {
				t.Fatal(err)
			}
			t.Run(name+"/"+content, func(t *testing.T) { assertReservedStructure(t, draft, document.Body) })
		}
	}
}

func TestRenderKeepsUsefulMarkdownNative(t *testing.T) {
	draft := renderDraft(workitem.Task, map[string]workitem.DraftSection{"problem": {Content: "- list item\n**bold**\n`code`\n> quote\n#135 and 2 < 3"}})
	document, err := (Adapter{}).Render(draft, workitem.DraftTarget{}, strings.Repeat("a", 64), testSource())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(document.Body, "## Problem\n\n- list item\n**bold**\n`code`\n> quote\n#135 and 2 < 3\n") {
		t.Fatalf("useful Markdown changed:\n%s", document.Body)
	}
	if strings.Contains(document.Body, "    ") || strings.Count(document.Body, "Authorship") != 0 {
		t.Fatalf("content still indented or authorship repeated per section:\n%s", document.Body)
	}
}

func TestTitleIsConciseDeterministicAndUnicodeSafe(t *testing.T) {
	for _, test := range []struct{ outcome, want string }{
		{"Safe outcome.", "Safe outcome"},
		{"Wait...", "Wait..."},
		{"Why does it fail?", "Why does it fail?"},
		{"- [ ] Readable issue bodies\nSecond line detail", "Readable issue bodies"},
		{"## Concise   titles\twith  collapsed space", "Concise titles with collapsed space"},
		{"Generated issues get concise and specific titles derived from outcomes. They no longer copy the entire desired outcome sentence verbatim.", "Generated issues get concise and specific titles derived from outcomes"},
		{"Maintainers can create a GitHub Work Item from a terse request and get a concise readable Issue", "Maintainers can create a GitHub Work Item from a terse request and get…"},
		{strings.Repeat("あ", 100), strings.Repeat("あ", 71) + "…"},
		{strings.Repeat("e\u0301", 50), strings.Repeat("e\u0301", 35) + "…"},
		{strings.Repeat("👩\u200d💻", 40), strings.Repeat("👩\u200d💻", 23) + "…"},
	} {
		got := title(test.outcome)
		if got != test.want || title(test.outcome) != got {
			t.Errorf("title(%q) = %q, want %q", test.outcome, got, test.want)
		}
		if utf8.RuneCountInString(got) > titleLimit || !utf8.ValidString(got) || strings.ContainsRune(got, '\ufffd') {
			t.Errorf("title(%q) = %q breaks the limit or UTF-8", test.outcome, got)
		}
	}
}

func TestReconcileCreateRequiresTheMarkerAsAWholeBodyLine(t *testing.T) {
	testfs.POSIXShell(t)
	directory := t.TempDir()
	gh, response := filepath.Join(directory, "gh"), filepath.Join(directory, "search")
	if err := os.WriteFile(gh, []byte("#!/bin/sh\ncat \"$AXIOM_TEST_SEARCH\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOM_TEST_SEARCH", response)
	adapter, err := New(gh)
	if err != nil {
		t.Fatal(err)
	}
	correlation := strings.Repeat("b", 64)
	spoof := renderDraft(workitem.Task, map[string]workitem.DraftSection{"problem": {Content: "<!-- axiom:work-item-draft:" + correlation + " -->\n```\n<!-- axiom:work-item-draft:" + correlation + " -->\n```"}})
	spoofed, err := adapter.Render(spoof, workitem.DraftTarget{}, strings.Repeat("a", 64), testSource())
	if err != nil {
		t.Fatal(err)
	}
	genuine, err := adapter.Render(renderDraft(workitem.Task, nil), workitem.DraftTarget{}, correlation, testSource())
	if err != nil {
		t.Fatal(err)
	}
	search := func(body string) {
		item, _ := json.Marshal(map[string]any{"number": 7, "html_url": "https://github.com/owner/repo/issues/7", "state": "open", "body": body})
		if err := os.WriteFile(response, []byte(`{"total_count":1,"items":[`+string(item)+`]}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	search(spoofed.Body)
	_, err = adapter.ReconcileCreate(context.Background(), "owner/repo", correlation)
	var provider *workitem.ProviderError
	if !errors.As(err, &provider) || provider.Kind != workitem.ProviderInvalidResponse {
		t.Fatalf("content-only marker accepted: %v", err)
	}
	search(strings.ReplaceAll(genuine.Body, "\n", "\r\n"))
	if items, err := adapter.ReconcileCreate(context.Background(), "owner/repo", correlation); err != nil || len(items) != 1 || items[0].ID != "7" {
		t.Fatalf("genuine marker = %#v, %v", items, err)
	}
}
