// Package githubissues adapts the GitHub Issues capability without leaking
// GitHub transport or formatting into Work Item application semantics.
package githubissues

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workitem"
)

const (
	outputLimit = 256 * 1024
	bodyLimit   = 64 * 1024
)

type Adapter struct {
	gh      string
	timeout time.Duration
}

func New(ghBinary string) (Adapter, error) {
	if ghBinary == "" {
		var err error
		ghBinary, err = exec.LookPath("gh")
		if err != nil {
			return Adapter{}, err
		}
	}
	if !strings.HasPrefix(ghBinary, "/") {
		return Adapter{}, errors.New("provider binary must be absolute")
	}
	return Adapter{gh: ghBinary, timeout: 15 * time.Second}, nil
}

func (Adapter) ProviderID() string { return "github" }

var segment = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func (Adapter) ValidResource(value string) bool {
	parts := strings.Split(value, "/")
	return len(parts) == 2 && parts[0] != "." && parts[0] != ".." && parts[1] != "." && parts[1] != ".." && segment.MatchString(parts[0]) && segment.MatchString(parts[1])
}

func (a Adapter) ValidExternal(resource, selector string, external workitem.External) bool {
	if !a.ValidResource(resource) || selector == "" || external.ID != selector {
		return false
	}
	number, err := strconv.Atoi(selector)
	return err == nil && number > 0 && external.URL == "https://github.com/"+resource+"/issues/"+selector && (external.State == "OPEN" || external.State == "CLOSED")
}

func (a Adapter) Render(draft workitem.Draft, _ workitem.DraftTarget, correlation string, source provenance.Value) (workitem.ProviderDocument, error) {
	if len(correlation) != 64 || !source.Valid() || len(draft.Sections) != 7 {
		return workitem.ProviderDocument{}, errors.New("invalid draft")
	}
	outcome := section(draft, "desired_outcome")
	if outcome == "" {
		return workitem.ProviderDocument{}, errors.New("missing outcome")
	}
	title := "Axiom: " + oneLine(outcome, 180)
	var body strings.Builder
	fmt.Fprintf(&body, "<!-- axiom:work-item-draft:%s -->\n", correlation)
	fmt.Fprintf(&body, "<!-- axiom:provenance:%s:%s:%s:%s -->\n\n", source.Product(), source.Version(), source.Revision(), source.SourceState())
	body.WriteString("_Axiom-authored structure; section content retains declared authorship._\n")
	for _, current := range draft.Sections {
		fmt.Fprintf(&body, "\n## %s\n\nAuthorship: `%s`\n\n", heading(current.Name), current.Authorship)
		for _, line := range strings.Split(current.Content, "\n") {
			body.WriteString("    ")
			body.WriteString(line)
			body.WriteByte('\n')
		}
	}
	if body.Len() > bodyLimit {
		return workitem.ProviderDocument{}, errors.New("provider body exceeds limit")
	}
	return workitem.ProviderDocument{Title: title, Body: body.String()}, nil
}

