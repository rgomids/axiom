package runtimeadapter

// SourceError retains a closed source class and an Is-compatible cause only.
// It never includes a source path, key or value in its message.
type SourceError struct {
	Source string
	Cause  error
}

func (e *SourceError) Error() string { return "runtime_configuration_refused" }
func (e *SourceError) Unwrap() error { return e.Cause }
