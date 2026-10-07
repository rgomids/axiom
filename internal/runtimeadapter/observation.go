package runtimeadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// IntegrationCapability is the only capability Lingo proves by itself: Axiom's
// own skill integration for that Runtime is installed and verified unchanged.
// Every other capability needs a proof source Lingo does not have, so the
// production observer leaves it unproven and resolution fails closed.
const IntegrationCapability = "axiom-skills"

const maxExecutableBytes = 1 << 30

var ErrExecutableUnavailable = errors.New("runtime executable unavailable")

// ExecutableIdentity is a presentation-safe identity of the regular executable
// file that path currently resolves to. It binds the resolved path and the
// exact content, so removal, replacement or another target fails or changes
// it. It never executes the file and never exposes the path.
func ExecutableIdentity(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", ErrExecutableUnavailable
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(resolved) {
		return "", ErrExecutableUnavailable
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.Mode().IsRegular() || !executableMode(info) || info.Size() > maxExecutableBytes {
		return "", ErrExecutableUnavailable
	}
	file, err := os.Open(resolved)
	if err != nil {
		return "", ErrExecutableUnavailable
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return "", ErrExecutableUnavailable
	}
	content := sha256.New()
	read, err := io.Copy(content, io.LimitReader(file, maxExecutableBytes+1))
	if err != nil || read != opened.Size() || read > maxExecutableBytes {
		return "", ErrExecutableUnavailable
	}
	identity := sha256.New()
	identity.Write([]byte("axiom-runtime-executable/v1\x00" + resolved + "\x00"))
	identity.Write(content.Sum(nil))
	return hex.EncodeToString(identity.Sum(nil)), nil
}

func executableMode(info os.FileInfo) bool {
	return runtime.GOOS == "windows" || info.Mode().Perm()&0o111 != 0
}

// IntegrationInspector reports, read-only, whether Axiom's skill integration
// for one Runtime is installed and verified unchanged.
type IntegrationInspector interface {
	IntegrationReady(context.Context) bool
}

// ObservedRuntime names the concrete executable a Runtime would dispatch.
// Executable stays internal; observations expose only its identity.
type ObservedRuntime struct {
	ID, Executable string
	Integration    IntegrationInspector
}

// ExecutableObserver derives observations from current machine-local state on
// every Observe. It never runs a Runtime, authenticates or reads credentials:
// installed/available mean the executable resolves to a regular runnable file,
// version stays unknown, and only IntegrationCapability can be proven.
type ExecutableObserver struct {
	revision   uint64
	observedAt time.Time
	runtimes   map[string]ObservedRuntime
}

func NewExecutableObserver(revision uint64, observedAt time.Time, runtimes []ObservedRuntime) (ExecutableObserver, error) {
	if revision == 0 || observedAt.IsZero() {
		return ExecutableObserver{}, ErrInvalidAdapterConfiguration
	}
	observer := ExecutableObserver{revision: revision, observedAt: observedAt.UTC(), runtimes: make(map[string]ObservedRuntime, len(runtimes))}
	for _, observed := range runtimes {
		if observed.ID != "codex" && observed.ID != "claude" || observed.Executable != "" && !filepath.IsAbs(observed.Executable) {
			return ExecutableObserver{}, ErrInvalidAdapterConfiguration
		}
		if _, exists := observer.runtimes[observed.ID]; exists {
			return ExecutableObserver{}, ErrInvalidAdapterConfiguration
		}
		observer.runtimes[observed.ID] = observed
	}
	return observer, nil
}

func (o ExecutableObserver) Observe(ctx context.Context, runtimeID string) (runtimeprofile.Observation, error) {
	observed, exists := o.runtimes[runtimeID]
	if !exists {
		return runtimeprofile.Observation{}, ErrInvalidAdapterConfiguration
	}
	if err := ctx.Err(); err != nil {
		return runtimeprofile.Observation{}, err
	}
	observation := runtimeprofile.Observation{RuntimeID: runtimeID, Adapter: runtimeID, Revision: o.revision, ObservedAt: o.observedAt, CapabilityStatus: map[string]runtimeprofile.CapabilityStatus{}}
	if observed.Executable == "" {
		return observation, nil
	}
	identity, err := ExecutableIdentity(observed.Executable)
	if err != nil {
		return observation, nil
	}
	observation.Installed, observation.Available, observation.ExecutableDigest = true, true, identity
	if observed.Integration != nil && observed.Integration.IntegrationReady(ctx) {
		observation.CapabilityStatus[IntegrationCapability] = runtimeprofile.CapabilityProven
	}
	return observation, nil
}
