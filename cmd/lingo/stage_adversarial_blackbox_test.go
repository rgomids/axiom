package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/executiongraph"
)

type singleCrashStatus struct {
	Single struct {
		ScopeDigest string `json:"scopeDigest"`
		Binding     struct {
			Workspace string `json:"workspace"`
		} `json:"binding"`
		Attempts []struct {
			AttemptID            string                              `json:"attemptId"`
			Status               string                              `json:"status"`
			ClassificationReason string                              `json:"classificationReason"`
			Process              *executiongraph.ProcessCorrelation  `json:"process"`
			Facts                []executiongraph.ReconciliationFact `json:"reconciliationFacts"`
		} `json:"attempts"`
	} `json:"single"`
	Health map[string]struct {
		Stale           bool `json:"stale"`
		Orphan          bool `json:"orphan"`
		LivenessUnknown bool `json:"livenessUnknown"`
	} `json:"health"`
	Next string `json:"next"`
}

func decodeSingleCrashStatus(t *testing.T, wire json.RawMessage) singleCrashStatus {
	t.Helper()
	var value singleCrashStatus
	if err := json.Unmarshal(wire, &value); err != nil {
		t.Fatalf("single crash status %s: %v", wire, err)
	}
	return value
}

func psField(t *testing.T, field string, pid int64) string {
	t.Helper()
	command := exec.Command("/bin/ps", "-o", field+"=", "-p", strconv.FormatInt(pid, 10))
	wire, err := command.Output()
	if err != nil {
		t.Fatalf("ps %s for PID %d: %v", field, pid, err)
	}
	return strings.TrimSpace(string(wire))
}

func verifiedSingleSupervisor(t *testing.T, j *stageJourney, pid int64, projectID, executionID, scopeDigest string) bool {
	t.Helper()
	args := strings.Fields(psField(t, "args", pid))
	return len(args) >= 7 && args[0] == j.axiom && args[1] == "__stage-supervisor" && args[2] == "single" && args[3] == projectID && args[4] == executionID && args[5] == scopeDigest
}

func recordedGroupID(t *testing.T, recorded executiongraph.ProcessCorrelation) int64 {
	t.Helper()
	_, tail, found := strings.Cut(recorded.GroupIdentity, ":")
	if !found {
		tail = recorded.GroupIdentity
	}
	if found {
		if _, next, more := strings.Cut(tail, ":"); more {
			tail = next
		}
	}
	group, err := strconv.ParseInt(tail, 10, 64)
	if err != nil || group <= 0 || group != recorded.PID {
		t.Fatalf("recorded fake process is not its own process-group leader: %+v", recorded)
	}
	return group
}

