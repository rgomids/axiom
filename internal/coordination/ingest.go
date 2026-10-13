package coordination

import (
	"context"
	"errors"
	"strconv"

	"github.com/rgomids/axiom/internal/executiongraph"
)

var ErrAttemptSealed = errors.New("coordination attempt sealed")
var ErrEpochFenced = errors.New("coordination epoch fenced")

// InboxDrain holds the same lock as child Append. It journals progress and
// bounded rejection categories, and seals only after the final callback succeeds.
// A sealed inbox rejects Append and never supplies new classifier input.
type InboxDrain interface {
	Consume(context.Context, ChildContext, bool, func(Candidate) (string, error)) error
}

// AttemptFence pins the current lease and running attempt through the entire
// drain. Implementations must serialize with terminal state writes and Close.
type AttemptFence interface {
	WithAttempt(context.Context, ChildContext, func() error) error
}

type Ingestor struct {
	Service         Service
	Reader          StreamReader
	Inbox           InboxDrain
	Fence           AttemptFence
	MaximumAttempts uint32
	// Trusted graph children, used only to resolve references in canonical streams.
	ReferenceChildren []string
}

// Drain may be called while children run. final drains then seals under the
// inbox lock before the supervisor's single CAS terminal save. A failed terminal
// save leaves the inbox sealed, preserving the drained set for recovery.
func (i Ingestor) Drain(ctx context.Context, c ChildContext, final bool) ([]executiongraph.AttemptRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !ValidChildContext(c) || i.Service.store == nil || i.Reader == nil || i.Inbox == nil || i.Fence == nil {
		return nil, ErrChildContext
	}
	if _, err := AttemptCapacity(i.MaximumAttempts); err != nil {
		return nil, err
	}
	var records []executiongraph.AttemptRecord
	err := i.Fence.WithAttempt(ctx, c, func() error {
		if err := i.Inbox.Consume(ctx, c, final, func(candidate Candidate) (string, error) {
			if candidate.Epoch != c.Epoch {
				return "quarantined_epoch", nil
			}
			r := candidate.Record
			if candidate.Sequence == 0 || !ValidRecord(r) || r.ParentID != c.ParentID || r.ChildID != c.ChildID || r.AttemptID != c.AttemptID || r.GraphRevision != c.GraphRevision {
				return "invalid_lineage", nil
			}
			if r.Revision != 1 || r.PreviousDigest != "" {
				return "stale_revision", nil
			}
			// Source provenance is assigned by ingestion, never trusted from the child.
			r.Provenance.SourceState = "inbox-epoch:" + strconv.FormatUint(c.Epoch, 10)
			r.Digest, _ = recordDigest(r)
			stream, found, err := i.Reader.ReadStream(ctx, c.ParentID, c.ChildID)
			if err != nil {
				return "", err
			}
			if !found {
				stream = Stream{}
			}
			if found && (!ValidStream(stream) || stream.ParentID != c.ParentID || stream.ChildID != c.ChildID) {
				return "", ErrInvalidRecord
			}
			count := 0
			previous := ""
			revision := uint64(1)
			for _, accepted := range stream.Records {
				if accepted.AttemptID == c.AttemptID {
					count++
				}
				// A crash after canonical publication but before the inbox receipt must
				// not duplicate a record or spend another reserved slot.
				if accepted.RecordID == r.RecordID {
					original := accepted
					original.Revision, original.PreviousDigest = 1, ""
					original.Digest, _ = recordDigest(original)
					if original.Digest == r.Digest {
						return "accepted", nil
					}
					return "duplicate_record", nil
				}
				previous, revision = accepted.Digest, accepted.Revision+1
			}
			if revision > maxStreamRecords || CheckAttemptCapacity(i.MaximumAttempts, count, r.Kind) != nil {
				return "capacity_exceeded", nil
			}
			if err := i.references(ctx, c, r); err != nil {
				if errors.Is(err, ErrRecordNotFound) {
					return "missing_reference", nil
				}
				return "", err
			}
			r.Revision, r.PreviousDigest = revision, previous
			r.Digest, err = recordDigest(r)
			if err != nil {
				return "", err
			}
			if _, err := Encode(r); err != nil {
				return "invalid_record", nil
			}
			if err := i.Service.store.Publish(ctx, r); err != nil {
				return "", err
			}
			return "accepted", nil
		}); err != nil {
			return err
		}
		stream, found, err := i.Reader.ReadStream(ctx, c.ParentID, c.ChildID)
		if err != nil {
			return err
		}
		if !found {
			stream = Stream{}
		}
		if found && (!ValidStream(stream) || stream.ParentID != c.ParentID || stream.ChildID != c.ChildID) {
			return ErrInvalidRecord
		}
		for _, r := range stream.Records {
			if r.AttemptID == c.AttemptID && r.GraphRevision == c.GraphRevision {
				records = append(records, ClassifierRecord(r))
			}
		}
		return nil
	})
	return records, err
}

func (i Ingestor) references(ctx context.Context, c ChildContext, r Record) error {
	kind, field := QuestionRequest, "question_reference"
	if r.Kind == ContractAcceptance {
		kind, field = ContractProposal, "contract_reference"
	} else if r.Kind != Answer {
		return nil
	}
	reference := ""
	for _, f := range r.Fields {
		if f.Name == field {
			reference = f.Value
		}
	}
	if reference == "" {
		return ErrRecordNotFound
	}
	for _, child := range i.ReferenceChildren {
		if !validID(child) {
			return ErrInvalidRecord
		}
		stream, found, err := i.Reader.ReadStream(ctx, c.ParentID, child)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		if !ValidStream(stream) || stream.ParentID != c.ParentID || stream.ChildID != child {
			return ErrInvalidRecord
		}
		for _, record := range stream.Records {
			if record.GraphRevision == c.GraphRevision && record.AttemptID != "" && record.Kind == kind && record.RecordID == reference {
				return nil
			}
		}
	}
	return ErrRecordNotFound
}

// Classify is the final-drain hook for the successor supervisor. It returns
// values for one terminal CAS save; rendering never calls it.
func (i Ingestor) Classify(ctx context.Context, c ChildContext, process executiongraph.ProcessResult, cancelled bool) (executiongraph.AttemptStatus, executiongraph.ClassificationReason, string, error) {
	records, err := i.Drain(ctx, c, true)
	if err != nil {
		return "", "", "", err
	}
	status, reason, reference := executiongraph.ClassifyAttempt(process, cancelled, executiongraph.AttemptCorrelation{ParentID: c.ParentID, ChildID: c.ChildID, AttemptID: c.AttemptID, GraphRevision: c.GraphRevision}, records)
	return status, reason, reference, nil
}
