package stageexecution

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

type fakeSingleMaterial struct{ m AttemptMaterial }

func (f fakeSingleMaterial) LoadSingleAttempt(context.Context, workflow.SingleDispatch, executiongraph.DispatchAuthority) (AttemptMaterial, error) {
	return f.m, nil
}

type fakeSingleRecorder struct {
	calls int
	fail  bool
}

func (f *fakeSingleRecorder) RecordTaskPacket(context.Context, runtimeadapter.AttemptPacketReceipt) error {
	f.calls++
	if f.fail {
		return errInjected
	}
	return nil
}

type fakeSingleInvocationSupervisor struct {
	calls    int
	recorder *fakeSingleRecorder
	inv      executiongraph.Invocation
	complete SingleCompletion
}

func (f *fakeSingleInvocationSupervisor) StartSingleInvocation(_ context.Context, _ workflow.SingleDispatch, _ executiongraph.DispatchAuthority, inv executiongraph.Invocation, complete SingleCompletion) (executiongraph.ProcessCorrelation, error) {
	if f.recorder.calls != 1 {
		return executiongraph.ProcessCorrelation{}, errInjected
	}
	f.calls++
	f.inv = inv
	f.complete = complete
	return executiongraph.ProcessCorrelation{PID: 44, StartIdentity: "fake-single", GroupIdentity: "fake-group"}, nil
}
func TestSinglePreparationReusesFrozenAdapterAndPacket(t *testing.T) {
	for _, kind := range []string{"positive", "input drift", "tuple drift", "auth blocked", "packet save failure", "foreign inbox"} {
		t.Run(kind, func(t *testing.T) {
			g := graphFixture(t)
			child := g.Children[0]
			p := *child.Envelope.Resolution.ResolvedProfile
			p.RuntimeVersion = "0.159.1"
			// Only synthetic paths are passed to fakes; no executable or directory is created.
			root := t.TempDir()
			workspace := filepath.Join(root, "workspace")
			inbox := filepath.Join(root, "inbox")
			axiom := filepath.Join(root, "axiom")
			id := "00000000-0000-4000-8000-000000000101"
			m := AttemptMaterial{Input: workflowcompiler.ExecutionInput{ProjectID: child.Envelope.Scope.ProjectID, RepositoryKey: "main", StageID: "implementation", AgentID: "implementer", Role: "implementer", Responsibilities: "Implement authorized work", Instructions: "Publish result", Scope: child.Envelope.Scope, Controls: child.Envelope.Controls, ResolvedProfile: &p, RuntimePreview: runtimeapplication.Preview{Choice: &runtimeprofile.Choice{ResolvedProfile: &p}}}, Invocation: runtimeadapter.StageInvocationRequest{Profile: runtimeadapter.CommandProfile{RuntimeID: p.RuntimeID, ModelProfileID: p.ProfileID, Model: p.Model, Executable: filepath.Join(root, "codex")}, Version: p.RuntimeVersion, Workspace: workspace, Inbox: inbox, AxiomPath: axiom}, Packet: runtimeadapter.PacketContext{AttemptID: id, Inbox: inbox, AxiomPath: axiom, AxiomDigest: strings.Repeat("a", 64)}, Context: coordination.ChildContext{ParentID: g.Parent.ExecutionID, ChildID: g.Parent.ExecutionID, AttemptID: id, GraphRevision: 1, Epoch: 1, Running: true, Workspace: workspace, Inbox: inbox}}
			digest, e := canonicalDigest(m.Input)
			if e != nil {
				t.Fatal(e)
			}
			single := workflow.SingleDispatch{State: workflow.State{ExecutionID: g.Parent.ExecutionID, Stage: "implementation", ProjectID: m.Input.ProjectID, RepositoryKey: "main", Revision: 1}, AttemptID: id, ResolvedProfile: &p, InputDigest: digest, Workspace: workspace, Inbox: inbox, AxiomDigest: m.Packet.AxiomDigest}
			a := executiongraph.DispatchAuthority{Children: []executiongraph.DispatchChildAuthority{{ChildID: single.State.ExecutionID, Effects: m.Input.Effects}}}
			checker := &fakeStageChecker{profile: &p}
			sources := &fakeStageSources{}
			auth := &fakeStageAuth{version: p.RuntimeVersion}
			recorder := &fakeSingleRecorder{}
			supervisor := &fakeSingleInvocationSupervisor{recorder: recorder}
			switch kind {
			case "input drift":
				m.Input.Instructions = "changed"
			case "tuple drift":
				changed := p
				changed.Model = "different"
				checker.profile = &changed
			case "auth blocked":
				auth.err = true
			case "packet save failure":
				recorder.fail = true
			case "foreign inbox":
				m.Packet.Inbox = workspace
			}
			starter := CompiledSingleStarter{Loader: fakeSingleMaterial{m}, Checker: checker, Sources: sources, Auth: auth, Recorder: recorder, Supervisor: supervisor}
			process, e := starter.StartSingle(context.Background(), single, a)
			if kind == "positive" {
				if e != nil || process.PID != 44 || supervisor.calls != 1 || supervisor.complete == nil || len(supervisor.inv.Stdin) == 0 || checker.calls != 1 || sources.calls != 2 || auth.calls != 1 {
					t.Fatal(process, e)
				}
				if strings.Contains(string(supervisor.inv.Stdin), p.Model) {
					t.Fatal("model in task packet")
				}
			} else if e == nil || supervisor.calls != 0 {
				t.Fatal("unsafe single dispatch", kind, e)
			}
		})
	}
}
