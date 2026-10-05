package cli

import (
	"errors"
	"strings"
)

// elaboratedSectionFlags transports Runtime proposals without relabeling them
// as user-authored. Section validation and authority remain in the domain.
type elaboratedSectionFlags map[string]string

func (e *elaboratedSectionFlags) String() string { return "" }
func (e *elaboratedSectionFlags) Set(value string) error {
	name, content, ok := strings.Cut(value, "=")
	if !ok || content == "" {
		return errors.New("invalid elaborated section")
	}
	switch name {
	case "problem", "desired_outcome", "context", "scope", "constraints", "non_goals", "acceptance_expectations":
	default:
		return errors.New("invalid elaborated section")
	}
	if *e == nil {
		*e = make(map[string]string)
	}
	if _, exists := (*e)[name]; exists {
		return errors.New("duplicate elaborated section")
	}
	(*e)[name] = content
	return nil
}
