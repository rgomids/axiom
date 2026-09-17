package project_test

import (
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/project"
)

func TestIdentityValidationSharedByPortableAndLocalMetadata(t *testing.T) {
	for _, input := range []struct{ id, slug string }{
		{"123e4567-e89b-42d3-a456-426614174000", "demo"},
		{"123E4567-e89b-42d3-a456-426614174000", "demo"},
		{"123e4567-e89b-12d3-a456-426614174000", "demo"},
		{"123e4567-e89b-42d3-7456-426614174000", "demo"},
		{"", "../demo"},
	} {
		_, full := project.New(project.State{SchemaVersion: 1, ID: input.id, Slug: input.slug, Name: "Demo"})
		identity := project.ValidateIdentity(input.id, input.slug)
		if !reflect.DeepEqual(full, identity) {
			t.Fatal("identity rules diverged")
		}
	}
}
