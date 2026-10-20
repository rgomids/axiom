package stageexecution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
)

type noSpawnRunner struct{ calls int }

func (r *noSpawnRunner) RunObserved(context.Context, executiongraph.Invocation, func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
	r.calls++
	return executiongraph.ProcessResult{SpawnFailed: true, FailureCode: "permission_denied", ExitCode: -1}
}

func TestSupervisorPersistsOperationalFailureBeforeSpawn(t *testing.T) {
	s, r, store, prep, runner := supervisorFixture(t)
	prep.failPrepare, prep.prepareError = 1, executiongraph.ErrAuthenticationBlocked
	if err := s.Run(context.Background(), r); !errors.Is(err, executiongraph.ErrAuthenticationBlocked) {
		t.Fatal(err)
	}
	if runner.calls.Load() != 0 {
		t.Fatal("refusal spawned an attempt")
	}
	var found bool
	for _, child := range store.g.Children {
		for _, a := range child.Attempts {
			found = true
			if a.Process != nil || a.Diagnostic == nil || a.Diagnostic.Code != "authentication_unavailable" || a.Diagnostic.EffectCertainty != "none" || a.Status != executiongraph.AttemptFailed {
				t.Fatalf("%+v", a)
			}
		}
	}
	if !found {
		t.Fatal("pre-spawn failure was not persisted")
	}
}

func TestSingleSupervisorPersistsOperationalFailureBeforeSpawn(t *testing.T) {
	store, r := singleFixture(t)
	runner := &fakeSingleRunner{}
	s := newSingleSupervisor(store, &fakeSinglePreparation{fail: executiongraph.ErrAuthenticationBlocked}, runner)
	if err := s.Run(context.Background(), r); !errors.Is(err, executiongraph.ErrAuthenticationBlocked) {
		t.Fatal(err)
	}
	if runner.calls != 0 || len(store.run.Attempts) != 1 {
		t.Fatal("wrong pre-spawn publication")
	}
	a := store.run.Attempts[0]
	if a.Process != nil || a.Diagnostic == nil || a.Diagnostic.Code != "authentication_unavailable" || a.Diagnostic.EffectCertainty != "none" {
		t.Fatalf("%+v", a)
	}
}

func TestSupervisorsPersistConfirmedSpawnFailure(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		store, r := singleFixture(t)
		runner := &noSpawnRunner{}
		if err := newSingleSupervisor(store, &fakeSinglePreparation{}, runner).Run(context.Background(), r); err != nil {
			t.Fatal(err)
		}
		if len(store.run.Attempts) != 1 || store.run.Attempts[0].Diagnostic == nil || store.run.Attempts[0].Diagnostic.Code != "permission_denied" || store.run.Attempts[0].Process != nil {
			t.Fatalf("%+v", store.run.Attempts)
		}
	})
	t.Run("graph", func(t *testing.T) {
		s, r, store, _, _ := supervisorFixture(t)
		s.Runner = &noSpawnRunner{}
		if err := s.Run(context.Background(), r); err != nil {
			t.Fatal(err)
		}
		var found bool
		for _, c := range store.g.Children {
			for _, a := range c.Attempts {
				found = true
				if a.Process != nil || a.Diagnostic == nil || a.Diagnostic.Code != "permission_denied" {
					t.Fatalf("%+v", a)
				}
			}
		}
		if !found {
			t.Fatal("spawn failure lost")
		}
	})
}

type diagnosticRunner struct{ result executiongraph.ProcessResult }

func (r diagnosticRunner) RunObserved(_ context.Context, _ executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
	if err := started(&executiongraph.ProcessCorrelation{PID: 42, StartIdentity: "synthetic-start", GroupIdentity: "synthetic-group"}); err != nil {
		return executiongraph.ProcessResult{OutcomeUnknown: true}
	}
	return r.result
}

func TestSupervisorsPersistSanitizedTerminalCauseWithoutChangingUncertainty(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		result := executiongraph.ProcessResult{ExitCode: 7, Output: []byte("error: authentication failed: secret-token /private/account\n"), OutcomeUnknown: unknown, EffectAmbiguous: unknown}
		check := func(t *testing.T, a executiongraph.SuccessorAttempt) {
			t.Helper()
			if a.Diagnostic == nil || a.Diagnostic.Cause != "authentication_unavailable" || a.Diagnostic.CauseConfidence != "hypothesis" {
				t.Fatalf("%+v", a)
			}
			if unknown && (a.Status != executiongraph.AttemptUnknown || a.Diagnostic.EffectCertainty != "unknown") {
				t.Fatal("uncertainty lost")
			}
			wire, _ := json.Marshal(a)
			for _, secret := range []string{"secret-token", "/private/account"} {
				if bytes.Contains(wire, []byte(secret)) {
					t.Fatal("raw output persisted")
				}
			}
		}
		t.Run(fmt.Sprintf("single/%v", unknown), func(t *testing.T) {
			store, r := singleFixture(t)
			if err := newSingleSupervisor(store, &fakeSinglePreparation{}, diagnosticRunner{result}).Run(context.Background(), r); err != nil {
				t.Fatal(err)
			}
			check(t, store.run.Attempts[0])
		})
		t.Run(fmt.Sprintf("graph/%v", unknown), func(t *testing.T) {
			s, r, store, _, _ := supervisorFixture(t)
			s.Runner = diagnosticRunner{result}
			if err := s.Run(context.Background(), r); err != nil {
				t.Fatal(err)
			}
			for _, c := range store.g.Children {
				for _, a := range c.Attempts {
					check(t, a)
				}
			}
		})
	}
}

type unobservedStartRunner struct{ calls int }

func (runner *unobservedStartRunner) RunObserved(context.Context, executiongraph.Invocation, func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
	runner.calls++
	return executiongraph.ProcessResult{ExitCode: -1, OutcomeUnknown: true}
}

func TestUnobservedStartCannotLeaveReusablePendingAuthority(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		store, r := singleFixture(t)
		s := newSingleSupervisor(store, &fakeSinglePreparation{}, &unobservedStartRunner{})
		if err := s.Run(context.Background(), r); err != nil {
			t.Fatal(err)
		}
		if len(store.run.Attempts) != 1 || store.run.Attempts[0].Status != executiongraph.AttemptUnknown || store.run.Attempts[0].Diagnostic.EffectCertainty != "unknown" {
			t.Fatalf("%+v", store.run.Attempts)
		}
		if _, err := verifySingle(r, store.run, store.authorities[r.AuthorityRef]); err == nil {
			t.Fatal("uncertain start left reusable first-attempt authority")
		}
	})
	t.Run("graph", func(t *testing.T) {
		s, r, store, _, _ := supervisorFixture(t)
		s.Runner = &unobservedStartRunner{}
		if err := s.Run(context.Background(), r); err != nil {
			t.Fatal(err)
		}
		for _, c := range store.g.Children {
			for _, a := range c.Attempts {
				if a.Status != executiongraph.AttemptUnknown || a.Diagnostic.EffectCertainty != "unknown" {
					t.Fatalf("%+v", a)
				}
			}
		}
		runner := s.Runner.(*unobservedStartRunner)
		calls := runner.calls
		_ = s.Run(context.Background(), r)
		if runner.calls != calls {
			t.Fatal("uncertain graph replayed first-attempt authority")
		}
	})
}
