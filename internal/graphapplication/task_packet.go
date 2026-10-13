package graphapplication

import (
	"context"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

// AttachCompiledTaskPacket projects the read-only compiler contract onto the
// adapter packet. Runtime/model/effort/configuration data has no packet field.
// The caller must persist the returned receipt with its successor attempt;
// this internal seam does not change historical graphs or wire public commands.
func AttachCompiledTaskPacket(invocation executiongraph.Invocation, input workflowcompiler.ExecutionInput, binding runtimeadapter.PacketContext) (executiongraph.Invocation, runtimeadapter.AttemptPacketReceipt, error) {
	return runtimeadapter.WithTaskPacket(invocation, projectPacketInput(input), binding)
}

// PrepareCompiledTaskPacket requires the admission ledger to record the digest
// before making the compiled packet available for spawn.
func PrepareCompiledTaskPacket(ctx context.Context, invocation executiongraph.Invocation, input workflowcompiler.ExecutionInput, binding runtimeadapter.PacketContext, recorder runtimeadapter.TaskPacketRecorder) (executiongraph.Invocation, runtimeadapter.AttemptPacketReceipt, error) {
	return runtimeadapter.PrepareTaskPacket(ctx, invocation, projectPacketInput(input), binding, recorder)
}

func projectPacketInput(input workflowcompiler.ExecutionInput) runtimeadapter.PacketInput {
	refs := make([]runtimeadapter.PacketInputBinding, len(input.Inputs))
	for i, ref := range input.Inputs {
		refs[i] = runtimeadapter.PacketInputBinding{InputID: ref.InputID, Reference: ref.Reference, Digest: ref.Digest}
	}
	return runtimeadapter.PacketInput{AgentID: input.AgentID, Role: input.Role, Responsibilities: input.Responsibilities, Purpose: input.Purpose, Instructions: input.Instructions, Inputs: refs, ExpectedOutputs: input.ExpectedOutputs, Criteria: input.Criteria, Scope: input.Scope, Effects: input.Effects, Controls: input.Controls}
}
