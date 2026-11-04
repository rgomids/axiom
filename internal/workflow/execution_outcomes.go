package workflow

import "slices"

// RepositoryOutcome is a read-only projection of canonical Execution facts.
// Ready stays false until a reader observes referenced content and targets.
// Blockers contain structural failures before that observation.
type RepositoryOutcome struct {
	ReadinessObserved bool                      `json:"readinessObserved"`
	RepositoryKey     string                    `json:"repositoryKey"`
	Results           []RepositoryAttemptResult `json:"results"`
	Integrations      []Integration             `json:"integrations"`
	Ready             bool                      `json:"ready"`
	Blockers          []string                  `json:"blockers"`
}

type RepositoryAttemptResult struct {
	StageID   string        `json:"stageId"`
	AgentID   string        `json:"agentId"`
	AttemptID string        `json:"attemptId"`
	Status    AttemptStatus `json:"status"`
	Current   bool          `json:"current"`
	Artifacts []ArtifactRef `json:"artifacts"`
}

// Claims reports whether an outcome row carries facts readiness must observe:
// the Work Item Repository always does, any other bound Repository only once
// an attempt or integration names it.
func (e Execution) Claims(o RepositoryOutcome) bool {
	return o.RepositoryKey == e.WorkItem.RepositoryKey || len(o.Results) != 0 || len(o.Integrations) != 0
}

// RepositoryOutcomes keeps historical attempts and Git provenance together
// without adding a persisted readiness boolean or another result owner.
func (e Execution) RepositoryOutcomes() []RepositoryOutcome {
	out := make([]RepositoryOutcome, 0, len(e.Repositories))
	valid := e.Validate() == nil
	for _, repo := range e.Repositories {
		o := RepositoryOutcome{RepositoryKey: repo.Key, Results: []RepositoryAttemptResult{}, Integrations: []Integration{}, Blockers: []string{}}
		if !valid {
			o.Blockers = append(o.Blockers, "execution_invalid")
			out = append(out, o)
			continue
		}
		if e.Status != RunCompleted || e.Cancel != nil {
			o.Blockers = append(o.Blockers, "execution_not_completed")
		}
		for _, stage := range e.Stages {
			for _, task := range stage.Tasks {
				for k, a := range task.Attempts {
					if a.Authority.RepositoryKey != repo.Key {
						continue
					}
					current := k == len(task.Attempts)-1
					o.Results = append(o.Results, RepositoryAttemptResult{StageID: stage.StageID, AgentID: task.AgentID, AttemptID: a.AttemptID, Status: a.Status, Current: current, Artifacts: append([]ArtifactRef{}, a.Artifacts...)})
					if current && a.Status != AttemptSucceeded {
						o.Blockers = append(o.Blockers, a.AttemptID+":"+string(a.Status))
					}
				}
			}
		}
		// Only the latest observation of each target governs current readiness;
		// all earlier outcomes remain visible, including failed/unknown attempts.
		latest := map[string]int{}
		for _, i := range e.Integrations {
			if i.RepositoryKey == repo.Key {
				i.Checks = slices.Clone(i.Checks)
				if i.EndedAt != nil {
					at := *i.EndedAt
					i.EndedAt = &at
				}
				if i.Resolution != nil {
					r := *i.Resolution
					i.Resolution = &r
				}
				latest[i.TargetRef] = len(o.Integrations)
				o.Integrations = append(o.Integrations, i)
			}
		}
		for k, i := range o.Integrations {
			if latest[i.TargetRef] == k && i.Status != IntegrationIntegrated {
				o.Blockers = append(o.Blockers, i.IntegrationID+":"+string(i.Status))
			}
		}
		if len(o.Results) == 0 && e.Claims(o) {
			o.Blockers = append(o.Blockers, "repository_result_missing")
		}
		out = append(out, o)
	}
	return out
}
