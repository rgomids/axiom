package stageexecution

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

type fakeMaterialLoader struct{ m AttemptMaterial }

func (f fakeMaterialLoader) LoadAttempt(context.Context, executiongraph.SuccessorGraph, executiongraph.SuccessorChild, string, uint64) (AttemptMaterial, error) {
	return f.m, nil
}

type fakeStageChecker struct {
	profile *runtimeprofile.ResolvedProfileV4
	err     error
	calls   int
}

func (f *fakeStageChecker) Check(context.Context, runtimeapplication.Preview) (runtimeapplication.Binding, error) {
	f.calls++
	return runtimeapplication.Binding{Choice: runtimeprofile.Choice{FrozenProfile: f.profile}}, f.err
}

type fakeStageSources struct {
	calls int
	err   error
}

func (f *fakeStageSources) CheckEffortSources(_ context.Context, p runtimeprofile.ResolvedProfileV4, _ executiongraph.Invocation) (string, error) {
	f.calls++
	return p.EffortSourceDigest, f.err
}

type fakeStageAuth struct {
	version string
	err     bool
	calls   int
}

func (f *fakeStageAuth) Check(_ context.Context, t runtimeadapter.AuthTarget) runtimeadapter.AuthReport {
	f.calls++
	status := runtimeadapter.AuthSubscriptionObserved
	if f.err {
		status = runtimeadapter.AuthUnproven
	}
	return runtimeadapter.AuthReport{Scope: runtimeadapter.AuthScopeInvocation, RuntimeID: t.RuntimeID, Status: status, Version: f.version, ExecutableDigest: t.ExpectedDigest}
}

func TestCompiledPreparationReusesFrozenChecksAndTaskPacket(t *testing.T) {
	for _, kind := range []string{"positive", "tuple-drift", "wrong-child", "wrong-epoch", "wrong-inbox", "source-conflict", "unproven-auth", "missing-loader"} {
		t.Run(kind, func(t *testing.T) {
			g := graphFixture(t)
			c := g.Children[0]
			p := *c.Envelope.Resolution.FrozenProfile

			c.Envelope.Resolution.FrozenProfile = &p
			root := t.TempDir()
			workspace := filepath.Join(root, "workspace")
			inbox := filepath.Join(root, "inbox")
			axiom := filepath.Join(root, "axiom")
			binary := filepath.Join(root, "codex")
			if err := os.WriteFile(binary, []byte("inert fixture; never execute"), 0o700); err != nil {
				t.Fatal(err)
			}
			identity, err := runtimeadapter.ExecutableIdentity(binary)
			if err != nil {
				t.Fatal(err)
			}
			p.ExecutableDigest = identity
			c.Envelope.Workspace = workspace
			g.Children[0] = c
			id := "00000000-0000-4000-8000-000000000101"
			m := AttemptMaterial{Input: workflowcompiler.ExecutionInput{ProjectID: c.Envelope.Scope.ProjectID, RepositoryKey: c.Envelope.Scope.RepositoryKey, AgentID: "implementer", Role: c.Envelope.Capability.Role, Responsibilities: "Implement the authorized change", Instructions: "Produce the required result", NodeKey: c.NodeKey, Scope: c.Envelope.Scope, Effects: c.Envelope.AllowedEffects, Controls: c.Envelope.Controls, FrozenProfile: &p, RuntimePreview: runtimeapplication.Preview{Choice: &runtimeprofile.Choice{FrozenProfile: &p}}}, Invocation: runtimeadapter.StageInvocationRequest{Profile: runtimeadapter.CommandProfile{RuntimeID: p.RuntimeID, ModelProfileID: p.ProfileID, Model: p.Model, Executable: binary}, Workspace: workspace, Inbox: inbox, AxiomPath: axiom}, Packet: runtimeadapter.PacketContext{AttemptID: id, Inbox: inbox, AxiomPath: axiom, AxiomDigest: p.ExecutableDigest}, Context: coordination.ChildContext{Workspace: workspace, Inbox: inbox, ParentID: g.Parent.ExecutionID, ChildID: c.ExecutionID, AttemptID: id, GraphRevision: g.Parent.GraphRevision, Epoch: 1, Running: true}}
			checker := &fakeStageChecker{profile: &p}
			sources := &fakeStageSources{}
			auth := &fakeStageAuth{version: "unknown"}
			prep := &CompiledSupervisorPreparation{Checker: checker, Sources: sources, Auth: auth}
			switch kind {
			case "tuple-drift":
				changed := p
				changed.Model = "different-model"
				checker.profile = &changed
			case "wrong-child":
				m.Context.ChildID = g.Children[2].ExecutionID
			case "wrong-epoch":
				m.Context.Epoch = 2
			case "wrong-inbox":
				m.Packet.Inbox = workspace
			case "source-conflict":
				sources.err = runtimeadapter.ErrEffortOverrideConflict
			case "unproven-auth":
				auth.err = true
			}
			if kind != "missing-loader" {
				prep.Loader = fakeMaterialLoader{m: m}
			}
			inv, e := prep.Prepare(context.Background(), g, c, id, 1)
			if kind == "positive" {
				if e != nil || len(inv.Stdin) == 0 || checker.calls != 1 || sources.calls != 2 || auth.calls != 1 {
					t.Fatal("composition failed", e)
				}
				// A forged attempt cannot drain the registered inbox.
				if _, e = prep.Drain(context.Background(), g, g.Children[2], executiongraph.SuccessorAttempt{AttemptID: id, LeaseEpoch: 1}, true); !errors.Is(e, coordination.ErrChildContext) {
					t.Fatal(e)
				}
			} else if e == nil || len(inv.Stdin) != 0 {
				t.Fatal("unsafe packet exposed", kind)
			}
		})
	}
}
