package projectapp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

const operationalTestID = "123e4567-e89b-42d3-a456-426614174000"

type operationalSpy struct {
	observation OperationalObservation
	inspectErr  error
	commitErr   error
	commits     []OperationalState
	expected    []string
}

func (s *operationalSpy) InspectOperational(context.Context, string) (OperationalObservation, error) {
	return s.observation, s.inspectErr
}

func (s *operationalSpy) CommitOperational(_ context.Context, _ string, expected string, next OperationalState) error {
	s.commits = append(s.commits, next)
	s.expected = append(s.expected, expected)
	return s.commitErr
}

func absentStore() *operationalSpy {
	return &operationalSpy{observation: OperationalObservation{Revision: OperationalRevisionAbsent, State: DefaultOperationalState()}}
}

type committedError struct{}

func (committedError) Error() string         { return "committed" }
func (committedError) EffectCommitted() bool { return true }

func TestOperationalStateValidation(t *testing.T) {
	for name, test := range map[string]struct {
		state OperationalState
		valid bool
	}{
		"default":            {DefaultOperationalState(), true},
		"archived with keys": {OperationalState{ProjectStatus: ProjectArchived, DisabledIntegrations: []string{"docs", "work-items"}}, true},
		"zero value":         {OperationalState{}, false},
		"nil keys":           {OperationalState{ProjectStatus: ProjectActive}, false},
		"unknown status":     {OperationalState{ProjectStatus: "deleted", DisabledIntegrations: []string{}}, false},
		"unsorted":           {OperationalState{ProjectStatus: ProjectActive, DisabledIntegrations: []string{"work-items", "docs"}}, false},
		"duplicate":          {OperationalState{ProjectStatus: ProjectActive, DisabledIntegrations: []string{"docs", "docs"}}, false},
		"unsafe key":         {OperationalState{ProjectStatus: ProjectActive, DisabledIntegrations: []string{"Docs"}}, false},
	} {
		if got := ValidOperationalState(test.state); got != test.valid {
			t.Errorf("%s: valid=%t", name, got)
		}
	}
	keys := make([]string, MaxDisabledIntegrations+1)
	for index := range keys {
		keys[index] = fmt.Sprintf("k%03d", index)
	}
	if ValidOperationalState(OperationalState{ProjectStatus: ProjectActive, DisabledIntegrations: keys}) {
		t.Fatal("unbounded disable set accepted")
	}
}

func TestProposeOperationalTransitionsAndNoOps(t *testing.T) {
	active, archived := DefaultOperationalState(), OperationalState{ProjectStatus: ProjectArchived, DisabledIntegrations: []string{"work-items"}}
	for _, test := range []struct {
		name      string
		current   OperationalState
		request   OperationalRequest
		want      OperationalState
		unchanged string
		failure   string
	}{
		{"archive", active, OperationalRequest{Operation: ArchiveProject}, OperationalState{ProjectStatus: ProjectArchived, DisabledIntegrations: []string{}}, "", ""},
		{"archive preserves disabled set", OperationalState{ProjectStatus: ProjectActive, DisabledIntegrations: []string{"work-items"}}, OperationalRequest{Operation: ArchiveProject}, archived, "", ""},
		{"archive archived", archived, OperationalRequest{Operation: ArchiveProject}, archived, ProjectAlreadyArchived, ""},
		{"reactivate preserves disabled set", archived, OperationalRequest{Operation: ReactivateProject}, OperationalState{ProjectStatus: ProjectActive, DisabledIntegrations: []string{"work-items"}}, "", ""},
		{"reactivate active", active, OperationalRequest{Operation: ReactivateProject}, active, ProjectAlreadyActive, ""},
		{"disable sorts", OperationalState{ProjectStatus: ProjectActive, DisabledIntegrations: []string{"work-items"}}, OperationalRequest{Operation: DisableIntegration, Integration: "docs"}, OperationalState{ProjectStatus: ProjectActive, DisabledIntegrations: []string{"docs", "work-items"}}, "", ""},
		{"disable disabled", archived, OperationalRequest{Operation: DisableIntegration, Integration: "work-items"}, archived, IntegrationAlreadyDisabled, ""},
		{"enable", archived, OperationalRequest{Operation: EnableIntegration, Integration: "work-items"}, OperationalState{ProjectStatus: ProjectArchived, DisabledIntegrations: []string{}}, "", ""},
		{"enable enabled", active, OperationalRequest{Operation: EnableIntegration, Integration: "work-items"}, active, IntegrationAlreadyEnabled, ""},
		{"archive with key", active, OperationalRequest{Operation: ArchiveProject, Integration: "work-items"}, OperationalState{}, "", OperationalInvalidInput},
		{"disable unsafe key", active, OperationalRequest{Operation: DisableIntegration, Integration: "../x"}, OperationalState{}, "", OperationalInvalidInput},
		{"enable missing key", active, OperationalRequest{Operation: EnableIntegration}, OperationalState{}, "", OperationalInvalidInput},
		{"unknown operation", active, OperationalRequest{Operation: "delete"}, OperationalState{}, "", OperationalInvalidInput},
		{"invalid current", OperationalState{}, OperationalRequest{Operation: ArchiveProject}, OperationalState{}, "", OperationalStateInvalid},
	} {
		t.Run(test.name, func(t *testing.T) {
			next, unchanged, failure := ProposeOperational(test.current, test.request)
			if failure != test.failure || unchanged != test.unchanged {
				t.Fatalf("unchanged=%q failure=%q", unchanged, failure)
			}
			if failure == "" && !next.Equal(test.want) {
				t.Fatalf("next = %+v", next)
			}
		})
	}
}

