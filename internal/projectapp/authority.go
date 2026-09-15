package projectapp

import (
	"context"
	"sort"
	"sync/atomic"
)

type Operation uint8

const (
	Create Operation = iota + 1
	Update
	Move
	Install
	ReplaceBindings
)

type WriteAction uint8

const (
	Replace WriteAction = iota + 1
	Remove
)

// Write lists the complete artifact effect at one opaque destination. Local
// records are ID-addressed; their only listed relative leaf is installation.json.
type Write struct {
	Destination Destination
	Name        string
	Action      WriteAction
}
type previewIdentity struct{ marker byte }
type Preview struct {
	identity      *previewIdentity
	operation     Operation
	from, to      Destination
	before, after ArtifactSnapshot
	local         LocalSnapshot
	expected      ExpectedRevisions
	writes        []Write
}

// PreviewPortable records complete normalized intent, exact before/after bytes,
// operation, expected source/target revisions and every write/removal. A move or
// create requires an absent destination; adapters enforce that condition along
// with namespace collision checks under their protected commit protocol.
func PreviewPortable(op Operation, from, to Destination, before, after ArtifactSnapshot) (Preview, []Issue) {
	if !to.valid() || !after.valid() {
		return Preview{}, problem(InvalidPreview)
	}
	if op == Create {
		if from.valid() || before.valid() {
			return Preview{}, problem(InvalidPreview)
		}
		return portablePreview(op, from, to, before, after, MissingPortableRevision()), nil
	}
	if op != Update && op != Move {
		return Preview{}, problem(WrongOperation)
	}
	if !from.valid() || !before.valid() || before.Project().State().ID != after.Project().State().ID {
		return Preview{}, problem(InvalidPreview)
	}
	renamed := before.Project().State().Slug != after.Project().State().Slug
	if (op == Update && (from != to || renamed)) || (op == Move && (from == to || !renamed)) {
		return Preview{}, problem(InvalidPreview)
	}
	return portablePreview(op, from, to, before, after, before.Revision()), nil
}
func portablePreview(op Operation, from, to Destination, before, after ArtifactSnapshot, expected PortableRevision) Preview {
	p := Preview{identity: &previewIdentity{1}, operation: op, from: from, to: to, before: before, after: after, expected: ExpectedRevisions{Portable: expected}}
	nextNames := map[string]bool{}
	for _, name := range after.names() {
		nextNames[name] = true
		p.writes = append(p.writes, Write{to, name, Replace})
	}
	for _, name := range before.names() {
		if from != to || !nextNames[name] {
			p.writes = append(p.writes, Write{from, name, Remove})
		}
	}
	sort.Slice(p.writes, func(i, j int) bool {
		if p.writes[i].Action != p.writes[j].Action {
			return p.writes[i].Action < p.writes[j].Action
		}
		return p.writes[i].Name < p.writes[j].Name
	})
	return p
}
func PreviewLocal(op Operation, destination Destination, expected LocalRevision, proposed LocalSnapshot) (Preview, []Issue) {
	if op != Install && op != ReplaceBindings {
		return Preview{}, problem(WrongOperation)
	}
	if !destination.valid() || !expected.valid || !proposed.valid || destination == proposed.state.Source {
		return Preview{}, problem(InvalidPreview)
	}
	if op == ReplaceBindings && !expected.present {
		return Preview{}, problem(InvalidPreview)
	}
	return Preview{identity: &previewIdentity{1}, operation: op, to: destination, local: proposed,
		expected: ExpectedRevisions{Portable: proposed.state.PortableRevision, Local: expected},
		writes:   []Write{{destination, "installation.json", Replace}},
	}, nil
}
func (p Preview) Expected() ExpectedRevisions              { return p.expected }
func (p Preview) Writes() []Write                          { return append([]Write(nil), p.writes...) }
func (p Preview) Operation() Operation                     { return p.operation }
func (p Preview) Destinations() (Destination, Destination) { return p.from, p.to }
func (p Preview) Before() ArtifactSnapshot                 { return p.before }
func (p Preview) After() ArtifactSnapshot                  { return p.after }
func (p Preview) Local() LocalSnapshot                     { return p.local }

// ExpectedDestination is distinct from the old source revision on create/move.
func (p Preview) ExpectedDestination() PortableRevision {
	if p.operation == Create || p.operation == Move {
		return MissingPortableRevision()
	}
	return p.expected.Portable
}

