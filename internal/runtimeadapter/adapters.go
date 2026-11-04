// Package runtimeadapter contains the two concrete S8 Runtime boundaries.
package runtimeadapter

import (
	"errors"
)

var ErrInvalidAdapterConfiguration = errors.New("invalid runtime adapter configuration")

type CommandProfile struct {
	RuntimeID, ModelProfileID string
	Executable                string
	Model                     string
	Arguments                 []string
	Environment               []string
	CredentialReference       string
	OutputMax                 int
}
