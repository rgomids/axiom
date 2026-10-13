//go:build !linux && !windows

package executiongraph

// Other POSIX platforms have no child subreaper: an escapee whose ancestry
// to the group leader is already gone cannot be observed. This is a recorded
// containment limitation; ancestry-visible escapes still fail closed.
func enableChildSubreaper() error { return nil }
