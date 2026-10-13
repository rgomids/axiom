package runtimeadapter

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/rgomids/axiom/internal/capabilityproof"
)

// InterfaceRunner is a bounded --version/--help port, never an inference or
// authentication port. No real runner is wired by this foundation.
type InterfaceRunner interface {
	Identity(context.Context, string) (string, error)
	Output(context.Context, string, []string, []string, int) (string, error)
}

func ObserveInterface(ctx context.Context, runner InterfaceRunner, executable, runtimeID, variant string, parent []string) (capabilityproof.InterfaceObservation, error) {
	unknown := capabilityproof.ParseInterface(runtimeID, variant, "", "")
	if runner == nil || !stagePath(executable) {
		return unknown, ErrInvalidAdapterConfiguration
	}
	env, err := StageEnvironment(runtimeID, parent)
	if err != nil {
		return unknown, err
	}
	before, err := runner.Identity(ctx, executable)
	identityBytes, identityError := hex.DecodeString(before)
	if err != nil || identityError != nil || len(identityBytes) != 32 {
		return unknown, ErrExecutableUnavailable
	}
	version, err := runner.Output(ctx, executable, []string{"--version"}, append([]string(nil), env...), 65536)
	if err != nil {
		return unknown, errors.New("interface version unavailable")
	}
	middle, err := runner.Identity(ctx, executable)
	if err != nil || middle != before {
		return unknown, ErrExecutableUnavailable
	}
	args := []string{"--help"}
	if runtimeID == "codex" {
		args = []string{"exec", "--help"}
	}
	help, err := runner.Output(ctx, executable, args, append([]string(nil), env...), 65536)
	if err != nil {
		return unknown, errors.New("interface help unavailable")
	}
	after, err := runner.Identity(ctx, executable)
	if err != nil || after != before {
		return unknown, ErrExecutableUnavailable
	}
	if len(version) > 65536 || len(help) > 65536 {
		return unknown, errors.New("interface output exceeds bound")
	}
	observation := capabilityproof.ParseInterface(runtimeID, variant, version, help)
	observation.ExecutableDigest = before
	return observation, nil
}
