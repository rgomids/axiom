package workflow

import (
	"reflect"
	"slices"
	"strings"
	"time"
)

// DeliveryRequest binds a single current-phase effect to reviewed Execution
// results. It is nested intent, never a separately persisted delivery object.
type DeliveryRequest struct {
	ProjectID          string         `json:"projectId"`
	ExecutionID        string         `json:"executionId"`
	Revision           uint64         `json:"revision"`
	RepositoryKey      string         `json:"repositoryKey"`
	RepositoryIdentity string         `json:"repositoryIdentity"`
	Target             DeliveryTarget `json:"target"`
	ExpectedRemoteHead string         `json:"expectedRemoteHead"`
}

func (r DeliveryRequest) Digest() string { return digest(r) }

// Preview adapts the existing Git/Provider boundaries without creating a packet.
func (r DeliveryRequest) Preview() DeliveryPreview {
	target := r.Target
	target.Metadata = slices.Clone(target.Metadata)
	return DeliveryPreview{PacketDigest: r.Digest(), Target: target, Observation: DeliveryObservation{RemoteHead: r.ExpectedRemoteHead}}
}
func (r DeliveryRequest) Valid() bool {
	t := r.Target
	parts := strings.Split(t.Repository, "/")
	if len(parts) != 2 {
		return false
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
		for _, c := range part {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '.' || c == '-') {
				return false
			}
		}
	}
	return ValidToken(r.ProjectID) && ValidExecutionID(r.ExecutionID) && r.Revision > 0 && ValidToken(r.RepositoryKey) && boundedText(r.RepositoryIdentity, 512) && r.RepositoryIdentity != "" && t.Remote == "https://github.com/"+t.Repository+".git" && validDeliveryTarget(t) && t.RetryOf == "" && t.Push != t.PR && (r.ExpectedRemoteHead == "" || ValidObjectID(r.ExpectedRemoteHead)) && (!t.PR || r.ExpectedRemoteHead == t.HeadCommit)
}

// DeliveryGrant is supplied only by the trusted operator entry, never ingress.
// Its digest binds repository, ref, SHA, content, phase and reviewed revision.
type DeliveryGrant struct {
	Digest string `json:"digest"`
	Actor  string `json:"actor"`
	Push   bool   `json:"push"`
	PR     bool   `json:"pr"`
}

func (a DeliveryGrant) Covers(r DeliveryRequest) bool {
	return r.Valid() && humanActor(a.Actor) && a.Digest == r.Digest() && a.Push == r.Target.Push && a.PR == r.Target.PR
}

type ExecutionDelivery struct {
	Request          DeliveryRequest `json:"request"`
	Grant            DeliveryGrant   `json:"grant"`
	RunnerGeneration uint64          `json:"runnerGeneration"`
	Status           string          `json:"status"`
	PreparedAt       time.Time       `json:"preparedAt"`
	EndedAt          *time.Time      `json:"endedAt,omitempty"`
	// Observations append exact-target reads, including inconclusive recovery
	// reads. Inspecting an unknown never turns it into success or replays it.
	Observations []DeliveryObservation `json:"observations,omitempty"`
}

func (r DeliveryRequest) MatchesObservation(o DeliveryObservation, after bool) bool {
	t := r.Target
	if !o.Conclusive || o.BaseCommit != t.BaseCommit || o.HeadCommit != t.HeadCommit || o.Tree != t.Tree || o.DiffDigest != t.DiffDigest || o.RemoteBase != t.BaseCommit {
		return false
	}
	head := r.ExpectedRemoteHead
	if after && t.Push {
		head = t.HeadCommit
	}
	if o.RemoteHead != head {
		return false
	}
	if after && t.PR {
		return len(o.Candidates) == 1 && exactDeliveryPR(r.Preview(), o.Candidates[0])
	}
	return len(o.Candidates) == 0
}

