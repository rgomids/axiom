package runtimeadapter

import (
	"context"
	_ "embed"
	"slices"
	"strings"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/strictjson"
)

// VersionRegistrySchema is the only accepted registry wire version.
const VersionRegistrySchema = "axiom-runtime-version-registry/v1"

const (
	maxVersionRegistryBytes   = 64 << 10
	maxVersionRegistryEntries = 256
)

// runtimeVersionRegistry is the repository-owned registry built into Axiom
// (Specification 007 AXM-22 PM-5). It is never read from the filesystem, the
// environment or the network, and holds no entry for an executable identity
// that has not been independently verified and approved.
//
//go:embed runtime-versions.json
var runtimeVersionRegistry []byte

type versionRegistryWire struct {
	Schema  string             `json:"schema"`
	Entries []versionEntryWire `json:"entries"`
}

type versionEntryWire struct {
	RuntimeID        string `json:"runtimeId"`
	Version          string `json:"version"`
	ExecutableDigest string `json:"executableDigest"`
}

type versionKey struct{ runtimeID, executableDigest string }

// RegistryVersionObserver passively maps a Runtime ID and its canonical
// ExecutableIdentity to the one registered Runtime version. It never runs,
// probes or opens the executable; the caller observes the identity and
// revalidates it for drift. A registered version is identity only: it proves
// no Runtime, model, effort, subscription or Provider capability. Its
// constructor in production is EmbeddedVersionObserver; the zero value refuses
// every lookup.
type RegistryVersionObserver struct {
	versions map[versionKey]string
}

// EmbeddedVersionObserver decodes the registry built into Axiom. No caller can
// select another registry source.
func EmbeddedVersionObserver() (RegistryVersionObserver, error) {
	return decodeVersionRegistry(runtimeVersionRegistry)
}

// Version has the BindingSource Version and ObserveEffortApplicability
// signature. An absent, unknown or mismatched identity is
// capabilityproof.ErrBlocked; there is no fallback to a filename, PATH, a
// declaration, a Profile, Evidence or executable invocation.
func (o RegistryVersionObserver) Version(ctx context.Context, runtimeID, executableDigest string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	version, ok := o.versions[versionKey{runtimeID, executableDigest}]
	if !ok {
		return "", capabilityproof.ErrBlocked
	}
	return version, nil
}

// decodeVersionRegistry is strict and bounded: exact member names, no unknown
// or duplicate members, one supported schema, entries in canonical
// (runtimeId, executableDigest) order, and one mapping per executable identity.
func decodeVersionRegistry(wire []byte) (RegistryVersionObserver, error) {
	var none RegistryVersionObserver
	var r versionRegistryWire
	if strictjson.Decode(wire, &r, maxVersionRegistryBytes) != nil || r.Schema != VersionRegistrySchema || r.Entries == nil || len(r.Entries) > maxVersionRegistryEntries {
		return none, capabilityproof.ErrBlocked
	}
	versions := make(map[versionKey]string, len(r.Entries))
	digests := make(map[string]bool, len(r.Entries))
	for i, e := range r.Entries {
		if _, err := AdapterVersion(e.RuntimeID); err != nil || !runtimeprofile.SafeRuntimeVersion(e.Version) || !lowerHexDigest(e.ExecutableDigest) || digests[e.ExecutableDigest] {
			return none, capabilityproof.ErrBlocked
		}
		if i > 0 && compareEntries(r.Entries[i-1], e) >= 0 {
			return none, capabilityproof.ErrBlocked
		}
		digests[e.ExecutableDigest] = true
		versions[versionKey{e.RuntimeID, e.ExecutableDigest}] = e.Version
	}
	return RegistryVersionObserver{versions: versions}, nil
}

func compareEntries(a, b versionEntryWire) int {
	if c := strings.Compare(a.RuntimeID, b.RuntimeID); c != 0 {
		return c
	}
	return strings.Compare(a.ExecutableDigest, b.ExecutableDigest)
}

func lowerHexDigest(s string) bool {
	return len(s) == 64 && !slices.ContainsFunc([]byte(s), func(c byte) bool { return !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') })
}
