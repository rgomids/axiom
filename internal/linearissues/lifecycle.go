package linearissues

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workitem"
)

const titleBytesLimit = 256

func notCommitted() error {
	return &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, EffectNotCommitted: true}
}

func validIssue(resource, selector string) bool {
	return (Adapter{}).ValidResource(resource) && issueIdentifierPattern.MatchString(selector) && strings.HasPrefix(selector, resource+"-")
}

// ReadDocument reads the current Issue identity, state, title and body.
func (a Adapter) ReadDocument(ctx context.Context, resource, selector string) (workitem.External, workitem.ProviderDocument, error) {
	if !validIssue(resource, selector) {
		return workitem.External{}, workitem.ProviderDocument{}, notCommitted()
	}
	raw, err := a.fetchIssue(ctx, selector)
	if err != nil {
		return workitem.External{}, workitem.ProviderDocument{}, err
	}
	external, err := decodeIssue(resource, selector, raw)
	if err != nil {
		return workitem.External{}, workitem.ProviderDocument{}, err
	}
	var response struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}
	if err := json.Unmarshal(raw, &response); err != nil || response.Title == nil || *response.Title == "" {
		return workitem.External{}, workitem.ProviderDocument{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	document := workitem.ProviderDocument{Title: *response.Title}
	if response.Description != nil {
		document.Body = *response.Description
	}
	return external, document, nil
}

// UpdateDocument mutates only the non-empty title and/or description.
func (a Adapter) UpdateDocument(ctx context.Context, resource, selector string, change workitem.DocumentChange) (workitem.External, error) {
	if !validIssue(resource, selector) || change == (workitem.DocumentChange{}) || len(change.Title) > titleBytesLimit || strings.ContainsAny(change.Title, "\r\n") || len(change.Body) > bodyLimit {
		return workitem.External{}, notCommitted()
	}
	input := map[string]any{"id": selector}
	if change.Title != "" {
		input["title"] = change.Title
	}
	if change.Body != "" {
		input["description"] = change.Body
	}
	const mutation = `mutation IssueUpdate($id: String!, $input: IssueUpdateInput!) {
  issueUpdate(id: $id, input: $input) {
    success
    issue { id identifier url state { type } }
  }
}`
	// Linear expects id as issue id (UUID) or identifier in some APIs; we pass identifier via issueUpdate(id:).
	vars := map[string]any{"id": selector, "input": map[string]any{}}
	if change.Title != "" {
		vars["input"].(map[string]any)["title"] = change.Title
	}
	if change.Body != "" {
		vars["input"].(map[string]any)["description"] = change.Body
	}
	_ = input
	data, err := a.graphql(ctx, mutation, vars)
	if err != nil {
		return workitem.External{}, err
	}
	var payload struct {
		IssueUpdate struct {
			Success bool            `json:"success"`
			Issue   json.RawMessage `json:"issue"`
		} `json:"issueUpdate"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || !payload.IssueUpdate.Success {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, Ambiguous: true}
	}
	return decodeIssue(resource, selector, payload.IssueUpdate.Issue)
}

// SetState maps OPEN/CLOSED onto a Linear workflow state of matching type.
func (a Adapter) SetState(ctx context.Context, resource, selector, state string) (workitem.External, error) {
	if !validIssue(resource, selector) || (state != workitem.OpenState && state != workitem.ClosedState) {
		return workitem.External{}, notCommitted()
	}
	stateID, err := a.workflowStateID(ctx, resource, state)
	if err != nil {
		var pe *workitem.ProviderError
		if errorsAsProvider(err, &pe) {
			copied := *pe
			copied.EffectNotCommitted = true
			return workitem.External{}, &copied
		}
		return workitem.External{}, err
	}
	const mutation = `mutation IssueUpdate($id: String!, $input: IssueUpdateInput!) {
  issueUpdate(id: $id, input: $input) {
    success
    issue { id identifier url state { type } }
  }
}`
	data, err := a.graphql(ctx, mutation, map[string]any{"id": selector, "input": map[string]any{"stateId": stateID}})
	if err != nil {
		return workitem.External{}, err
	}
	var payload struct {
		IssueUpdate struct {
			Success bool            `json:"success"`
			Issue   json.RawMessage `json:"issue"`
		} `json:"issueUpdate"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || !payload.IssueUpdate.Success {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, Ambiguous: true}
	}
	external, err := decodeIssue(resource, selector, payload.IssueUpdate.Issue)
	if err == nil && external.State != state {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	return external, err
}

func errorsAsProvider(err error, target **workitem.ProviderError) bool {
	if err == nil {
		return false
	}
	if pe, ok := err.(*workitem.ProviderError); ok {
		*target = pe
		return true
	}
	return false
}

func (a Adapter) workflowStateID(ctx context.Context, teamKey, desired string) (string, error) {
	const query = `query TeamStates($key: String!) {
  teams(filter: { key: { eq: $key } }, first: 1) {
    nodes {
      states { nodes { id type } }
    }
  }
}`
	data, err := a.graphql(ctx, query, map[string]any{"key": teamKey})
	if err != nil {
		return "", err
	}
	var payload struct {
		Teams struct {
			Nodes []struct {
				States struct {
					Nodes []struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"nodes"`
				} `json:"states"`
			} `json:"nodes"`
		} `json:"teams"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || len(payload.Teams.Nodes) != 1 {
		return "", &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	wantType := "unstarted"
	if desired == workitem.ClosedState {
		wantType = "completed"
	}
	matches := []string{}
	for _, state := range payload.Teams.Nodes[0].States.Nodes {
		if strings.EqualFold(state.Type, wantType) || (desired == workitem.OpenState && (strings.EqualFold(state.Type, "backlog") || strings.EqualFold(state.Type, "unstarted") || strings.EqualFold(state.Type, "started"))) {
			if desired == workitem.OpenState && strings.EqualFold(state.Type, "unstarted") {
				return state.ID, nil
			}
			if desired == workitem.ClosedState && strings.EqualFold(state.Type, "completed") {
				return state.ID, nil
			}
			matches = append(matches, state.ID)
		}
	}
	if desired == workitem.OpenState {
		for _, state := range payload.Teams.Nodes[0].States.Nodes {
			if strings.EqualFold(state.Type, "backlog") {
				return state.ID, nil
			}
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", &workitem.ProviderError{Kind: workitem.ProviderAmbiguous, EffectNotCommitted: true}
	}
	return "", &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, EffectNotCommitted: true}
}

// Comment creates one reviewed comment on the issue.
func (a Adapter) Comment(ctx context.Context, resource, selector, message string) error {
	if !validIssue(resource, selector) || message == "" || len(message) > bodyLimit {
		return notCommitted()
	}
	const mutation = `mutation CommentCreate($input: CommentCreateInput!) {
  commentCreate(input: $input) { success }
}`
	data, err := a.graphql(ctx, mutation, map[string]any{"input": map[string]any{"issueId": selector, "body": message}})
	if err != nil {
		return err
	}
	var payload struct {
		CommentCreate struct {
			Success bool `json:"success"`
		} `json:"commentCreate"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || !payload.CommentCreate.Success {
		return &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, Ambiguous: true}
	}
	return nil
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
