package githubissues

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workitem"
)

// Issue #230 (I230-T06) GitHub Issue lifecycle effects. Only these Provider
// effects exist: GET the issue, PATCH title/body, PATCH state open/closed and
// POST one comment. There is no Issue deletion, no label or classification
// mutation and no arbitrary field update.

// titleBytesLimit bounds a reviewed title well inside GitHub's own limit.
const titleBytesLimit = 256

func issueNumber(repository, selector string) (int, bool) {
	number, err := strconv.Atoi(selector)
	return number, err == nil && number > 0 && strconv.Itoa(number) == selector && (Adapter{}).ValidResource(repository)
}

func notCommitted() error {
	return &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, EffectNotCommitted: true}
}

// ReadDocument reads the current Issue identity, state, title and body.
func (a Adapter) ReadDocument(ctx context.Context, repository, selector string) (workitem.External, workitem.ProviderDocument, error) {
	number, ok := issueNumber(repository, selector)
	if !ok {
		return workitem.External{}, workitem.ProviderDocument{}, notCommitted()
	}
	output, err := a.run(ctx, nil, "api", "repos/"+repository+"/issues/"+strconv.Itoa(number))
	if err != nil {
		return workitem.External{}, workitem.ProviderDocument{}, err
	}
	external, err := decodeIssue(repository, selector, output)
	if err != nil {
		return workitem.External{}, workitem.ProviderDocument{}, err
	}
	var response struct {
		Title *string `json:"title"`
		Body  *string `json:"body"`
	}
	if err := json.Unmarshal(output, &response); err != nil || response.Title == nil || *response.Title == "" {
		return workitem.External{}, workitem.ProviderDocument{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	document := workitem.ProviderDocument{Title: *response.Title}
	if response.Body != nil {
		document.Body = *response.Body
	}
	return external, document, nil
}

// UpdateDocument PATCHes only the non-empty title and/or body.
func (a Adapter) UpdateDocument(ctx context.Context, repository, selector string, change workitem.DocumentChange) (workitem.External, error) {
	number, ok := issueNumber(repository, selector)
	if !ok || change == (workitem.DocumentChange{}) || len(change.Title) > titleBytesLimit || strings.ContainsAny(change.Title, "\r\n") || len(change.Body) > bodyLimit {
		return workitem.External{}, notCommitted()
	}
	payload, err := json.Marshal(struct {
		Title string `json:"title,omitempty"`
		Body  string `json:"body,omitempty"`
	}{change.Title, change.Body})
	if err != nil {
		return workitem.External{}, notCommitted()
	}
	output, err := a.run(ctx, payload, "api", "--method", "PATCH", "repos/"+repository+"/issues/"+strconv.Itoa(number), "--input", "-")
	if err != nil {
		return workitem.External{}, err
	}
	return decodeIssue(repository, selector, output)
}

// SetState PATCHes only the Issue state to open or closed. It sends no
// state_reason: closing is not acceptance or workflow completion.
func (a Adapter) SetState(ctx context.Context, repository, selector, state string) (workitem.External, error) {
	number, ok := issueNumber(repository, selector)
	value := map[string]string{workitem.OpenState: "open", workitem.ClosedState: "closed"}[state]
	if !ok || value == "" {
		return workitem.External{}, notCommitted()
	}
	payload, _ := json.Marshal(map[string]string{"state": value})
	output, err := a.run(ctx, payload, "api", "--method", "PATCH", "repos/"+repository+"/issues/"+strconv.Itoa(number), "--input", "-")
	if err != nil {
		return workitem.External{}, err
	}
	external, err := decodeIssue(repository, selector, output)
	if err == nil && external.State != state {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	return external, err
}

// Comment POSTs one reviewed comment through stdin.
func (a Adapter) Comment(ctx context.Context, repository, selector, message string) error {
	number, ok := issueNumber(repository, selector)
	if !ok || message == "" || len(message) > bodyLimit {
		return notCommitted()
	}
	payload, _ := json.Marshal(map[string]string{"body": message})
	_, err := a.run(ctx, payload, "api", "--method", "POST", "repos/"+repository+"/issues/"+strconv.Itoa(number)+"/comments", "--input", "-")
	return err
}

// ReviseDocument applies a reviewed revision to the current Issue document.
// A title revision replaces only the title. A section revision replaces only
// the content region of each named Axiom section and the authorship statement
// of the provenance footer; every other byte of the body is preserved. A body
// without the exact Axiom structure is never guessed at.
func (Adapter) ReviseDocument(current workitem.ProviderDocument, revision workitem.DocumentRevision) (workitem.ProviderDocument, error) {
	revised := workitem.ProviderDocument{Title: current.Title, Body: current.Body}
	if revision.Title != "" {
		revised.Title = revision.Title
	}
	if len(revision.Sections) == 0 {
		return revised, nil
	}
	body, err := reviseBody(current.Body, revision.Sections)
	if err != nil {
		return workitem.ProviderDocument{}, err
	}
	if len(body) > bodyLimit {
		return workitem.ProviderDocument{}, workitem.ErrDocumentTooLarge
	}
	revised.Body = body
	return revised, nil
}

var displaySections = map[string]string{
	"Story beneficiary": "beneficiary", "Story value": "value",
	"Problem": "problem", "Desired outcome": "desired_outcome", "Context": "context",
	"Scope": "scope", "Constraints": "constraints", "Non-goals": "non_goals",
	"Acceptance expectations": "acceptance_expectations",
}

func reviseBody(body string, sections []workitem.DraftSection) (string, error) {
	lines := strings.Split(body, "\n")
	plain := func(index int) string { return strings.TrimSuffix(lines[index], "\r") }
	// Section content can never produce a column-0 thematic break or heading
	// (sectionMarkdown escapes both), so the last "---" is the footer.
	separator := -1
	for index := len(lines) - 1; index >= 0; index-- {
		if plain(index) == "---" {
			separator = index
			break
		}
	}
	if separator < 0 || separator+3 >= len(lines) || plain(separator+1) != "" || !strings.HasPrefix(plain(separator+2), "_Structure authored by ") {
		return "", workitem.ErrDocumentUnrecognized
	}
	type heading struct {
		name string
		line int
	}
	var headings []heading
	known := map[string]int{}
	var present []string
	for index := 0; index < separator; index++ {
		line := plain(index)
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		name := displaySections[line[3:]]
		if name != "" {
			if _, duplicate := known[name]; duplicate {
				return "", workitem.ErrDocumentUnrecognized
			}
			known[name] = len(headings)
			present = append(present, line[3:])
		}
		headings = append(headings, heading{name, index})
	}
	authorship, ok := parseAuthorship(plain(separator+3), present)
	if !ok {
		return "", workitem.ErrDocumentUnrecognized
	}
	type region struct {
		end   int
		lines []string
	}
	replacements := map[int]region{}
	for _, section := range sections {
		position, found := known[section.Name]
		if !found {
			return "", workitem.ErrDocumentUnrecognized
		}
		start, end := headings[position].line+1, separator
		if position+1 < len(headings) {
			end = headings[position+1].line
		}
		if end-start < 2 || plain(start) != "" || plain(end-1) != "" {
			return "", workitem.ErrDocumentUnrecognized
		}
		content := strings.Split(strings.TrimSuffix(sectionMarkdown(section.Content), "\n"), "\n")
		replaced := append(append([]string{""}, content...), "")
		if strings.HasSuffix(lines[headings[position].line], "\r") {
			for index := range replaced {
				replaced[index] += "\r"
			}
		}
		replacements[start] = region{end, replaced}
		authorship[displayName(section.Name)] = section.Authorship
	}
	statement := authorshipStatement(present, authorship)
	if strings.HasSuffix(lines[separator+3], "\r") {
		statement += "\r"
	}
	revised := make([]string, 0, len(lines))
	for index := 0; index < len(lines); index++ {
		if replacement, found := replacements[index]; found {
			revised = append(revised, replacement.lines...)
			index = replacement.end - 1
			continue
		}
		if index == separator+3 {
			revised = append(revised, statement)
			continue
		}
		revised = append(revised, lines[index])
	}
	return strings.Join(revised, "\n"), nil
}

// parseAuthorship reads the footer authorship statement provenanceFooter
// writes. Every present Axiom section must be attributed exactly once.
func parseAuthorship(line string, present []string) (map[string]provenance.Authorship, bool) {
	result := make(map[string]provenance.Authorship, len(present))
	all := func(value provenance.Authorship) (map[string]provenance.Authorship, bool) {
		for _, name := range present {
			result[name] = value
		}
		return result, len(present) != 0
	}
	switch line {
	case "_All section content is user-authored._":
		return all(provenance.UserAuthored)
	case "_All section content is Axiom-authored._":
		return all(provenance.AxiomAuthored)
	}
	inner, found := strings.CutPrefix(line, "_User-authored: ")
	if !found {
		return nil, false
	}
	inner, found = strings.CutSuffix(inner, "._")
	if !found {
		return nil, false
	}
	user, axiom, found := strings.Cut(inner, ". Axiom-authored: ")
	if !found {
		return nil, false
	}
	allowed := make(map[string]bool, len(present))
	for _, name := range present {
		allowed[name] = true
	}
	for _, group := range []struct {
		names string
		value provenance.Authorship
	}{{user, provenance.UserAuthored}, {axiom, provenance.AxiomAuthored}} {
		for _, name := range strings.Split(group.names, ", ") {
			if _, seen := result[name]; seen || !allowed[name] {
				return nil, false
			}
			result[name] = group.value
		}
	}
	return result, len(result) == len(present)
}