func TestProposeOperationalNeverAliasesCurrentState(t *testing.T) {
	current := OperationalState{ProjectStatus: ProjectActive, DisabledIntegrations: []string{"b"}}
	next, _, _ := ProposeOperational(current, OperationalRequest{Operation: DisableIntegration, Integration: "a"})
	next.DisabledIntegrations[0] = "z"
	if current.DisabledIntegrations[0] != "b" || len(current.DisabledIntegrations) != 1 {
		t.Fatalf("current mutated: %+v", current)
	}
}

func TestApplyOperationalRequiresExactReviewedLocalAuthority(t *testing.T) {
	request := OperationalRequest{ProjectID: operationalTestID, Operation: ArchiveProject}
	store := absentStore()
	preview := ApplyOperational(context.Background(), store, request, "", false)
	if preview.Status != OperationalPreviewed || preview.Category != OperationalPreviewReady || preview.Preview == nil || len(store.commits) != 0 {
		t.Fatalf("preview = %+v commits=%d", preview, len(store.commits))
	}
	reviewed := *preview.Preview
	if reviewed.Revision != OperationalRevisionAbsent || len(reviewed.Effects) != 1 || reviewed.Effects[0] != (EditEffect{Scope: LocalScope, Code: "archive_project"}) {
		t.Fatalf("reviewed preview = %+v", reviewed)
	}
	if reviewed.Boundary != (OperationalBoundary{Portable: "unchanged", Provider: "none", Credentials: "unchanged"}) || reviewed.Current.ProjectStatus != ProjectActive || reviewed.Result.ProjectStatus != ProjectArchived {
		t.Fatalf("reviewed preview disclosure = %+v", reviewed)
	}
	for name, test := range map[string]struct {
		digest    string
		authorize bool
	}{
		"digest without authority": {reviewed.Digest, false},
		"authority without digest": {"", true},
		"authority with stale":     {strings.Repeat("0", 64), true},
	} {
		denied := ApplyOperational(context.Background(), store, request, test.digest, test.authorize)
		if denied.Status != OperationalDenied || denied.Category != OperationalAuthorityDenied || len(store.commits) != 0 {
			t.Fatalf("%s: %+v commits=%d", name, denied, len(store.commits))
		}
	}
	applied := ApplyOperational(context.Background(), store, request, reviewed.Digest, true)
	if applied.Status != OperationalCommitted || applied.Category != OperationalApplied || !applied.Committed {
		t.Fatalf("applied = %+v", applied)
	}
	if len(store.commits) != 1 || store.expected[0] != OperationalRevisionAbsent || !store.commits[0].Archived() {
		t.Fatalf("commits = %+v expected=%v", store.commits, store.expected)
	}
}

func TestApplyOperationalDigestBindsObservedRevisionAndRequest(t *testing.T) {
	store := absentStore()
	archive := OperationalRequest{ProjectID: operationalTestID, Operation: ArchiveProject}
	first, _, _, _ := PreviewOperational(context.Background(), store, archive)
	again, _, _, _ := PreviewOperational(context.Background(), store, archive)
	if first.Digest != again.Digest || len(first.Digest) != 64 {
		t.Fatalf("digest not deterministic: %s %s", first.Digest, again.Digest)
	}
	disable, _, _, _ := PreviewOperational(context.Background(), store, OperationalRequest{ProjectID: operationalTestID, Operation: DisableIntegration, Integration: "work-items"})
	if disable.Digest == first.Digest {
		t.Fatal("different requests share a digest")
	}
	// A concurrent writer changes the observed revision: the reviewed digest
	// no longer authorizes anything.
	store.observation = OperationalObservation{Exists: true, Revision: strings.Repeat("a", 64), State: OperationalState{ProjectStatus: ProjectActive, DisabledIntegrations: []string{"docs"}}}
	result := ApplyOperational(context.Background(), store, archive, first.Digest, true)
	if result.Status != OperationalDenied || len(store.commits) != 0 {
		t.Fatalf("stale reviewed digest = %+v commits=%d", result, len(store.commits))
	}
}

