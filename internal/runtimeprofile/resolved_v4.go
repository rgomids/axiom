package runtimeprofile

import (
	"encoding/json"

	"github.com/rgomids/axiom/internal/strictjson"
)

type ResolvedProfileV4 struct {
	ProfileID             string `json:"profileId"`
	RuntimeID             string `json:"runtimeId"`
	Model                 string `json:"model"`
	Effort                Effort `json:"effort"`
	ProfileDigest         string `json:"profileDigest"`
	ConfigurationRevision uint64 `json:"configurationRevision"`
	ConfigurationDigest   string `json:"configurationDigest"`
	ExecutableDigest      string `json:"executableDigest"`
	InvocationMode        string `json:"invocationMode"`
	ObservationRevision   uint64 `json:"observationRevision"`
	ObservationDigest     string `json:"observationDigest"`
	EffortSourceDigest    string `json:"effortSourceDigest"`
}

func ValidateResolvedProfileV4(p ResolvedProfileV4) error {
	if !validToken(p.ProfileID) || !SafeIdentifier(p.ProfileID) || (p.RuntimeID != "codex" && p.RuntimeID != "claude") || !SafeIdentifier(p.Model) || !ValidEffort(p.Effort) || p.ConfigurationRevision == 0 || p.ObservationRevision == 0 || p.InvocationMode != "noninteractive" {
		return ErrInvalidConfiguration
	}
	for _, d := range []string{p.ProfileDigest, p.ConfigurationDigest, p.ExecutableDigest, p.ObservationDigest, p.EffortSourceDigest} {
		if !validDigest(d) {
			return ErrInvalidConfiguration
		}
	}
	return nil
}

func DecodeResolvedProfileV4(wire []byte) (ResolvedProfileV4, error) {
	var p ResolvedProfileV4
	if strictjson.Decode(wire, &p, MaxConfigurationBytes) != nil || ValidateResolvedProfileV4(p) != nil {
		return ResolvedProfileV4{}, ErrInvalidConfiguration
	}
	return p, nil
}

func ResolvedProfileDigestV4(p ResolvedProfileV4) (string, error) {
	if err := ValidateResolvedProfileV4(p); err != nil {
		return "", err
	}
	wire, err := json.Marshal(p)
	if err != nil {
		return "", ErrInvalidConfiguration
	}
	return hash(wire), nil
}

func operationalObservationDigest(o OperationalObservation) (string, error) {
	wire, err := json.Marshal(o)
	if err != nil {
		return "", ErrInvalidConfiguration
	}
	return hash(wire), nil
}
