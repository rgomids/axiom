package workflow

import (
	"bytes"
	"context"
	"encoding/json"
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

type DeliveryEffects interface {
	ObserveDelivery(context.Context, DeliveryPreview) (DeliveryObservation, error)
	PushDelivery(context.Context, DeliveryPreview) error
	CreateDeliveryPR(context.Context, DeliveryPreview) error
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

func exactDeliveryPR(p DeliveryPreview, r DeliveryPR) bool {
	t := p.Target
	return validText(r.ID) && r.URL == "https://github.com/"+t.Repository+"/pull/"+r.ID && r.Repository == t.Repository && r.BaseRef == t.BaseRef && r.HeadRef == t.HeadRef && r.BaseCommit == t.BaseCommit && r.HeadCommit == t.HeadCommit && r.Title == t.Title && r.Body == p.PRBody() && sameJSON(r.Metadata, t.Metadata)
}

// sameJSON compares JSON by compact form: a request may carry insignificant
// whitespace that persistence drops, which must not turn a confirmed effect
// into an unknown one.
func sameJSON(a, b json.RawMessage) bool {
	var ca, cb bytes.Buffer
	return json.Compact(&ca, a) == nil && json.Compact(&cb, b) == nil && bytes.Equal(ca.Bytes(), cb.Bytes())
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
