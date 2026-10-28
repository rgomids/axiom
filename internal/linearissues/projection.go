package linearissues

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workitem"
)

// Inspect reads bounded Linear state needed to reconcile one committed local transition.
func (a Adapter) Inspect(ctx context.Context, item workflow.WorkItem, desiredLabel, projectionKey string) (workflow.ProjectionObservation, error) {
	if !a.validProjection(item, desiredLabel, projectionKey) {
		return workflow.ProjectionObservation{}, &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse}
	}
	teamLabels, err := a.teamLabelNames(ctx, item.Resource)
	if err != nil {
		return workflow.ProjectionObservation{}, projectionError(err, false)
	}
	raw, err := a.fetchIssue(ctx, item.ExternalID)
	if err != nil {
		return workflow.ProjectionObservation{}, projectionError(err, false)
	}
	var issue struct {
		Identifier string `json:"identifier"`
		URL        string `json:"url"`
		State      struct {
			Type string `json:"type"`
		} `json:"state"`
		Labels struct {
			Nodes []struct {
				Name string `json:"name"`
			} `json:"nodes"`
		} `json:"labels"`
	}
	if err := json.Unmarshal(raw, &issue); err != nil {
		return workflow.ProjectionObservation{}, &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse}
	}
	observed := workitem.External{ID: issue.Identifier, URL: issue.URL, State: mapLinearState(issue.State.Type)}
	if !a.ValidExternal(item.Resource, item.ExternalID, observed) || len(issue.Labels.Nodes) > 100 {
		return workflow.ProjectionObservation{}, &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse}
	}
	issueLabels := make([]string, 0, len(issue.Labels.Nodes))
	for _, label := range issue.Labels.Nodes {
		if label.Name == "" || len(label.Name) > 256 {
			return workflow.ProjectionObservation{}, &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse}
		}
		issueLabels = append(issueLabels, label.Name)
	}
	present, err := a.commentPresent(ctx, item.ExternalID, projectionKey)
	if err != nil {
		return workflow.ProjectionObservation{}, projectionError(err, false)
	}
	return workflow.ProjectionObservation{
		Provider:         item.Provider,
		Resource:         item.Resource,
		IssueExternalID:  observed.ID,
		IssueURL:         observed.URL,
		IssueState:       observed.State,
		RepositoryLabels: teamLabels,
		IssueLabels:      issueLabels,
		CommentPresent:   present,
	}, nil
}

func (a Adapter) Apply(ctx context.Context, item workflow.WorkItem, effect workflow.ProjectionEffect) error {
	if !a.validProjectionItem(item) || effect.Value == "" {
		return &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse, EffectNotCommitted: true}
	}
	switch effect.Kind {
	case workflow.CreateStageLabel:
		if !workflow.ValidDesiredProjectionLabel(effect.Value) {
			return &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse, EffectNotCommitted: true}
		}
		return a.createLabel(ctx, item.Resource, effect.Value)
	case workflow.AddStageLabel:
		if !workflow.ValidDesiredProjectionLabel(effect.Value) {
			return &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse, EffectNotCommitted: true}
		}
		return a.addLabel(ctx, item.ExternalID, item.Resource, effect.Value)
	case workflow.RemoveStageLabel:
		if !workflow.ValidProjectionLabel(effect.Value) {
			return &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse, EffectNotCommitted: true}
		}
		return a.removeLabel(ctx, item.ExternalID, item.Resource, effect.Value)
	case workflow.PostTransitionComment:
		if len(effect.Value) > bodyLimit || !strings.Contains(effect.Value, "<!-- axiom:workflow-projection:") {
			return &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse, EffectNotCommitted: true}
		}
		if err := a.Comment(ctx, item.Resource, item.ExternalID, effect.Value); err != nil {
			return projectionError(err, true)
		}
		return nil
	default:
		return &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse, EffectNotCommitted: true}
	}
}

