package executiongraph

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

func singleStageFixture(t *testing.T) (SingleStageRun, SingleStageAuthority) {
	t.Helper()
	profile := &runtimeprofile.ResolvedProfile{ProfileID: "codex-high", RuntimeID: "codex", Model: "gpt-synthetic", Effort: runtimeprofile.Effort{Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"}, ProfileDigest: strings.Repeat("1", 64), ConfigurationRevision: 1, ConfigurationDigest: strings.Repeat("2", 64), ExecutableDigest: strings.Repeat("3", 64), RuntimeVersion: "1.0.0", InvocationMode: "noninteractive", ObservationRevision: 1, ObservationDigest: strings.Repeat("4", 64), EffortSourceDigest: strings.Repeat("5", 64), CapabilityProofReference: "evidence-1"}
	scope := SingleStageScope{ProjectID: "project-1", RepositoryKey: "main", WorkItem: "github:owner/repo#7", ExecutionID: "00000000-0000-4000-8000-000000000001", StageID: "implementation", StageOrdinal: 1}
	digest, err := SingleStageScopeDigest(scope)
	if err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	binding := SingleStageBinding{ExecutionGeneration: strings.Repeat("6", 64), WorkflowRevision: 3, PreviewDigest: strings.Repeat("7", 64), ProposalDigest: strings.Repeat("8", 64), InputDigest: strings.Repeat("9", 64), AxiomDigest: strings.Repeat("a", 64), Workspace: "/synthetic/worktrees/one", Inbox: "/synthetic/inboxes/one", Effects: []Effect{{Kind: "repository-write", Target: "src"}}, Controls: ExecutionControls{Timeout: time.Hour, MaximumAttempts: 3}, Resolution: Resolution{ResolvedProfile: profile, RuntimeID: "codex", ModelProfileID: "codex-high", ConfigurationRevision: 1, ObservationRevision: 1}}
	a := SingleStageAuthority{FormatVersion: 2, Scope: SingleStageAuthorityScope, ProjectID: scope.ProjectID, ExecutionID: scope.ExecutionID, StageID: scope.StageID, StageOrdinal: scope.StageOrdinal, ScopeDigest: digest, ExecutionGeneration: binding.ExecutionGeneration, WorkflowRevision: binding.WorkflowRevision, PreviewDigest: binding.PreviewDigest, DispatchDigest: binding.PreviewDigest, ProposalDigest: binding.ProposalDigest, AttemptNumber: 1, Effects: binding.Effects, Disposition: DispositionNotRequired, ActorKind: "human-operator", Actor: "synthetic-operator", RecordedAt: created}
	ref, err := SingleStageAuthorityDigest(a)
	if err != nil {
		t.Fatal(err)
	}
	return SingleStageRun{Kind: SingleStageRunKind, FormatVersion: 1, ScopeDigest: digest, Scope: scope, Binding: binding, CreatedAt: created, Authorities: []string{ref}, Attempts: []SuccessorAttempt{}}, a
}

func finishedAttempt(id string, number uint32, epoch uint64, status AttemptStatus, reason ClassificationReason, start time.Time) SuccessorAttempt {
	finish := start.Add(time.Minute)
	a := SuccessorAttempt{AttemptID: id, Number: number, Status: status, StartedAt: &start, FinishedAt: &finish, LeaseEpoch: epoch, PacketDigest: strings.Repeat("c", 64), ClassificationReason: reason, Process: &ProcessCorrelation{PID: 10, StartIdentity: "s", GroupIdentity: "g"}, ReconciliationFacts: []ReconciliationFact{}}
	if status == AttemptFailed && reason == ProcessFailed {
		a.ExitCode = 7
	}
	return a
}

func TestSingleStageCanonicalVectors(t *testing.T) {
	r, a := singleStageFixture(t)
	start := r.CreatedAt.Add(time.Second)
	r.RunEpoch = 1
	r.Attempts = []SuccessorAttempt{finishedAttempt("00000000-0000-4000-8000-000000000002", 1, 1, AttemptFailed, ProcessFailed, start)}
	for _, vector := range []struct {
		file   string
		encode func() ([]byte, error)
		decode func([]byte) error
	}{
		{"testdata/single-stage/run-v1.json", func() ([]byte, error) { return EncodeSingleStageRun(r) }, func(b []byte) error { _, err := DecodeSingleStageRun(b); return err }},
		{"testdata/single-stage/authority-v2.json", func() ([]byte, error) { return EncodeSingleStageAuthority(a) }, func(b []byte) error { _, err := DecodeSingleStageAuthority(b); return err }},
	} {
		wire, err := vector.encode()
		if err != nil {
			t.Fatal(err)
		}
		golden, err := os.ReadFile(vector.file)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(wire, golden) || vector.decode(golden) != nil {
			t.Fatalf("%s is not the canonical encoding:\n%s", vector.file, wire)
		}
	}
	decoded, err := DecodeSingleStageRun(mustRead(t, "testdata/single-stage/run-v1.json"))
	if err != nil || decoded.ScopeDigest != r.ScopeDigest || decoded.Authorities[0] != r.Authorities[0] {
		t.Fatal("golden ledger lost its scope or authority binding")
	}
	digests := mustRead(t, "testdata/single-stage/digests.txt")
	want := fmt.Sprintf("scope %s\nauthority %s\nrun %x\n", r.ScopeDigest, r.Authorities[0], sha256.Sum256(mustRead(t, "testdata/single-stage/run-v1.json")))
	if string(digests) != want {
		t.Fatalf("digests = %s want %s", digests, want)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSingleStageVersionAdmissionFailsClosed(t *testing.T) {
	r, a := singleStageFixture(t)
	runWire, _ := EncodeSingleStageRun(r)
	receipt, _ := EncodeSingleStageAuthority(a)
	mutate := func(wire []byte, key string, value any) []byte {
		var m map[string]any
		if err := json.Unmarshal(wire, &m); err != nil {
			t.Fatal(err)
		}
		m[key] = value
		b, _ := json.Marshal(m)
		return b
	}
	if _, err := DecodeSingleStageRun(mutate(runWire, "formatVersion", 3)); !errors.Is(err, ErrUnsupportedSingleStageFormat) {
		t.Fatalf("newer ledger = %v", err)
	}
	if _, err := DecodeSingleStageRun(mutate(runWire, "kind", "execution_graph")); err == nil || errors.Is(err, ErrUnsupportedSingleStageFormat) {
		t.Fatal("foreign kind admitted")
	}
	if _, err := DecodeSingleStageRun(mutate(runWire, "stageAttempts", []any{})); err == nil {
		t.Fatal("unknown member admitted")
	}
	for _, version := range []int{1, 3} {
		if _, err := DecodeSingleStageAuthority(mutate(receipt, "formatVersion", version)); !errors.Is(err, ErrUnsupportedSingleStageFormat) {
			t.Fatalf("authority version %d = %v", version, err)
		}
	}
	if _, err := DecodeDispatchAuthority(receipt); err == nil {
		t.Fatal("version-2 single receipt decoded by the version-1 codec")
	}
	legacy := DispatchAuthority{FormatVersion: 1, ProjectID: "project-1", PreviewDigest: strings.Repeat("a", 64), Scope: "single-first-attempt", ParentID: r.Scope.ExecutionID, Revision: 3, ProposalDigest: strings.Repeat("b", 64), Children: []DispatchChildAuthority{{ChildID: r.Scope.ExecutionID, Effects: []Effect{}}}, SingleStageID: "implementation", SingleStageOrdinal: 1, SingleAttemptID: "00000000-0000-4000-8000-000000000009", LedgerDigest: strings.Repeat("c", 64), ActorKind: "human-operator", Actor: "op", RecordedAt: r.CreatedAt}
	legacyWire, err := EncodeDispatchAuthority(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeSingleStageAuthority(legacyWire); err == nil {
		t.Fatal("version-1 receipt mapped onto the single-stage codec")
	}
}

func TestSingleStageBoundsAndScope(t *testing.T) {
	r, _ := singleStageFixture(t)
	for name, change := range map[string]func(*SingleStageRun){
		"attempts>10":   func(r *SingleStageRun) { r.Binding.Controls.MaximumAttempts = 11 },
		"zero attempts": func(r *SingleStageRun) { r.Binding.Controls.MaximumAttempts = 0 },
		"timeout>24h":   func(r *SingleStageRun) { r.Binding.Controls.Timeout = 24*time.Hour + time.Second },
		"scope digest":  func(r *SingleStageRun) { r.Scope.StageOrdinal = 2 },
		"no authority":  func(r *SingleStageRun) { r.Authorities = nil },
		"same paths":    func(r *SingleStageRun) { r.Binding.Inbox = r.Binding.Workspace },
		"two pending":   func(r *SingleStageRun) { r.Authorities = append(r.Authorities, strings.Repeat("d", 64)) },
		"epoch": func(r *SingleStageRun) {
			r.Attempts = []SuccessorAttempt{finishedAttempt("00000000-0000-4000-8000-000000000002", 1, 2, AttemptFailed, ProcessFailed, r.CreatedAt)}
		},
	} {
		c := CloneSingleStageRun(r)
		change(&c)
		if ValidateSingleStageRun(c) == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if _, err := SingleStageScopeDigest(SingleStageScope{ProjectID: "p", RepositoryKey: "r", WorkItem: "w", ExecutionID: "not-a-uuid", StageID: "s"}); err == nil {
		t.Fatal("invalid scope digested")
	}
}

func TestSingleStageUpdateIsAppendOnly(t *testing.T) {
	r, _ := singleStageFixture(t)
	start := r.CreatedAt.Add(time.Second)
	running := SuccessorAttempt{AttemptID: "00000000-0000-4000-8000-000000000002", Number: 1, Status: AttemptRunning, StartedAt: &start, LeaseEpoch: 1, PacketDigest: strings.Repeat("c", 64), Process: &ProcessCorrelation{PID: 10, StartIdentity: "s", GroupIdentity: "g"}, ReconciliationFacts: []ReconciliationFact{}}
	withRunning := CloneSingleStageRun(r)
	withRunning.RunEpoch = 1
	withRunning.Attempts = []SuccessorAttempt{running}
	if err := ValidateSingleStageUpdate(r, withRunning); err != nil {
		t.Fatal(err)
	}
	done := CloneSingleStageRun(withRunning)
	done.Attempts[0] = finishedAttempt(running.AttemptID, 1, 1, AttemptFailed, ProcessFailed, start)
	if err := ValidateSingleStageUpdate(withRunning, done); err != nil {
		t.Fatal(err)
	}
	for name, next := range map[string]func() SingleStageRun{
		"terminal rewrite": func() SingleStageRun {
			c := CloneSingleStageRun(done)
			c.Attempts[0] = finishedAttempt(running.AttemptID, 1, 1, AttemptBlocked, ProcessOutcomeUnknown, start)
			return c
		},
		"attempt removal": func() SingleStageRun { c := CloneSingleStageRun(done); c.Attempts = nil; return c },
		"epoch rollback":  func() SingleStageRun { c := CloneSingleStageRun(done); c.RunEpoch = 0; return c },
		"binding change": func() SingleStageRun {
			c := CloneSingleStageRun(done)
			c.Binding.PreviewDigest = strings.Repeat("e", 64)
			return c
		},
		"process rewrite": func() SingleStageRun {
			c := CloneSingleStageRun(withRunning)
			p := *c.Attempts[0].Process
			p.PID = 11
			c.Attempts[0].Process = &p
			return c
		},
	} {
		base := done
		if name == "process rewrite" {
			base = withRunning
		}
		if ValidateSingleStageUpdate(base, next()) == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestSingleStageDispatchAndRetryAuthority(t *testing.T) {
	r, a := singleStageFixture(t)
	if n, err := CheckSingleStageDispatch(r, a); err != nil || n != 1 {
		t.Fatalf("first dispatch = %d %v", n, err)
	}
	cancelled := CloneSingleStageRun(r)
	cancelled.CancelRequest = &CancelRequest{RequestID: "00000000-0000-4000-8000-000000000020", ParentID: r.Scope.ExecutionID, GraphRevision: 3, ActorKind: "human-operator", Actor: "op", RequestedAt: r.CreatedAt, AuthorityRef: strings.Repeat("f", 64)}
	if _, err := CheckSingleStageDispatch(cancelled, a); err == nil {
		t.Fatal("dispatch after cancellation")
	}
	drift := a
	drift.ExecutionGeneration = strings.Repeat("0", 64)
	if _, err := CheckSingleStageDispatch(r, drift); err == nil {
		t.Fatal("foreign Execution3 generation dispatched")
	}
	failed := CloneSingleStageRun(r)
	failed.RunEpoch = 1
	failed.Attempts = []SuccessorAttempt{finishedAttempt("00000000-0000-4000-8000-000000000002", 1, 1, AttemptFailed, ProcessFailed, r.CreatedAt)}
	if _, err := CheckSingleStageDispatch(failed, a); err == nil {
		t.Fatal("consumed first-attempt receipt reused")
	}
	wire, _ := EncodeSingleStageRun(failed)
	failed.StorageRevision = sha256.Sum256(wire)
	retry := a
	retry.AttemptNumber, retry.PriorAttemptID, retry.LedgerGeneration, retry.Disposition = 2, failed.Attempts[0].AttemptID, fmt.Sprintf("%x", failed.StorageRevision), DispositionPreserve
	if err := CheckSingleStageRetryAuthority(failed, retry); err != nil {
		t.Fatal(err)
	}
	missing := retry
	missing.Disposition = DispositionNotRequired
	if CheckSingleStageRetryAuthority(failed, missing) == nil {
		t.Fatal("repository-write retry without disposition")
	}
	stale := retry
	stale.LedgerGeneration = strings.Repeat("0", 64)
	if CheckSingleStageRetryAuthority(failed, stale) == nil {
		t.Fatal("retry on a foreign ledger generation")
	}
	ref, _ := SingleStageAuthorityDigest(retry)
	pending := CloneSingleStageRun(failed)
	pending.Authorities = append(pending.Authorities, ref)
	if n, err := CheckSingleStageDispatch(pending, retry); err != nil || n != 2 {
		t.Fatalf("retry dispatch = %d %v", n, err)
	}
	if ok, _ := SingleStageRetryEligible(pending); ok {
		t.Fatal("second retry receipt while one is pending")
	}
	exhausted := CloneSingleStageRun(failed)
	exhausted.Binding.Controls.MaximumAttempts = 1
	if ok, _ := SingleStageRetryEligible(exhausted); ok {
		t.Fatal("exhausted budget eligible")
	}
	unknown := CloneSingleStageRun(failed)
	unknown.Attempts[0] = finishedAttempt(failed.Attempts[0].AttemptID, 1, 1, AttemptUnknown, ProcessOutcomeUnknown, r.CreatedAt)
	unknown.Attempts[0].AmbiguousEffect = true
	if ok, _ := SingleStageRetryEligible(unknown); ok {
		t.Fatal("unreconciled unknown eligible")
	}
}

func TestSingleStageReconciliationNeverChangesStatus(t *testing.T) {
	r, _ := singleStageFixture(t)
	r.RunEpoch = 1
	r.Attempts = []SuccessorAttempt{finishedAttempt("00000000-0000-4000-8000-000000000002", 1, 1, AttemptUnknown, ProcessOutcomeUnknown, r.CreatedAt)}
	o := ReconciliationObservations{ParentID: r.Scope.ExecutionID, ChildID: r.Scope.ExecutionID, GraphRevision: 3, AttemptID: r.Attempts[0].AttemptID, LeaseEpoch: 1, WorktreeDigest: strings.Repeat("1", 64), CanonicalRecordsDigest: strings.Repeat("2", 64), QuarantinedInboxDigest: strings.Repeat("3", 64)}
	digest, _ := o.Digest()
	f := ReconciliationFact{FactID: "00000000-0000-4000-8000-000000000030", ParentID: r.Scope.ExecutionID, ChildID: r.Scope.ExecutionID, GraphRevision: 3, AttemptID: r.Attempts[0].AttemptID, ObservationsDigest: digest, Classification: "retry_safe", AuthorityRef: strings.Repeat("4", 64), ActorKind: "human-operator", Actor: "op", RecordedAt: r.CreatedAt.Add(time.Hour)}
	next, err := AppendSingleStageReconciliation(r, f, o)
	if err != nil {
		t.Fatal(err)
	}
	if next.Attempts[0].Status != AttemptUnknown || len(next.Attempts[0].ReconciliationFacts) != 1 {
		t.Fatal("reconciliation changed status")
	}
	if ok, _ := SingleStageRetryEligible(next); !ok {
		t.Fatal("retry_safe unknown not eligible for explicit retry")
	}
	alive := o
	alive.ProcessAlive = true
	if _, err := AppendSingleStageReconciliation(r, f, alive); err == nil {
		t.Fatal("retry_safe recorded with a live process")
	}
	f.AuthorityRef = r.Authorities[0]
	if _, err := AppendSingleStageReconciliation(r, f, o); err == nil {
		t.Fatal("dispatch authority reused as reconciliation authority")
	}
}
