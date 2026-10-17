package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/rgomids/axiom/internal/strictjson"
)

type DeliveryTarget struct {
	RetryOf    string          `json:"retryOf,omitempty"`
	Kind       string          `json:"kind"`
	Repository string          `json:"repository"`
	Remote     string          `json:"remote"`
	BaseRef    string          `json:"baseRef"`
	HeadRef    string          `json:"headRef"`
	BaseCommit string          `json:"baseCommit"`
	HeadCommit string          `json:"headCommit"`
	Tree       string          `json:"tree"`
	DiffDigest string          `json:"diffDigest"`
	Title      string          `json:"title"`
	Body       string          `json:"body"`
	Metadata   json.RawMessage `json:"metadata"`
	Push       bool            `json:"push"`
	PR         bool            `json:"pr"`
}

type DeliveryPR struct {
	ID         string          `json:"id"`
	URL        string          `json:"url"`
	Repository string          `json:"repository"`
	BaseRef    string          `json:"baseRef"`
	HeadRef    string          `json:"headRef"`
	BaseCommit string          `json:"baseCommit"`
	HeadCommit string          `json:"headCommit"`
	Title      string          `json:"title"`
	Body       string          `json:"body"`
	Metadata   json.RawMessage `json:"metadata"`
}

type DeliveryObservation struct {
	BaseCommit string       `json:"baseCommit"`
	HeadCommit string       `json:"headCommit"`
	Tree       string       `json:"tree"`
	DiffDigest string       `json:"diffDigest"`
	RemoteBase string       `json:"remoteBase"`
	RemoteHead string       `json:"remoteHead"`
	Candidates []DeliveryPR `json:"candidates"`
	Conclusive bool         `json:"conclusive"`
}

type DeliveryPreview struct {
	Packet       DeliveryPacket      `json:"packet"`
	PacketDigest string              `json:"packetDigest"`
	Target       DeliveryTarget      `json:"target"`
	Observation  DeliveryObservation `json:"observation"`
}

func (p DeliveryPreview) Digest() string { return digest(p) }
func (p DeliveryPreview) Correlation() string {
	return digest(struct {
		Packet string
		Target DeliveryTarget
	}{p.PacketDigest, p.Target})
}
func (p DeliveryPreview) PRBody() string {
	return p.Target.Body + "\n\n<!-- axiom:delivery:" + p.Correlation() + " -->"
}

type DeliveryPublication struct {
	FormatVersion int             `json:"formatVersion"`
	Kind          string          `json:"kind"`
	Preview       DeliveryPreview `json:"preview"`
	Status        string          `json:"status"`
	Push          string          `json:"push"`
	PR            string          `json:"pr"`
	Artifact      string          `json:"artifact"`
	ConfirmedPR   *DeliveryPR     `json:"confirmedPr,omitempty"`
	Next          string          `json:"next"`
}

type DeliveryPublicationStore interface {
	LoadDeliveryPublication(context.Context, string, string) (DeliveryPublication, error)
	SaveDeliveryPublication(context.Context, State, DeliveryPublication, *DeliveryPublication) error
}

type DeliveryEffects interface {
	ObserveDelivery(context.Context, DeliveryPreview) (DeliveryObservation, error)
	PushDelivery(context.Context, DeliveryPreview) error
	CreateDeliveryPR(context.Context, DeliveryPreview) error
}

type DeliveryAuthority struct {
	Digest          string
	Local, Push, PR bool
}

type DeliveryPublisher struct {
	Store   DeliveryPublicationStore
	Effects DeliveryEffects
}

