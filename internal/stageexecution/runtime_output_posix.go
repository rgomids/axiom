//go:build unix

package stageexecution

import "golang.org/x/sys/unix"

// Refuse a symlink swapped in after Lstat, and never block on a swapped FIFO.
const noFollowRuntimeOutput = unix.O_NOFOLLOW | unix.O_NONBLOCK

const runtimeOutputSupported = true
