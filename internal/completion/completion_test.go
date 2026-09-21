package completion

import (
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/provenance"
)

func TestCanonicalStatusesAreClosed(t *testing.T) {
	want := []Status{Success, Failure, ValidationFailure, DeniedAuthority, Partial, Interrupted, RetryableFailure}
	if got := Statuses(); !reflect.DeepEqual(got, want) {
		t.Fatalf("statuses = %#v, want %#v", got, want)
	}
	for _, status := range want {
		if !status.Valid() {
			t.Fatalf("status %q is invalid", status)
		}
	}
	if Status("cancelled").Valid() || Status("error").Valid() {
		t.Fatal("legacy status accepted")
	}
}

func TestStatusEffectClassificationMatrix(t *testing.T) {
	tests := []struct {
		name  string
		facts Facts
		want  Status
	}{
		{"confirmed requested effect", Facts{RequestedEffectConfirmed: true}, Success},
		{"read only success", Facts{Completed: true}, Success},
		{"ordinary failure", Facts{Failed: true}, Failure},
		{"validation failure", Facts{ValidationFailed: true}, ValidationFailure},
		{"authority denied", Facts{AuthorityDenied: true}, DeniedAuthority},
		{"confirmed effect then failure", Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}, Partial},
		{"interrupted", Facts{WasInterrupted: true}, Interrupted},
		{"safe retry", Facts{RetrySafeFailure: true}, RetryableFailure},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Classify(test.facts)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("status = %q, want %q", got, test.want)
			}
		})
	}
}

func TestClassificationRejectsAmbiguousOrFalseSuccess(t *testing.T) {
	tests := []Facts{
		{},
		{Completed: true, Failed: true},
		{RequestedEffectConfirmed: true, Failed: true},
		{RequestedEffectConfirmed: true, SecondaryFailure: true, AuthorityDenied: true},
		{SecondaryFailure: true},
	}
	for _, facts := range tests {
		if _, err := Classify(facts); err == nil {
			t.Fatalf("accepted ambiguous facts: %#v", facts)
		}
	}
}

func TestResultPreservesSixCanonicalFieldsAndCopiesReferences(t *testing.T) {
	prov := developmentProvenance(t)
	statement := authored(t, "Project state is valid")
	next := authored(t, "Continue with explicit setup")
	references := []string{"repository:main", "project:123"}

	result, err := New(Facts{Completed: true}, statement, references, next, "artifact:123", prov)
	if err != nil {
		t.Fatal(err)
	}
	references[0] = "changed"
	got := result.References()
	got[1] = "changed"

	if result.Status() != Success || result.Result() != statement || result.Next() != next || result.Details() != "artifact:123" || result.Provenance() != prov {
		t.Fatalf("result fields changed: %#v", result)
	}
	if want := []string{"project:123", "repository:main"}; !reflect.DeepEqual(result.References(), want) {
		t.Fatalf("references = %#v, want %#v", result.References(), want)
	}
}

func TestResultRejectsTransportedUserSummaryAndUnsafeMetadata(t *testing.T) {
	prov := developmentProvenance(t)
	transported, err := provenance.NewText("transported-user-sentinel", provenance.UserAuthored)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(Facts{Completed: true}, transported, nil, provenance.Text{}, "", prov); err == nil {
		t.Fatal("transported user content accepted as Axiom result")
	}
	if _, err := New(Facts{Completed: true}, authored(t, "ok"), []string{"reference\nleak"}, provenance.Text{}, "", prov); err == nil {
		t.Fatal("unsafe reference accepted")
	}
}

func TestResultRequiresPartialAndRetryContext(t *testing.T) {
	prov := developmentProvenance(t)
	for _, facts := range []Facts{{RequestedEffectConfirmed: true, SecondaryFailure: true}, {RetrySafeFailure: true}} {
		if _, err := New(facts, authored(t, "operation incomplete"), nil, provenance.Text{}, "", prov); err == nil {
			t.Fatalf("%#v accepted without next action", facts)
		}
	}
	if _, err := New(Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}, authored(t, "effect confirmed"), nil, authored(t, "retry local persistence"), "", prov); err == nil {
		t.Fatal("partial accepted without confirmed reference")
	}
}

func authored(t *testing.T, value string) provenance.Text {
	t.Helper()
	text, err := provenance.NewText(value, provenance.AxiomAuthored)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func developmentProvenance(t *testing.T) provenance.Value {
	t.Helper()
	value, err := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: provenance.Unavailable, SourceState: provenance.Unknown}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
