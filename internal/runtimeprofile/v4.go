package runtimeprofile

import (
	"encoding/json"
	"sort"

	"github.com/rgomids/axiom/internal/strictjson"
)

const FormatVersionV4 = 4

type ConfigurationV4 struct {
	FormatVersion int            `json:"formatVersion"`
	Revision      uint64         `json:"revision"`
	Runtimes      []RuntimeV4    `json:"runtimes"`
	ModelProfiles []ModelProfile `json:"modelProfiles"`
	Preferences   []Preference   `json:"preferences,omitempty"`
}

type RuntimeV4 struct {
	ID                    string   `json:"id"`
	Adapter               string   `json:"adapter"`
	Enabled               bool     `json:"enabled"`
	AllowlistedProfileIDs []string `json:"allowlistedProfileIds"`
}

func ValidateV4(c ConfigurationV4) error {
	if c.FormatVersion != FormatVersionV4 {
		if c.FormatVersion <= 0 {
			return ErrInvalidConfiguration
		}
		return ErrUnsupportedConfigurationFormat
	}
	runtimes := make([]Runtime, len(c.Runtimes))
	for i, r := range c.Runtimes {
		runtimes[i] = Runtime{ID: r.ID, Adapter: r.Adapter, Enabled: r.Enabled, AllowlistedProfileIDs: r.AllowlistedProfileIDs}
	}
	return Validate(Configuration{FormatVersion: FormatVersionV2, Revision: c.Revision, Runtimes: runtimes, ModelProfiles: c.ModelProfiles, Preferences: c.Preferences})
}

func DecodeV4(wire []byte) (ConfigurationV4, error) {
	var c ConfigurationV4
	if strictjson.Decode(wire, &c, MaxConfigurationBytes) != nil {
		return ConfigurationV4{}, ErrInvalidConfiguration
	}
	if err := ValidateV4(c); err != nil {
		return ConfigurationV4{}, err
	}
	return c, nil
}

func EncodeV4(c ConfigurationV4) ([]byte, error) {
	if err := ValidateV4(c); err != nil {
		return nil, err
	}
	wire, err := json.Marshal(canonicalV4(c))
	if err != nil || len(wire)+1 > MaxConfigurationBytes {
		return nil, ErrInvalidConfiguration
	}
	return append(wire, '\n'), nil
}

func DigestV4(c ConfigurationV4) (string, error) {
	wire, err := EncodeV4(c)
	if err != nil {
		return "", err
	}
	return hash(wire), nil
}

func canonicalV4(c ConfigurationV4) ConfigurationV4 {
	n := c
	n.Runtimes = make([]RuntimeV4, len(c.Runtimes))
	for i, r := range c.Runtimes {
		r.AllowlistedProfileIDs = sortedStrings(r.AllowlistedProfileIDs)
		n.Runtimes[i] = r
	}
	sort.Slice(n.Runtimes, func(i, j int) bool { return n.Runtimes[i].ID < n.Runtimes[j].ID })
	n.ModelProfiles = append([]ModelProfile(nil), c.ModelProfiles...)
	for i := range n.ModelProfiles {
		n.ModelProfiles[i].Capabilities = sortedStrings(n.ModelProfiles[i].Capabilities)
		n.ModelProfiles[i].Complexities = sortedStrings(n.ModelProfiles[i].Complexities)
	}
	sort.Slice(n.ModelProfiles, func(i, j int) bool { return n.ModelProfiles[i].ID < n.ModelProfiles[j].ID })
	n.Preferences = append([]Preference(nil), c.Preferences...)
	sort.Slice(n.Preferences, func(i, j int) bool {
		return n.Preferences[i].Role+"\x00"+n.Preferences[i].Complexity < n.Preferences[j].Role+"\x00"+n.Preferences[j].Complexity
	})
	return n
}
