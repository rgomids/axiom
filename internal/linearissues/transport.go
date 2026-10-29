package linearissues

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/workitem"
)

const (
	defaultAPIURL = "https://api.linear.app/graphql"
	outputLimit   = 256 * 1024
	bodyLimit     = 64 * 1024
)

// Adapter is the Linear Work Item provider. Empty bin uses HTTPS GraphQL;
// a non-empty absolute bin is preferred for tests and optional wrappers.
type Adapter struct {
	bin     string
	apiURL  string
	timeout time.Duration
	client  *http.Client
	getenv  func(string) string
}

// New resolves the optional helper binary. An empty path selects the built-in
// HTTPS GraphQL transport (LINEAR_API_KEY required at call time).
func New(bin string) (Adapter, error) {
	if bin == "" {
		return Adapter{apiURL: defaultAPIURL, timeout: 15 * time.Second, getenv: os.Getenv, client: &http.Client{Timeout: 15 * time.Second}}, nil
	}
	if !filepath.IsAbs(bin) {
		return Adapter{}, errors.New("provider binary must be absolute")
	}
	return Adapter{bin: bin, apiURL: defaultAPIURL, timeout: 15 * time.Second, getenv: os.Getenv, client: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (Adapter) ProviderID() string { return "linear" }

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlEnvelope struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (a Adapter) graphql(parent context.Context, query string, variables map[string]any) ([]byte, error) {
	payload, err := json.Marshal(graphqlRequest{Query: query, Variables: variables})
	if err != nil {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, EffectNotCommitted: true}
	}
	if a.bin != "" {
		return a.runBin(parent, payload)
	}
	return a.runHTTP(parent, payload)
}

func (a Adapter) runBin(parent context.Context, payload []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, a.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, a.bin)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderUnavailable, Retryable: true, EffectNotCommitted: true}
	}
	if err != nil {
		msg := strings.ToLower(stderr.String() + err.Error())
		kind := workitem.ProviderUnavailable
		if strings.Contains(msg, "auth") || strings.Contains(msg, "unauthorized") || strings.Contains(msg, "401") {
			kind = workitem.ProviderUnauthenticated
		}
		return nil, &workitem.ProviderError{Kind: kind, Retryable: kind == workitem.ProviderUnavailable, EffectNotCommitted: false, Ambiguous: true}
	}
	out := stdout.Bytes()
	if len(out) > outputLimit {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, Ambiguous: true}
	}
	return decodeEnvelope(out)
}

func (a Adapter) runHTTP(parent context.Context, payload []byte) ([]byte, error) {
	getenv := a.getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	key := strings.TrimSpace(getenv("LINEAR_API_KEY"))
	if key == "" || strings.ContainsAny(key, "\r\n\x00") {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderUnauthenticated, EffectNotCommitted: true}
	}
	ctx, cancel := context.WithTimeout(parent, a.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, EffectNotCommitted: true}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", key)
	client := a.client
	if client == nil {
		client = &http.Client{Timeout: a.timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, &workitem.ProviderError{Kind: workitem.ProviderUnavailable, Retryable: true, EffectNotCommitted: true}
		}
		return nil, &workitem.ProviderError{Kind: workitem.ProviderUnavailable, Retryable: true, Ambiguous: true}
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, outputLimit+1)
	body, err := io.ReadAll(limited)
	if err != nil || len(body) > outputLimit {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, Ambiguous: true}
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderUnauthenticated, EffectNotCommitted: resp.StatusCode == http.StatusUnauthorized}
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderRateLimited, Retryable: true, Ambiguous: true}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderUnavailable, Retryable: resp.StatusCode >= 500, Ambiguous: true}
	}
	return decodeEnvelope(body)
}

func decodeEnvelope(raw []byte) ([]byte, error) {
	var envelope graphqlEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, Ambiguous: true}
	}
	if len(envelope.Errors) != 0 {
		msg := strings.ToLower(envelope.Errors[0].Message)
		kind := workitem.ProviderInvalidResponse
		if strings.Contains(msg, "auth") || strings.Contains(msg, "unauthorized") {
			kind = workitem.ProviderUnauthenticated
		}
		return nil, &workitem.ProviderError{Kind: kind, Ambiguous: true}
	}
	if len(envelope.Data) == 0 || !json.Valid(envelope.Data) {
		return nil, &workitem.ProviderError{Kind: workitem.ProviderInvalidResponse, Ambiguous: true}
	}
	return envelope.Data, nil
}