type Approver uint8

const (
	Human Approver = iota + 1
	System
)

type approval struct {
	preview *previewIdentity
	by      Approver
	revoked atomic.Bool
}
type Authority struct{ grant *approval }

// Confirm must be called by trusted presentation/system policy only after review
// of this exact preview. It records scoped consent; it is not authentication or a
// durable grant protocol. AI/draft input cannot supply an Approver. No remote/Git
// authority exists. A fresh preview always requires fresh consent, even if equal.
func Confirm(p Preview, by Approver) (Authority, []Issue) {
	if p.identity == nil || (by != Human && by != System) {
		return Authority{}, problem(InvalidPreview)
	}
	return Authority{&approval{preview: p.identity, by: by}}, nil
}
func (a Authority) Revoke() {
	if a.grant != nil {
		a.grant.revoked.Store(true)
	}
}
func check(p Preview, a Authority, observed ExpectedRevisions) []Issue {
	if p.identity == nil {
		return problem(InvalidPreview)
	}
	if a.grant == nil {
		return problem(MissingAuthority)
	}
	if a.grant.revoked.Load() {
		return problem(RevokedAuthority)
	}
	if a.grant.preview != p.identity {
		return problem(StaleAuthority)
	}
	if p.expected != observed {
		return problem(RevisionMismatch)
	}
	return nil
}

type AuthorizedPortable struct {
	preview   Preview
	authority Authority
}
type AuthorizedLocal struct {
	preview   Preview
	authority Authority
}

func AuthorizePortable(p Preview, a Authority, observed ExpectedRevisions) (AuthorizedPortable, []Issue) {
	if issues := check(p, a, observed); len(issues) != 0 {
		return AuthorizedPortable{}, issues
	}
	if !p.after.valid() {
		return AuthorizedPortable{}, problem(WrongOperation)
	}
	return AuthorizedPortable{p, a}, nil
}
func AuthorizeLocal(p Preview, a Authority, observed ExpectedRevisions) (AuthorizedLocal, []Issue) {
	if issues := check(p, a, observed); len(issues) != 0 {
		return AuthorizedLocal{}, issues
	}
	if !p.local.valid {
		return AuthorizedLocal{}, problem(WrongOperation)
	}
	return AuthorizedLocal{p, a}, nil
}
func (a AuthorizedPortable) Valid() bool {
	return a.preview.after.valid() && len(check(a.preview, a.authority, a.preview.expected)) == 0
}
func (a AuthorizedLocal) Valid() bool {
	return a.preview.local.valid && len(check(a.preview, a.authority, a.preview.expected)) == 0
}
func (a AuthorizedPortable) Snapshot() ArtifactSnapshot { return a.preview.after }
func (a AuthorizedLocal) Snapshot() LocalSnapshot       { return a.preview.local }
func (a AuthorizedPortable) Preview() Preview           { return a.preview }
func (a AuthorizedLocal) Preview() Preview              { return a.preview }

// ApplyPortable/ApplyLocal are small contract gates, not create/update/install
// use cases. They call no port on denied consent, including no reads or metadata
// allocation. Observations supplied here never replace adapter commit-time CAS.
func ApplyPortable(ctx context.Context, store PortableWriter, p Preview, a Authority, observed ExpectedRevisions) MutationResult {
	request, issues := AuthorizePortable(p, a, observed)
	if len(issues) != 0 {
		return MutationResult{Commit: NotCommitted, Status: Denied, Issues: issues}
	}
	if ctx.Err() != nil {
		return MutationResult{Commit: NotCommitted, Status: Cancelled}
	}
	if store == nil {
		return MutationResult{Commit: NotCommitted, Status: Failed}
	}
	return store.CommitPortable(ctx, request).Normalized()
}
func ApplyLocal(ctx context.Context, store LocalWriter, p Preview, a Authority, observed ExpectedRevisions) MutationResult {
	request, issues := AuthorizeLocal(p, a, observed)
	if len(issues) != 0 {
		return MutationResult{Commit: NotCommitted, Status: Denied, Issues: issues}
	}
	if ctx.Err() != nil {
		return MutationResult{Commit: NotCommitted, Status: Cancelled}
	}
	if store == nil {
		return MutationResult{Commit: NotCommitted, Status: Failed}
	}
	return store.CommitLocal(ctx, request).Normalized()
}
