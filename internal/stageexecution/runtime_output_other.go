//go:build !unix

package stageexecution

const noFollowRuntimeOutput = 0

// Until a no-follow file handle primitive is implemented, fail closed.
const runtimeOutputSupported = false
