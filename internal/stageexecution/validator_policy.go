package stageexecution

import (
	"context"
	"time"

	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/validatorpolicy"
)

type ValidatorPolicyStore interface {
	Create(context.Context, validatorpolicy.Policy) (string, error)
	Load(context.Context, string) (validatorpolicy.Policy, error)
}

type ValidatorPolicyService struct {
	Store ValidatorPolicyStore
	Now   func() time.Time
}
type ValidatorPolicyApply struct {
	Definition     validatorpolicy.Definition
	PreviewDigest  string
	AuthorizeLocal bool
	Source         string
	Actor          string
	AuthorityRef   string
}

func (s ValidatorPolicyService) Preview(d validatorpolicy.Definition) (string, error) {
	return validatorpolicy.PreviewDigest(d)
}

// Apply accepts only a separately confirmed operator policy, never an agent
// record or a Plan. A stale review cannot authorize changed argv or environment.
func (s ValidatorPolicyService) Apply(ctx context.Context, r ValidatorPolicyApply) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !r.AuthorizeLocal || r.Source != "human-operator" || s.Store == nil {
		return "", validatorpolicy.ErrAuthority
	}
	actual, e := s.Preview(r.Definition)
	if e != nil {
		return "", e
	}
	if actual != r.PreviewDigest {
		return "", validatorpolicy.ErrDigest
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	p := validatorpolicy.Policy{Definition: r.Definition, Review: validatorpolicy.Review{PreviewDigest: actual, ActorKind: "human-operator", Actor: r.Actor, AuthorityRef: r.AuthorityRef, ReviewedAt: now().UTC()}}
	if validatorpolicy.Validate(p) != nil {
		return "", validatorpolicy.ErrInvalid
	}
	return s.Store.Create(ctx, p)
}
func (s ValidatorPolicyService) Commands(ctx context.Context, projectID, digest string) ([]gitworkspace.ValidationCommand, error) {
	if s.Store == nil {
		return nil, validatorpolicy.ErrInvalid
	}
	p, e := s.Store.Load(ctx, digest)
	if e != nil {
		return nil, e
	}
	if p.ProjectID != projectID {
		return nil, validatorpolicy.ErrAuthority
	}
	return validatorpolicy.Commands(p, digest)
}
