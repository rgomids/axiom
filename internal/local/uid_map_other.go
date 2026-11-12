//go:build !linux

package local

func ancestorUIDMapping(uint32) string { return "unknown" }
