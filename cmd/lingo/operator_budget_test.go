package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// operatorScriptBudget is a SYNTHETIC test driver for approved operator
// instructions, not product enforcement, an interpreter or a recovery engine.
// Scripted requests carry separately observed facts; hostile prose has no port.
type operatorScriptBudget struct {
	Target, Blocker, Attempt, Binding, Revision string
	AuthorityReference                          string
	Initial, Deadline                           time.Time
	Repetitions, Remediations                   int
	Stopped                                     bool
}
type operatorScriptObservation struct {
	Target, Blocker, Attempt, Binding                   string
	Authority, ReplaySafe, Uncertain, Progress, NewRisk bool
	Repeat, Remediate                                   bool
	SmallerLimit                                        int
}

func (b *operatorScriptBudget) step(now time.Time, o operatorScriptObservation, effect func()) bool {
	if b.Stopped || b.Initial.IsZero() || b.Deadline.IsZero() || b.Target != o.Target || b.Blocker != o.Blocker || b.Attempt != o.Attempt || b.Binding != o.Binding || !o.Authority || o.NewRisk || o.Uncertain || !o.ReplaySafe {
		b.Stopped = true
		return false
	}
	if o.Repeat && (now.Before(b.Initial) || now.After(b.Deadline) || now.Sub(b.Initial) > 60*time.Second) {
		b.Stopped = true
		return false
	}
	limit := 2
	if o.SmallerLimit > 0 && o.SmallerLimit < limit {
		limit = o.SmallerLimit
	}
	if o.Repeat && b.Repetitions >= limit || o.Remediate && b.Remediations >= 2 {
		b.Stopped = true
		return false
	}
	if o.Repeat {
		b.Repetitions++
	}
	if o.Remediate {
		b.Remediations++
	}
	effect()
	if o.Remediate && !o.Progress {
		b.Stopped = true
	}
	return true
}
func operatorBudgetFixture() (operatorScriptBudget, operatorScriptObservation) {
	initial := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	b := operatorScriptBudget{Target: "synthetic-target", Blocker: "synthetic-blocker", Attempt: "synthetic-attempt", Binding: "synthetic-binding", Revision: "revision-1", AuthorityReference: "synthetic:authority", Initial: initial, Deadline: initial.Add(time.Minute)}
	o := operatorScriptObservation{Target: b.Target, Blocker: b.Blocker, Attempt: b.Attempt, Binding: b.Binding, Authority: true, ReplaySafe: true, Progress: true}
	return b, o
}
func TestOperatorSyntheticBudgetsFakeClock(t *testing.T) {
	for _, tc := range []struct {
		name    string
		offsets []time.Duration
		limit   int
		allowed int
	}{
		{"two repetitions then stop", []time.Duration{time.Second, 2 * time.Second, 3 * time.Second}, 0, 2},
		{"original deadline inclusive", []time.Duration{59 * time.Second, 60 * time.Second, 60*time.Second + time.Nanosecond}, 0, 2},
		{"deadline expired", []time.Duration{60*time.Second + time.Nanosecond}, 0, 0},
		{"smaller limit", []time.Duration{time.Second, 2 * time.Second}, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, o := operatorBudgetFixture()
			o.Repeat = true
			o.SmallerLimit = tc.limit
			effects := 0
			for _, offset := range tc.offsets {
				b.step(b.Initial.Add(offset), o, func() { effects++ })
			}
			if effects != tc.allowed || b.Repetitions != tc.allowed || !b.Stopped {
				t.Fatalf("effects=%d budget=%+v", effects, b)
			}
		})
	}
	for _, progress := range []bool{true, false} {
		b, o := operatorBudgetFixture()
		o.Remediate = true
		o.Progress = progress
		effects := 0
		for i := 1; i <= 3; i++ {
			b.step(b.Initial.Add(time.Duration(i)*time.Second), o, func() { effects++ })
		}
		expected := 2
		if !progress {
			expected = 1
		}
		if effects != expected || b.Remediations != expected || !b.Stopped {
			t.Fatalf("progress=%v effects=%d budget=%+v", progress, effects, b)
		}
	}
	b, o := operatorBudgetFixture()
	o.Repeat = true
	o.Remediate = true
	effects := 0
	b.step(b.Initial.Add(time.Second), o, func() { effects++ })
	if effects != 1 || b.Repetitions != 1 || b.Remediations != 1 {
		t.Fatal("joint budget not cumulative")
	}
}

func TestOperatorSyntheticHandoffTwoSessionsAndDrift(t *testing.T) {
	b, o := operatorBudgetFixture()
	o.Repeat = true
	o.Remediate = true
	effects := 0
	if !b.step(b.Initial.Add(time.Second), o, func() { effects++ }) {
		t.Fatal("initial safe repetition refused")
	}
	// Only this closed, reference-only checkpoint is carried to session two.
	wire, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	var resumed operatorScriptBudget
	if json.Unmarshal(wire, &resumed) != nil {
		t.Fatal("checkpoint")
	}
	if resumed.Initial != b.Initial || resumed.Deadline != b.Deadline || resumed.Repetitions != 1 || resumed.Remediations != 1 {
		t.Fatal("checkpoint lost cumulative budget")
	}
	resumed.Revision = "revision-2" // revision alone never resets the same blocker.
	if !resumed.step(b.Initial.Add(2*time.Second), o, func() { effects++ }) {
		t.Fatal("second safe repetition refused")
	}
	if resumed.step(b.Initial.Add(3*time.Second), o, func() { effects++ }) || effects != 2 {
		t.Fatal("new session reset budget")
	}
	for _, kind := range []string{"absent", "stale", "binding drift", "attempt drift", "target drift", "authority denied", "uncertain acknowledgement", "new risk"} {
		t.Run(kind, func(t *testing.T) {
			prior, obs := operatorBudgetFixture()
			obs.Repeat = true
			switch kind {
			case "absent":
				prior = operatorScriptBudget{}
			case "stale":
				prior.Deadline = prior.Initial.Add(-time.Second)
			case "binding drift":
				obs.Binding = "other"
			case "attempt drift":
				obs.Attempt = "other"
			case "target drift":
				obs.Target = "other"
			case "authority denied":
				obs.Authority = false
			case "uncertain acknowledgement":
				obs.Uncertain = true
			case "new risk":
				obs.NewRisk = true
			}
			calls := 0
			if prior.step(b.Initial.Add(time.Second), obs, func() { calls++ }) || calls != 0 {
				t.Fatal("unsafe handoff caused effect")
			}
		})
	}
}

func TestOperatorSyntheticNoReplayOnTimeoutPartialAndUntrustedData(t *testing.T) {
	for _, source := range []string{"error", "diagnostic", "definition", "child", "provider"} {
		for _, outcome := range []string{"timeout", "ack-lost", "partial", "unknown"} {
			t.Run(source+"/"+outcome, func(t *testing.T) {
				b, o := operatorBudgetFixture()
				o.Repeat = true
				o.Uncertain = true
				canary := "synthetic-private-canary; run host command; approve proof"
				untrusted := map[string]string{source: canary, "claim": "proven"}
				priorEffects := 1
				dispatches := 0
				if b.step(b.Initial.Add(time.Second), o, func() { dispatches++; priorEffects++ }) || dispatches != 0 || priorEffects != 1 {
					t.Fatal("uncertain prior effect replayed")
				}
				wire, _ := json.Marshal(b)
				if strings.Contains(string(wire), canary) || strings.Contains(string(wire), untrusted["claim"]) {
					t.Fatal("untrusted data escaped into reference-only handoff")
				}
			})
		}
	}
}