func (a Adapter) Create(parent context.Context, request workitem.CreateRequest) (workitem.External, error) {
	if !a.ValidResource(request.Resource) || len(request.Correlation) != 64 || request.Document.Title == "" || request.Document.Body == "" || len(request.Document.Body) > bodyLimit {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	payload, err := json.Marshal(map[string]string{"title": request.Document.Title, "body": request.Document.Body})
	if err != nil {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	output, err := a.run(parent, payload, "api", "--method", "POST", "repos/"+request.Resource+"/issues", "--input", "-")
	if err != nil {
		var provider *workitem.ProviderError
		if errors.As(err, &provider) {
			provider.Ambiguous = true
		}
		return workitem.External{}, err
	}
	return decodeIssue(request.Resource, "", output)
}

func (a Adapter) ReconcileCreate(ctx context.Context, repository, correlation string) ([]workitem.External, error) {
	if !a.ValidResource(repository) || len(correlation) != 64 {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	marker := "axiom:work-item-draft:" + correlation
	query := marker + " repo:" + repository + " in:body type:issue"
	output, err := a.run(ctx, nil, "api", "--method", "GET", "search/issues", "-f", "q="+query, "-f", "per_page=10")
	if err != nil {
		return nil, err
	}
	var response struct {
		TotalCount int               `json:"total_count"`
		Items      []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(output, &response); err != nil || response.TotalCount != len(response.Items) || len(response.Items) > 10 {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	items := make([]workitem.External, 0, len(response.Items))
	for _, raw := range response.Items {
		var inspected struct {
			Body string `json:"body"`
		}
		if err := json.Unmarshal(raw, &inspected); err != nil || !strings.Contains(inspected.Body, "<!-- "+marker+" -->") {
			return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
		}
		item, err := decodeIssue(repository, "", raw)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (a Adapter) Read(ctx context.Context, repository, selector string) (workitem.External, error) {
	number, parseErr := strconv.Atoi(selector)
	if !a.ValidResource(repository) || parseErr != nil || number <= 0 {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	output, err := a.run(ctx, nil, "api", "repos/"+repository+"/issues/"+strconv.Itoa(number))
	if err != nil {
		return workitem.External{}, err
	}
	return decodeIssue(repository, selector, output)
}

// Historical POC-only operations. S3 does not invoke them.
func (a Adapter) Comment(ctx context.Context, repository, selector, message string) error {
	number, parseErr := strconv.Atoi(selector)
	if !a.ValidResource(repository) || parseErr != nil || number <= 0 || message == "" {
		return &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	payload, _ := json.Marshal(map[string]string{"body": message})
	_, err := a.run(ctx, payload, "api", "--method", "POST", "repos/"+repository+"/issues/"+strconv.Itoa(number)+"/comments", "--input", "-")
	return err
}

func (a Adapter) Close(ctx context.Context, repository, selector string) (workitem.External, error) {
	number, parseErr := strconv.Atoi(selector)
	if !a.ValidResource(repository) || parseErr != nil || number <= 0 {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	payload, _ := json.Marshal(map[string]string{"state": "closed"})
	output, err := a.run(ctx, payload, "api", "--method", "PATCH", "repos/"+repository+"/issues/"+strconv.Itoa(number), "--input", "-")
	if err != nil {
		return workitem.External{}, err
	}
	return decodeIssue(repository, selector, output)
}

func (a Adapter) run(parent context.Context, input []byte, arguments ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, a.timeout)
	defer cancel()
	var stdout, stderr boundedBuffer
	command := exec.CommandContext(ctx, a.gh, arguments...)
	command.Stdin = bytes.NewReader(input)
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if errors.Is(stdout.err, errOutputLimit) || errors.Is(stderr.err, errOutputLimit) {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	if err == nil {
		return append([]byte(nil), stdout.Bytes()...), nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderAmbiguous, Retryable: true, Ambiguous: true}
	}
	lower := strings.ToLower(stderr.String())
	if strings.Contains(lower, "rate limit") || strings.Contains(lower, "http 429") {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderRateLimited, Retryable: true}
	}
	if strings.Contains(lower, "authentication") || strings.Contains(lower, "http 401") || strings.Contains(lower, "bad credentials") {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderUnauthenticated}
	}
	return nil, &workitem.ProviderError{Kind: workitem.ProviderUnavailable, Retryable: true}
}

func decodeIssue(repository, expected string, source []byte) (workitem.External, error) {
	var response struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
	}
	if err := json.Unmarshal(source, &response); err != nil || response.Number <= 0 {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	selector := strconv.Itoa(response.Number)
	if expected != "" && selector != expected {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	state := strings.ToUpper(response.State)
	external := workitem.External{ID: selector, URL: response.HTMLURL, State: state}
	if !validIssue(repository, external) {
		return workitem.External{}, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse}
	}
	return external, nil
}

func validIssue(repository string, external workitem.External) bool {
	return external.URL == "https://github.com/"+repository+"/issues/"+external.ID && (external.State == "OPEN" || external.State == "CLOSED")
}

func section(draft workitem.Draft, name string) string {
	for _, current := range draft.Sections {
		if current.Name == name {
			return current.Content
		}
	}
	return ""
}

func oneLine(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return strings.TrimSpace(string(runes[:limit-3])) + "..."
}

func heading(value string) string {
	return map[string]string{
		"problem": "Problem", "desired_outcome": "Desired outcome", "context": "Context",
		"scope": "Scope", "constraints": "Constraints", "non_goals": "Non-goals",
		"acceptance_expectations": "Acceptance expectations",
	}[value]
}

var errOutputLimit = errors.New("provider output exceeded limit")

type boundedBuffer struct {
	bytes.Buffer
	err error
}

func (b *boundedBuffer) Write(input []byte) (int, error) {
	remaining := outputLimit - b.Len()
	if remaining <= 0 {
		b.err = errOutputLimit
		return 0, b.err
	}
	if len(input) > remaining {
		_, _ = b.Buffer.Write(input[:remaining])
		b.err = errOutputLimit
		return remaining, b.err
	}
	return b.Buffer.Write(input)
}
