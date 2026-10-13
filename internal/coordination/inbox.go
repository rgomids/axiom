package coordination

import (
	"context"
	"errors"
	"path/filepath"
	"time"
)

var ErrChildContext = errors.New("running child context required")

// ChildContext is resolved by the trusted launcher, never from publish flags.
// Epoch and workspace are revalidated by the inbox adapter on every operation.
type ChildContext struct {
	Workspace, Inbox, ParentID, ChildID, AttemptID string
	GraphRevision, Epoch                           uint64
	Running                                        bool
}

type Candidate struct {
	Sequence uint64
	Epoch    uint64
	Record   Record
}

// AttemptInbox owns exclusive creation and atomic per-attempt sequence allocation.
// Append must revalidate the binding, reject foreign/replaced inboxes and return
// a detached candidate. It cannot write canonical state. Ingestion revalidates it.
type AttemptInbox interface {
	Create(context.Context, ChildContext) error
	Append(context.Context, ChildContext, Record) (Candidate, error)
}

type ChildResolver interface {
	ResolveChild(context.Context, string) (ChildContext, error)
}

type InboxPublisher struct {
	Resolver   ChildResolver
	Inbox      AttemptInbox
	AllocateID func() (string, error)
	Now        func() time.Time
}

func ValidChildContext(c ChildContext) bool {
	return c.Running && childLocation(c.Workspace) && childLocation(c.Inbox) && c.Inbox != c.Workspace && validID(c.ParentID) && validID(c.ChildID) && validID(c.AttemptID) && c.GraphRevision > 0 && c.Epoch > 0
}

func childLocation(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && filepath.Dir(path) != path
}

func (p InboxPublisher) Create(ctx context.Context, c ChildContext) error {
	if p.Inbox == nil || !ValidChildContext(c) {
		return ErrChildContext
	}
	return p.Inbox.Create(ctx, c)
}

// Publish accepts content only; all lineage comes from the resolved child.
func (p InboxPublisher) Publish(ctx context.Context, workspace string, kind Kind, fields []Field, usage *UsageObservation, provenance Provenance) (Candidate, error) {
	if err := ctx.Err(); err != nil {
		return Candidate{}, err
	}
	if p.Resolver == nil || p.Inbox == nil {
		return Candidate{}, ErrChildContext
	}
	c, err := p.Resolver.ResolveChild(ctx, workspace)
	if err != nil || !ValidChildContext(c) || c.Workspace != workspace {
		return Candidate{}, ErrChildContext
	}
	allocate := p.AllocateID
	if allocate == nil {
		allocate = randomID
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	id, err := allocate()
	if err != nil {
		return Candidate{}, err
	}
	// Revision 1 is a validation envelope, not a canonical revision. The inbox
	// sequence is independent; T23 assigns canonical chain fields at ingestion.
	r := Record{FormatVersion: FormatVersion, RecordID: id, Kind: kind, ParentID: c.ParentID, ChildID: c.ChildID, AttemptID: c.AttemptID, GraphRevision: c.GraphRevision, Revision: 1, Provenance: provenance, Fields: sortedFields(fields), Usage: usage, CreatedAt: now().UTC()}
	r.Digest, err = recordDigest(r)
	if err != nil {
		return Candidate{}, err
	}
	if _, err = Encode(r); err != nil {
		return Candidate{}, err
	}
	candidate, err := p.Inbox.Append(ctx, c, r)
	if err != nil {
		return Candidate{}, err
	}
	if candidate.Sequence == 0 || candidate.Epoch != c.Epoch || candidate.Record.Digest != r.Digest || !ValidRecord(candidate.Record) {
		return Candidate{}, ErrChildContext
	}
	return candidate, nil
}
