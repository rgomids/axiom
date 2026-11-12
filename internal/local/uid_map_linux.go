package local

import (
	"io"
	"os"
	"strconv"
	"strings"
)

func ancestorUIDMapping(uid uint32) string {
	file, err := os.Open("/proc/self/uid_map")
	if err != nil {
		return "unknown"
	}
	defer file.Close()
	wire, err := io.ReadAll(io.LimitReader(file, 65537))
	if err != nil || len(wire) > 65536 {
		return "unknown"
	}
	return uidMapping(wire, uid)
}

// uidMapping describes only visibility in this namespace, never host identity.
func uidMapping(wire []byte, uid uint32) string {
	lines := strings.Split(strings.TrimSpace(string(wire)), "\n")
	if len(lines) == 0 || len(lines) > 340 {
		return "unknown"
	}
	mapped := false
	var previousEnd uint64
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return "unknown"
		}
		var values [3]uint64
		for j, field := range fields {
			value, err := strconv.ParseUint(field, 10, 32)
			if err != nil {
				return "unknown"
			}
			values[j] = value
		}
		start, outside, count := values[0], values[1], values[2]
		end := start + count
		if count == 0 || end > 1<<32 || outside+count > 1<<32 || (i > 0 && start < previousEnd) {
			return "unknown"
		}
		previousEnd = end
		mapped = mapped || uint64(uid) >= start && uint64(uid) < end
	}
	if mapped {
		return "mapped"
	}
	return "unmapped"
}
