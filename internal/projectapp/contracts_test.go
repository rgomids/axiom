package projectapp_test

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

func TestSnapshotsAreCompleteDetachedAndRevisioned(t *testing.T) {
	state := definition(t).State()
	state.BusinessContext = project.Configured(project.BusinessContext{Documents: project.Configured([]string{"context/a.md", "context/b.md"})})
	p, domainIssues := project.New(state)
	if len(domainIssues) != 0 {
		t.Fatal(domainIssues)
	}
	codec := fixtureCodec{p}
	docs := []projectapp.Document{{Name: "context/b.md", Content: []byte("B")}, {Name: "context/a.md", Content: []byte("A")}}
	manifest := []byte("synthetic complete manifest")
	s, issues := projectapp.ReadSnapshot(codec, manifest, docs)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	ordered, issues := projectapp.ReadSnapshot(codec, manifest, []projectapp.Document{docs[1], docs[0]})
	if len(issues) != 0 || s.Revision() != ordered.Revision() {
		t.Fatal("enumeration changes revision")
	}
	manifest[0] = 'X'
	docs[0].Content[0] = 'X'
	extracted := s.Documents()
	extracted[0].Content[0] = 'X'
	extracted[0].Name = "changed"
	output := s.Manifest()
	output[0] = 'X'
	if s.Revision() != ordered.Revision() || string(s.Manifest()) != "synthetic complete manifest" || string(s.Documents()[0].Content) != "A" {
		t.Fatal("aliased snapshot")
	}
	for _, invalid := range [][]projectapp.Document{
		nil,
		{{Name: "context/a.md"}},
		{{Name: "context/a.md"}, {Name: "context/a.md"}},
		{{Name: "/absolute"}}, {{Name: "../outside"}}, {{Name: "context/./a.md"}}, {{Name: "axiom.yaml"}}, {{Name: "context//a.md"}}, {{Name: "context\\a.md"}},
	} {
		if _, issues := projectapp.ReadSnapshot(codec, []byte("synthetic"), invalid); len(issues) == 0 {
			t.Fatal("incomplete/ambiguous artifact set accepted")
		}
	}
	for _, c := range []projectapp.ManifestCodec{nil, fixtureCodec{}} {
		if _, issues := projectapp.ReadSnapshot(c, []byte("synthetic"), nil); len(issues) == 0 {
			t.Fatal("invalid definition accepted")
		}
	}
	if _, issues := projectapp.ReadSnapshot(codec, nil, docs); len(issues) == 0 {
		t.Fatal("absent manifest accepted")
	}
	if projectapp.MissingPortableRevision() == (projectapp.PortableRevision{}) || projectapp.MissingLocalRevision() == (projectapp.LocalRevision{}) || projectapp.MissingLocalRevision() == projectapp.ObserveLocalRevision(nil) {
		t.Fatal("absence collapsed into invalid/empty")
	}
	changed, issues := projectapp.ReadSnapshot(fixtureCodec{definition(t)}, s.Manifest(), []projectapp.Document{{Name: "context/a.md", Content: []byte("different")}})
	if len(issues) != 0 || changed.Revision() == s.Revision() {
		t.Fatal("document change omitted from revision")
	}
}

func TestReadSnapshotRejectsInvalidUTF8DocumentName(t *testing.T) {
	name := string([]byte{'c', 'o', 'n', 't', 'e', 'x', 't', '/', 0xff, '.', 'm', 'd'})
	snapshot, issues := projectapp.ReadSnapshot(fixtureCodec{definition(t)}, []byte("synthetic manifest"), []projectapp.Document{{Name: name, Content: []byte("content")}})
	if len(issues) == 0 {
		t.Error("invalid UTF-8 document name accepted")
	}
	if snapshot.Digests() != nil {
		t.Error("invalid document produced reusable digests")
	}
	if projectapp.ValidDocumentName(name) {
		t.Error("document-name contract accepted invalid UTF-8")
	}
}