func TestApplyOperationalNoOpNeedsNoAuthorityAndWritesNothing(t *testing.T) {
	store := &operationalSpy{observation: OperationalObservation{Exists: true, Revision: strings.Repeat("b", 64), State: OperationalState{ProjectStatus: ProjectArchived, DisabledIntegrations: []string{"work-items"}}}}
	for request, want := range map[OperationalRequest]string{
		{ProjectID: operationalTestID, Operation: ArchiveProject}:                                ProjectAlreadyArchived,
		{ProjectID: operationalTestID, Operation: DisableIntegration, Integration: "work-items"}: IntegrationAlreadyDisabled,
		{ProjectID: operationalTestID, Operation: EnableIntegration, Integration: "docs"}:        IntegrationAlreadyEnabled,
	} {
		result := ApplyOperational(context.Background(), store, request, "", false)
		if result.Status != OperationalUnchanged || result.Category != want || result.Preview == nil || len(result.Preview.Effects) != 0 {
			t.Fatalf("%+v: %+v", request, result)
		}
	}
	if len(store.commits) != 0 {
		t.Fatalf("no-op wrote %d times", len(store.commits))
	}
}

func TestApplyOperationalMapsStoreFailuresTruthfully(t *testing.T) {
	request := OperationalRequest{ProjectID: operationalTestID, Operation: ReactivateProject}
	archivedObservation := OperationalObservation{Exists: true, Revision: strings.Repeat("c", 64), State: OperationalState{ProjectStatus: ProjectArchived, DisabledIntegrations: []string{}}}
	for name, test := range map[string]struct {
		inspectErr, commitErr error
		status                OperationalStatus
		category              string
		committed             bool
	}{
		"not installed":     {inspectErr: ErrNotFound, status: OperationalFailed, category: OperationalNotInstalled},
		"malformed":         {inspectErr: ErrUnsafe, status: OperationalFailed, category: OperationalStateInvalid},
		"interrupted":       {inspectErr: ErrRecoveryRequired, status: OperationalFailed, category: OperationalRecoveryRequired},
		"concurrent writer": {commitErr: ErrConflict, status: OperationalFailed, category: OperationalConflict},
		"storage":           {commitErr: errors.New("disk"), status: OperationalFailed, category: OperationalStorageFailure},
		"committed cleanup": {commitErr: committedError{}, status: OperationalCommitted, category: OperationalAppliedRecovery, committed: true},
	} {
		t.Run(name, func(t *testing.T) {
			store := &operationalSpy{observation: archivedObservation, inspectErr: test.inspectErr, commitErr: test.commitErr}
			preview, _, _, _ := PreviewOperational(context.Background(), &operationalSpy{observation: archivedObservation}, request)
			result := ApplyOperational(context.Background(), store, request, preview.Digest, true)
			if result.Status != test.status || result.Category != test.category || result.Committed != test.committed {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestInspectOperationalRejectsIncoherentObservations(t *testing.T) {
	for name, observation := range map[string]OperationalObservation{
		"exists with absent revision":  {Exists: true, Revision: OperationalRevisionAbsent, State: DefaultOperationalState()},
		"missing with digest revision": {Revision: strings.Repeat("d", 64), State: DefaultOperationalState()},
		"empty revision":               {State: DefaultOperationalState()},
		"invalid state":                {Revision: OperationalRevisionAbsent},
	} {
		if _, category := InspectOperational(context.Background(), &operationalSpy{observation: observation}, operationalTestID); category != OperationalStateInvalid {
			t.Errorf("%s: category=%q", name, category)
		}
	}
	if _, category := InspectOperational(context.Background(), absentStore(), "sample"); category != OperationalInvalidInput {
		t.Fatalf("slug accepted as Project ID: %q", category)
	}
	if result := ApplyOperational(context.Background(), absentStore(), OperationalRequest{ProjectID: "sample", Operation: ArchiveProject}, "", false); result.Category != OperationalInvalidInput {
		t.Fatalf("invalid project id = %+v", result)
	}
}
