package provenance

import "testing"

func TestBuildProvenanceMatrix(t *testing.T) {
	tests := []struct {
		name     string
		input    Build
		settings []Setting
		want     Value
	}{
		{
			name:  "released",
			input: Build{Release: true, Version: "1.2.3", Revision: "abc123def456", SourceState: Clean},
			want:  value("1.2.3", "abc123def456", Clean),
		},
		{
			name:  "development clean from observed settings",
			input: Build{Version: Development, Revision: Unavailable, SourceState: Unknown},
			settings: []Setting{
				{Key: "vcs.revision", Value: "abc123def4567890"},
				{Key: "vcs.modified", Value: "false"},
			},
			want: value(Development, "abc123def456", Clean),
		},
		{
			name:  "development dirty from observed settings",
			input: Build{Version: Development, Revision: Unavailable, SourceState: Unknown},
			settings: []Setting{
				{Key: "vcs.revision", Value: "abc123def4567890"},
				{Key: "vcs.modified", Value: "true"},
			},
			want: value(Development, "abc123def456", Dirty),
		},
		{
			name:  "development unavailable revision",
			input: Build{Version: Development, Revision: Unavailable, SourceState: Unknown},
			want:  value(Development, Unavailable, Unknown),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := FromBuild(test.input, test.settings)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("provenance = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestReleaseProvenanceFailsClosed(t *testing.T) {
	tests := []Build{
		{Release: true, Version: Development, Revision: "abc123", SourceState: Clean},
		{Release: true, Version: "1.2", Revision: "abc123", SourceState: Clean},
		{Release: true, Version: "1.2.3-01", Revision: "abc123", SourceState: Clean},
		{Release: true, Version: "1.2.3", Revision: Unavailable, SourceState: Clean},
		{Release: true, Version: "1.2.3", Revision: "abc123", SourceState: Dirty},
		{Release: true, Version: "1.2.3", Revision: "abc123", SourceState: Unknown},
	}
	for _, input := range tests {
		if _, err := FromBuild(input, nil); err == nil {
			t.Fatalf("accepted invalid release input: %#v", input)
		}
	}
}

func TestDevelopmentRejectsUntrustedBuildMetadata(t *testing.T) {
	tests := []Build{
		{Version: "poc-abc", Revision: Unavailable, SourceState: Unknown},
		{Version: Development, Revision: "bad\nrevision", SourceState: Clean},
		{Version: Development, Revision: "abc123", SourceState: "modified"},
	}
	for _, input := range tests {
		if _, err := FromBuild(input, nil); err == nil {
			t.Fatalf("accepted invalid development input: %#v", input)
		}
	}
}

func TestAuthoredTextRejectsTransportedUserContent(t *testing.T) {
	const sentinel = "transported-user-sentinel"
	transported, err := NewText(sentinel, UserAuthored)
	if err != nil {
		t.Fatal(err)
	}
	if transported.Authorship() != UserAuthored || transported.String() != sentinel {
		t.Fatalf("transported text = %#v", transported)
	}
	if _, err := RequireAxiomAuthored(transported); err == nil {
		t.Fatal("transported user content accepted as Axiom-authored")
	}
}

func value(version, revision string, state SourceState) Value {
	return Value{product: Product, version: version, revision: revision, sourceState: state}
}