func TestInstalledSyntheticSingleLifecycleAdversarial(t *testing.T) {
	j := newStageJourney(t)
	j.run(t, 0, "runtime", "codex", "install")
	j.seedEvidence(t, "codex", "profile-codex", capabilityproof.RepositoryWriteVariant, stageJourneyCaps...)
	unit := writeUnit("owner", []string{"docs"}, "docs/intake.md")
	plan := writeStagePlanDocument(t, []executiongraph.WorkUnit{unit}, []executiongraph.Effect{{Kind: "repository-write", Target: "docs/intake.md"}})

	saveTargetLink(t, j.env.state, workItemSourceProjectID, "20", "owner/repo")
	view := j.startExecution(t, "github:owner/repo#20")
	args := stageRunArgs(view, "intake", plan, j.policy, j.workspace)
	j.script(t, fakeStageScript{Agents: map[string][]fakeStageStep{"owner": {{Op: "sleep", Millis: 15000}, {Op: "publish", Kind: "result", Fields: []string{"outcome=succeeded"}}}}})
	digest := j.run(t, 0, append(args, "--authorize-local")...).reference(t, "stage-preview:")
	j.run(t, 0, append(args, "--preview-digest", digest, "--authorize-local", "--authorize-external")...)
	running := j.waitStatus(t, view, func(stage json.RawMessage) bool {
		var value struct {
			Single struct {
				Attempts []struct {
					Status string `json:"status"`
				} `json:"attempts"`
			} `json:"single"`
		}
		return json.Unmarshal(stage, &value) == nil && len(value.Single.Attempts) == 1 && value.Single.Attempts[0].Status == "running"
	})
	before := decodeSingle(t, running)
	attemptID := before.Single.Attempts[0].AttemptID
	cancel := append([]string{"workflow", "stage", "cancel"}, stageSelector(view)...)
	cancelDigest := j.run(t, 0, cancel...).reference(t, "stage-preview:")
	if refused := j.run(t, 1, append(cancel, "--preview-digest", cancelDigest)...); refused.Status != "denied_authority" {
		t.Fatalf("cancel without local authority: %+v", refused)
	}
	if refused := j.run(t, 1, append(cancel, "--preview-digest", strings.Repeat("0", 64), "--authorize-local")...); refused.Category != "stale_preview" || refused.Diagnostic == nil {
		t.Fatalf("cancel with stale preview authority: %+v", refused)
	}
	if event := j.run(t, 0, append(cancel, "--preview-digest", cancelDigest, "--authorize-local")...); event.Category != "cancellation_requested" {
		t.Fatalf("cancel request: %+v", event)
	}
	settled := j.waitStatus(t, view, latestSettled)
	status := decodeSingle(t, settled)
	if len(status.Single.Attempts) != 1 || status.Single.Attempts[0].AttemptID != attemptID || status.Single.Attempts[0].Status != "cancelled" || status.Single.Attempts[0].ClassificationReason != "cancel_confirmed" {
		t.Fatalf("cancelled state: %+v", status)
	}
	reopened := decodeSingle(t, j.run(t, 0, append([]string{"workflow", "stage", "status"}, stageSelector(view)...)...).Stage)
	if len(reopened.Single.Attempts) != 1 || reopened.Single.Attempts[0].AttemptID != attemptID || reopened.Single.Attempts[0].Status != "cancelled" {
		t.Fatalf("persisted status changed on a fresh CLI process: %+v", reopened)
	}
	if len(j.invocations(t)) != 1 {
		t.Fatalf("cancellation dispatched another Runtime: %+v", j.invocations(t))
	}

	for _, scenario := range []struct {
		name       string
		workItem   string
		steps      []fakeStageStep
		wantReason string
	}{
		{name: "process failure", workItem: "github:owner/repo#21", steps: []fakeStageStep{{Op: "exit", Code: 7}}, wantReason: "process_failed"},
		{name: "missing result", workItem: "github:owner/repo#22", steps: nil, wantReason: "result_missing"},
		{name: "timeout", workItem: "github:owner/repo#23", steps: []fakeStageStep{{Op: "sleep", Millis: 5000}}, wantReason: "timeout_stopped"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			externalID := strings.TrimPrefix(scenario.workItem, "github:owner/repo#")
			saveTargetLink(t, j.env.state, workItemSourceProjectID, externalID, "owner/repo")
			current := j.startExecution(t, scenario.workItem)
			currentUnit := unit
			if scenario.name == "timeout" {
				currentUnit.Controls.Timeout = 250 * time.Millisecond
			}
			currentPlan := writeStagePlanDocument(t, []executiongraph.WorkUnit{currentUnit}, []executiongraph.Effect{{Kind: "repository-write", Target: "docs/intake.md"}})
			currentArgs := stageRunArgs(current, "intake", currentPlan, j.policy, j.workspace)
			j.script(t, fakeStageScript{Agents: map[string][]fakeStageStep{"owner": scenario.steps}})
			preview := j.run(t, 0, append(currentArgs, "--authorize-local")...).reference(t, "stage-preview:")
			j.run(t, 0, append(currentArgs, "--preview-digest", preview, "--authorize-local", "--authorize-external")...)
			persisted := decodeSingle(t, j.waitStatus(t, current, latestSettled))
			if len(persisted.Single.Attempts) != 1 || persisted.Single.Attempts[0].Status != "failed" || persisted.Single.Attempts[0].ClassificationReason != scenario.wantReason {
				t.Fatalf("attempt classification: %+v", persisted)
			}
			attempt := persisted.Single.Attempts[0]
			reconcile := append([]string{"workflow", "stage", "reconcile"}, stageSelector(current)...)
			reconcile = append(reconcile, "--child", current.ExecutionID, "--attempt", attempt.AttemptID)
			refused := j.run(t, 1, reconcile...)
			if refused.Category != "reconcile_not_required" {
				t.Fatalf("terminal attempt reconciliation: %+v", refused)
			}
			after := decodeSingle(t, j.run(t, 0, append([]string{"workflow", "stage", "status"}, stageSelector(current)...)...).Stage)
			if len(after.Single.Attempts) != 1 || after.Single.Attempts[0].Status != "failed" || after.Single.Attempts[0].ClassificationReason != scenario.wantReason {
				t.Fatalf("reconcile changed terminal status: %+v", after)
			}
		})
	}

	saveTargetLink(t, j.env.state, workItemSourceProjectID, "24", "owner/repo")
	invalidated := j.startExecution(t, "github:owner/repo#24")
	invalidatedArgs := stageRunArgs(invalidated, "intake", plan, j.policy, j.workspace)
	j.script(t, fakeStageScript{Agents: map[string][]fakeStageStep{"owner": {{Op: "publish", Kind: "result", Fields: []string{"outcome=succeeded"}}}}})
	invalidatedDigest := j.run(t, 0, append(invalidatedArgs, "--authorize-local")...).reference(t, "stage-preview:")
	capabilityArgs := []string{"runtime", "codex", "capability", "invalidate", "--profile", "profile-codex", "--variant", "repository-write", "--repository-root", j.repository, "--workspace-root", j.workspace, "--authorize-local"}
	capabilityPreview := j.run(t, 0, capabilityArgs...)
	var invalidation struct {
		PreviewDigest string `json:"previewDigest"`
	}
	if err := json.Unmarshal(capabilityPreview.Capability, &invalidation); err != nil || invalidation.PreviewDigest == "" {
		t.Fatalf("capability invalidation preview: %s %v", capabilityPreview.Capability, err)
	}
	if applied := j.run(t, 0, append(capabilityArgs, "--preview-digest", invalidation.PreviewDigest)...); applied.Category != "capability_invalidated" {
		t.Fatalf("capability invalidation: %+v", applied)
	}
	if refused := j.run(t, 1, append(invalidatedArgs, "--preview-digest", invalidatedDigest, "--authorize-local", "--authorize-external")...); refused.Category != "runtime_unresolvable" {
		t.Fatalf("invalidated Evidence admitted dispatch: %+v", refused)
	}
	fakeRuntime := filepath.Join(j.bin, "codex")
	file, err := os.OpenFile(fakeRuntime, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.WriteString("\nchanged-after-preview\n"); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	refused := j.run(t, 1, append(invalidatedArgs, "--preview-digest", invalidatedDigest, "--authorize-local", "--authorize-external")...)
	if refused.Category != "runtime_unresolvable" {
		t.Fatalf("dispatch after capability invalidation: %+v", refused)
	}
	if calls := j.invocations(t); len(calls) != 4 {
		t.Fatalf("invalidated capability launched Runtime; calls=%+v", calls)
	}
}

