package workflow

// OutputSubject is supplied by the fenced launcher, never by child content.
// A retained artifact must match every identity before it can satisfy output.
type OutputSubject struct {
	// Isolated selects bounded artifact-lock acquisition for enrolled attempts;
	// it is transient collection context, never persisted authority.
	Isolated                                                         bool
	ExecutionID, AttemptID, StageID, DefinitionDigest, RepositoryKey string
	Check                                                            OutputCheck
}