func (a Adapter) teamLabelNames(ctx context.Context, teamKey string) ([]string, error) {
	const query = `query TeamLabels($key: String!) {
  teams(filter: { key: { eq: $key } }, first: 1) {
    nodes { labels(first: 100) { nodes { name } } }
  }
}`
	data, err := a.graphql(ctx, query, map[string]any{"key": teamKey})
	if err != nil {
		return nil, err
	}
	var payload struct {
		Teams struct {
			Nodes []struct {
				Labels struct {
					Nodes []struct {
						Name string `json:"name"`
					} `json:"nodes"`
				} `json:"labels"`
			} `json:"nodes"`
		} `json:"teams"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || len(payload.Teams.Nodes) != 1 || len(payload.Teams.Nodes[0].Labels.Nodes) >= 100 {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	result := make([]string, 0, len(payload.Teams.Nodes[0].Labels.Nodes))
	for _, label := range payload.Teams.Nodes[0].Labels.Nodes {
		if label.Name == "" || len(label.Name) > 256 {
			return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
		}
		result = append(result, label.Name)
	}
	return result, nil
}

func (a Adapter) commentPresent(ctx context.Context, identifier, projectionKey string) (bool, error) {
	const query = `query IssueComments($id: String!) {
  issue(id: $id) { comments(first: 100) { nodes { body } } }
}`
	data, err := a.graphql(ctx, query, map[string]any{"id": identifier})
	if err != nil {
		return false, err
	}
	var payload struct {
		Issue struct {
			Comments struct {
				Nodes []struct {
					Body string `json:"body"`
				} `json:"nodes"`
			} `json:"comments"`
		} `json:"issue"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || len(payload.Issue.Comments.Nodes) >= 100 {
		return false, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	marker := "<!-- axiom:workflow-projection:" + projectionKey + " -->"
	present := false
	for _, comment := range payload.Issue.Comments.Nodes {
		if len(comment.Body) > bodyLimit {
			return false, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
		}
		if strings.Contains(comment.Body, marker) {
			if present {
				return false, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
			}
			present = true
		}
	}
	return present, nil
}

func (a Adapter) createLabel(ctx context.Context, teamKey, name string) error {
	teamID, err := a.teamID(ctx, teamKey)
	if err != nil {
		return projectionError(err, false)
	}
	const mutation = `mutation LabelCreate($input: IssueLabelCreateInput!) {
  issueLabelCreate(input: $input) { success }
}`
	data, err := a.graphql(ctx, mutation, map[string]any{"input": map[string]any{"teamId": teamID, "name": name, "color": "#5319e7"}})
	if err != nil {
		return projectionError(err, true)
	}
	var payload struct {
		IssueLabelCreate struct {
			Success bool `json:"success"`
		} `json:"issueLabelCreate"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || !payload.IssueLabelCreate.Success {
		return &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse, Ambiguous: true}
	}
	return nil
}

func (a Adapter) labelID(ctx context.Context, teamKey, name string) (string, error) {
	const query = `query TeamLabel($key: String!) {
  teams(filter: { key: { eq: $key } }, first: 1) {
    nodes { labels(first: 100) { nodes { id name } } }
  }
}`
	data, err := a.graphql(ctx, query, map[string]any{"key": teamKey})
	if err != nil {
		return "", err
	}
	var payload struct {
		Teams struct {
			Nodes []struct {
				Labels struct {
					Nodes []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"nodes"`
				} `json:"labels"`
			} `json:"nodes"`
		} `json:"teams"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || len(payload.Teams.Nodes) != 1 {
		return "", &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	matches := []string{}
	for _, label := range payload.Teams.Nodes[0].Labels.Nodes {
		if label.Name == name {
			matches = append(matches, label.ID)
		}
	}
	if len(matches) != 1 {
		if len(matches) > 1 {
			return "", &workitem.ProviderError{Kind: workitem.ProviderAmbiguous}
		}
		return "", &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	return matches[0], nil
}

func (a Adapter) addLabel(ctx context.Context, identifier, teamKey, name string) error {
	labelID, err := a.labelID(ctx, teamKey, name)
	if err != nil {
		return projectionError(err, false)
	}
	const mutation = `mutation IssueAddLabel($id: String!, $labelId: String!) {
  issueAddLabel(id: $id, labelId: $labelId) { success }
}`
	data, err := a.graphql(ctx, mutation, map[string]any{"id": identifier, "labelId": labelID})
	if err != nil {
		return projectionError(err, true)
	}
	var payload struct {
		IssueAddLabel struct {
			Success bool `json:"success"`
		} `json:"issueAddLabel"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || !payload.IssueAddLabel.Success {
		return &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse, Ambiguous: true}
	}
	return nil
}

func (a Adapter) removeLabel(ctx context.Context, identifier, teamKey, name string) error {
	labelID, err := a.labelID(ctx, teamKey, name)
	if err != nil {
		return projectionError(err, false)
	}
	const mutation = `mutation IssueRemoveLabel($id: String!, $labelId: String!) {
  issueRemoveLabel(id: $id, labelId: $labelId) { success }
}`
	data, err := a.graphql(ctx, mutation, map[string]any{"id": identifier, "labelId": labelID})
	if err != nil {
		return projectionError(err, true)
	}
	var payload struct {
		IssueRemoveLabel struct {
			Success bool `json:"success"`
		} `json:"issueRemoveLabel"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || !payload.IssueRemoveLabel.Success {
		return &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse, Ambiguous: true}
	}
	return nil
}

func (a Adapter) validProjection(item workflow.WorkItem, label, key string) bool {
	return a.validProjectionItem(item) && workflow.ValidProjectionLabel(label) && len(key) == 64
}

func (a Adapter) validProjectionItem(item workflow.WorkItem) bool {
	return item.Provider == "linear" && a.ValidResource(item.Resource) && a.ValidExternal(item.Resource, item.ExternalID, workitem.External{ID: item.ExternalID, URL: item.URL, State: item.State})
}

func projectionError(err error, ambiguous bool) error {
	var provider *workitem.ProviderError
	if errors.As(err, &provider) {
		kind := workflow.ProjectionUnavailable
		switch provider.Kind {
		case workitem.ProviderUnauthenticated:
			kind = workflow.ProjectionUnauthenticated
		case workitem.ProviderRateLimited:
			kind = workflow.ProjectionRateLimited
		case workitem.ProviderInvalidResponse, workitem.ProviderAmbiguous, workitem.ProviderClassificationUnsupported:
			kind = workflow.ProjectionInvalidResponse
		}
		return &workflow.ProjectionError{Kind: kind, Retryable: provider.Retryable, Ambiguous: ambiguous || provider.Ambiguous, EffectNotCommitted: provider.EffectNotCommitted}
	}
	return &workflow.ProjectionError{Kind: workflow.ProjectionInvalidResponse, Ambiguous: ambiguous}
}
