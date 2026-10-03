package githubissues

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/workitem"
)

type createMetadata struct {
	Labels []string `json:"labels"`
}

// Verification is read-only, including after a reconciled or confirmed retry.
// A dropped label must never become a successful classification claim.
func (a Adapter) VerifyDocument(ctx context.Context, request workitem.CreateRequest, external workitem.External) (bool, error) {
	metadata, err := decodeCreateMetadata(request.Document.Metadata)
	if err != nil {
		return false, err
	}
	if len(metadata.Labels) == 0 {
		return true, nil
	}
	if !a.ValidExternal(request.Resource, external.ID, external) {
		return false, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	output, err := a.run(ctx, nil, "api", "repos/"+request.Resource+"/issues/"+external.ID)
	if err != nil {
		return false, err
	}
	if _, err := decodeIssue(request.Resource, external.ID, output); err != nil {
		return false, err
	}
	var response struct {
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return false, err
	}
	labels := make([]string, 0, len(response.Labels))
	for _, label := range response.Labels {
		labels = append(labels, label.Name)
	}
	return labelsExist(metadata.Labels, labels), nil
}

func decodeCreateMetadata(raw json.RawMessage) (createMetadata, error) {
	var metadata createMetadata
	if len(raw) == 0 {
		return metadata, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&metadata); err != nil {
		return metadata, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return metadata, fmt.Errorf("invalid metadata")
	}
	if len(metadata.Labels) > 16 {
		return metadata, fmt.Errorf("too many labels")
	}
	seen := map[string]bool{}
	for _, label := range metadata.Labels {
		if !validCreationLabel(label) || seen[label] {
			return metadata, fmt.Errorf("invalid label")
		}
		seen[label] = true
	}
	return metadata, nil
}

func validCreationLabel(label string) bool {
	if label == "" || len(label) > 256 || !utf8.ValidString(label) || strings.TrimSpace(label) != label {
		return false
	}
	for _, r := range label {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// ClassifyDocument only uses the repository's observed label catalog. No label
// creation, workflow-stage inference, or GitHub issue-type dependency is implied.
func (a Adapter) ClassifyDocument(ctx context.Context, draft workitem.Draft, target workitem.DraftTarget, document workitem.ProviderDocument, explicit []string) (workitem.ProviderDocument, error) {
	if !a.ValidResource(target.Resource) || !draft.Type.Valid() || len(explicit) > 16 {
		return document, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	available, err := a.creationLabels(ctx, target.Resource)
	if err != nil {
		return document, err
	}
	selected := make([]string, 0)
	if len(explicit) != 0 {
		if !labelsExist(explicit, available) {
			return document, &workitem.ProviderError{Kind: workitem.ProviderClassificationUnsupported, EffectNotCommitted: true}
		}
		seen := map[string]bool{}
		for _, label := range explicit {
			if !seen[label] {
				selected = append(selected, label)
				seen[label] = true
			}
		}
	} else {
		candidates := []string{"type:" + string(draft.Type), string(draft.Type)}
		if draft.Type == workitem.Story {
			candidates = append(candidates, "enhancement")
		}
		for _, candidate := range candidates {
			for _, label := range available {
				if strings.EqualFold(candidate, label) {
					selected = append(selected, label)
					break
				}
			}
			if len(selected) != 0 {
				break
			}
		}
		if len(selected) == 0 {
			document.Notices = []string{"no_existing_label_for_item_type"}
		}
	}
	sort.Strings(selected)
	document.Metadata, err = json.Marshal(createMetadata{Labels: selected})
	return document, err
}

func labelsExist(requested, available []string) bool {
	set := make(map[string]bool, len(available))
	for _, label := range available {
		set[label] = true
	}
	for _, label := range requested {
		if !validCreationLabel(label) || !set[label] {
			return false
		}
	}
	return true
}

// Bound pagination and aggregate size; reaching the bound fails explicitly,
// rather than selecting from an incomplete catalog.
func (a Adapter) creationLabels(ctx context.Context, resource string) ([]string, error) {
	labels := make([]string, 0)
	seen := map[string]bool{}
	for page := 1; page <= 10; page++ {
		output, err := a.run(ctx, nil, "api", fmt.Sprintf("repos/%s/labels?per_page=100&page=%d", resource, page))
		if err != nil {
			return nil, err
		}
		var rows []struct {
			Name string `json:"name"`
		}
		output = bytes.TrimSpace(output)
		if len(output) == 0 || output[0] != '[' || json.Unmarshal(output, &rows) != nil || len(rows) > 100 {
			return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
		}
		for _, row := range rows {
			if !validCreationLabel(row.Name) || seen[row.Name] {
				return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
			}
			labels = append(labels, row.Name)
			seen[row.Name] = true
		}
		if len(rows) < 100 {
			return labels, nil
		}
	}
	return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
}
