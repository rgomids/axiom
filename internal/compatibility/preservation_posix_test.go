//go:build unix

package compatibility

import "syscall"

var edquot error = syscall.EDQUOT
