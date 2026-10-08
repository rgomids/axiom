package local

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/projectapp"
)

// I230-T03 adapter Evidence: cross-store Project edit publication, owned
// recovery state, fail-closed readers, and recovery classification.

const editTestProjectID = "123e4567-e89b-42d3-a456-426614174000"

type editPublicationFixture struct {
	projects, state string
	portable        PortableStore
	installation    InstallationStore
	source          string
}

func newEditPublicationFixture(t *testing.T) editPublicationFixture {
	t.Helper()
	f := editPublicationFixture{projects: privateTestRoot(t), state: privateTestRoot(t)}
	var err error
	if f.portable, err = NewPortableStore(f.projects); err != nil {
		t.Fatal(err)
	}
	if f.installation, err = NewInstallationStore(f.state); err != nil {
		t.Fatal(err)
	}
	if err := f.portable.Create(context.Background(), "sample", editTestManifest("Sample")); err != nil {
		t.Fatal(err)
	}
	f.source = filepath.Join(f.portable.root, "sample")
	if got := f.installation.Install(context.Background(), f.source); got.Status != InstallationApplied {
		t.Fatalf("install = %+v", got)
	}
	return f
}

func editTestManifest(name string) []byte {
	return []byte("schemaVersion: 1\nproject:\n  id: " + editTestProjectID + "\n  slug: sample\n  name: " + name + "\n")
}

func (f editPublicationFixture) manifestPath() string {
	return filepath.Join(f.projects, "sample", manifestName)
}
func (f editPublicationFixture) recordPath() string {
	return filepath.Join(f.state, "projects", editTestProjectID, installationRecord)
}

func fileBytes(t *testing.T, path string) []byte {
	t.Helper()
	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

// request builds the exact publication of a rename to name.
func (f editPublicationFixture) request(t *testing.T, name string) projectapp.EditPublicationRequest {
	t.Helper()
	next := editTestManifest(name)
	snapshot, issues := projectapp.ReadSnapshot(manifest.Codec{}, next, nil)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	current := fileBytes(t, f.recordPath())
	record, recordIssues := DecodeRecord(current)
	if len(recordIssues) != 0 {
		t.Fatal(recordIssues)
	}
	state := record.State()
	state.PortableRevision, state.ArtifactDigests = snapshot.Revision(), snapshot.Digests()
	nextRecord, recordIssues := NewRecord(state)
	if len(recordIssues) != 0 {
		t.Fatal(recordIssues)
	}
	localNext, recordIssues := EncodeRecord(nextRecord)
	if len(recordIssues) != 0 {
		t.Fatal(recordIssues)
	}
	return projectapp.EditPublicationRequest{
		ProjectID: editTestProjectID, Slug: "sample", PortableDestination: f.source,
		LocalDestination: filepath.Join(f.installation.root, "projects", editTestProjectID),
		PortableExpected: fileBytes(t, f.manifestPath()), PortableNext: next,
		LocalExpected: current, LocalNext: localNext,
	}
}

func (f editPublicationFixture) publisher(fault func(EditStage) error) ProjectEditPublisher {
	return ProjectEditPublisher{Installation: f.installation, Portable: f.portable, Fault: fault}
}

func (f editPublicationFixture) roots() RecoveryRoots {
	return RecoveryRoots{State: f.state, Projects: f.projects}
}

func (f editPublicationFixture) editStates(t *testing.T) []string {
	t.Helper()
	names, err := filepath.Glob(filepath.Join(f.state, "projects", editTestProjectID, editRecoveryPrefix+"*"))
	if err != nil {
		t.Fatal(err)
	}
	return names
}

func (f editPublicationFixture) applyPlan(t *testing.T, want RecoveryAction) RecoveryResult {
	t.Helper()
	plan := singlePlan(t, f.roots())
	if plan.Protocol != protocolProjectEdit || plan.Action != want || plan.Directory != "projects/"+editTestProjectID {
		t.Fatalf("plan = %+v", plan)
	}
	authority, err := AuthorizeRecovery(plan, plan.Digest)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ApplyRecovery(context.Background(), f.roots(), plan, authority)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.editStates(t)) != 0 {
		t.Fatal("recovery left the edit state")
	}
	return result
}

