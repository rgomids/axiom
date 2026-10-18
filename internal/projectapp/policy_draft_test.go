package projectapp_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

func policyProject(t *testing.T, version int) project.Project {
	t.Helper()
	p, issues := project.New(project.State{SchemaVersion: version, ID: editProjectID, Slug: "sample", Name: "Sample",
		Runtimes: project.Configured([]project.Runtime{{ID: "codex"}, {ID: "claude"}}),
		ModelProfiles: project.Configured([]project.ModelProfile{
			{Key: "coding", RuntimeRef: project.Configured("codex"), Model: project.Configured("model")},
			{Key: "review", RuntimeRef: project.Configured("codex"), Model: project.Configured("model")},
		}), RuntimePreferences: project.Configured([]project.RuntimePreference{{Role: "implementer", Complexity: "standard", ModelProfileRef: "coding"}}),
		BusinessContext:      project.Configured(project.BusinessContext{Text: project.Configured("Preserved context"), Documents: project.Configured([]string{"context.md"})}),
		Policies:             project.Configured([]string{"policy.md"}),
		CredentialReferences: project.Configured([]project.CredentialReference{{Key: "retained-reference"}}),
	})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return p
}

func policyDraft(t *testing.T, wire string) projectapp.PolicyDraft {
	t.Helper()
	d, f := projectapp.DecodePolicyDraft([]byte(wire))
	if f != projectapp.EditOK {
		t.Fatalf("draft rejected: %d", f)
	}
	return d
}

func TestPolicyDraftPartialMembersAndPreservation(t *testing.T) {
	for _, version := range []int{2, 3} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			original := policyProject(t, version)
			for _, wire := range []string{
				`{"runtimes":{"mode":"members","add":[{"id":"other"}],"remove":["claude"]}}`,
				`{"modelProfiles":{"mode":"members","replace":[{"key":"review","runtimeRef":"codex","model":"new-model"}]}}`,
				`{"runtimePreferences":{"mode":"members","replace":[{"role":"implementer","complexity":"standard","modelProfileRef":"review"}]}}`,
				`{"modelProfiles":{"mode":"members","add":[{"key":"pending","state":"unconfigured"}]}}`,
				`{"modelProfiles":{"mode":"members","remove":["coding"]},"runtimePreferences":{"mode":"members","replace":[{"role":"implementer","complexity":"standard","modelProfileRef":"review"}]}}`,
			} {
				d := policyDraft(t, wire)
				candidate, f := d.Propose(original)
				if f != projectapp.EditOK {
					t.Fatalf("candidate rejected: %d", f)
				}
				before, after := original.State(), candidate.State()
				after.Runtimes, after.ModelProfiles, after.RuntimePreferences = before.Runtimes, before.ModelProfiles, before.RuntimePreferences
				if !reflect.DeepEqual(before, after) {
					t.Fatal("unrelated fields changed")
				}
				encoded, issues := (manifest.Codec{}).Encode(candidate)
				if len(issues) != 0 {
					t.Fatal(issues)
				}
				decoded, issues := (manifest.Codec{}).Decode(encoded)
				if len(issues) != 0 || !decoded.Equivalent(candidate) {
					t.Fatal("same-schema round trip changed candidate")
				}
				if !reflect.DeepEqual(original.State(), before) {
					t.Fatal("original mutated")
				}
			}
		})
	}
}

