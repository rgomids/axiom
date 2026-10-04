//go:build !windows

package compatibility

import "syscall"

var edquot error = syscall.EDQUOT