func TestReadSnapshotPreservesValidReplacementRune(t *testing.T) {
	const name = "context/api\ufffdlegacy.md"
	if !projectapp.ValidDocumentName(name) {
		t.Fatal("valid U+FFFD rejected by document-name contract")
	}
	snapshot, issues := projectapp.ReadSnapshot(fixtureCodec{definition(t)}, []byte("synthetic manifest"), []projectapp.Document{{Name: name, Content: []byte("content")}})
	if len(issues) != 0 {
		t.Fatal("valid U+FFFD rejected", issues)
	}
	digests := snapshot.Digests()
	if len(digests) != 2 || digests[1].Name != name {
		t.Fatal("valid U+FFFD changed in artifact metadata")
	}
}

func TestWriteSetContainsDeletionsAndMoveDestinations(t *testing.T) {
	p := definition(t)
	old, issues := projectapp.ReadSnapshot(fixtureCodec{p}, []byte("old"), []projectapp.Document{{Name: "context/removed.md", Content: []byte("old document")}})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	next := snapshot(t, "next")
	destination := projectapp.NewDestination()
	update, issues := projectapp.PreviewPortable(projectapp.Update, destination, destination, old, next)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	want := []projectapp.Write{{Destination: destination, Name: "axiom.yaml", Action: projectapp.Replace}, {Destination: destination, Name: "context/removed.md", Action: projectapp.Remove}}
	if !reflect.DeepEqual(update.Writes(), want) {
		t.Fatal("deletion not approved")
	}
	changed := update.Writes()
	changed[0].Name = "unreviewed"
	if !reflect.DeepEqual(update.Writes(), want) {
		t.Fatal("mutable approval write set")
	}
	renamed, issuesDomain := p.Propose(project.Intent{Slug: project.Set("renamed")})
	if len(issuesDomain) != 0 {
		t.Fatal(issuesDomain)
	}
	moved, issues := projectapp.ReadSnapshot(fixtureCodec{renamed}, []byte("renamed manifest"), nil)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	target := projectapp.NewDestination()
	move, issues := projectapp.PreviewPortable(projectapp.Move, destination, target, old, moved)
	if len(issues) != 0 || len(move.Writes()) != 3 || move.ExpectedDestination() != projectapp.MissingPortableRevision() {
		t.Fatal("incomplete move authority")
	}
	if update.ExpectedDestination() != old.Revision() {
		t.Fatal("update expected revision lost")
	}
	from, to := move.Destinations()
	if from != destination || to != target || move.Operation() != projectapp.Move || !move.Before().Project().Equivalent(p) || !move.After().Project().Equivalent(renamed) {
		t.Fatal("move intent lost")
	}
	store := &portableSpy{}
	if got := projectapp.ApplyPortable(context.Background(), store, move, approved(t, update), move.Expected()); got.Status != projectapp.Denied || store.calls != 0 {
		t.Fatal("update consent reused for move")
	}
}

func TestPreviewRejectsInvalidOperationIdentityAndScope(t *testing.T) {
	old, next := snapshot(t, "old"), snapshot(t, "next")
	destination := projectapp.NewDestination()
	for _, tc := range []struct {
		op            projectapp.Operation
		from, to      projectapp.Destination
		before, after projectapp.ArtifactSnapshot
	}{
		{projectapp.Create, destination, destination, old, next},
		{projectapp.Update, destination, projectapp.Destination{}, old, next},
		{projectapp.Update, destination, destination, projectapp.ArtifactSnapshot{}, next},
		{projectapp.Update, destination, destination, old, projectapp.ArtifactSnapshot{}},
		{projectapp.Move, destination, projectapp.NewDestination(), old, next},
		{projectapp.Install, destination, destination, old, next},
	} {
		if _, issues := projectapp.PreviewPortable(tc.op, tc.from, tc.to, tc.before, tc.after); len(issues) == 0 {
			t.Fatal("invalid portable preview accepted")
		}
	}
	state := definition(t).State()
	state.ID = "87654321-1234-4abc-8def-123456789abc"
	other, _ := project.New(state)
	s, issues := projectapp.ReadSnapshot(fixtureCodec{other}, []byte("other identity"), nil)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if _, issues := projectapp.PreviewPortable(projectapp.Update, destination, destination, old, s); len(issues) == 0 {
		t.Fatal("identity reassignment accepted")
	}
	if _, issues := projectapp.Confirm(projectapp.Preview{}, projectapp.Human); len(issues) == 0 {
		t.Fatal("zero preview consent")
	}
	if _, issues := projectapp.PreviewLocal(projectapp.Create, destination, projectapp.MissingLocalRevision(), projectapp.LocalSnapshot{}); len(issues) == 0 {
		t.Fatal("wrong local operation")
	}
	if _, issues := projectapp.PreviewLocal(projectapp.Install, destination, projectapp.LocalRevision{}, projectapp.LocalSnapshot{}); len(issues) == 0 {
		t.Fatal("invalid local snapshot")
	}
}

