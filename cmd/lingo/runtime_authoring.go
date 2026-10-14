package main

import (
	"context"
	"os/exec"
	"path/filepath"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimebootstrap"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type runtimeAuthoringStore struct{ local.RuntimeProfileStore }

func authoringSnapshot(s local.RuntimeProfileSnapshot) runtimeapplication.LocalSnapshot {
	return runtimeapplication.LocalSnapshot{State: s.State, Revision: s.Revision, Legacy: s.Legacy, V3: s.V3, Token: s.Token}
}

// Inspect reports canonical v3 state; a pre-stable record is carried only for
// its explicit retirement (Specification 007 PM-11).
func (s runtimeAuthoringStore) Inspect(ctx context.Context) (runtimeapplication.LocalSnapshot, error) {
	v, e := s.RuntimeProfileStore.Inspect(ctx)
	return authoringSnapshot(v), e
}
func (s runtimeAuthoringStore) Publish(ctx context.Context, token string, wire []byte) (runtimeapplication.LocalSnapshot, error) {
	v, e := s.RuntimeProfileStore.Publish(ctx, token, wire)
	return authoringSnapshot(v), e
}
func (s lifecycleService) RuntimeAuthoring(ctx context.Context, req runtimeapplication.AuthoringRequest) cli.Result {
	store, e := local.NewRuntimeProfileStore(s.stateRoot)
	if e != nil {
		r := canonicalCompletion(completion.Facts{ValidationFailed: true}, "Local Runtime configuration unavailable", nil, "Inspect scoped local recovery", s.provenance)
		r.Category = "configuration_unavailable"
		return r
	}
	app := runtimeapplication.Authoring{Impacts: s.runtimeImpacts, Passive: passiveRuntimeInventory, Store: runtimeAuthoringStore{store}, Installed: func(id string) bool { p, e := exec.LookPath(id); return e == nil && filepath.IsAbs(p) }, ReadDeclaration: local.ReadEffortDeclarationInput}
	// Declaration applicability uses the existing passive version seam. No
	// production version reader is composed here (BLD-001B.3c), so it stays
	// effort_configuration_unproven; nothing is probed or executed.
	version := s.runtimeObservationSources().version
	app.Applicability = func(ctx context.Context, id string) (runtimeprofile.EffortApplicability, error) {
		return runtimeadapter.ObserveEffortApplicability(ctx, exec.LookPath, version, id)
	}
	report, e := app.Run(ctx, req)
	facts := completion.Facts{Completed: e == nil}
	if e != nil {
		facts = completion.Facts{ValidationFailed: true}
		if report.Category == "publication_unconfirmed" {
			facts = completion.Facts{Failed: true}
		}
	}
	message := "Local Runtime configuration " + report.Category
	r := canonicalCompletion(facts, message, nil, report.Next, s.provenance)
	r.Category = report.Category
	r.RuntimeAuthoring = &report
	return r
}

func (s runtimeAuthoringStore) Recover(ctx context.Context, r runtimeapplication.AuthoringRequest) (runtimeapplication.AuthoringReport, error) {
	report := runtimeapplication.AuthoringReport{State: "pending", ImpactInventory: "scoped canonical runtime configuration only", Authentication: "unknown", ModelSupport: "unproven", EffortSupport: "unproven"}
	plan, present, e := s.InspectRuntimeRecovery(ctx)
	if e != nil {
		report.Category = "recovery_unavailable"
		return report, runtimeapplication.ErrAuthoring
	}
	if !present {
		report.Category = "unchanged"
		return report, nil
	}
	report.ExpectedRevision = plan.Digest
	report.PreviewDigest = plan.Digest
	report.Effects = []string{string(plan.Action)}
	report.Category = "preview"
	if !r.AuthorizeLocal {
		if r.ExpectedRevision != "" || r.PreviewDigest != "" {
			report.Category = "local_authority_required"
			return report, runtimeapplication.ErrAuthoring
		}
		return report, nil
	}
	if r.ExpectedRevision != plan.Digest || r.PreviewDigest != plan.Digest {
		report.Category = "stale_preview"
		return report, runtimeapplication.ErrAuthoring
	}
	authority, e := local.AuthorizeRecovery(plan, r.PreviewDigest)
	if e != nil {
		report.Category = "stale_preview"
		return report, runtimeapplication.ErrAuthoring
	}
	_, e = s.ApplyRuntimeRecovery(ctx, plan, authority)
	if e != nil {
		report.Category = "recovery_unconfirmed"
		return report, runtimeapplication.ErrAuthoring
	}
	report.Category = "applied"
	report.Next = "Inspect canonical configuration after scoped recovery"
	return report, nil
}

func passiveRuntimeInventory(ctx context.Context) []runtimeapplication.DetectedRuntime {
	inventory := runtimebootstrap.Detect(ctx, exec.LookPath, []runtimebootstrap.Runtime{{ID: "codex", Executable: "codex"}, {ID: "claude", Executable: "claude"}})
	out := []runtimeapplication.DetectedRuntime{}
	for _, r := range inventory.Runtimes {
		out = append(out, runtimeapplication.DetectedRuntime{RuntimeID: r.ID, Installed: r.Present, Integration: "unobserved"})
	}
	return out
}
func (s lifecycleService) runtimeImpacts(ctx context.Context) ([]runtimeapplication.LocalImpact, string) {
	records, e := s.installation.ListInstalled(ctx)
	if e != nil {
		return nil, "incomplete: installed Project inventory unavailable"
	}
	out := []runtimeapplication.LocalImpact{}
	classification := "bounded installed Projects only; repositories outside catalog excluded"
	for i, r := range records {
		if i >= 64 {
			classification = "incomplete: installed Project bound reached"
			break
		}
		p, e := s.installation.ReadInstalledProject(ctx, r)
		if e != nil {
			classification = "incomplete: a Project definition is inaccessible"
			continue
		}
		profiles, _ := p.State().ModelProfiles.Value()
		for _, profile := range profiles {
			if len(out) >= 256 {
				return out, "incomplete: Profile reference bound reached"
			}
			out = append(out, runtimeapplication.LocalImpact{ProjectID: r.ID, ProfileID: profile.Key})
		}
	}
	// ListInstalled includes archived local membership; it is not a visibility filter.
	return out, classification
}
