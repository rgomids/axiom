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
		wire, err := EncodeSingleStageRun(r)
		if err != nil {
			t.Fatal(err)
		}
		c, err := DecodeSingleStageRun(wire)
		if err != nil {
			t.Fatal(err)
		}
		change(&c)
		if ValidateSingleStageRun(c) == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if _, err := SingleStageScopeDigest(SingleStageScope{ProjectID: "p", RepositoryKey: "r", WorkItem: "w", ExecutionID: "not-a-uuid", StageID: "s"}); err == nil {
		t.Fatal("invalid scope digested")
	}
}
