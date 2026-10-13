//go:build linux

package executiongraph

import (
	"strings"
	"testing"
)

func TestProcessIdentityKernelTicks(t *testing.T) {
	fields := make([]string, 21)
	for i := range fields {
		fields[i] = "0"
	}
	fields[0], fields[1], fields[2], fields[19] = "S", "1", "123", "456"
	stat := "123 (name with ) parens) " + strings.Join(fields, " ")
	p, err := parseProcessCorrelation(123, stat, "boot-one")
	if err != nil {
		t.Fatal(err)
	}
	if p.StartIdentity != "boot-one:456" || p.GroupIdentity != "boot-one:123" {
		t.Fatal(p)
	}
	fields[19] = "789"
	reused, err := parseProcessCorrelation(123, "123 (other) "+strings.Join(fields, " "), "boot-one")
	if err != nil || *p == *reused {
		t.Fatal(reused, err)
	}

	if _, err := parseProcessCorrelation(123, "malformed", "boot-one"); err == nil {
		t.Fatal("accepted malformed stat")
	}
}