func TestLocalSnapshotKeepsReferenceMetadataDetached(t *testing.T) {
	p := definition(t)
	state := projectapp.LocalState{Source: projectapp.NewDestination(), PortableRevision: snapshot(t, "manifest").Revision(), Credentials: []projectapp.CredentialBinding{{ReferenceKey: "logical", SourceKind: "environment", ItemReference: "SYNTHETIC_ENV_NAME"}}}
	s, issues := projectapp.NewLocalSnapshot(p, state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	state.Credentials[0].ItemReference = "changed"
	copy := s.State()
	copy.Credentials[0].ItemReference = "changed"
	if s.State().Credentials[0].ItemReference != "SYNTHETIC_ENV_NAME" || s.ProjectID() != identity || s.Slug() != "sample" {
		t.Fatal("local snapshot aliased")
	}
	preview, issues := projectapp.PreviewLocal(projectapp.ReplaceBindings, projectapp.NewDestination(), projectapp.ObserveLocalRevision([]byte("old")), s)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	permit, issues := projectapp.AuthorizeLocal(preview, approved(t, preview), preview.Expected())
	if len(issues) != 0 || !permit.Valid() || permit.Snapshot().ProjectID() != identity || permit.Preview().Local().ProjectID() != identity {
		t.Fatal("local permit lost scope")
	}
	if _, issues := projectapp.PreviewLocal(projectapp.ReplaceBindings, projectapp.NewDestination(), projectapp.MissingLocalRevision(), s); len(issues) == 0 {
		t.Fatal("replacement of missing record")
	}
	if _, issues := projectapp.NewLocalSnapshot(project.Project{}, state); len(issues) == 0 {
		t.Fatal("invalid local identity accepted")
	}
}

func TestOutcomeClassificationPreservesActualCommit(t *testing.T) {
	for _, tc := range []struct {
		name            string
		portable, local projectapp.MutationResult
		attempted       bool
		want            projectapp.Classification
	}{
		{"pre commit", projectapp.MutationResult{Commit: projectapp.NotCommitted, Status: projectapp.Failed}, projectapp.MutationResult{}, false, projectapp.PreCommitFailure},
		{"committed", projectapp.MutationResult{Commit: projectapp.Committed, Status: projectapp.Applied}, projectapp.MutationResult{}, false, projectapp.MutationCommitted},
		{"committed local failure", projectapp.MutationResult{Commit: projectapp.Committed, Status: projectapp.Applied}, projectapp.MutationResult{Commit: projectapp.NotCommitted, Status: projectapp.Failed}, true, projectapp.CommittedWithLocalFailure},
		{"local uncertain after commit", projectapp.MutationResult{Commit: projectapp.Committed, Status: projectapp.Applied}, projectapp.MutationResult{}, true, projectapp.CommittedWithLocalFailure},
		{"post commit error", projectapp.MutationResult{Commit: projectapp.Committed, Status: projectapp.Failed}, projectapp.MutationResult{}, false, projectapp.RecoveryNeededOutcome},
		{"unknown", projectapp.MutationResult{}, projectapp.MutationResult{}, false, projectapp.RecoveryNeededOutcome},
		{"no op", projectapp.MutationResult{Commit: projectapp.NotCommitted, Status: projectapp.Unchanged}, projectapp.MutationResult{}, false, projectapp.NoMutation},
		{"local only commit", projectapp.MutationResult{Commit: projectapp.NotCommitted, Status: projectapp.Unchanged}, projectapp.MutationResult{Commit: projectapp.Committed, Status: projectapp.Applied}, true, projectapp.MutationCommitted},
		{"local only failure", projectapp.MutationResult{Commit: projectapp.NotCommitted, Status: projectapp.Unchanged}, projectapp.MutationResult{Commit: projectapp.NotCommitted, Status: projectapp.Failed}, true, projectapp.PreCommitFailure},
		{"local only uncertain", projectapp.MutationResult{Commit: projectapp.NotCommitted, Status: projectapp.Unchanged}, projectapp.MutationResult{}, true, projectapp.RecoveryNeededOutcome},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := projectapp.Outcome{Portable: tc.portable, Local: tc.local, LocalAttempted: tc.attempted, Readiness: projectapp.ReadinessUnverified}
			if o.Classify() != tc.want {
				t.Fatalf("wrong classification: %v", o.Classify())
			}
			if o.Portable.Commit != tc.portable.Commit || o.Readiness != projectapp.ReadinessUnverified {
				t.Fatal("classification rewrote facts")
			}
		})
	}
	for _, status := range []projectapp.MutationStatus{projectapp.Denied, projectapp.Cancelled, projectapp.Conflict, projectapp.Unchanged, projectapp.Failed} {
		r := (projectapp.MutationResult{Commit: projectapp.Committed, Status: status}).Normalized()
		if r.Commit != projectapp.Committed || r.Status != projectapp.RecoveryRequired {
			t.Fatal("false rollback")
		}
	}
	for _, r := range []projectapp.MutationResult{{Commit: projectapp.NotCommitted, Status: projectapp.Applied}, {Commit: projectapp.CommitStatus(99)}, {Commit: projectapp.NotCommitted, Status: projectapp.MutationStatus(99)}} {
		if r.Normalized().Status != projectapp.RecoveryRequired {
			t.Fatal("invalid outcome claimed success")
		}
	}
}

