package runtimeprofile

import "testing"

func TestNoHistoricalFormatBecomesV3(t *testing.T) {
	v2 := Configuration{FormatVersion: FormatVersionV2, Revision: 4,
		Runtimes:      []Runtime{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"codex-high"}}},
		ModelProfiles: []ModelProfile{{ID: "codex-high", RuntimeID: "codex", Model: "local-codex-model", Capabilities: []string{"go"}, Complexities: []string{"high"}, Effort: Effort{Mode: "explicit", Value: "high"}}},
	}
	wire, err := Encode(v2)
	if err != nil {
		t.Fatal(err)
	}
	v, err := DecodeVersioned(wire)
	if err != nil || v.V3 != nil || v.FormatVersion != FormatVersionV2 {
		t.Fatalf("v2 reinterpreted: %+v %v", v, err)
	}
}
