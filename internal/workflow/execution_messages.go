package workflow

import (
	"errors"
	"time"
)

// Messages are bounded attempt-owned facts (ADR-0037 §6). The sender is the
// launcher-issued attempt context, never a payload field; content is data and
// never grants authority, satisfies a gate or changes another attempt.
const (
	MaxExecutionMessages = 256
	MaxMessageBytes      = 4 * 1024
)

var (
	ErrMessageRejected  = errors.New("message_rejected")
	ErrMessageDuplicate = errors.New("message_duplicate")
)

type MessageKind string

const (
	MessageQuestion MessageKind = "question"
	MessageAnswer   MessageKind = "answer"
	MessageBlocker  MessageKind = "blocker"
	MessageResult   MessageKind = "result"
)

type Message struct {
	ID               string      `json:"id"`
	SenderAttempt    string      `json:"senderAttempt"`
	Sender           string      `json:"sender"`
	Recipient        string      `json:"recipient"`
	Kind             MessageKind `json:"kind"`
	ReplyTo          string      `json:"replyTo,omitempty"`
	Content          string      `json:"content"`
	RunnerGeneration uint64      `json:"runnerGeneration"`
	ReceivedAt       time.Time   `json:"receivedAt"`
}

// IngressContext is issued by the launcher for one running attempt. A child
// can present only its own private ingress, so it cannot choose its sender.
type IngressContext struct {
	ExecutionID string
	AttemptID   string
	AgentID     string
	Generation  uint64
}

// MessageCandidate is the untrusted child payload.
type MessageCandidate struct {
	ID        string      `json:"id"`
	Recipient string      `json:"recipient"`
	Kind      MessageKind `json:"kind"`
	ReplyTo   string      `json:"replyTo,omitempty"`
	Content   string      `json:"content"`
}

func (e Execution) agentExists(id string) bool {
	for _, s := range e.Workflow.Stages {
		for _, a := range s.Agents {
			if a.ID == id {
				return true
			}
		}
	}
	return false
}

func (e Execution) message(id string) (Message, bool) {
	for _, m := range e.Messages {
		if m.ID == id {
			return m, true
		}
	}
	return Message{}, false
}

// AcceptMessage validates a candidate against the current attempt identity
// and runner fence. An exact redelivery is idempotent (ErrMessageDuplicate,
// no state change); any other conflict, forged, stale, wrong, oversized or
// late candidate is rejected without being stored.
func (e *Execution) AcceptMessage(ctx IngressContext, c MessageCandidate, now time.Time) error {
	if ctx.ExecutionID != e.ExecutionID || ctx.Generation == 0 || ctx.Generation != e.RunnerGeneration {
		return ErrMessageRejected
	}
	sender, ref, ok := e.FindAttempt(ctx.AttemptID)
	if !ok || ref.AgentID != ctx.AgentID || sender.RunnerGeneration != ctx.Generation {
		return ErrMessageRejected
	}
	if e.Cancel != nil || sender.Status != AttemptRunning {
		return ErrMessageRejected
	}
	if !ValidToken(c.ID) || len(c.Content) > MaxMessageBytes || !boundedText(c.Content, MaxMessageBytes) {
		return ErrMessageRejected
	}
	candidate := Message{
		ID: c.ID, SenderAttempt: ctx.AttemptID, Sender: ctx.AgentID, Recipient: c.Recipient,
		Kind: c.Kind, ReplyTo: c.ReplyTo, Content: c.Content, RunnerGeneration: ctx.Generation, ReceivedAt: now.UTC(),
	}
	if prior, dup := e.message(c.ID); dup {
		candidate.ReceivedAt = prior.ReceivedAt
		if prior == candidate {
			return ErrMessageDuplicate
		}
		return ErrMessageRejected
	}
	if len(e.Messages) >= MaxExecutionMessages {
		return ErrMessageRejected
	}
	e.Messages = append(e.Messages, candidate)
	if err := validateMessages(*e); err != nil {
		e.Messages = e.Messages[:len(e.Messages)-1]
		return ErrMessageRejected
	}
	e.touch(now)
	return e.Validate()
}

// validateMessages checks the stored log: known recipients, no self-address,
// and an answer replies once to a question addressed to its sender.
func validateMessages(e Execution) error {
	if len(e.Messages) > MaxExecutionMessages {
		return invalid("messages")
	}
	seen := map[string]Message{}
	answered := map[string]bool{}
	for _, m := range e.Messages {
		if !ValidToken(m.ID) || !ValidToken(m.SenderAttempt) || !e.agentExists(m.Sender) || !e.agentExists(m.Recipient) || m.Sender == m.Recipient {
			return invalid("message %q", m.ID)
		}
		if _, dup := seen[m.ID]; dup || !boundedText(m.Content, MaxMessageBytes) || m.RunnerGeneration == 0 || m.RunnerGeneration > e.RunnerGeneration || m.ReceivedAt.IsZero() {
			return invalid("message %q", m.ID)
		}
		switch m.Kind {
		case MessageQuestion, MessageBlocker, MessageResult:
			if m.ReplyTo != "" {
				return invalid("message %q reply", m.ID)
			}
		case MessageAnswer:
			q, ok := seen[m.ReplyTo]
			if !ok || q.Kind != MessageQuestion || q.Recipient != m.Sender || q.Sender != m.Recipient || answered[m.ReplyTo] {
				return invalid("message %q reply", m.ID)
			}
			answered[m.ReplyTo] = true
		default:
			return invalid("message %q kind", m.ID)
		}
		seen[m.ID] = m
	}
	return nil
}