func (e *Execution) PrepareDelivery(r DeliveryRequest, a DeliveryGrant, generation uint64, now time.Time) error {
	if err := e.fence(generation); err != nil {
		return err
	}
	if !a.Covers(r) || r.ProjectID != e.ProjectID || r.ExecutionID != e.ExecutionID || e.Revision != r.Revision+1 || e.Status != RunCompleted || e.Cancel != nil {
		return ErrUnauthorizedEffect
	}
	bound := false
	for _, b := range e.Repositories {
		bound = bound || b.Key == r.RepositoryKey && b.Identity == r.RepositoryIdentity
	}
	claimed := false
	for _, o := range e.RepositoryOutcomes() {
		if o.RepositoryKey == r.RepositoryKey && len(o.Results) > 0 {
			claimed = true
		}
	}
	if !bound || !claimed {
		return ErrUnauthorizedEffect
	}
	for _, d := range e.Deliveries {
		if d.Status != "confirmed" {
			return ErrUnknownOutstanding
		}
		// A new digest or revision cannot replay an already effected ref/SHA/phase.
		if d.Request.RepositoryIdentity == r.RepositoryIdentity && d.Request.Target.Remote == r.Target.Remote && d.Request.Target.HeadRef == r.Target.HeadRef && d.Request.Target.HeadCommit == r.Target.HeadCommit && d.Request.Target.Push == r.Target.Push {
			return ErrTransition
		}
	}
	e.Deliveries = append(e.Deliveries, ExecutionDelivery{Request: r, Grant: a, RunnerGeneration: generation, Status: "prepared", PreparedAt: now.UTC()})
	e.touch(now)
	return e.Validate()
}

func (e *Execution) FinishDelivery(index int, generation uint64, status string, observation *DeliveryObservation, now time.Time) error {
	if err := e.fence(generation); err != nil {
		return err
	}
	if index < 0 || index >= len(e.Deliveries) {
		return ErrTransition
	}
	d := &e.Deliveries[index]
	if d.Status != "prepared" || status != "unknown" && status != "confirmed" || status == "confirmed" && (d.RunnerGeneration != generation || observation == nil || !d.Request.MatchesObservation(*observation, true)) {
		return ErrTransition
	}
	d.Status = status
	at := now.UTC()
	d.EndedAt = &at
	if observation != nil {
		d.Observations = append(d.Observations, *observation)
	}
	e.touch(now)
	return e.Validate()
}

func (e *Execution) ObserveDelivery(index int, generation uint64, a DeliveryGrant, o DeliveryObservation, now time.Time) error {
	if err := e.fence(generation); err != nil {
		return err
	}
	if index < 0 || index >= len(e.Deliveries) {
		return ErrTransition
	}
	d := &e.Deliveries[index]
	if d.Status != "unknown" || !a.Covers(d.Request) || a != d.Grant {
		return ErrUnauthorizedEffect
	}
	d.Observations = append(d.Observations, o)
	e.touch(now)
	return e.Validate()
}

func validateDeliveries(e Execution) error {
	if len(e.Deliveries) > 32 {
		return invalid("delivery bounds")
	}
	for _, d := range e.Deliveries {
		r := d.Request
		bound := false
		for _, b := range e.Repositories {
			bound = bound || b.Key == r.RepositoryKey && b.Identity == r.RepositoryIdentity
		}
		if !bound || !d.Grant.Covers(r) || r.ExecutionID != e.ExecutionID || r.ProjectID != e.ProjectID || r.Revision >= e.Revision || d.RunnerGeneration == 0 || d.RunnerGeneration > e.RunnerGeneration || d.PreparedAt.IsZero() || len(d.Observations) > 32 {
			return invalid("delivery intent")
		}
		for _, o := range d.Observations {
			if len(o.Candidates) > 100 {
				return invalid("delivery observation bounds")
			}
		}
		switch d.Status {
		case "prepared":
			if d.EndedAt != nil || len(d.Observations) != 0 {
				return invalid("prepared delivery")
			}
		case "unknown":
			if d.EndedAt == nil {
				return invalid("unknown delivery")
			}
		case "confirmed":
			if d.EndedAt == nil || len(d.Observations) != 1 || !r.MatchesObservation(d.Observations[0], true) {
				return invalid("confirmed delivery")
			}
		default:
			return invalid("delivery status")
		}
	}
	return nil
}

// PreservesDeliveryHistory is enforced at the same store CAS as attempts.
func PreservesDeliveryHistory(prior, next []ExecutionDelivery) bool {
	if len(next) < len(prior) {
		return false
	}
	for _, d := range next[len(prior):] {
		if d.Status != "prepared" {
			return false
		}
	}
	for i, d := range prior {
		n := next[i]
		if len(n.Observations) < len(d.Observations) || !reflect.DeepEqual(d.Observations, n.Observations[:len(d.Observations)]) && len(d.Observations) > 0 {
			return false
		}
		if d.Status == "prepared" {
			n.Status = d.Status
			n.EndedAt = d.EndedAt
			n.Observations = d.Observations
		} else if d.Status == "unknown" {
			n.Observations = d.Observations
		}
		if !reflect.DeepEqual(d, n) {
			return false
		}
	}
	return true
}
