package stageexecution

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/validatorpolicy"
)

type memoryPolicies map[string]validatorpolicy.Policy

func (s memoryPolicies) Create(_ context.Context, p validatorpolicy.Policy) (string, error) {
	id, err := validatorpolicy.Digest(p)
	if err == nil {
		s[id] = p
	}
	return id, err
}
func (s memoryPolicies) Load(_ context.Context, id string) (validatorpolicy.Policy, error) {
	p, ok := s[id]
	if !ok {
		return p, validatorpolicy.ErrInvalid
	}
	return p, nil
}
func TestPolicyPreviewApplyRefusesAgentAuthorityAndDrift(t *testing.T) {
	ctx := context.Background()
	store := memoryPolicies{}
	service := ValidatorPolicyService{Store: store, Now: func() time.Time { return time.Unix(10, 0) }}
	path, _ := filepath.Abs("synthetic-validator")
	d := validatorpolicy.Definition{FormatVersion: 1, ProjectID: "project-1", Commands: []validatorpolicy.Command{{Reference: "test", Argv: []string{path, "test"}, Env: []string{}, OutputMax: 4096}}}
	preview, err := service.Preview(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(store) != 0 {
		t.Fatal("preview wrote state")
	}
	request := ValidatorPolicyApply{Definition: d, PreviewDigest: preview, AuthorizeLocal: true, Source: "human-operator", Actor: "operator", AuthorityRef: strings.Repeat("a", 64)}
	for _, source := range []string{"agent-record", "plan-document", ""} {
		r := request
		r.Source = source
		if _, err := service.Apply(ctx, r); !errors.Is(err, validatorpolicy.ErrAuthority) {
			t.Fatal(source, err)
		}
	}
	r := request
	r.AuthorizeLocal = false
	if _, err := service.Apply(ctx, r); !errors.Is(err, validatorpolicy.ErrAuthority) {
		t.Fatal(err)
	}
	r = request
	r.PreviewDigest = strings.Repeat("b", 64)
	if _, err := service.Apply(ctx, r); !errors.Is(err, validatorpolicy.ErrDigest) {
		t.Fatal(err)
	}
	if len(store) != 0 {
		t.Fatal("denied requests wrote state")
	}
	id, err := service.Apply(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	commands, err := service.Commands(ctx, "project-1", id)
	if err != nil || commands[0].Argv[0] != path {
		t.Fatal(commands, err)
	}
	if _, err := service.Commands(ctx, "other", id); !errors.Is(err, validatorpolicy.ErrAuthority) {
		t.Fatal(err)
	}
	p := store[id]
	p.Commands[0].Argv[1] = "unreviewed"
	store[id] = p
	if _, err := service.Commands(ctx, "project-1", id); err == nil {
		t.Fatal("digest drift permitted")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := service.Apply(cancelled, request); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