func TestPolicyDraftFormsAndLastRemoval(t *testing.T) {
	for _, group := range []string{"runtimes", "modelProfiles", "runtimePreferences"} {
		for _, mode := range []string{"absent", "unconfigured", "replace", "members"} {
			for _, form := range []project.Form{project.Absent, project.NotConfigured, project.Present} {
				s := project.State{SchemaVersion: 2, ID: editProjectID, Slug: "sample", Name: "Sample"}
				if form == project.NotConfigured {
					s.Runtimes = project.Unconfigured[[]project.Runtime]()
					s.ModelProfiles = project.Unconfigured[[]project.ModelProfile]()
					s.RuntimePreferences = project.Unconfigured[[]project.RuntimePreference]()
				} else if form == project.Present {
					s.Runtimes = project.Configured([]project.Runtime{})
					s.ModelProfiles = project.Configured([]project.ModelProfile{})
					s.RuntimePreferences = project.Configured([]project.RuntimePreference{})
				}
				current, issues := project.New(s)
				if len(issues) != 0 {
					t.Fatal(issues)
				}
				payload := ""
				if mode == "replace" {
					payload = `,"values":[]`
				}
				d := policyDraft(t, fmt.Sprintf(`{"%s":{"mode":"%s"%s}}`, group, mode, payload))
				next, f := d.Propose(current)
				if f != projectapp.EditOK {
					t.Fatalf("%s/%s/%d: %d", group, mode, form, f)
				}
				n := next.State()
				got := map[string]project.Form{"runtimes": n.Runtimes.Form(), "modelProfiles": n.ModelProfiles.Form(), "runtimePreferences": n.RuntimePreferences.Form()}[group]
				want := map[string]project.Form{"absent": project.Absent, "unconfigured": project.NotConfigured, "replace": project.Present, "members": project.Present}[mode]
				if got != want {
					t.Fatalf("%s/%s form %d != %d", group, mode, got, want)
				}
			}
		}
	}
	current := policyProject(t, 2)
	d := policyDraft(t, `{"runtimes":{"mode":"members","remove":["codex","claude"]},"modelProfiles":{"mode":"members","remove":["coding","review"]},"runtimePreferences":{"mode":"members","remove":[{"role":"implementer","complexity":"standard"}]}}`)
	next, f := d.Propose(current)
	if f != projectapp.EditOK {
		t.Fatal(f)
	}
	s := next.State()
	r, _ := s.Runtimes.Value()
	p, _ := s.ModelProfiles.Value()
	prefs, _ := s.RuntimePreferences.Value()
	if len(r)+len(p)+len(prefs) != 0 || s.Runtimes.Form() != project.Present || s.ModelProfiles.Form() != project.Present || s.RuntimePreferences.Form() != project.Present {
		t.Fatal("last removal lost configured empty")
	}
}

func TestPolicyDraftStrictRejections(t *testing.T) {
	for _, wire := range []string{
		`null`, `[]`, `{"runtime":{}}`, `{"Runtimes":{"mode":"absent"}}`, `{"runtimes":null}`,
		`{"runtimes":{"mode":"absent","mode":"replace"}}`,
		`{"runtimes":{"mode":"absent","values":[]}}`, `{"runtimes":{"mode":"replace"}}`,
		`{"runtimes":{"mode":"replace","values":null}}`, `{"runtimes":{"mode":"members","values":[]}}`,
		`{"runtimes":{"mode":"members","add":null}}`, `{"runtimes":{"mode":"members","add":[{"ID":"codex"}]}}`,
		`{"runtimes":{"mode":"members","add":[{}]}}`, `{"runtimes":{"mode":"members","add":[{"id":"codex","executable":"sentinel"}]}}`,
		`{"modelProfiles":{"mode":"replace","values":[{"key":"p","runtimeRef":null,"model":"model"}]}}`,
		`{"runtimePreferences":{"mode":"members","remove":[{"role":"x"}]}}`,
		`{"authorizeLocal":true}`, `{"runtimes":{"mode":"absent"}} {}`,
		`{"runtimes":{"mode":"replace","values":[{"id":"\ud800"}]}}`,
	} {
		if _, f := projectapp.DecodePolicyDraft([]byte(wire)); f == projectapp.EditOK {
			t.Errorf("accepted malformed draft %s", wire)
		}
	}
	for _, wire := range []string{
		`{"runtimes":{"mode":"members","add":[{"id":"codex"}]}}`,
		`{"runtimes":{"mode":"members","replace":[{"id":"missing"}]}}`,
		`{"runtimes":{"mode":"members","remove":["missing"]}}`,
		`{"runtimes":{"mode":"members","remove":["claude","claude"]}}`,
		`{"runtimes":{"mode":"members","replace":[{"id":"claude"}],"remove":["claude"]}}`,
		`{"runtimes":{"mode":"members","add":[{"id":"new"}],"remove":["new"]}}`,
		`{"runtimes":{"mode":"members","remove":["codex"]}}`,
		`{"modelProfiles":{"mode":"members","remove":["coding"]}}`,
		`{"modelProfiles":{"mode":"members","replace":[{"key":"coding","state":"unconfigured"}]}}`,
		`{"modelProfiles":{"mode":"members","add":[{"key":"x","state":"unconfigured","model":"model"}]}}`,
		`{"modelProfiles":{"mode":"members","add":[{"key":"x","runtimeRef":"codex"}]}}`,
		`{"runtimePreferences":{"mode":"members","add":[{"role":"implementer","complexity":"standard","modelProfileRef":"review"}]}}`,
		`{"runtimes":{"mode":"replace","values":[{"id":"codex"},{"id":"codex"}]}}`,
		`{"modelProfiles":{"mode":"replace","values":[{"key":"coding","runtimeRef":"missing","model":"model"}]}}`,
		`{"modelProfiles":{"mode":"members","replace":[{"key":"coding","runtimeRef":"codex","model":"/sentinel-machine-path"}]}}`,
		`{"runtimePreferences":{"mode":"replace","values":[{"role":"role","complexity":"standard","modelProfileRef":"missing"}]}}`,
	} {
		d, f := projectapp.DecodePolicyDraft([]byte(wire))
		if f == projectapp.EditOK {
			_, f = d.Propose(policyProject(t, 2))
		}
		if f == projectapp.EditOK {
			t.Errorf("accepted invalid candidate %s", wire)
		}
	}
}

