package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
)

func capabilityPreviewDigest(p any) (string, error) {
	wire, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(wire)
	return hex.EncodeToString(sum[:]), nil
}

func (s lifecycleService) RuntimeCapability(ctx context.Context, in cli.RuntimeCapabilityInput) cli.RuntimeCapabilityResponse {
	service, err := productionCapabilityService(s.stateRoot, s.runtimes.lookPath, stageOperation{repository: in.RepositoryRoot, workspaceRoot: in.WorkspaceRoot, consented: in.AuthorizeLocal})
	if err != nil {
		return s.capabilityFailure("capability_unknown", nil)
	}
	return s.capabilityOperation(ctx, service, in)
}

func (s lifecycleService) capabilityFailure(category string, payload any) cli.RuntimeCapabilityResponse {
	r := canonicalCompletion(completion.Facts{ValidationFailed: true}, "Capability operation refused: "+category, nil, "Inspect the exact binding and authority; no capability is inferred", s.provenance)
	r.Category = category
	return cli.RuntimeCapabilityResponse{Result: r, Payload: payload}
}

func (s lifecycleService) capabilitySuccess(category string, payload any) cli.RuntimeCapabilityResponse {
	r := canonicalCompletion(completion.Facts{Completed: true}, "Capability observation: "+category, nil, "Inspect Evidence freshness, variant and unknown states before dispatch", s.provenance)
	r.Category = category
	return cli.RuntimeCapabilityResponse{Result: r, Payload: payload}
}

func (s lifecycleService) capabilityOperation(ctx context.Context, service capabilityproof.Service, in cli.RuntimeCapabilityInput) cli.RuntimeCapabilityResponse {
	target := capabilityproof.Target{RuntimeID: in.RuntimeID, ProfileID: in.ProfileID, Variant: in.Variant, Process: in.Process}
	switch in.Operation {
	case "show":
		view, err := service.Show(ctx, target)
		if err != nil {
			return s.capabilityFailure("capability_unknown", view)
		}
		return s.capabilitySuccess("capability_observed", view)
	case "invalidate":
		if in.PreviewDigest != "" && !in.AuthorizeLocal {
			return s.capabilityFailure("local_authority_required", nil)
		}
		p, err := service.PreviewInvalidate(ctx, target)
		if err != nil {
			return s.capabilityFailure("capability_unknown", nil)
		}
		digest, err := capabilityPreviewDigest(p)
		if err != nil {
			return s.capabilityFailure("capability_unknown", nil)
		}
		if in.PreviewDigest == "" {
			return s.capabilitySuccess("capability_preview_ready", map[string]any{"previewDigest": digest, "preview": p})
		}
		if digest != in.PreviewDigest {
			return s.capabilityFailure("stale_preview", nil)
		}
		if err := service.Invalidate(ctx, p, capabilityproof.LocalAuthority{Actor: stageActor(), Preview: p}); err != nil {
			return s.capabilityFailure("stale_preview", nil)
		}
		return s.capabilitySuccess("capability_invalidated", map[string]string{"evidenceRef": p.EvidenceRef})
	default:
		return s.capabilityFailure("invalid_command", nil)
	}
}