func TestDeterministicSafeIssueOrderingAndInspectionWarnings(t *testing.T) {
	input := []projectapp.Issue{
		{Phase: projectapp.PersistencePhase, Field: projectapp.ArtifactsField, Code: projectapp.StoreFailure},
		{Phase: projectapp.ApprovalPhase, Field: projectapp.RevisionField, Code: projectapp.RevisionMismatch},
		{Phase: projectapp.ApprovalPhase, Field: projectapp.AuthorityField, Code: projectapp.StaleAuthority, Index: 2},
		{Phase: projectapp.ApprovalPhase, Field: projectapp.AuthorityField, Code: projectapp.MissingAuthority},
		{Phase: projectapp.ApprovalPhase, Field: projectapp.AuthorityField, Code: projectapp.StaleAuthority, Index: 1},
	}
	want := projectapp.OrderedIssues(input)
	for i := 0; i < len(input); i++ {
		input = append(input[1:], input[0])
		if !reflect.DeepEqual(want, projectapp.OrderedIssues(input)) {
			t.Fatal("unstable issue order")
		}
	}
	if want[0].Code != projectapp.MissingAuthority || want[1].Index != 1 {
		t.Fatal("wrong diagnostic ordering")
	}
	for _, i := range want {
		if i.Severity() != "error" || i.Category() == "" || i.Remedy() == "" {
			t.Fatal("incomplete safe issue")
		}
	}
	if projectapp.Code(255).String() != "invalid_issue" || projectapp.Field(255).String() != "project" {
		t.Fatal("unsafe unknown diagnostic")
	}
	for _, status := range []projectapp.InspectionStatus{projectapp.InspectionUnavailable, projectapp.InspectionCompleted, projectapp.InspectionFinding, projectapp.InspectionFailed} {
		issues := projectapp.InspectionWarnings(projectapp.Inspection{Status: status})
		if status == projectapp.InspectionCompleted {
			if len(issues) != 0 {
				t.Fatal("successful scan warning")
			}
			continue
		}
		if len(issues) != 1 || issues[0].Severity() != "warning" || issues[0].Category() != "inspection" {
			t.Fatal("optional scanner invalidates input")
		}
		if strings.Contains(issues[0].Remedy(), "secret absent") {
			t.Fatal("scanner certifies absence")
		}
	}
}

