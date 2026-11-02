package main

import (
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

// fakeGitHubStateVariable switches the test binary into a stateful fake `gh`
// so the executable exercises the real GitHub adapter without a network.
const fakeGitHubStateVariable = "AXIOM_TEST_FAKE_GH_STATE"

func TestMain(m *testing.M) {
	if path := os.Getenv(fakeGitHubStateVariable); path != "" {
		os.Exit(runFakeGitHub(path, os.Args[1:], os.Stdin, os.Stdout))
	}
	os.Exit(m.Run())
}

type fakeGitHub struct {
	IssueNumber      int      `json:"issueNumber,omitempty"`
	RepositoryLabels []string `json:"repositoryLabels"`
	IssueLabels      []string `json:"issueLabels"`
	Comments         []string `json:"comments"`
	IssueCreated     bool     `json:"issueCreated"`
	Mutations        []string `json:"mutations"`
}

func runFakeGitHub(path string, args []string, stdin io.Reader, stdout io.Writer) int {
	wire, err := os.ReadFile(path)
	if err != nil {
		return 1
	}
	var state fakeGitHub
	if err := json.Unmarshal(wire, &state); err != nil {
		return 1
	}
	method, endpoint := "GET", ""
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "api", "--include":
		case "--method":
			index++
			method = args[index]
		case "--input", "-f":
			index++
		default:
			if endpoint == "" {
				endpoint = args[index]
			}
		}
	}
	body, _ := io.ReadAll(stdin)
	number := state.IssueNumber
	if number == 0 {
		number = 7
	}
	issue := "repos/owner/repo/issues/" + strconv.Itoa(number)
	issueURL := "https://github.com/owner/repo/issues/" + strconv.Itoa(number)
	names := func(values []string) []map[string]string {
		result := make([]map[string]string, 0, len(values))
		for _, value := range values {
			result = append(result, map[string]string{"name": value})
		}
		return result
	}
	var response any
	switch {
	case method == "GET" && endpoint == "search/issues":
		response = map[string]any{"total_count": 0, "items": []any{}}
	case method == "POST" && endpoint == "repos/owner/repo/issues":
		var request struct{ Labels []string }
		if json.Unmarshal(body, &request) != nil {
			return 1
		}
		state.IssueLabels = append([]string(nil), request.Labels...)
		state.IssueCreated = true
		state.Mutations = append(state.Mutations, "create_issue")
		response = map[string]any{"number": number, "html_url": issueURL, "state": "open", "labels": names(state.IssueLabels)}
	case method == "GET" && strings.HasPrefix(endpoint, "repos/owner/repo/labels?per_page=100"):
		response = names(state.RepositoryLabels)
	case !state.IssueCreated:
		return 1
	case method == "POST" && endpoint == "repos/owner/repo/labels":
		var label struct{ Name string }
		if json.Unmarshal(body, &label) != nil || label.Name == "" {
			return 1
		}
		state.RepositoryLabels = append(state.RepositoryLabels, label.Name)
		state.Mutations = append(state.Mutations, "create_label:"+label.Name)
		response = map[string]string{"name": label.Name}
	case method == "GET" && endpoint == issue:
		response = map[string]any{"number": number, "html_url": issueURL, "state": "open", "labels": names(state.IssueLabels)}
	case method == "POST" && endpoint == issue+"/labels":
		var request struct{ Labels []string }
		if json.Unmarshal(body, &request) != nil || len(request.Labels) != 1 {
			return 1
		}
		state.IssueLabels = append(state.IssueLabels, request.Labels[0])
		state.Mutations = append(state.Mutations, "add_label:"+request.Labels[0])
		response = names(state.IssueLabels)
	case method == "DELETE" && strings.HasPrefix(endpoint, issue+"/labels/"):
		name := strings.TrimPrefix(endpoint, issue+"/labels/")
		kept := []string{}
		for _, label := range state.IssueLabels {
			if label != name {
				kept = append(kept, label)
			}
		}
		if len(kept) == len(state.IssueLabels) {
			return 1
		}
		state.IssueLabels = kept
		state.Mutations = append(state.Mutations, "remove_label:"+name)
		response = names(state.IssueLabels)
	case method == "GET" && endpoint == issue+"/comments?per_page=100":
		comments := make([]map[string]string, 0, len(state.Comments))
		for _, comment := range state.Comments {
			comments = append(comments, map[string]string{"body": comment})
		}
		response = comments
	case method == "POST" && endpoint == issue+"/comments":
		var comment struct{ Body string }
		if json.Unmarshal(body, &comment) != nil || comment.Body == "" {
			return 1
		}
		state.Comments = append(state.Comments, comment.Body)
		state.Mutations = append(state.Mutations, "comment")
		response = map[string]int{"id": len(state.Comments)}
	default:
		return 1
	}
	updated, err := json.Marshal(state)
	if err != nil || os.WriteFile(path, updated, 0o600) != nil {
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(response); err != nil {
		return 1
	}
	return 0
}

// TestExecutableClaudeRuntimeAndFirstProjectionOfCreatedWorkItem reproduces
// both Major findings of the S9 dogfood against v0.1.2-rc.1: an Execution
// conducted by Claude was persisted as codex, and `work-item create` →
// `workflow start` → first transition → `workflow reconcile` required
// recovery because the new Issue carried no axiom:stage:* marker.
// Both concrete Runtimes start only through the protocol the installed Work
// Item skills teach, on the Runtime Lingo itself observes.