func validDeliveryTarget(t DeliveryTarget) bool {
	if t.RetryOf != "" && !validDigest(t.RetryOf) {
		return false
	}
	if t.Kind == "local" {
		return t.RetryOf == "" && !t.Push && !t.PR && t.Remote == "" && t.Repository == ""
	}
	if t.Kind != "github" || !validText(t.Repository) || !validText(t.Remote) || !validText(t.Title) || len(t.Body) > 64<<10 || len(t.Metadata) > 8192 || !json.Valid(t.Metadata) || !validGitRef(t.BaseRef) || !validGitRef(t.HeadRef) || t.BaseRef == t.HeadRef {
		return false
	}
	return validGitObject(t.BaseCommit) && validGitObject(t.HeadCommit) && validGitObject(t.Tree) && validDigest(t.DiffDigest) && (t.Push || t.PR)
}
func validGitObject(s string) bool {
	return (len(s) == 40 || len(s) == 64) && validDigest(s+strings.Repeat("0", 64-len(s)))
}
func validGitRef(s string) bool {
	return strings.HasPrefix(s, "refs/heads/") && len(s) > 11 && len(s) < 256 && !strings.ContainsAny(s, " ~^:?*[\\\r\n\x00") && !strings.Contains(s, "..") && !strings.Contains(s, "@{") && !strings.HasSuffix(s, "/") && !strings.HasSuffix(s, ".lock")
}

func deliveryPacketCurrent(s State, p DeliveryPacket) bool {
	return s.Binding != nil && p.ProjectID == s.ProjectID && p.RepositoryKey == s.RepositoryKey && p.WorkItem == s.WorkItem && p.ExecutionID == s.ExecutionID && p.ExecutionRevision == s.Revision && p.LedgerDigest == fmtDigest(s.StorageRevision) && p.BindingDigest == digest(s.Binding) && p.Definition == s.Binding.Definition
}

func (d DeliveryPublisher) Preview(ctx context.Context, s State, p DeliveryPacket, t DeliveryTarget) (DeliveryPreview, error) {
	ref, err := DeliveryPacketDigest(p)
	if err != nil || !deliveryPacketCurrent(s, p) || !validDeliveryTarget(t) {
		return DeliveryPreview{}, ErrDelivery
	}
	v := DeliveryPreview{Packet: p, PacketDigest: ref, Target: t}
	if t.Kind == "local" {
		return v, nil
	}
	found := false
	var tree, diff string
	for _, stage := range p.Stages {
		if stage.ResultTree != "" {
			found = true
			tree, diff = stage.ResultTree, stage.DiffDigest
		}
	}
	if found && (tree != t.Tree || diff != t.DiffDigest) {
		return v, ErrDelivery
	}
	if !found || d.Effects == nil {
		return v, ErrDelivery
	}
	v.Observation, err = d.Effects.ObserveDelivery(ctx, v)
	if err != nil || !validDeliveryObservation(v) {
		return v, ErrDelivery
	}
	if err = d.validateRetry(ctx, v); err != nil {
		return v, err
	}
	return v, nil
}
func validDeliveryObservation(p DeliveryPreview) bool {
	o, t := p.Observation, p.Target
	return o.Conclusive && o.BaseCommit == t.BaseCommit && o.HeadCommit == t.HeadCommit && o.Tree == t.Tree && o.DiffDigest == t.DiffDigest && o.RemoteBase == t.BaseCommit && (o.RemoteHead == "" || o.RemoteHead == t.HeadCommit) && len(o.Candidates) == 0
}
func exactDeliveryPR(p DeliveryPreview, r DeliveryPR) bool {
	t := p.Target
	return validText(r.ID) && r.URL == "https://github.com/"+t.Repository+"/pull/"+r.ID && r.Repository == t.Repository && r.BaseRef == t.BaseRef && r.HeadRef == t.HeadRef && r.BaseCommit == t.BaseCommit && r.HeadCommit == t.HeadCommit && r.Title == t.Title && r.Body == p.PRBody() && reflect.DeepEqual(r.Metadata, t.Metadata)
}