func TestInstalledSyntheticSingleSupervisorCrashAndReconcile(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("process identity and /bin/ps verification are supported on Linux and macOS")
	}
	j := newStageJourney(t)
	j.run(t, 0, "runtime", "codex", "install")
	j.seedEvidence(t, "codex", "profile-codex", capabilityproof.RepositoryWriteVariant, stageJourneyCaps...)
	saveTargetLink(t, j.env.state, workItemSourceProjectID, "30", "owner/repo")
	view := j.startExecution(t, "github:owner/repo#30")
	unit := writeUnit("owner", []string{"docs"}, "docs/intake.md")
	unit.Controls.MaximumAttempts = 2
	unit.Controls.Timeout = 2 * time.Minute
	plan := writeStagePlanDocument(t, []executiongraph.WorkUnit{unit}, []executiongraph.Effect{{Kind: "repository-write", Target: "docs/intake.md"}})
	args := stageRunArgs(view, "intake", plan, j.policy, j.workspace)
	j.script(t, fakeStageScript{StartupMillis: 1500, Agents: map[string][]fakeStageStep{"owner": {{Op: "sleep", Millis: 60000}, {Op: "publish", Kind: "result", Fields: []string{"outcome=succeeded"}}}}})
	digest := j.run(t, 0, append(args, "--authorize-local")...).reference(t, "stage-preview:")
	j.run(t, 0, append(args, "--preview-digest", digest, "--authorize-local", "--authorize-external")...)
	runningWire := j.waitStatus(t, view, func(stage json.RawMessage) bool {
		value := decodeSingleCrashStatus(t, stage)
		if len(value.Single.Attempts) != 1 || value.Single.Attempts[0].Status != "running" || value.Single.Attempts[0].Process == nil {
			return false
		}
		for _, call := range j.invocations(t) {
			if call.Attempt == value.Single.Attempts[0].AttemptID && call.Runtime == "codex" && call.Agent == "owner" {
				return true
			}
		}
		return false
	})
	running := decodeSingleCrashStatus(t, runningWire)
	attempt := running.Single.Attempts[0]
	recorded := *attempt.Process
	calls := j.invocations(t)
	fakeCall := false
	for _, call := range calls {
		fakeCall = fakeCall || call.Attempt == attempt.AttemptID && call.Runtime == "codex" && call.Agent == "owner"
	}
	if !fakeCall {
		t.Fatalf("recorded attempt did not match the FAKE invocation: attempt=%s calls=%+v", attempt.AttemptID, calls)
	}
	fakeArgs := psField(t, "args", recorded.PID)
	if !strings.Contains(fakeArgs, filepath.Join(j.bin, "codex")) {
		t.Fatalf("recorded process is not the fake codex executable: %q", fakeArgs)
	}
	supervisorPID, err := strconv.ParseInt(psField(t, "ppid", recorded.PID), 10, 64)
	if err != nil || supervisorPID <= 1 || !verifiedSingleSupervisor(t, j, supervisorPID, workItemSourceProjectID, view.ExecutionID, running.Single.ScopeDigest) {
		t.Fatalf("attempt parent is not its scoped single-stage supervisor: ppid=%d err=%v args=%q", supervisorPID, err, psField(t, "args", supervisorPID))
	}
	groupID := recordedGroupID(t, recorded)
	groupField := psField(t, "pgid", recorded.PID)
	observedGroup, err := strconv.ParseInt(groupField, 10, 64)
	if err != nil || observedGroup != groupID {
		t.Fatalf("FAKE attempt process group differs from its recorded group: recorded=%d ps=%q", groupID, groupField)
	}

	var supervisorStopped bool
	var fakeGroupStopped bool
	t.Cleanup(func() {
		if !supervisorStopped {
			wire, psErr := exec.Command("/bin/ps", "-o", "args=", "-p", strconv.FormatInt(supervisorPID, 10)).Output()
			fields := strings.Fields(string(wire))
			if psErr == nil && len(fields) >= 7 && fields[0] == j.axiom && fields[1] == "__stage-supervisor" && fields[2] == "single" && fields[3] == workItemSourceProjectID && fields[4] == view.ExecutionID && fields[5] == running.Single.ScopeDigest {
				_ = syscall.Kill(int(supervisorPID), syscall.SIGKILL)
			}
		}
		if !fakeGroupStopped {
			current, observeErr := executiongraph.ObserveRecordedProcess(recorded)
			if observeErr == nil && current != nil && *current == recorded {
				_ = syscall.Kill(-int(groupID), syscall.SIGKILL)
			}
		}
	})
	if !verifiedSingleSupervisor(t, j, supervisorPID, workItemSourceProjectID, view.ExecutionID, running.Single.ScopeDigest) {
		t.Fatal("single-stage supervisor identity changed before kill")
	}
	if err := syscall.Kill(int(supervisorPID), syscall.SIGKILL); err != nil {
		t.Fatalf("kill verified single-stage supervisor: %v", err)
	}
	supervisorStopped = true
	staleWire := j.waitStatus(t, view, func(stage json.RawMessage) bool {
		value := decodeSingleCrashStatus(t, stage)
		health := value.Health[attempt.AttemptID]
		return value.Next == "stop_orphan_then_reconcile" && health.Stale && health.Orphan && !health.LivenessUnknown
	})
	stale := decodeSingleCrashStatus(t, staleWire)
	if stale.Single.Attempts[0].AttemptID != attempt.AttemptID || stale.Single.Attempts[0].Status != "running" || stale.Single.Attempts[0].Process == nil || *stale.Single.Attempts[0].Process != recorded {
		t.Fatalf("supervisor crash changed recorded attempt identity: %+v", stale)
	}
	reconcile := append([]string{"workflow", "stage", "reconcile"}, stageSelector(view)...)
	reconcile = append(reconcile, "--child", view.ExecutionID, "--attempt", attempt.AttemptID)
	livePreview := j.run(t, 0, reconcile...).reference(t, "stage-preview:")
	if refused := j.run(t, 1, append(reconcile, "--preview-digest", livePreview, "--authorize-local")...); refused.Category != "stop_orphan_then_reconcile" || refused.Diagnostic == nil || refused.Diagnostic.Capability != "unknown" {
		t.Fatalf("reconcile accepted a live FAKE orphan: %+v", refused)
	}
	current, err := executiongraph.ObserveRecordedProcess(recorded)
	if err != nil || current == nil || *current != recorded {
		t.Fatalf("refused reconciliation did not retain the exact live FAKE attempt: current=%+v err=%v", current, err)
	}
	if err := syscall.Kill(-int(groupID), syscall.SIGKILL); err != nil {
		t.Fatalf("kill exact recorded FAKE attempt group %d: %v", groupID, err)
	}
	fakeGroupStopped = true
	deadWire := j.waitStatus(t, view, func(stage json.RawMessage) bool {
		value := decodeSingleCrashStatus(t, stage)
		health := value.Health[attempt.AttemptID]
		return health.Stale && !health.Orphan && !health.LivenessUnknown
	})
	dead := decodeSingleCrashStatus(t, deadWire)
	if dead.Single.Attempts[0].AttemptID != attempt.AttemptID || dead.Single.Attempts[0].Status != "running" || dead.Single.Attempts[0].Process == nil || *dead.Single.Attempts[0].Process != recorded {
		t.Fatalf("confirmed stop rewrote persisted attempt identity/status: %+v", dead)
	}
	safePreviewEvent := j.run(t, 0, reconcile...)
	safePreview := safePreviewEvent.reference(t, "stage-preview:")
	if !strings.Contains(string(safePreviewEvent.Stage), `"next":"explicit_retry_required"`) {
		t.Fatal("reconcile after exact FAKE process-group stop did not preview an explicit retry")
	}
	if event := j.run(t, 0, append(reconcile, "--preview-digest", safePreview, "--authorize-local")...); event.Category != "reconciliation_recorded" {
		t.Fatalf("safe single reconciliation: %+v", event)
	}
	reconciled := decodeSingleCrashStatus(t, j.run(t, 0, append([]string{"workflow", "stage", "status"}, stageSelector(view)...)...).Stage)
	if len(reconciled.Single.Attempts) != 1 || reconciled.Single.Attempts[0].Status != "running" || len(reconciled.Single.Attempts[0].Facts) != 1 || reconciled.Single.Attempts[0].Facts[0].Classification != "retry_safe" || reconciled.Single.Attempts[0].AttemptID != attempt.AttemptID || reconciled.Single.Attempts[0].Process == nil || *reconciled.Single.Attempts[0].Process != recorded {
		t.Fatalf("reconciliation did not append a fact while preserving running identity: %+v", reconciled)
	}
	retry := append([]string{"workflow", "stage", "retry"}, stageSelector(view)...)
	retry = append(retry, "--expected-revision", strconv.FormatUint(view.Revision, 10), "--stage", "intake", "--plan", plan, "--validator-policy", j.policy, "--child", view.ExecutionID)
	if refused := j.run(t, 1, retry...); refused.Category != "retry_not_eligible" {
		t.Fatalf("retry bypassed unresolved running attempt: %+v", refused)
	}
	resumeDigest := j.run(t, 0, append(args, "--authorize-local")...).reference(t, "stage-preview:")
	resumed := j.run(t, 1, append(args, "--preview-digest", resumeDigest, "--authorize-local", "--authorize-external")...)
	if resumed.Category != "recovery_required" {
		t.Fatalf("resume bypassed unresolved running attempt: %+v", resumed)
	}
	if calls := j.invocations(t); len(calls) != 1 || calls[0].Attempt != attempt.AttemptID {
		t.Fatalf("refused retry/resume changed fake invocation identity: %+v", calls)
	}
	final := decodeSingleCrashStatus(t, j.run(t, 0, append([]string{"workflow", "stage", "status"}, stageSelector(view)...)...).Stage)
	if len(final.Single.Attempts) != 1 || final.Single.Attempts[0].AttemptID != attempt.AttemptID || final.Single.Attempts[0].Status != "running" || len(final.Single.Attempts[0].Facts) != 1 {
		t.Fatalf("refused retry/resume mutated persisted state: %+v", final)
	}
}

