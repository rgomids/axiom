package githubissues

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workitem"
)

func DeliveryMetadata(repository, observation string, policy workitem.MetadataPolicy, explicit map[string][]string) (json.RawMessage, error) {
	resolution, err := workitem.ResolveMetadata(workitem.PullRequestSurface, policy, explicit, nil, map[string]bool{"classification": false, "owner": false, "reviewers": false})
	if err != nil {
		return nil, err
	}
	preview, err := PrepareMetadataPreview(workitem.PullRequestSurface, repository, observation, resolution)
	if err != nil || len(preview.Effects) != 0 || len(preview.Unsupported) != 0 {
		return nil, errors.New("pull request metadata unresolved or unsupported")
	}
	return json.Marshal(preview)
}

func (a Adapter) ObserveDeliveryPR(ctx context.Context, p workflow.DeliveryPreview) ([]workflow.DeliveryPR, bool, error) {
	t := p.Target
	if !a.ValidResource(t.Repository) {
		return nil, false, workflow.ErrDelivery
	}
	output, err := a.run(ctx, nil, "api", "--hostname", "github.com", "--method", "GET", "repos/"+t.Repository+"/pulls", "-f", "state=all", "-f", "head="+strings.Split(t.Repository, "/")[0]+":"+strings.TrimPrefix(t.HeadRef, "refs/heads/"), "-f", "per_page=100")
	if err != nil {
		return nil, false, err
	}
	var items []json.RawMessage
	if json.Unmarshal(output, &items) != nil || len(items) >= 100 {
		return nil, false, workflow.ErrRecoveryRequired
	}
	result := []workflow.DeliveryPR{}
	for _, wire := range items {
		r, err := decodeDeliveryPR(t.Repository, wire, t.Metadata)
		if err != nil {
			return nil, false, err
		}
		if r.HeadRef == t.HeadRef && r.BaseRef == t.BaseRef {
			result = append(result, r)
		}
	}
	return result, true, nil
}
func (a Adapter) CreateDeliveryPR(ctx context.Context, p workflow.DeliveryPreview) error {
	t := p.Target
	if !a.ValidResource(t.Repository) || t.Title == "" || len(p.PRBody()) > bodyLimit {
		return workflow.ErrDelivery
	}
	var metadata MetadataPreview
	if json.Unmarshal(t.Metadata, &metadata) != nil || metadata.Surface != workitem.PullRequestSurface || metadata.Target != t.Repository || len(metadata.Effects) != 0 || !(MetadataAuthority{Target: metadata.Target, ObservationDigest: metadata.ObservationDigest, PreviewDigest: metadata.Digest}).Covers(metadata) {
		return workflow.ErrDelivery
	}
	payload, err := json.Marshal(struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Base  string `json:"base"`
		Head  string `json:"head"`
		Draft bool   `json:"draft"`
	}{t.Title, p.PRBody(), strings.TrimPrefix(t.BaseRef, "refs/heads/"), strings.TrimPrefix(t.HeadRef, "refs/heads/"), true})
	if err != nil {
		return err
	}
	_, err = a.run(ctx, payload, "api", "--hostname", "github.com", "--method", "POST", "repos/"+t.Repository+"/pulls", "--input", "-")
	return err
}
func decodeDeliveryPR(repository string, wire []byte, metadata json.RawMessage) (workflow.DeliveryPR, error) {
	var raw struct {
		Number int    `json:"number"`
		URL    string `json:"html_url"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		Head   struct {
			Ref, SHA string
			Repo     struct {
				FullName string `json:"full_name"`
			}
		}
		Base struct {
			Ref, SHA string
			Repo     struct {
				FullName string `json:"full_name"`
			}
		}
	}
	if json.Unmarshal(wire, &raw) != nil || raw.Number < 1 || raw.Base.Repo.FullName != repository || raw.Head.Repo.FullName != repository {
		return workflow.DeliveryPR{}, workflow.ErrDelivery
	}
	r := workflow.DeliveryPR{ID: strconv.Itoa(raw.Number), URL: raw.URL, Repository: repository, BaseRef: "refs/heads/" + raw.Base.Ref, HeadRef: "refs/heads/" + raw.Head.Ref, BaseCommit: raw.Base.SHA, HeadCommit: raw.Head.SHA, Title: raw.Title, Body: raw.Body, Metadata: metadata}
	if r.URL != "https://github.com/"+repository+"/pull/"+r.ID {
		return r, workflow.ErrDelivery
	}
	return r, nil
}

func (a Adapter) DeliveryGitCredential(ctx context.Context) (string, error) {
	wire, err := a.run(ctx, nil, "auth", "token", "--hostname", "github.com")
	if err != nil {
		return "", errors.New("GitHub authentication unavailable")
	}
	token := strings.TrimSpace(string(wire))
	if token == "" || len(token) > 16384 || strings.ContainsAny(token, " \r\n\x00") {
		return "", errors.New("GitHub authentication unavailable")
	}
	return token, nil
}