func TestPolicyDraftBounds(t *testing.T) {
	for _, size := range []int{projectapp.MaxPolicyDraftBytes - 1, projectapp.MaxPolicyDraftBytes, projectapp.MaxPolicyDraftBytes + 1} {
		wire := []byte("{}" + strings.Repeat(" ", size-2))
		_, f := projectapp.DecodePolicyDraft(wire)
		if (f == projectapp.EditOK) != (size <= projectapp.MaxPolicyDraftBytes) {
			t.Fatalf("byte boundary %d", size)
		}
	}
	for _, count := range []int{8, 9} {
		runtimes := []string{}
		profiles := []string{}
		prefs := []string{}
		for i := 0; i < count; i++ {
			runtimes = append(runtimes, fmt.Sprintf(`{"id":"runtime-%d"}`, i))
		}
		for i := 0; i < 32; i++ {
			profiles = append(profiles, fmt.Sprintf(`{"key":"profile-%d","runtimeRef":"runtime-0","model":"model"}`, i))
			prefs = append(prefs, fmt.Sprintf(`{"role":"role-%d","complexity":"standard","modelProfileRef":"profile-%d"}`, i, i))
		}
		for _, extra := range []bool{false, true} {
			ps, fs := strings.Join(profiles, ","), strings.Join(prefs, ",")
			if extra {
				ps += `,{"key":"profile-32","runtimeRef":"runtime-0","model":"model"}`
				fs += `,{"role":"role-32","complexity":"standard","modelProfileRef":"profile-32"}`
			}
			wire := fmt.Sprintf(`{"runtimes":{"mode":"replace","values":[%s]},"modelProfiles":{"mode":"replace","values":[%s]},"runtimePreferences":{"mode":"replace","values":[%s]}}`, strings.Join(runtimes, ","), ps, fs)
			d, f := projectapp.DecodePolicyDraft([]byte(wire))
			if f == projectapp.EditOK {
				_, f = d.Propose(policyProject(t, 2))
			}
			if (f == projectapp.EditOK) != (count == 8 && !extra) {
				t.Fatalf("collection boundary %d/%t: %d", count, extra, f)
			}
		}
	}
	for _, spec := range []struct {
		group, member string
		limit         int
	}{
		{"runtimes", `{"id":"new-%d"}`, 16}, {"modelProfiles", `{"key":"new-%d","state":"unconfigured"}`, 64},
		{"runtimePreferences", `{"role":"role-%d","complexity":"standard","modelProfileRef":"coding"}`, 64},
	} {
		for _, n := range []int{spec.limit, spec.limit + 1} {
			entries := []string{}
			for i := 0; i < n; i++ {
				entries = append(entries, fmt.Sprintf(spec.member, i))
			}
			_, f := projectapp.DecodePolicyDraft([]byte(fmt.Sprintf(`{"%s":{"mode":"members","add":[%s]}}`, spec.group, strings.Join(entries, ","))))
			if (f == projectapp.EditOK) != (n == spec.limit) {
				t.Fatalf("operation boundary %s/%d", spec.group, n)
			}
		}
	}
}

