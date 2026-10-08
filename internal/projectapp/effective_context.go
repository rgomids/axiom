package projectapp

import "context"

// Project preferences contain canonical identities only, outside portable state.
type ContextPreferences interface {
	ReadProjectContext(context.Context, string) (string, error)
	WriteProjectContext(context.Context, string, string) error
}

type ContextResolver interface {
	ResolveContextProject(context.Context, string) (string, string)
}

type EffectiveContext struct {
	Default   string `json:"default,omitempty"`
	Session   string `json:"session,omitempty"`
	Effective string `json:"effective,omitempty"`
	Source    string `json:"source"`
	Issue     string `json:"issue,omitempty"`
}

type ProjectContext struct {
	Preferences ContextPreferences
	Resolver    ContextResolver
}

type sessionKey struct{}

func ValidProjectSession(session string) bool {
	if len(session) > 128 {
		return false
	}
	for _, c := range session {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func WithProjectSession(ctx context.Context, session string) context.Context {
	return context.WithValue(ctx, sessionKey{}, session)
}

func ProjectSession(ctx context.Context) string {
	session, _ := ctx.Value(sessionKey{}).(string)
	return session
}

// Effective reads only the winning tier. A stale lower tier cannot defeat an
// explicit selector; a stale winning tier never falls back to another tier.
func (s ProjectContext) Effective(ctx context.Context, explicit string) (EffectiveContext, string) {
	view := EffectiveContext{Source: "explicit"}
	selector := explicit
	if selector == "" {
		if !ValidProjectSession(ProjectSession(ctx)) {
			return view, "invalid_project_session"
		}
		if session := ProjectSession(ctx); session != "" {
			id, err := s.Preferences.ReadProjectContext(ctx, session)
			if err != nil {
				return view, "invalid_project_context"
			}
			selector, view.Session, view.Source = id, id, "session"
		}
		if selector == "" {
			id, err := s.Preferences.ReadProjectContext(ctx, "")
			if err != nil {
				return view, "invalid_project_context"
			}
			selector, view.Default, view.Source = id, id, "default"
		}
	}
	if selector == "" {
		view.Source = "unresolved"
		return view, "project_context_unresolved"
	}
	id, category := s.Resolver.ResolveContextProject(ctx, selector)
	if category != "" {
		return view, category
	}
	view.Effective = id
	return view, ""
}

// Inspect discloses all tiers and validates every stored reference.
func (s ProjectContext) Inspect(ctx context.Context, explicit string) (EffectiveContext, string) {
	view, category := s.Effective(ctx, explicit)
	for _, session := range []string{"", ProjectSession(ctx)} {
		if session == "" && view.Default != "" || session != "" && view.Session != "" {
			continue
		}
		id, err := s.Preferences.ReadProjectContext(ctx, session)
		if err != nil {
			return view, "invalid_project_context"
		}
		if session == "" {
			view.Default = id
		} else {
			view.Session = id
		}
		if id != "" {
			if _, issue := s.Resolver.ResolveContextProject(ctx, id); issue != "" {
				return view, issue
			}
		}
	}
	return view, category
}

func (s ProjectContext) Set(ctx context.Context, session, selector string) string {
	if !ValidProjectSession(session) {
		return "invalid_project_session"
	}
	id, category := s.Resolver.ResolveContextProject(ctx, selector)
	if category != "" {
		return category
	}
	if err := s.Preferences.WriteProjectContext(ctx, session, id); err != nil {
		return "invalid_project_context"
	}
	return ""
}

func (s ProjectContext) Clear(ctx context.Context, session string) string {
	if !ValidProjectSession(session) {
		return "invalid_project_session"
	}
	if err := s.Preferences.WriteProjectContext(ctx, session, ""); err != nil {
		return "invalid_project_context"
	}
	return ""
}