func (d DeliveryPublisher) validateRetry(ctx context.Context, p DeliveryPreview) error {
	if p.Target.RetryOf == "" {
		return nil
	}
	prior, err := d.Store.LoadDeliveryPublication(ctx, p.Packet.ProjectID, p.Target.RetryOf)
	if err != nil || prior.Status == "success" || prior.Preview.PacketDigest != p.PacketDigest || !p.Observation.Conclusive || len(p.Observation.Candidates) != 0 {
		return ErrDelivery
	}
	old, next := prior.Preview.Target, p.Target
	old.RetryOf = ""
	next.RetryOf = ""
	if old.Push && !next.Push {
		if p.Observation.RemoteHead != next.HeadCommit {
			return ErrDelivery
		}
		old.Push = false
	}
	if !reflect.DeepEqual(old, next) {
		return ErrDelivery
	}
	return nil
}

func (d DeliveryPublisher) Apply(ctx context.Context, s State, p DeliveryPreview, a DeliveryAuthority) (DeliveryPublication, error) {
	ref, err := DeliveryPacketDigest(p.Packet)
	if err != nil || ref != p.PacketDigest || !deliveryPacketCurrent(s, p.Packet) || !p.Packet.Ready || !validDeliveryTarget(p.Target) || a.Digest != p.Digest() || !a.Local || p.Target.Push && !a.Push || p.Target.PR && !a.PR {
		return DeliveryPublication{}, ErrDelivery
	}
	prior, err := d.Store.LoadDeliveryPublication(ctx, s.ProjectID, p.Digest())
	if err == nil {
		if prior.Status != "success" {
			return prior, ErrRecoveryRequired
		}
		return prior, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return prior, err
	}
	if s.DeliveryDecision != nil {
		return prior, ErrDelivery
	}
	if p.Target.Kind != "local" {
		observed, e := d.Effects.ObserveDelivery(ctx, p)
		if e != nil || !reflect.DeepEqual(observed, p.Observation) || !validDeliveryObservation(p) {
			return prior, ErrDelivery
		}
	}
	if err = d.validateRetry(ctx, p); err != nil {
		return prior, err
	}
	r := DeliveryPublication{FormatVersion: 1, Kind: "delivery_publication", Preview: p, Status: "failure", Push: "not_requested", PR: "not_requested", Next: "inspect/reconcile publication; never retry an uncertain effect"}
	if p.Target.Push {
		r.Push = "unknown"
	}
	if p.Target.PR {
		r.PR = "unknown"
	}
	if err = d.Store.SaveDeliveryPublication(ctx, s, r, nil); err != nil {
		return r, err
	}
	if p.Target.Kind == "local" {
		next := r
		next.Status = "success"
		next.Artifact = "delivery-packet:" + p.PacketDigest
		next.Next = "record explicit human delivery decision"
		return d.save(ctx, s, r, next)
	}
	if p.Target.Push {
		if err = d.Effects.PushDelivery(ctx, p); err != nil {
			return r, err
		}
		o, e := d.Effects.ObserveDelivery(ctx, p)
		if e != nil || o.RemoteHead != p.Target.HeadCommit {
			return r, ErrRecoveryRequired
		}
		next := r
		next.Push = "confirmed"
		next.Artifact = p.Target.Remote + "#" + p.Target.HeadRef + "@" + p.Target.HeadCommit
		next.Status = "partial"
		r, err = d.save(ctx, s, r, next)
		if err != nil {
			return r, err
		}
		if o.RemoteBase != p.Target.BaseCommit || len(o.Candidates) != 0 {
			return r, ErrRecoveryRequired
		}
	}
	if p.Target.PR {
		if !p.Target.Push && p.Observation.RemoteHead != p.Target.HeadCommit {
			return r, ErrDelivery
		}
		if err = d.Effects.CreateDeliveryPR(ctx, p); err != nil {
			return r, err
		}
	}
	return d.Reconcile(ctx, s, r)
}
func (d DeliveryPublisher) save(ctx context.Context, s State, prior, next DeliveryPublication) (DeliveryPublication, error) {
	if err := d.Store.SaveDeliveryPublication(ctx, s, next, &prior); err != nil {
		return next, ErrRecoveryRequired
	}
	return next, nil
}
func (d DeliveryPublisher) Reconcile(ctx context.Context, s State, r DeliveryPublication) (DeliveryPublication, error) {
	if !deliveryPacketCurrent(s, r.Preview.Packet) || r.Preview.Target.Kind != "github" {
		return r, ErrDelivery
	}
	o, err := d.Effects.ObserveDelivery(ctx, r.Preview)
	if err != nil || !o.Conclusive {
		return r, ErrRecoveryRequired
	}
	next := r
	if r.Preview.Target.Push && o.RemoteHead == r.Preview.Target.HeadCommit {
		next.Push = "confirmed"
		next.Artifact = r.Preview.Target.Remote + "#" + r.Preview.Target.HeadRef + "@" + o.RemoteHead
	}
	if r.Preview.Target.PR {
		if len(o.Candidates) != 1 || !exactDeliveryPR(r.Preview, o.Candidates[0]) {
			if next.Push == "confirmed" {
				next.Status = "partial"
				if !reflect.DeepEqual(r, next) {
					saved, e := d.save(ctx, s, r, next)
					if e != nil {
						return saved, e
					}
					return saved, ErrRecoveryRequired
				}
			}
			return r, ErrRecoveryRequired
		}
		pr := o.Candidates[0]
		next.PR = "confirmed"
		next.ConfirmedPR = &pr
		next.Artifact = pr.URL
	}
	if (!r.Preview.Target.Push || next.Push == "confirmed") && (!r.Preview.Target.PR || next.PR == "confirmed") {
		next.Status = "success"
		next.Next = "record explicit human delivery decision"
	}
	if reflect.DeepEqual(r, next) {
		return r, nil
	}
	return d.save(ctx, s, r, next)
}
func EncodeDeliveryPublication(r DeliveryPublication) ([]byte, error) {
	ref, err := DeliveryPacketDigest(r.Preview.Packet)
	if err != nil || r.FormatVersion != 1 || r.Kind != "delivery_publication" || ref != r.Preview.PacketDigest || !validDeliveryTarget(r.Preview.Target) || !validText(r.Next) {
		return nil, ErrDelivery
	}
	if r.Status != "failure" && r.Status != "partial" && r.Status != "success" {
		return nil, ErrDelivery
	}
	for _, effect := range []string{r.Push, r.PR} {
		if effect != "not_requested" && effect != "unknown" && effect != "confirmed" {
			return nil, ErrDelivery
		}
	}
	if r.ConfirmedPR != nil && !exactDeliveryPR(r.Preview, *r.ConfirmedPR) {
		return nil, ErrDelivery
	}
	if r.Status == "success" {
		if !r.Preview.Packet.Ready {
			return nil, ErrDelivery
		}
		if r.Preview.Target.Kind == "local" {
			if r.Artifact != "delivery-packet:"+r.Preview.PacketDigest {
				return nil, ErrDelivery
			}
		} else {
			if r.Preview.Target.Push && r.Push != "confirmed" || r.Preview.Target.PR && (r.PR != "confirmed" || r.ConfirmedPR == nil) {
				return nil, ErrDelivery
			}
		}
	}
	wire, err := json.Marshal(r)
	if err != nil || len(wire)+1 > MaxDeliveryBytes {
		return nil, ErrDelivery
	}
	return append(wire, '\n'), nil
}
func DecodeDeliveryPublication(wire []byte) (DeliveryPublication, error) {
	var r DeliveryPublication
	if strictjson.Decode(wire, &r, MaxDeliveryBytes) != nil {
		return r, ErrDelivery
	}
	_, err := EncodeDeliveryPublication(r)
	return r, err
}
