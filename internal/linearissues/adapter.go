package linearissues

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workitem"
)

// Team key: Linear team keys are uppercase alphanumeric (e.g. AXM, ENG).
var teamKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{0,9}$`)

// issueIdentifierPattern matches TEAM-N where N is a positive decimal.
var issueIdentifierPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{0,9}-[1-9][0-9]{0,9}$`)

func (Adapter) ValidResource(value string) bool {
	return teamKeyPattern.MatchString(value)
}

func (a Adapter) ValidExternal(resource, selector string, external workitem.External) bool {
	if !a.ValidResource(resource) || selector == "" || external.ID != selector {
		return false
	}
	if !issueIdentifierPattern.MatchString(selector) || !strings.HasPrefix(selector, resource+"-") {
		return false
	}
	if external.State != workitem.OpenState && external.State != workitem.ClosedState {
		return false
	}
	return validLinearURL(selector, external.URL)
}

func validLinearURL(identifier, raw string) bool {
	if raw == "" || len(raw) > 512 {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "linear.app" {
		return false
	}
	// /<workspace>/issue/<TEAM-N> or /<workspace>/issue/<TEAM-N>/...
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 3 || parts[1] != "issue" || parts[2] != identifier {
		return false
	}
	return parts[0] != "" && !strings.ContainsAny(parts[0], "/\\")
}

func (a Adapter) Render(draft workitem.Draft, _ workitem.DraftTarget, correlation string, source provenance.Value) (workitem.ProviderDocument, error) {
	if len(correlation) != 64 || !source.Valid() || len(draft.Sections) != 7 {
		return workitem.ProviderDocument{}, errors.New("invalid draft")
	}
	outcome := section(draft, "desired_outcome")
	if outcome == "" {
		return workitem.ProviderDocument{}, errors.New("missing outcome")
	}
	if !draft.Type.Valid() {
		return workitem.ProviderDocument{}, errors.New("missing item type")
	}
	sections := draft.Sections
	if draft.Type == workitem.Story {
		if draft.Beneficiary == nil || draft.Value == nil {
			return workitem.ProviderDocument{}, errors.New("missing story value")
		}
		sections = append([]workitem.DraftSection{*draft.Beneficiary, *draft.Value}, draft.Sections...)
	}
	var body strings.Builder
	fmt.Fprintf(&body, "<!-- axiom:work-item-draft:%s -->\n", correlation)
	fmt.Fprintf(&body, "<!-- axiom:provenance:%s:%s:%s:%s -->\n\n", source.Product(), source.Version(), source.Revision(), source.SourceState())
	fmt.Fprintf(&body, "**Work Item type:** `%s`\n", draft.Type)
	for _, current := range sections {
		fmt.Fprintf(&body, "\n## %s\n\n%s", displayName(current.Name), sectionMarkdown(current.Content))
	}
	body.WriteString("\n" + provenanceFooter(source, sections))
	if body.Len() > bodyLimit {
		return workitem.ProviderDocument{}, errors.New("provider body exceeds limit")
	}
	return workitem.ProviderDocument{Title: title(outcome), Body: body.String()}, nil
}

func section(draft workitem.Draft, name string) string {
	for _, current := range draft.Sections {
		if current.Name == name {
			return current.Content
		}
	}
	return ""
}

func (a Adapter) Create(parent context.Context, request workitem.CreateRequest) (workitem.External, error) {
	if !a.ValidResource(request.Resource) || len(request.Correlation) != 64 || request.Document.Title == "" || request.Document.Body == "" || len(request.Document.Body) > bodyLimit {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, EffectNotCommitted: true}
	}
	teamID, err := a.teamID(parent, request.Resource)
	if err != nil {
		var provider *workitem.ProviderError
		if errors.As(err, &provider) {
			copied := *provider
			copied.EffectNotCommitted = true
			return workitem.External{}, &copied
		}
		return workitem.External{}, err
	}
	const mutation = `mutation IssueCreate($input: IssueCreateInput!) {
  issueCreate(input: $input) {
    success
    issue { id identifier url state { type } description }
  }
}`
	data, err := a.graphql(parent, mutation, map[string]any{
		"input": map[string]any{
			"teamId":      teamID,
			"title":       request.Document.Title,
			"description": request.Document.Body,
		},
	})
	if err != nil {
		var provider *workitem.ProviderError
		if errors.As(err, &provider) && !provider.EffectNotCommitted {
			provider.Ambiguous = true
		}
		return workitem.External{}, err
	}
	var payload struct {
		IssueCreate struct {
			Success bool            `json:"success"`
			Issue   json.RawMessage `json:"issue"`
		} `json:"issueCreate"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || !payload.IssueCreate.Success {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, Ambiguous: true}
	}
	external, err := decodeIssue(request.Resource, "", payload.IssueCreate.Issue)
	if err != nil {
		var provider *workitem.ProviderError
		if errors.As(err, &provider) {
			provider.Ambiguous = true
		}
		return workitem.External{}, err
	}
	return external, nil
}

func (a Adapter) ReconcileCreate(ctx context.Context, resource, correlation string) ([]workitem.External, error) {
	if !a.ValidResource(resource) || len(correlation) != 64 {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	marker := "axiom:work-item-draft:" + correlation
	const query = `query IssueSearch($term: String!) {
  searchIssues(term: $term, first: 10) {
    nodes { id identifier url state { type } description }
  }
}`
	data, err := a.graphql(ctx, query, map[string]any{"term": marker})
	if err != nil {
		return nil, err
	}
	var payload struct {
		SearchIssues struct {
			Nodes []json.RawMessage `json:"nodes"`
		} `json:"searchIssues"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || len(payload.SearchIssues.Nodes) > 10 {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	items := make([]workitem.External, 0, len(payload.SearchIssues.Nodes))
	for _, raw := range payload.SearchIssues.Nodes {
		var inspected struct {
			Description string `json:"description"`
			Identifier  string `json:"identifier"`
		}
		if err := json.Unmarshal(raw, &inspected); err != nil || !hasReservedHeader(inspected.Description, "<!-- "+marker+" -->") {
			continue
		}
		if !strings.HasPrefix(inspected.Identifier, resource+"-") {
			continue
		}
		item, err := decodeIssue(resource, "", raw)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (a Adapter) Read(ctx context.Context, resource, selector string) (workitem.External, error) {
	if !a.ValidResource(resource) || !issueIdentifierPattern.MatchString(selector) || !strings.HasPrefix(selector, resource+"-") {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	raw, err := a.fetchIssue(ctx, selector)
	if err != nil {
		return workitem.External{}, err
	}
	return decodeIssue(resource, selector, raw)
}

func (a Adapter) teamID(ctx context.Context, teamKey string) (string, error) {
	const query = `query Team($key: String!) {
  teams(filter: { key: { eq: $key } }, first: 2) {
    nodes { id key }
  }
}`
	data, err := a.graphql(ctx, query, map[string]any{"key": teamKey})
	if err != nil {
		return "", err
	}
	var payload struct {
		Teams struct {
			Nodes []struct {
				ID  string `json:"id"`
				Key string `json:"key"`
			} `json:"nodes"`
		} `json:"teams"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	if len(payload.Teams.Nodes) != 1 || payload.Teams.Nodes[0].Key != teamKey || payload.Teams.Nodes[0].ID == "" {
		if len(payload.Teams.Nodes) > 1 {
			return "", &workitem.ProviderError{Kind: workitem.ProviderAmbiguous}
		}
		return "", &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, EffectNotCommitted: true}
	}
	return payload.Teams.Nodes[0].ID, nil
}

func (a Adapter) fetchIssue(ctx context.Context, identifier string) (json.RawMessage, error) {
	const query = `query Issue($id: String!) {
  issue(id: $id) { id identifier url state { type } title description labels { nodes { name } } }
}`
	data, err := a.graphql(ctx, query, map[string]any{"id": identifier})
	if err != nil {
		return nil, err
	}
	var payload struct {
		Issue json.RawMessage `json:"issue"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || len(payload.Issue) == 0 || string(payload.Issue) == "null" {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	return payload.Issue, nil
}

func decodeIssue(resource, selector string, raw []byte) (workitem.External, error) {
	var issue struct {
		Identifier string `json:"identifier"`
		URL        string `json:"url"`
		State      struct {
			Type string `json:"type"`
		} `json:"state"`
	}
	if err := json.Unmarshal(raw, &issue); err != nil || issue.Identifier == "" || issue.URL == "" {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	if selector != "" && issue.Identifier != selector {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	if !strings.HasPrefix(issue.Identifier, resource+"-") {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	external := workitem.External{ID: issue.Identifier, URL: issue.URL, State: mapLinearState(issue.State.Type)}
	if !(Adapter{}).ValidExternal(resource, external.ID, external) {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	return external, nil
}

func mapLinearState(stateType string) string {
	switch strings.ToLower(stateType) {
	case "completed", "canceled", "cancelled":
		return workitem.ClosedState
	default:
		return workitem.OpenState
	}
}

func hasReservedHeader(body, marker string) bool {
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == marker {
			return true
		}
	}
	return false
}
