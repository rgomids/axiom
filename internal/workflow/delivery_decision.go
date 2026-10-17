package workflow

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/rgomids/axiom/internal/strictjson"
)

type DeliveryDecision struct {
	FormatVersion int                  `json:"formatVersion"`
	Kind          string               `json:"kind"`
	Packet        DeliveryPacket       `json:"packet"`
	PacketDigest  string               `json:"packetDigest"`
	Publication   *DeliveryPublication `json:"publication,omitempty"`
	Outcome       string               `json:"outcome"`
	ActorKind     string               `json:"actorKind"`
	Actor         string               `json:"actor"`
	Authority     Reference            `json:"authority"`
	Reason        string               `json:"reason"`
	Evidence      []Reference          `json:"evidence"`
}

func (d DeliveryDecision) Digest() string { return digest(d) }

type DeliveryDecisions interface {
	LoadDeliveryDecision(context.Context, string, string) (DeliveryDecision, error)
	RecordDeliveryDecision(context.Context, State, DeliveryDecision) error
}

func PrepareDeliveryDecision(ctx context.Context, s State, p DeliveryPacket, publication *DeliveryPublication, outcome, actor, reason string, authority Reference, evidence []Reference, validator ReferenceValidator, repository string) (DeliveryDecision, error) {
	ref, err := DeliveryPacketDigest(p)
	d := DeliveryDecision{FormatVersion: 1, Kind: "delivery_decision", Packet: p, PacketDigest: ref, Publication: publication, Outcome: outcome, ActorKind: "human-operator", Actor: actor, Authority: authority, Reason: reason, Evidence: evidence}
	if err != nil || !deliveryPacketCurrent(s, p) || validator == nil {
		return d, ErrDelivery
	}
	if _, err = EncodeDeliveryDecision(d); err != nil {
		return d, err
	}
	for _, r := range append([]Reference{authority}, evidence...) {
		if err = validator.Validate(ctx, s.ExecutionID, repository, r); err != nil {
			return d, err
		}
	}
	return d, nil
}
func ApplyDeliveryDecision(ctx context.Context, store DeliveryDecisions, s State, d DeliveryDecision, preview string, local, human bool) (DeliveryDecision, error) {
	if _, err := EncodeDeliveryDecision(d); err != nil || !local || !human || preview != d.Digest() || !deliveryPacketCurrent(s, d.Packet) {
		return d, ErrDelivery
	}
	return d, store.RecordDeliveryDecision(ctx, s, d)
}
func EncodeDeliveryDecision(d DeliveryDecision) ([]byte, error) {
	ref, err := DeliveryPacketDigest(d.Packet)
	if err != nil || d.FormatVersion != 1 || d.Kind != "delivery_decision" || ref != d.PacketDigest || d.ActorKind != "human-operator" || !validText(d.Actor) || !validText(d.Reason) || !validReference(d.Authority) || len(d.Evidence) > 16 || (d.Outcome != "accept" && d.Outcome != "reject") {
		return nil, ErrDelivery
	}
	for _, r := range d.Evidence {
		if !validReference(r) {
			return nil, ErrDelivery
		}
	}
	if d.Publication != nil {
		if _, err = EncodeDeliveryPublication(*d.Publication); err != nil || d.Publication.Preview.PacketDigest != ref {
			return nil, ErrDelivery
		}
	}
	if d.Outcome == "accept" && (!d.Packet.Ready || d.Publication == nil || d.Publication.Status != "success" || d.Publication.Artifact == "") {
		return nil, ErrDelivery
	}
	wire, err := json.Marshal(d)
	if err != nil || len(wire)+1 > MaxDeliveryBytes {
		return nil, ErrDelivery
	}
	return append(wire, '\n'), nil
}
func DecodeDeliveryDecision(wire []byte) (DeliveryDecision, error) {
	var d DeliveryDecision
	if strictjson.Decode(wire, &d, MaxDeliveryBytes) != nil {
		return d, ErrDelivery
	}
	_, err := EncodeDeliveryDecision(d)
	return d, err
}
func DeliveryDisposition(s State) (string, bool) {
	if s.Binding == nil {
		return "", false
	}
	if s.DeliveryDecision == nil {
		return "pending", false
	}
	d := s.DeliveryDecision
	if _, err := EncodeDeliveryDecision(*d); err != nil || !deliveryPacketCurrent(s, d.Packet) {
		return "recovery_required", false
	}
	if d.Outcome == "accept" {
		return "accepted", true
	}
	return "rejected", false
}
func SameDeliveryDecision(a, b DeliveryDecision) bool { return reflect.DeepEqual(a, b) }