func TestProjectEditPublisherPublishesBothGenerations(t *testing.T) {
	f := newEditPublicationFixture(t)
	request := f.request(t, "Renamed")
	if got := f.publisher(nil).PublishEdit(context.Background(), request); got.Status != projectapp.EditPublicationApplied {
		t.Fatalf("publish = %+v", got)
	}
	if !bytes.Equal(fileBytes(t, f.manifestPath()), request.PortableNext) || !bytes.Equal(fileBytes(t, f.recordPath()), request.LocalNext) || len(f.editStates(t)) != 0 {
		t.Fatal("published generations differ or recovery state remains")
	}
	if report, err := InspectRecovery(context.Background(), f.roots()); err != nil || len(report.Plans) != 0 {
		t.Fatalf("recovery after success = %+v %v", report, err)
	}
	// A replay of the same request is now stale against both stores.
	if got := f.publisher(nil).PublishEdit(context.Background(), request); got.Status != projectapp.EditPublicationConflict {
		t.Fatalf("stale replay = %+v", got)
	}
}

func TestProjectEditPublisherStaleObservationsWriteNothing(t *testing.T) {
	for name, mutate := range map[string]func(t *testing.T, f editPublicationFixture, request *projectapp.EditPublicationRequest){
		"portable drift": func(t *testing.T, f editPublicationFixture, _ *projectapp.EditPublicationRequest) {
			if err := os.WriteFile(f.manifestPath(), editTestManifest("Drift"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"local drift": func(_ *testing.T, _ editPublicationFixture, request *projectapp.EditPublicationRequest) {
			request.LocalExpected = append(append([]byte(nil), request.LocalExpected...), ' ')
		},
		"source outside projects root": func(t *testing.T, _ editPublicationFixture, request *projectapp.EditPublicationRequest) {
			request.PortableDestination = filepath.Join(t.TempDir(), "sample")
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newEditPublicationFixture(t)
			request := f.request(t, "Renamed")
			mutate(t, f, &request)
			portable, record := fileBytes(t, f.manifestPath()), fileBytes(t, f.recordPath())
			got := f.publisher(nil).PublishEdit(context.Background(), request)
			if got.Status != projectapp.EditPublicationConflict && got.Status != projectapp.EditPublicationFailed {
				t.Fatalf("publish = %+v", got)
			}
			if !bytes.Equal(portable, fileBytes(t, f.manifestPath())) || !bytes.Equal(record, fileBytes(t, f.recordPath())) || len(f.editStates(t)) != 0 {
				t.Fatal("stale publication wrote state")
			}
		})
	}
}

func TestProjectEditStateFailsReadersClosedAndRestoresPrior(t *testing.T) {
	f := newEditPublicationFixture(t)
	request := f.request(t, "Renamed")
	got := f.publisher(func(stage EditStage) error {
		if stage == EditStageRecorded {
			return ErrSimulatedInterruption
		}
		return nil
	}).PublishEdit(context.Background(), request)
	if got.Status != projectapp.EditPublicationRecoveryRequired || len(f.editStates(t)) != 1 {
		t.Fatalf("interrupted = %+v states=%v", got, f.editStates(t))
	}
	if _, category := f.installation.Inspect(context.Background(), editTestProjectID); category != "recovery_required" {
		t.Fatalf("inspect = %q", category)
	}
	if selected := f.installation.Select(context.Background(), "sample"); selected.Category != "recovery_required" {
		t.Fatalf("select = %+v", selected)
	}
	operational, err := NewOperationalStore(f.state)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := operational.InspectOperational(context.Background(), editTestProjectID); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("operational inspect = %v", err)
	}
	if err := f.installation.ReplaceRecord(context.Background(), editTestProjectID, request.LocalExpected, request.LocalNext); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("replace during recovery = %v", err)
	}
	if again := f.publisher(nil).PublishEdit(context.Background(), request); again.Status != projectapp.EditPublicationRecoveryRequired {
		t.Fatalf("publish during recovery = %+v", again)
	}
	inventory, err := InspectStateInventory(context.Background(), f.state)
	if err != nil {
		t.Fatal(err)
	}
	recovered := false
	for _, entry := range inventory.Entries {
		recovered = recovered || strings.Contains(entry.Relative, editRecoveryPrefix) && entry.Kind == InventoryRecovery
	}
	if !recovered {
		t.Fatalf("inventory does not classify edit state as recovery: %+v", inventory.Entries)
	}
	f.applyPlan(t, RestorePrior)
	if !bytes.Equal(fileBytes(t, f.manifestPath()), request.PortableExpected) || !bytes.Equal(fileBytes(t, f.recordPath()), request.LocalExpected) {
		t.Fatal("restore_prior changed a generation")
	}
	if got := f.publisher(nil).PublishEdit(context.Background(), request); got.Status != projectapp.EditPublicationApplied {
		t.Fatalf("publish after recovery = %+v", got)
	}
}

