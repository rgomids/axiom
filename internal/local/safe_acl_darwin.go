//go:build darwin && cgo

package local

/*
#include <sys/types.h>
#include <sys/acl.h>
#include <errno.h>

static int axiom_extended_acl_present(int fd) {
	errno = 0;
	acl_t acl = acl_get_fd_np(fd, ACL_TYPE_EXTENDED);
	if (acl == NULL) {
		return errno == ENOENT ? 0 : -1;
	}
	acl_free(acl);
	return 1;
}
*/
import "C"

import "os"

func checkPrivateACL(file *os.File) error {
	if C.axiom_extended_acl_present(C.int(file.Fd())) != 0 {
		return ErrUnsafe
	}
	return nil
}