// Denial spies are test-only traps, not production process/network/secret ports.
// Static import/symbol checks close ambient escape routes for this package.
type deniedEffects struct{ writes, processes, networks, secrets int }

func (d *deniedEffects) write()   { d.writes++ }
func (d *deniedEffects) process() { d.processes++ }
func (d *deniedEffects) network() { d.networks++ }
func (d *deniedEffects) secret()  { d.secrets++ }

type deniedStore struct{ effects *deniedEffects }

func (s deniedStore) CommitPortable(context.Context, projectapp.AuthorizedPortable) projectapp.MutationResult {
	s.effects.write()
	s.effects.process()
	s.effects.network()
	s.effects.secret()
	return projectapp.MutationResult{}
}
func TestDeniedEffectsAndCancelledGate(t *testing.T) {
	effects := &deniedEffects{}
	p := preview(t)
	projectapp.ApplyPortable(context.Background(), deniedStore{effects}, p, projectapp.Authority{}, p.Expected())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := projectapp.ApplyPortable(ctx, deniedStore{effects}, p, approved(t, p), p.Expected())
	if result.Status != projectapp.Cancelled || *effects != (deniedEffects{}) {
		t.Fatal("denied external effect")
	}
	local := localPreview(t)
	store := &localSpy{}
	if result := projectapp.ApplyLocal(ctx, store, local, approved(t, local), local.Expected()); result.Status != projectapp.Cancelled || store.calls != 0 {
		t.Fatal("cancelled local side effect")
	}
	if result := projectapp.ApplyPortable(context.Background(), nil, p, approved(t, p), p.Expected()); result.Commit != projectapp.NotCommitted {
		t.Fatal("absent store commit")
	}
	if result := projectapp.ApplyLocal(context.Background(), nil, local, approved(t, local), local.Expected()); result.Commit != projectapp.NotCommitted {
		t.Fatal("absent local store commit")
	}
}

// Barrier and fault injection belong to the substitutable writer. This fake
// proves request propagation/CAS obligations, not a filesystem protocol.
type barrierStore struct {
	entered  chan struct{}
	release  chan struct{}
	mu       sync.Mutex
	revision projectapp.PortableRevision
}

func (s *barrierStore) CommitPortable(_ context.Context, request projectapp.AuthorizedPortable) projectapp.MutationResult {
	s.entered <- struct{}{}
	<-s.release
	s.mu.Lock()
	defer s.mu.Unlock()
	if !request.Valid() {
		return projectapp.MutationResult{Commit: projectapp.NotCommitted, Status: projectapp.Denied}
	}
	if s.revision != request.Preview().Expected().Portable {
		return projectapp.MutationResult{Commit: projectapp.NotCommitted, Status: projectapp.Conflict}
	}
	s.revision = request.Snapshot().Revision()
	return projectapp.MutationResult{Commit: projectapp.Committed, Status: projectapp.Applied}
}
func TestBarrierSeamSupportsConflictAndLateRevocation(t *testing.T) {
	for _, revoke := range []bool{false, true} {
		p := preview(t)
		a := approved(t, p)
		store := &barrierStore{entered: make(chan struct{}, 2), release: make(chan struct{}), revision: p.Expected().Portable}
		results := make(chan projectapp.MutationResult, 2)
		for i := 0; i < 2; i++ {
			go func() { results <- projectapp.ApplyPortable(context.Background(), store, p, a, p.Expected()) }()
		}
		<-store.entered
		<-store.entered
		if revoke {
			a.Revoke()
		}
		close(store.release)
		first, second := <-results, <-results
		committed := 0
		for _, r := range []projectapp.MutationResult{first, second} {
			if r.Commit == projectapp.Committed {
				committed++
			}
		}
		if !revoke && committed != 1 {
			t.Fatal("fake CAS allowed conflicting commits")
		}
		if revoke && committed != 0 {
			t.Fatal("late revocation ignored")
		}
	}
}

