package main

import (
	"context"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
)

func TestApplyOperationalRendersEveryOutcomeCanonically(t *testing.T) {
	env := newAdmissionEnv(t)
	service := env.service.(lifecycleService)
	request := projectapp.OperationalRequest{ProjectID: env.projectID, Operation: projectapp.ArchiveProject}
	preview := service.applyOperational(context.Background(), request, "", false, "Project archive")
	if preview.Operational == nil || preview.Category != projectapp.OperationalPreviewReady || preview.Completion.Status() != completion.Success {
		t.Fatalf("preview = %+v", preview)
	}
	if denied := service.applyOperational(context.Background(), request, "stale", true, "Project archive"); denied.Category != projectapp.OperationalAuthorityDenied || denied.Completion.Status() != completion.DeniedAuthority {
		t.Fatalf("denied = %+v", denied)
	}
	applied := service.applyOperational(context.Background(), request, preview.Operational.Digest, true, "Project archive")
	if applied.Category != projectapp.OperationalApplied || applied.Completion.Status() != completion.Success || applied.Operational.Result.ProjectStatus != projectapp.ProjectArchived {
		t.Fatalf("applied = %+v", applied)
	}
	if again := service.applyOperational(context.Background(), request, "", false, "Project archive"); again.Category != projectapp.ProjectAlreadyArchived || len(again.Operational.Effects) != 0 {
		t.Fatalf("no-op = %+v", again)
	}
	missing := service.applyOperational(context.Background(), projectapp.OperationalRequest{ProjectID: "123e4567-e89b-42d3-a456-426614174999", Operation: projectapp.ArchiveProject}, "", false, "Project archive")
	if missing.Category != projectapp.OperationalNotInstalled || missing.Completion.Status() != completion.ValidationFailure {
		t.Fatalf("not installed = %+v", missing)
	}
}
