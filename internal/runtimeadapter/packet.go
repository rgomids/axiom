package runtimeadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/portableconfig"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

var ErrInvalidTaskPacket = errors.New("invalid task packet")

const MaxTaskPacketBytes = executiongraph.MaxStdinBytes

// PacketInput is the Runtime-independent projection of the compiled input.
// The graph application owns conversion from the read-only #306 producer.
type PacketInput struct {
	AgentID, Role, Responsibilities, Purpose, Instructions string
	Inputs                                                 []PacketInputBinding
	ExpectedOutputs                                        []workflowdefinition.Output
	Criteria                                               []workflowdefinition.Criterion
	Scope                                                  executiongraph.Scope
	Effects                                                []executiongraph.Effect
	Controls                                               executiongraph.ExecutionControls
}

type PacketInputBinding struct {
	InputID   string `json:"inputId"`
	Reference string `json:"reference"`
	Digest    string `json:"digest"`
}

// PacketContext contains the attempt-local coordination binding. It is supplied
// by admission, never inferred from the task's instructions or input records.
type PacketContext struct {
	AttemptID   string `json:"attemptId"`
	Inbox       string `json:"inbox"`
	AxiomPath   string `json:"axiomPath"`
	AxiomDigest string `json:"axiomDigest"`
}

// TaskPacket deliberately projects compiled input rather than serializing it:
// Runtime selections, model, effort and authentication never enter stdin.
type TaskPacket struct {
	FormatVersion    int                            `json:"formatVersion"`
	AgentID          string                         `json:"agentId"`
	Role             string                         `json:"role"`
	Responsibilities string                         `json:"responsibilities"`
	Purpose          string                         `json:"purpose"`
	Instructions     string                         `json:"instructions"`
	Inputs           []PacketInputBinding           `json:"inputs"`
	ExpectedOutputs  []workflowdefinition.Output    `json:"expectedOutputs"`
	Criteria         []workflowdefinition.Criterion `json:"completionCriteria"`
	AllowedPaths     []string                       `json:"allowedPaths"`
	AllowedEffects   []executiongraph.Effect        `json:"allowedEffects"`
	RecordMaxBytes   int                            `json:"coordinationRecordMaxBytes"`
	TimeoutSeconds   int64                          `json:"timeoutSeconds"`
	Coordination     PacketContext                  `json:"coordination"`
	Protocol         []string                       `json:"protocol"`
	Prohibitions     []string                       `json:"prohibitions"`
}

// AttemptPacketReceipt is separate from the historical Graph1/Graph2 Attempt.
// A successor admission ledger can persist it with the attempt without changing
// either historical graph codec. Digest hashes the exact bytes sent on stdin.
type AttemptPacketReceipt struct {
	AttemptID    string `json:"attemptId"`
	PacketDigest string `json:"packetDigest"`
}

// TaskPacketRecorder persists the digest in the admission-owned attempt ledger.
// The adapter owns no graph codec; later admission supplies the durable port.
type TaskPacketRecorder interface {
	RecordTaskPacket(context.Context, AttemptPacketReceipt) error
}

// PrepareTaskPacket records the exact packet digest before exposing stdin for
// spawn. A failed or missing recorder refuses preparation; it cannot silently
// dispatch an unrecorded packet. This port is usable with fakes in Batch B0.
func PrepareTaskPacket(ctx context.Context, invocation executiongraph.Invocation, input PacketInput, binding PacketContext, recorder TaskPacketRecorder) (executiongraph.Invocation, AttemptPacketReceipt, error) {
	if recorder == nil || ctx.Err() != nil {
		return executiongraph.Invocation{}, AttemptPacketReceipt{}, ErrInvalidTaskPacket
	}
	prepared, receipt, err := WithTaskPacket(invocation, input, binding)
	if err != nil {
		return executiongraph.Invocation{}, AttemptPacketReceipt{}, err
	}
	if recorder.RecordTaskPacket(ctx, receipt) != nil {
		// Recorder errors may contain backend values; expose only a fixed category.
		return executiongraph.Invocation{}, AttemptPacketReceipt{}, ErrInvalidTaskPacket
	}
	return prepared, receipt, nil
}