func TestInstalledSyntheticSingleLedgerCorruptionFailsClosed(t *testing.T) {
	j := newStageJourney(t)
	j.run(t, 0, "runtime", "codex", "install")
	j.seedEvidence(t, "codex", "profile-codex", capabilityproof.RepositoryWriteVariant, stageJourneyCaps...)
	saveTargetLink(t, j.env.state, workItemSourceProjectID, "31", "owner/repo")
	view := j.startExecution(t, "github:owner/repo#31")
	plan := writeStagePlanDocument(t, []executiongraph.WorkUnit{writeUnit("owner", []string{"docs"}, "docs/intake.md")}, []executiongraph.Effect{{Kind: "repository-write", Target: "docs/intake.md"}})
	args := stageRunArgs(view, "intake", plan, j.policy, j.workspace)
	j.script(t, fakeStageScript{Agents: map[string][]fakeStageStep{"owner": {{Op: "publish", Kind: "result", Fields: []string{"outcome=succeeded"}}}}})
	digest := j.run(t, 0, append(args, "--authorize-local")...).reference(t, "stage-preview:")
	j.run(t, 0, append(args, "--preview-digest", digest, "--authorize-local", "--authorize-external")...)
	settled := decodeSingleCrashStatus(t, j.waitStatus(t, view, latestSettled))
	if len(settled.Single.Attempts) != 1 || settled.Single.Attempts[0].Status != "succeeded" {
		t.Fatalf("corruption fixture did not settle: %+v", settled)
	}
	ledger := filepath.Join(j.env.state, "single-stage-runs", "v1", workItemSourceProjectID, settled.Single.ScopeDigest+".json")
	file, err := os.OpenFile(ledger, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.WriteString("\ncorrupt-after-settlement\n"); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	code, refused, _ := j.invoke(t, j.axiom, append([]string{"workflow", "stage", "status"}, stageSelector(view)...)...)
	if code == 0 || refused.Category != "recovery_required" {
		t.Fatalf("corrupt settled ledger did not fail closed: code=%d event=%+v", code, refused)
	}
	if calls := j.invocations(t); len(calls) != 1 || calls[0].Attempt != settled.Single.Attempts[0].AttemptID {
		t.Fatalf("corrupt-ledger refusal started or changed a Runtime: %+v", calls)
	}
}

func TestInstalledSyntheticSingleLineageCapacityAndAdmission(t *testing.T) {
	j := newStageJourney(t)
	j.run(t, 0, "runtime", "codex", "install")
	j.seedEvidence(t, "codex", "profile-codex", capabilityproof.RepositoryWriteVariant, stageJourneyCaps...)
	unit := writeUnit("owner", []string{"docs"}, "docs/intake.md")
	plan := writeStagePlanDocument(t, []executiongraph.WorkUnit{unit}, []executiongraph.Effect{{Kind: "repository-write", Target: "docs/intake.md"}})

	saveTargetLink(t, j.env.state, workItemSourceProjectID, "32", "owner/repo")
	lineageView := j.startExecution(t, "github:owner/repo#32")
	lineageArgs := stageRunArgs(lineageView, "intake", plan, j.policy, j.workspace)
	j.script(t, fakeStageScript{Agents: map[string][]fakeStageStep{"owner": {
		{Op: "publish-denied", Kind: "progress", Fields: []string{"phase=spoof", "percent=1"}, Environment: []string{"AXIOM_ATTEMPT_ID=123e4567-e89b-42d3-a456-426614174999", "AXIOM_GRAPH_ID=123e4567-e89b-42d3-a456-426614174998"}},
		{Op: "publish", Kind: "result", Fields: []string{"outcome=succeeded"}},
	}}})
	lineageDigest := j.run(t, 0, append(lineageArgs, "--authorize-local")...).reference(t, "stage-preview:")
	j.run(t, 0, append(lineageArgs, "--preview-digest", lineageDigest, "--authorize-local", "--authorize-external")...)
	lineageStatus := decodeSingleCrashStatus(t, j.waitStatus(t, lineageView, latestSettled))
	if len(lineageStatus.Single.Attempts) != 1 || lineageStatus.Single.Attempts[0].Status != "succeeded" || lineageStatus.Single.Attempts[0].ClassificationReason != "result_recorded" {
		t.Fatalf("valid result after rejected lineage spoof: %+v", lineageStatus)
	}
	lineageRecords := j.run(t, 0, append([]string{"workflow", "coordination", "list"}, stageSelector(lineageView)...)...)
	var lineageStream struct {
		Records []struct {
			Kind      string `json:"kind"`
			AttemptID string `json:"attemptId"`
		} `json:"records"`
	}
	if err := json.Unmarshal(lineageRecords.Stage, &lineageStream); err != nil || len(lineageStream.Records) != 1 || lineageStream.Records[0].Kind != "result" || lineageStream.Records[0].AttemptID != lineageStatus.Single.Attempts[0].AttemptID {
		t.Fatalf("spoofed lineage entered canonical records: %s %v", lineageRecords.Stage, err)
	}

	saveTargetLink(t, j.env.state, workItemSourceProjectID, "33", "owner/repo")
	capacityView := j.startExecution(t, "github:owner/repo#33")
	capacityArgs := stageRunArgs(capacityView, "intake", plan, j.policy, j.workspace)
	steps := make([]fakeStageStep, 0, 66)
	for i := 0; i < 65; i++ {
		steps = append(steps, fakeStageStep{Op: "publish", Kind: "progress", Fields: []string{"phase=capacity", "percent=1"}})
	}
	steps = append(steps, fakeStageStep{Op: "publish", Kind: "result", Fields: []string{"outcome=failed", "summary=capacity-fixture"}})
	j.script(t, fakeStageScript{Agents: map[string][]fakeStageStep{"owner": steps}})
	capacityDigest := j.run(t, 0, append(capacityArgs, "--authorize-local")...).reference(t, "stage-preview:")
	j.run(t, 0, append(capacityArgs, "--preview-digest", capacityDigest, "--authorize-local", "--authorize-external")...)
	capacityStatus := decodeSingleCrashStatus(t, j.waitStatus(t, capacityView, latestSettled))
	if len(capacityStatus.Single.Attempts) != 1 || capacityStatus.Single.Attempts[0].Status != "failed" || capacityStatus.Single.Attempts[0].ClassificationReason != "result_not_successful" {
		t.Fatalf("capacity attempt did not fail on its accepted terminal result: %+v", capacityStatus)
	}
	capacityRecords := j.run(t, 0, append([]string{"workflow", "coordination", "list"}, stageSelector(capacityView)...)...)
	var capacityStream struct {
		Records []struct {
			Kind      string `json:"kind"`
			AttemptID string `json:"attemptId"`
		} `json:"records"`
	}
	if err := json.Unmarshal(capacityRecords.Stage, &capacityStream); err != nil {
		t.Fatalf("capacity stream: %s %v", capacityRecords.Stage, err)
	}
	progress, results := 0, 0
	for _, record := range capacityStream.Records {
		if record.AttemptID != capacityStatus.Single.Attempts[0].AttemptID {
			t.Fatalf("capacity record crossed attempt lineage: %+v", record)
		}
		switch record.Kind {
		case "progress":
			progress++
		case "result":
			results++
		}
	}
	if len(capacityStream.Records) != 64 || progress != 63 || results != 1 || capacityStream.Records[len(capacityStream.Records)-1].Kind != "result" {
		t.Fatalf("capacity bound or reserved terminal slot violated: total=%d progress=%d results=%d records=%+v", len(capacityStream.Records), progress, results, capacityStream.Records)
	}
	inbox := attemptInbox(capacityStatus.Single.Binding.Workspace, capacityStatus.Single.Attempts[0].AttemptID)
	ingestion, err := os.ReadFile(filepath.Join(inbox, "ingestion.json"))
	if err != nil || !bytes.Contains(ingestion, []byte(`"capacity_exceeded"`)) {
		t.Fatalf("capacity rejection was not recorded by installed ingestion: %s %v", ingestion, err)
	}

	for _, scenario := range []struct {
		id       string
		workItem string
		args     []string
		want     string
	}{
		{id: "stale execution revision", workItem: "github:owner/repo#34", want: "stale_execution_revision"},
		{id: "invalid timeout limit", workItem: "github:owner/repo#35", want: "invalid_stage_plan"},
	} {
		t.Run(scenario.id, func(t *testing.T) {
			externalID := strings.TrimPrefix(scenario.workItem, "github:owner/repo#")
			saveTargetLink(t, j.env.state, workItemSourceProjectID, externalID, "owner/repo")
			current := j.startExecution(t, scenario.workItem)
			currentUnit := unit
			currentPlan := plan
			if scenario.id == "invalid timeout limit" {
				currentUnit.Controls.Timeout = 25 * time.Hour
				currentPlan = writeStagePlanDocument(t, []executiongraph.WorkUnit{currentUnit}, []executiongraph.Effect{{Kind: "repository-write", Target: "docs/intake.md"}})
			}
			currentArgs := stageRunArgs(current, "intake", currentPlan, j.policy, j.workspace)
			if scenario.id == "stale execution revision" {
				for i := range currentArgs {
					if currentArgs[i] == "--expected-revision" {
						currentArgs[i+1] = strconv.FormatUint(current.Revision+1, 10)
					}
				}
			}
			refused := j.run(t, 1, append(currentArgs, "--authorize-local")...)
			if refused.Category != scenario.want {
				t.Fatalf("public dispatch admission: want %s, got %+v", scenario.want, refused)
			}
		})
	}
	if calls := j.invocations(t); len(calls) != 2 {
		t.Fatalf("spoof, capacity, or rejected admission launched an extra Runtime: %+v", calls)
	}
}
