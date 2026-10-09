package project

import (
	"encoding/hex"
	"regexp"
)

func (v *validation) workflowSelection(s State) {
	if s.WorkflowSelection.form == Absent {
		return
	}
	if s.SchemaVersion != 4 {
		v.add("workflowSelection", "unsupported_field")
		return
	}
	r := s.WorkflowSelection.value
	digest, e := hex.DecodeString(r.Digest)
	if s.WorkflowSelection.form != Present || !regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`).MatchString(r.WorkflowID) || r.Revision < 1 || r.Revision > 2147483647 || e != nil || len(digest) != 32 || hex.EncodeToString(digest) != r.Digest || (r.Source != "builtin" && r.Source != "project") {
		v.add("workflowSelection", "invalid_reference")
	}
}

// SelectWorkflow explicitly upgrades portable intent, retaining legacy fields
// and declaration presence. It does not resolve or publish the reference.
func (p Project) SelectWorkflow(r WorkflowSelection) (Project, []Issue) {
	s := p.State()
	if !p.valid {
		return Project{}, []Issue{{"project", "invalid_current_state"}}
	}
	if s.SchemaVersion == 1 {
		switch s.Runtime.form {
		case Present:
			s.Runtimes = Configured([]Runtime{s.Runtime.value})
		case NotConfigured:
			s.Runtimes = Unconfigured[[]Runtime]()
		}
		s.Runtime = Declaration[Runtime]{}
	}
	s.SchemaVersion = 4
	s.WorkflowSelection = Configured(r)
	return New(s)
}