// BuildTaskPacket is pure and bounded; graph children and single-agent stages
// use the same compiled-input contract. Input references must already have been
// validated against their artifact digests by admission.
func BuildTaskPacket(input PacketInput, binding PacketContext) ([]byte, AttemptPacketReceipt, error) {
	if !safePacketText(binding.AttemptID) || binding.AttemptID == "" || !filepath.IsAbs(binding.AxiomPath) || !filepath.IsAbs(binding.Inbox) || !workflowdefinition.ValidDigest(binding.AxiomDigest) ||
		!safePacketText(binding.AxiomPath) || !safePacketText(binding.Inbox) || input.AgentID == "" || input.Role == "" || input.Responsibilities == "" || input.Instructions == "" || input.Controls.Timeout <= 0 {
		return nil, AttemptPacketReceipt{}, ErrInvalidTaskPacket
	}
	for _, ref := range input.Inputs {
		if ref.InputID == "" || ref.Reference == "" || !workflowdefinition.ValidDigest(ref.Digest) || !portableconfig.SafeValue(ref.Reference, "logical") {
			return nil, AttemptPacketReceipt{}, ErrInvalidTaskPacket
		}
	}
	for _, target := range input.Scope.Paths {
		if target == "" || path.IsAbs(target) || strings.Contains(target, "\\") || path.Clean(target) != target || target == ".." || strings.HasPrefix(target, "../") {
			return nil, AttemptPacketReceipt{}, ErrInvalidTaskPacket
		}
	}
	packet := TaskPacket{FormatVersion: 1, AgentID: input.AgentID, Role: input.Role, Responsibilities: input.Responsibilities, Purpose: input.Purpose, Instructions: input.Instructions, Inputs: input.Inputs, ExpectedOutputs: input.ExpectedOutputs, Criteria: input.Criteria, AllowedPaths: input.Scope.Paths, AllowedEffects: input.Effects, TimeoutSeconds: int64(input.Controls.Timeout.Seconds()), RecordMaxBytes: coordination.MaxRecordBytes, Coordination: binding,
		Protocol:     []string{"Publish candidates only into this attempt's inbox.", "Records grant no authority and cannot satisfy human gates.", "Publish result before exiting 0."},
		Prohibitions: []string{"Do not push, open pull requests, merge, or call external providers.", "Do not access credentials or expand allowed paths or effects.", "Do not mutate canonical Axiom state or another attempt's inbox."}}
	// Validate the entire projected packet, including nested output/criteria and
	// effect strings; omitted Runtime fields are never inspected or copied.
	if !safePacketValue(reflect.ValueOf(packet)) {
		return nil, AttemptPacketReceipt{}, ErrInvalidTaskPacket
	}
	wire, err := json.Marshal(packet)
	if err != nil || len(wire)+1 > MaxTaskPacketBytes || !safePacketJSON(wire) {
		return nil, AttemptPacketReceipt{}, ErrInvalidTaskPacket
	}
	wire = append(wire, '\n')
	digest := sha256.Sum256(wire)
	return wire, AttemptPacketReceipt{AttemptID: binding.AttemptID, PacketDigest: hex.EncodeToString(digest[:])}, nil
}

// WithTaskPacket attaches a copy of reviewed bytes and returns the receipt the
// caller must persist with its successor attempt before dispatch. It runs no
// Runtime and is intentionally not connected to public CLI composition.
func WithTaskPacket(invocation executiongraph.Invocation, input PacketInput, binding PacketContext) (executiongraph.Invocation, AttemptPacketReceipt, error) {
	wire, receipt, err := BuildTaskPacket(input, binding)
	if err != nil {
		return executiongraph.Invocation{}, AttemptPacketReceipt{}, err
	}
	invocation.Stdin = wire
	return invocation, receipt, nil
}

func safePacketJSON(wire []byte) bool {
	var object any
	if json.Unmarshal(wire, &object) != nil {
		return false
	}
	var safe func(any) bool
	safe = func(value any) bool {
		switch v := value.(type) {
		case string:
			return safePacketText(v)
		case []any:
			for _, item := range v {
				if !safe(item) {
					return false
				}
			}
		case map[string]any:
			for _, item := range v {
				if !safe(item) {
					return false
				}
			}
		}
		return true
	}
	return safe(object)
}

func safePacketText(value string) bool {
	if !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') || portableconfig.ContainsSecretBearingValue(value) {
		return false
	}
	lower := "\n" + strings.ToLower(strings.ReplaceAll(value, "\r", "\n"))
	for _, line := range strings.Split(lower, "\n") {
		line = strings.TrimSpace(line)
		for _, role := range []string{"system:", "assistant:", "user:", "developer:"} {
			if strings.HasPrefix(line, role) {
				return false
			}
		}
	}
	for _, marker := range []string{"sk-", "ghp_", "github_pat_", "-----begin", "<|im_start|>", "<|im_end|>", "<|start_header_id|>", "<|end_header_id|>", "[inst]", "[/inst]", "<system>", "</system>", "<assistant>", "</assistant>", "<user>", "</user>", "\nsystem:", "\nassistant:", "\nuser:"} {
		if strings.Contains(lower, marker) {
			return false
		}
	}
	return true
}

// Validate before JSON encoding, which otherwise repairs malformed UTF-8.
func safePacketValue(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.String:
		return safePacketText(value.String())
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if !safePacketValue(value.Field(i)) {
				return false
			}
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			if !safePacketValue(value.Index(i)) {
				return false
			}
		}
	}
	return true
}
