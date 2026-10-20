package operationaldocs

import (
	"strings"
	"testing"
)

func TestAssociationBindsDocumentaryInputs(t *testing.T) {
	d := Descriptor{Generation: strings.Repeat("a", 64), SourceCommit: strings.Repeat("b", 40), Platform: "macos", Architecture: "arm64", SkillsSHA256: strings.Repeat("c", 64)}
	d.Identity = Association(d.Generation, d.SourceCommit, d.Platform, d.Architecture, d.SkillsSHA256)
	if !d.Valid() {
		t.Fatal("valid descriptor refused")
	}
	mutations := []func(*Descriptor){
		func(d *Descriptor) { d.Generation = strings.Repeat("d", 64) },
		func(d *Descriptor) { d.SourceCommit = strings.Repeat("d", 40) },
		func(d *Descriptor) { d.Platform = "linux" },
		func(d *Descriptor) { d.Architecture = "amd64" },
		func(d *Descriptor) { d.SkillsSHA256 = strings.Repeat("d", 64) },
	}
	for i, mutate := range mutations {
		changed := d
		mutate(&changed)
		if changed.Valid() {
			t.Fatalf("mutation %d accepted", i)
		}
	}
}

func TestClosedBindingDecodeRejectsDuplicateAndUnknownMembers(t *testing.T) {
	type record struct {
		Name string `json:"name"`
	}
	var value record
	if err := DecodeClosed([]byte(`{"name":"A"}`), &value, []string{"name"}); err != nil || value.Name != "A" {
		t.Fatal(err)
	}
	for _, wire := range []string{`{"name":"A","name":"B"}`, `{"name":"A","\u006eame":"B"}`, `{"name":"A","extra":1}`, `{}`, `{"name":null}`, `{"name":1}`, `{"name":"A"} {}`, `{"name":"A"`} {
		if DecodeClosed([]byte(wire), &value, []string{"name"}) == nil {
			t.Fatalf("accepted %s", wire)
		}
	}
}

func TestSkillsIdentityRequiresExactActiveInventory(t *testing.T) {
	skills := map[string]string{}
	for _, name := range []string{"axiom-project", "axiom-runtime", "axiom-work-item", "axiom-workflow"} {
		skills[name] = strings.Repeat("a", 64)
	}
	first, err := SkillsIdentity(skills)
	if err != nil {
		t.Fatal(err)
	}
	skills["axiom-runtime"] = strings.Repeat("b", 64)
	second, err := SkillsIdentity(skills)
	if err != nil || first == second {
		t.Fatal("skill mutation not bound")
	}
	skills["foreign"] = strings.Repeat("a", 64)
	if _, err := SkillsIdentity(skills); err == nil {
		t.Fatal("extra skill accepted")
	}
}
