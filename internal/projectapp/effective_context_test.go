package projectapp

import (
	"context"
	"errors"
	"testing"
)

type contextFixture struct {
	values map[string]string
	broken bool
}

func (f *contextFixture) ReadProjectContext(_ context.Context, session string) (string, error) {
	if f.broken {
		return "", errors.New("invalid")
	}
	return f.values[session], nil
}
func (f *contextFixture) WriteProjectContext(_ context.Context, session, id string) error {
	f.values[session] = id
	return nil
}
func (f *contextFixture) ResolveContextProject(_ context.Context, selector string) (string, string) {
	if selector == "stale" {
		return "", "project_not_found"
	}
	return selector, ""
}

func TestEffectiveProjectPrecedenceMatrix(t *testing.T) {
	for mask := 0; mask < 8; mask++ {
		f := &contextFixture{values: map[string]string{}}
		explicit, want, source, category := "", "", "unresolved", "project_context_unresolved"
		if mask&1 != 0 {
			f.values[""] = "default"
			want, source, category = "default", "default", ""
		}
		if mask&2 != 0 {
			f.values["one"] = "session"
			want, source, category = "session", "session", ""
		}
		if mask&4 != 0 {
			explicit = "explicit"
			want, source, category = "explicit", "explicit", ""
		}
		s := ProjectContext{Preferences: f, Resolver: f}
		view, got := s.Effective(WithProjectSession(context.Background(), "one"), explicit)
		if view.Effective != want || view.Source != source || got != category {
			t.Fatalf("mask=%d view=%+v category=%s", mask, view, got)
		}
	}
}

func TestEffectiveProjectSessionIsolationClearAndReset(t *testing.T) {
	f := &contextFixture{values: map[string]string{}}
	s := ProjectContext{Preferences: f, Resolver: f}
	ctx := context.Background()
	if s.Set(ctx, "", "default") != "" || s.Set(ctx, "one", "override") != "" {
		t.Fatal("set failed")
	}
	for _, session := range []string{"", "two"} {
		view, category := s.Effective(WithProjectSession(ctx, session), "")
		if view.Effective != "default" || category != "" {
			t.Fatal(view, category)
		}
	}
	if s.Clear(ctx, "one") != "" {
		t.Fatal("clear failed")
	}
	view, _ := s.Effective(WithProjectSession(ctx, "one"), "")
	if view.Effective != "default" || f.values[""] != "default" {
		t.Fatal("session changed default")
	}
	if s.Clear(ctx, "") != "" {
		t.Fatal("clear default failed")
	}
	if _, category := s.Effective(ctx, ""); category != "project_context_unresolved" {
		t.Fatal(category)
	}
}

func TestEffectiveProjectStaleWinningTierAndExplicitBypass(t *testing.T) {
	for _, tier := range []string{"", "one"} {
		f := &contextFixture{values: map[string]string{"": "default", tier: "stale"}}
		s := ProjectContext{Preferences: f, Resolver: f}
		ctx := WithProjectSession(context.Background(), "one")
		if _, category := s.Effective(ctx, ""); category != "project_not_found" {
			t.Fatal(category)
		}
		view, category := s.Effective(ctx, "explicit")
		if category != "" || view.Effective != "explicit" || f.values[tier] != "stale" {
			t.Fatal(view, category)
		}
		if _, category := s.Effective(ctx, "stale"); category != "project_not_found" {
			t.Fatal(category)
		}
	}
	f := &contextFixture{broken: true}
	s := ProjectContext{Preferences: f, Resolver: f}
	if _, category := s.Effective(context.Background(), ""); category != "invalid_project_context" {
		t.Fatal(category)
	}
	if view, category := s.Effective(context.Background(), "explicit"); category != "" || view.Effective != "explicit" {
		t.Fatal(view, category)
	}
}