func TestProjectEditPortableCommittedLocalFailedFinalizes(t *testing.T) {
	f := newEditPublicationFixture(t)
	request := f.request(t, "Renamed")
	got := f.publisher(func(stage EditStage) error {
		if stage == EditStagePortableCommitted {
			return errors.New("injected")
		}
		return nil
	}).PublishEdit(context.Background(), request)
	if got.Status != projectapp.EditPublicationPartial || got.LocalCommitted {
		t.Fatalf("partial = %+v", got)
	}
	if !bytes.Equal(fileBytes(t, f.manifestPath()), request.PortableNext) || !bytes.Equal(fileBytes(t, f.recordPath()), request.LocalExpected) {
		t.Fatal("state is not portable-new / local-prior")
	}
	result := f.applyPlan(t, FinalizeCommitted)
	if len(result.Published) != 1 || result.Published[0] != installationRecord {
		t.Fatalf("finalize result = %+v", result)
	}
	if !bytes.Equal(fileBytes(t, f.manifestPath()), request.PortableNext) || !bytes.Equal(fileBytes(t, f.recordPath()), request.LocalNext) {
		t.Fatal("finalize did not publish the recorded local generation or rolled back portable")
	}
	if observation, category := f.installation.Inspect(context.Background(), editTestProjectID); category != "" || !observation.Exists {
		t.Fatalf("inspect after finalize = %q", category)
	}
}

func TestProjectEditRetainedStateAfterBothCommitsFinalizes(t *testing.T) {
	f := newEditPublicationFixture(t)
	request := f.request(t, "Renamed")
	installation := f.installation
	installation.removeAttempt = func(root *os.Root, name string) error {
		if strings.HasPrefix(name, editRecoveryPrefix) {
			return errors.New("injected removal failure")
		}
		return root.Remove(name)
	}
	got := ProjectEditPublisher{Installation: installation, Portable: f.portable}.PublishEdit(context.Background(), request)
	if got.Status != projectapp.EditPublicationPartial || !got.LocalCommitted {
		t.Fatalf("retained = %+v", got)
	}
	result := f.applyPlan(t, FinalizeCommitted)
	if len(result.Published) != 0 {
		t.Fatalf("finalize republished a committed generation: %+v", result)
	}
	if !bytes.Equal(fileBytes(t, f.recordPath()), request.LocalNext) {
		t.Fatal("local generation changed")
	}
}

