package cli

// ExecutionTargetView contains only validated, presentation-safe identities.
// It is a preflight payload, not a second persisted Execution format.
type ExecutionTargetView struct {
	ProjectID     string `json:"projectId"`
	ProjectSource string `json:"projectSource"`
	RepositoryKey string `json:"repositoryKey"`
	Provider      string `json:"provider"`
	Resource      string `json:"resource"`
	ExternalID    string `json:"externalId"`
	WorkItem      string `json:"workItem"`
}

// ParseWorkItemSelector shares the existing exact CLI wire-selector parser with
// direct application callers. Domain/link validation stays in the application.
func ParseWorkItemSelector(value string) (string, string, string, bool) {
	return parseWorkItemSelector(value)
}
