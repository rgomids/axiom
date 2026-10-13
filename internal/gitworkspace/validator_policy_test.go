package gitworkspace_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/validatorpolicy"
)

// The concrete local store has separate publication/recovery tests. This test
// isolates reviewed policy mapping and execution in real temporary worktrees.
type integrationPolicyStore map[string]validatorpolicy.Policy

func (s integrationPolicyStore) Create(_ context.Context, p validatorpolicy.Policy) (string, error) {
	id, err := validatorpolicy.Digest(p)
	if err == nil {
		s[id] = p
	}
	return id, err
}
func (s integrationPolicyStore) Load(_ context.Context, id string) (validatorpolicy.Policy, error) {
	p, ok := s[id]
	if !ok {
		return p, validatorpolicy.ErrInvalid
	}
	return p, nil
}
func TestReviewedValidatorPolicyRunsDuringIntegration(t *testing.T) {
	ctx := context.Background()
	fixture := newGitFixture(t, graphOptions{})
	fixture.prepare(t)
	before := fixture.sharedState(t)
	graph := fixture.succeed(t, "a", "b")
	writeFile(t, filepath.Join(fixture.childFrom(graph, "a").Envelope.Workspace, "a.txt"), "child-a\n")
	writeFile(t, filepath.Join(fixture.childFrom(graph, "b").Envelope.Workspace, "b.txt"), "child-b\n")
	store := integrationPolicyStore{}
	service := stageexecution.ValidatorPolicyService{Store: store, Now: func() time.Time { return time.Unix(10, 0) }}
	definition := validatorpolicy.Definition{FormatVersion: 1, ProjectID: graph.Children[0].Envelope.Scope.ProjectID, Commands: []validatorpolicy.Command{{Reference: "reviewed-diff-check", Argv: []string{nativeGit(t), "diff", "--check"}, Env: []string{"LC_ALL=C"}, OutputMax: 4096}}}
	preview, err := service.Preview(definition)
	if err != nil {
		t.Fatal(err)
	}
	id, err := service.Apply(ctx, stageexecution.ValidatorPolicyApply{Definition: definition, PreviewDigest: preview, AuthorizeLocal: true, Source: "human-operator", Actor: "operator", AuthorityRef: strings.Repeat("a", 64)})
	if err != nil {
		t.Fatal(err)
	}
	commands, err := service.Commands(ctx, definition.ProjectID, id)
	if err != nil {
		t.Fatal(err)
	}
	validator, err := gitworkspace.NewCombinedValidator(fixture.manager, commands)
	if err != nil {
		t.Fatal(err)
	}
	observation, results, _, err := fixture.manager.ObserveIntegration(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	integration := executiongraph.NewIntegrationService(fixture.manager, validator, fixture.store(), nil, nil)
	ip, err := integration.Preview(graph, observation, results)
	if err != nil {
		t.Fatal(err)
	}
	result, err := integration.Execute(ctx, graph, ip, executiongraph.IntegrationAuthority{PreviewDigest: ip.Digest, TargetRevision: ip.TargetRevision, Effects: ip.Effects, Reference: "authority:integration"})
	if err != nil || result.Rollup.Status != "success" || len(result.Rollup.ValidationResults) != 1 || result.Rollup.ValidationResults[0].CommandReference != "reviewed-diff-check" || result.Rollup.ValidationResults[0].ExitCode != 0 {
		t.Fatal(result.Rollup, err)
	}
	if fixture.sharedState(t) != before {
		t.Fatal("shared checkout changed")
	}
}