func TestPolicyDraftEditSnapshotNoopAndIsolation(t *testing.T) {
	f := newEditFixture(t)
	p := policyProject(t, 3)
	docs := []projectapp.Document{{Name: "context.md", Content: []byte("original context\n")}, {Name: "policy.md", Content: []byte("original policy\n")}}
	update := func(p project.Project) {
		wire, issues := (manifest.Codec{}).Encode(p)
		if len(issues) != 0 {
			t.Fatal(issues)
		}
		wire = append([]byte("# original formatting\n"), wire...)
		snapshot, issues := projectapp.ReadSnapshot(manifest.Codec{}, wire, docs)
		if len(issues) != 0 {
			t.Fatal(issues)
		}
		f.selection.Portable = snapshot
		f.selection.Local.Repositories = nil
		f.selection.Local.PortableRevision = snapshot.Revision()
		f.selection.Local.ArtifactDigests = snapshot.Digests()
		f.selection.LocalWire, _ = (localCodecFixture{}).EncodeLocal(f.selection.Local)
		digest := sha256.Sum256(f.selection.LocalWire)
		f.selection.LocalRevision = hex.EncodeToString(digest[:])
		portableDigest, _ := snapshot.Revision().Digest()
		f.selection.PortableRevision = hex.EncodeToString(portableDigest[:])
	}
	update(p)
	before := f.selection
	d := policyDraft(t, `{"runtimes":{"mode":"replace","values":[{"id":"codex"},{"id":"claude"}]}}`)
	proposal := mustPreview(t, f, projectapp.EditIntent{Policy: &d})
	if len(proposal.Preview().Effects) != 0 || !bytes.Equal(proposal.Manifest(), before.Portable.Manifest()) || !bytes.Equal(proposal.LocalWire(), before.LocalWire) {
		t.Fatal("semantic no-op rewrote formatting or local state")
	}
	d = policyDraft(t, `{"runtimes":{"mode":"members","remove":["claude"]}}`)
	proposal = mustPreview(t, f, projectapp.EditIntent{Policy: &d})
	snapshot, issues := projectapp.ReadSnapshot(manifest.Codec{}, proposal.Manifest(), docs)
	if len(issues) != 0 || !reflect.DeepEqual(snapshot.Documents(), before.Portable.Documents()) {
		t.Fatal("companion snapshots lost")
	}
	nextLocal := proposal.Local()
	nextLocal.PortableRevision = before.Local.PortableRevision
	nextLocal.ArtifactDigests = before.Local.ArtifactDigests
	if !reflect.DeepEqual(nextLocal, before.Local) {
		t.Fatal("local bindings changed")
	}
	if !reflect.DeepEqual(f.selection, before) {
		t.Fatal("preview changed source")
	}
	for _, intent := range []projectapp.EditIntent{{Policy: &d, Name: set("new")}, {Policy: &d, RepositoryRemovals: []string{"api"}}, {Policy: &d, RemoveWorkItemProvider: true}, {Policy: &d, IntegrationRemovals: []string{"chat"}}} {
		if projectapp.ValidateEditIntent(intent) == projectapp.EditOK {
			t.Fatal("mixed intent accepted")
		}
	}
	result := projectapp.ApplyEdit(context.Background(), projectapp.EditPorts{}, nil, projectapp.EditIntent{Policy: &d}, projectapp.EditAuthority{})
	if result.Failure != projectapp.EditUnavailable {
		t.Fatal("unguarded policy publication exposed")
	}
	for _, version := range []int{1, 4} {
		s := p.State()
		s.SchemaVersion = version
		if version == 1 {
			s.Runtimes = project.Declaration[[]project.Runtime]{}
			s.RuntimePreferences = project.Declaration[[]project.RuntimePreference]{}
			s.Runtime = project.Configured(project.Runtime{ID: "codex"})
		}
		current, issues := project.New(s)
		if len(issues) != 0 {
			t.Fatal(issues)
		}
		if _, failure := d.Propose(current); failure == projectapp.EditOK {
			t.Fatalf("schema %d accepted", version)
		}
	}
}