type failedCodec struct{}

func (failedCodec) Decode([]byte) (project.Project, []projectapp.Issue) {
	return project.Project{}, []projectapp.Issue{{Code: projectapp.InvalidSnapshot}}
}
func (failedCodec) Encode(project.Project) ([]byte, []projectapp.Issue) {
	return nil, []projectapp.Issue{{Code: projectapp.InvalidSnapshot}}
}
func TestBoundaryFailuresCannotBeReinterpretedAsSuccess(t *testing.T) {
	if _, issues := projectapp.ReadSnapshot(failedCodec{}, []byte("SYNTHETIC_REJECTED_VALUE"), nil); len(issues) != 1 || issues[0].Code != projectapp.InvalidSnapshot {
		t.Fatal("codec failure lost")
	}
	p := localPreview(t)
	if _, issues := projectapp.PreviewLocal(projectapp.Install, p.Local().State().Source, projectapp.MissingLocalRevision(), p.Local()); len(issues) == 0 {
		t.Fatal("portable source authorized as local destination")
	}
	o := projectapp.Outcome{Portable: projectapp.MutationResult{Commit: projectapp.NotCommitted, Status: projectapp.Failed}, Local: projectapp.MutationResult{Commit: projectapp.Committed, Status: projectapp.Applied}, LocalAttempted: true}
	if o.Classify() != projectapp.PreCommitFailure {
		t.Fatal("local success hid portable failure")
	}
	for _, code := range []projectapp.Code{projectapp.InvalidSnapshot, projectapp.RecoveryNeeded} {
		issue := projectapp.Issue{Code: code}
		if issue.Category() == "" || issue.Remedy() == "" {
			t.Fatal("missing category/remedy")
		}
	}
	input := []projectapp.Issue{{Field: projectapp.Field(254), Code: projectapp.Code(254), Index: 1}, {Field: projectapp.Field(255), Code: projectapp.Code(255), Index: 2}, {Field: projectapp.Field(255), Code: projectapp.Code(254), Index: 1}}
	expected := projectapp.OrderedIssues(input)
	for i := 0; i < len(input); i++ {
		input = append(input[1:], input[0])
		if !reflect.DeepEqual(expected, projectapp.OrderedIssues(input)) {
			t.Fatal("unknown diagnostic vocabulary breaks deterministic order")
		}
	}
}

func TestRevisionMetadataCanBeRecordedWithoutBodiesOrFreshnessClaims(t *testing.T) {
	s := snapshot(t, "synthetic content")
	digest, known := s.Revision().Digest()
	if !known || projectapp.RecordedPortableRevision(digest) != s.Revision() {
		t.Fatal("portable revision cannot round trip through record metadata")
	}
	if _, known := projectapp.MissingPortableRevision().Digest(); known {
		t.Fatal("absence became present content")
	}
	if _, known := projectapp.MissingLocalRevision().Digest(); known {
		t.Fatal("missing local digest known")
	}
	if _, known := projectapp.ObserveLocalRevision([]byte("record")).Digest(); !known {
		t.Fatal("observed local digest missing")
	}
	if len((projectapp.ArtifactSnapshot{}).Digests()) != 0 {
		t.Fatal("zero snapshot has digest metadata")
	}
	complete, issues := projectapp.ReadSnapshot(fixtureCodec{definition(t)}, s.Manifest(), []projectapp.Document{{Name: "context/a.md", Content: []byte("text")}})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	digests := complete.Digests()
	if len(digests) != 2 || digests[0].Name != "axiom.yaml" || digests[1].Name != "context/a.md" {
		t.Fatal("incomplete digest metadata")
	}
	local, issues := projectapp.NewLocalSnapshot(definition(t), projectapp.LocalState{Source: projectapp.NewDestination(), PortableRevision: complete.Revision(), ArtifactDigests: digests})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	digests[0].Name = "changed"
	if local.State().ArtifactDigests[0].Name != "axiom.yaml" {
		t.Fatal("record metadata alias")
	}
}