func TestProjectEditContradictoryOrMalformedStateIsPreserved(t *testing.T) {
	for name, mutate := range map[string]func(t *testing.T, f editPublicationFixture, request projectapp.EditPublicationRequest){
		"third local generation": func(t *testing.T, f editPublicationFixture, request projectapp.EditPublicationRequest) {
			if err := os.WriteFile(f.recordPath(), append(append([]byte(nil), request.LocalExpected...), '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"third portable generation": func(t *testing.T, f editPublicationFixture, _ projectapp.EditPublicationRequest) {
			if err := os.WriteFile(f.manifestPath(), editTestManifest("Third"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"missing portable": func(t *testing.T, f editPublicationFixture, _ projectapp.EditPublicationRequest) {
			if err := os.Remove(f.manifestPath()); err != nil {
				t.Fatal(err)
			}
		},
		"noncanonical state": func(t *testing.T, f editPublicationFixture, _ projectapp.EditPublicationRequest) {
			path := f.editStates(t)[0]
			wire := fileBytes(t, path)
			if err := os.WriteFile(path, append([]byte(" "), wire...), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"pending portable protocol": func(t *testing.T, f editPublicationFixture, _ projectapp.EditPublicationRequest) {
			if err := os.WriteFile(filepath.Join(f.projects, "sample", ".axiom-recovery-0001"), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newEditPublicationFixture(t)
			request := f.request(t, "Renamed")
			got := f.publisher(func(stage EditStage) error {
				if stage == EditStagePortableCommitted {
					return errors.New("injected")
				}
				return nil
			}).PublishEdit(context.Background(), request)
			if got.Status != projectapp.EditPublicationPartial {
				t.Fatalf("partial = %+v", got)
			}
			mutate(t, f, request)
			report, err := InspectRecovery(context.Background(), f.roots())
			if err != nil {
				t.Fatal(err)
			}
			var plan *RecoveryPlan
			for index := range report.Plans {
				if report.Plans[index].Scope == RecoveryScopeState {
					plan = &report.Plans[index]
				}
			}
			if plan == nil || plan.Action != PreservedReview {
				t.Fatalf("plans = %+v", report.Plans)
			}
			if _, err := AuthorizeRecovery(*plan, plan.Digest); err == nil {
				t.Fatal("preserved plan was authorized")
			}
			if len(f.editStates(t)) != 1 {
				t.Fatal("preserved edit state was removed")
			}
		})
	}
}

func TestReplaceRecordIsExactCompareAndSwap(t *testing.T) {
	f := newEditPublicationFixture(t)
	request := f.request(t, "Renamed")
	if err := f.installation.ReplaceRecord(context.Background(), editTestProjectID, request.LocalNext, request.LocalNext); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale expected = %v", err)
	}
	if err := f.installation.ReplaceRecord(context.Background(), editTestProjectID, request.LocalExpected, request.LocalNext); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fileBytes(t, f.recordPath()), request.LocalNext) {
		t.Fatal("record not replaced")
	}
	if err := f.installation.ReplaceRecord(context.Background(), editTestProjectID, request.LocalNext, []byte("{}")); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("invalid next record = %v", err)
	}
}

func TestEditRecoveryStateCodecIsStrict(t *testing.T) {
	f := newEditPublicationFixture(t)
	request := f.request(t, "Renamed")
	state := editRecoveryState{
		FormatVersion: editRecoveryFormatVersion, Protocol: editRecoveryProtocol, OperationID: strings.Repeat("a", 32),
		ProjectID: editTestProjectID, Slug: "sample", PortableSource: f.source, PortableObject: manifestName, LocalObject: installationRecord,
		PriorPortableRevision: hexDigest(request.PortableExpected), NewPortableRevision: hexDigest(request.PortableNext),
		PriorLocalRevision: hexDigest(request.LocalExpected), NewLocalRevision: hexDigest(request.LocalNext), NextLocalRecord: request.LocalNext,
	}
	wire, err := encodeEditRecoveryState(state)
	if err != nil {
		t.Fatal(err)
	}
	if decoded, err := decodeEditRecoveryState(wire); err != nil || decoded.OperationID != state.OperationID {
		t.Fatalf("round trip = %v", err)
	}
	for name, candidate := range map[string][]byte{
		"unknown field":  bytes.Replace(wire, []byte(`{"formatVersion"`), []byte(`{"extra":1,"formatVersion"`), 1),
		"other version":  bytes.Replace(wire, []byte(`"formatVersion":1`), []byte(`"formatVersion":2`), 1),
		"noncanonical":   append([]byte(" "), wire...),
		"trailing value": append(append([]byte(nil), wire...), []byte("{}\n")...),
		"empty":          nil,
	} {
		if _, err := decodeEditRecoveryState(candidate); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	tampered := state
	tampered.NewLocalRevision = tampered.PriorLocalRevision
	if _, err := encodeEditRecoveryState(tampered); err == nil {
		t.Fatal("state whose next local wire does not match its revision was encoded")
	}
}