func TestPolicyDiffDeterministicAndSafe(t *testing.T) {
	original := policyProject(t, 3)
	drafts := []string{
		`{"runtimes":{"mode":"replace","values":[{"id":"other"},{"id":"codex"}]}}`,
		`{"runtimes":{"values":[{"id":"codex"},{"id":"other"}],"mode":"replace"}}`,
	}
	var first []byte
	for _, wire := range drafts {
		next, f := policyDraft(t, wire).Propose(original)
		if f != projectapp.EditOK {
			t.Fatal(f)
		}
		diff := projectapp.PolicyDiff(original, next)
		if len(diff) != 2 || diff[0].Member != "claude" || diff[1].Member != "other" {
			t.Fatalf("incomplete diff: %+v", diff)
		}
		encoded, err := json.Marshal(struct {
			Policy projectapp.PolicyView
			Diff   []projectapp.PolicyChange
		}{projectapp.ProjectPolicy(next), diff})
		if err != nil {
			t.Fatal(err)
		}
		if first != nil && !bytes.Equal(first, encoded) {
			t.Fatal("ordering changed semantic diff")
		}
		first = encoded
		for _, sentinel := range []string{"retained-reference", "Preserved context", "context.md", "policy.md"} {
			if bytes.Contains(encoded, []byte(sentinel)) {
				t.Fatal("unrelated portable state leaked")
			}
		}
	}
	if len(projectapp.PolicyDiff(original, original)) != 0 {
		t.Fatal("equal policy has diff")
	}
	for _, mode := range []string{"absent", "unconfigured", "replace"} {
		payload := ""
		if mode == "replace" {
			payload = `,"values":[]`
		}
		next, f := policyDraft(t, `{"runtimePreferences":{"mode":"`+mode+`"`+payload+`}}`).Propose(original)
		if f != projectapp.EditOK {
			t.Fatal(f)
		}
		diff := projectapp.PolicyDiff(original, next)
		expected := 1
		if mode != "replace" {
			expected = 2
		}
		if len(diff) != expected {
			t.Fatalf("presence diff incomplete for %s: %+v", mode, diff)
		}
	}
}

func TestPolicyDraftAddsToAbsentAndUnconfigured(t *testing.T) {
	for _, unconfigured := range []bool{false, true} {
		s := project.State{SchemaVersion: 2, ID: editProjectID, Slug: "sample", Name: "Sample"}
		if unconfigured {
			s.Runtimes = project.Unconfigured[[]project.Runtime]()
			s.ModelProfiles = project.Unconfigured[[]project.ModelProfile]()
			s.RuntimePreferences = project.Unconfigured[[]project.RuntimePreference]()
		}
		original, issues := project.New(s)
		if len(issues) != 0 {
			t.Fatal(issues)
		}
		draft := policyDraft(t, `{"runtimes":{"mode":"members","add":[{"id":"codex"}]},"modelProfiles":{"mode":"members","add":[{"key":"coding","runtimeRef":"codex","model":"model"}]},"runtimePreferences":{"mode":"members","add":[{"role":"implementer","complexity":"standard","modelProfileRef":"coding"}]}}`)
		next, f := draft.Propose(original)
		if f != projectapp.EditOK {
			t.Fatal(f)
		}
		if !project.RuntimePolicyDeclared(next.State()) {
			t.Fatal("add did not configure policy")
		}
	}
}

func TestPolicyDraftIndividualProfileAndPreferenceBounds(t *testing.T) {
	original := policyProject(t, 2)
	for _, group := range []string{"modelProfiles", "runtimePreferences"} {
		entries := []string{}
		for i := 0; i < 33; i++ {
			if group == "modelProfiles" {
				entries = append(entries, fmt.Sprintf(`{"key":"profile-%d","state":"unconfigured"}`, i))
			} else {
				entries = append(entries, fmt.Sprintf(`{"role":"role-%d","complexity":"standard","modelProfileRef":"coding"}`, i))
			}
		}
		wire := fmt.Sprintf(`{"%s":{"mode":"replace","values":[%s]}}`, group, strings.Join(entries, ","))
		if _, failure := projectapp.DecodePolicyDraft([]byte(wire)); failure == projectapp.EditOK {
			t.Fatalf("33 accepted for %s", group)
		}
		add := strings.Replace(wire, `"mode":"replace","values"`, `"mode":"members","add"`, 1)
		d := policyDraft(t, add)
		if _, f := d.Propose(original); f == projectapp.EditOK {
			t.Fatalf("merged 33 accepted for %s", group)
		}
	}
}
