//go:build linux || darwin

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/detailartifact"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/workflow"
)

type criterionArtifactLauncher struct {
	stageexecution.RuntimeLauncher
	t                *testing.T
	artifacts        local.ArtifactStore
	missing, invalid bool
}

func (l criterionArtifactLauncher) Build(ctx context.Context, r stageexecution.LaunchRequest) (executiongraph.Invocation, error) {
	inv, err := l.RuntimeLauncher.Build(ctx, r)
	if err != nil {
		return inv, err
	}
	if len(r.Stage.OutputChecks) == 0 {
		for _, name := range r.Stage.Outputs {
			if err := os.WriteFile(filepath.Join(l.InboxRoot, r.Attempt.AttemptID, name), []byte("result"), 0600); err != nil {
				l.t.Fatal(err)
			}
		}
	}
	for _, check := range r.Stage.OutputChecks {
		if l.missing && check.Name == "evidence" {
			continue
		}
		refs := []detailartifact.Reference{{Kind: "workflow-stage", Value: r.Ref.StageID}, {Kind: "workflow-attempt", Value: r.Attempt.AttemptID}, {Kind: "workflow-repository", Value: r.Execution.WorkItem.RepositoryKey}, {Kind: "workflow-definition", Value: strings.TrimPrefix(r.Stage.DefinitionDigest, "sha256:")}, {Kind: "workflow-output", Value: check.Name}}
		if l.invalid && check.Name == "evidence" {
			refs[1].Value = "wrong-attempt"
		}
		artifact, err := l.artifacts.Create(ctx, detailartifact.Draft{ExecutionID: r.Execution.ExecutionID, Category: check.Kind, Outcome: "success", Retention: detailartifact.Evidence, Markdown: []byte("Validated scoped output"), Provenance: currentProvenance(), References: refs})
		if err != nil {
			l.t.Fatal(err)
		}
		wire, _ := json.Marshal(map[string]string{"kind": "artifact", "id": artifact.ID, "digest": hex.EncodeToString(artifact.Digest[:])})
		if err := os.WriteFile(filepath.Join(l.InboxRoot, r.Attempt.AttemptID, check.Name), wire, 0600); err != nil {
			l.t.Fatal(err)
		}
	}
	return inv, nil
}

func TestCriterionReferencedOptionalArtifactRequiresValidatedPresence(t *testing.T) {
	for _, mode := range []string{"absent", "invalid", "present"} {
		t.Run(mode, func(t *testing.T) {
			f := newExecutionRunFixture(t)
			// The bounded stub consumes its real packet; fixture launcher supplies the
			// retained references so the production Collect validator checks exact identity.
			if err := os.WriteFile(filepath.Join(os.Getenv("PATH"), "codex"), []byte("#!/bin/sh\nIFS= read -r packet\nexit 0\n"), 0700); err != nil {
				t.Fatal(err)
			}
			c, id := pendingExecutionDefinition(t, f, criterionWorkflow(true))
			ctx := context.Background()
			base, _, err := c.existingLauncher(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			artifacts, err := local.NewArtifactStore(f.env.state)
			if err != nil {
				t.Fatal(err)
			}
			e := c.engine(criterionArtifactLauncher{RuntimeLauncher: base, t: t, artifacts: artifacts, missing: mode == "absent", invalid: mode == "invalid"})
			runner := &countingRunner{base: e.Runner}
			e.Runner = runner
			x, err := e.Run(ctx, c.project.ID, id, stageexecution.RunOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if mode == "present" {
				if x.Status != workflow.RunCompleted || runner.spawns != 2 {
					t.Fatalf("valid evidence refused: %s spawns=%d", x.Status, runner.spawns)
				}
			} else if x.Status == workflow.RunCompleted || runner.spawns != 1 || attemptsIn(x, 1) != 0 {
				t.Fatalf("%s evidence dispatched successor: %s spawns=%d", mode, x.Status, runner.spawns)
			}
		})
	}
}
