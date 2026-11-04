package workflow

// OutputSubject is supplied by the fenced launcher, never by child content.
// A retained artifact must match every identity before it can satisfy output.
type OutputSubject struct {
	ExecutionID, AttemptID, StageID, DefinitionDigest, RepositoryKey string
	Check                                                            OutputCheck
}
