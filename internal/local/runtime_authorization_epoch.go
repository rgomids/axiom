package local

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/rgomids/axiom/internal/strictjson"
)

// Authorization epoch (Specification 007 AXM-22 PM-6). The internal record
// runtime-profiles/v1/authorization-epoch.json holds a persistent, monotonic
// counter that participates in every Runtime/Profile snapshot token and
// recovery plan identity. It lives beside, never inside, the configuration
// bytes, so restoring a prior configuration generation cannot restore it. It is
// advanced under the store's exclusive lock before every authorized
// publication reaches its commit point and before every scoped recovery acts,
// so a preview issued before a failed, rolled-back or recovered publication
// never becomes valid again. It is not a configuration field, Evidence, digest
// or authority, and no command reads or writes it directly.
const (
	authorizationEpochObject = "authorization-epoch.json"
	authorizationEpochSchema = "axiom-runtime-authorization-epoch/v1"
	authorizationEpochStage  = ".axiom-stage-epoch-"
	maxAuthorizationEpoch    = 256
	// faultEpoch is a test-only interruption point between the staged and the
	// committed epoch; it is never persisted in a protocol marker.
	faultEpoch FaultStage = "epoch"
)

type authorizationEpochWire struct {
	Schema string `json:"schema"`
	Epoch  uint64 `json:"epoch"`
}

func encodeAuthorizationEpoch(epoch uint64) []byte {
	wire, _ := json.Marshal(authorizationEpochWire{Schema: authorizationEpochSchema, Epoch: epoch})
	return append(wire, '\n')
}

// decodeAuthorizationEpoch accepts only the exact canonical bytes of a
// positive epoch; every other spelling is ambiguous and refused.
func decodeAuthorizationEpoch(wire []byte) (uint64, error) {
	var r authorizationEpochWire
	if len(wire) > maxAuthorizationEpoch || strictjson.Decode(wire, &r, maxAuthorizationEpoch) != nil || r.Schema != authorizationEpochSchema || r.Epoch == 0 || !bytes.Equal(wire, encodeAuthorizationEpoch(r.Epoch)) {
		return 0, ErrUnsafe
	}
	return r.Epoch, nil
}

// readAuthorizationEpoch reads the committed epoch below the locked version
// directory. An absent record is epoch 0 and present reports false; a
// symlinked, foreign, permissive, oversized or non-canonical record is unsafe.
func readAuthorizationEpoch(version *os.Root) (uint64, bool, error) {
	wire, err := readPrivateFileBounded(version, authorizationEpochObject, maxAuthorizationEpoch)
	if errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, ErrUnsafe
	}
	epoch, err := decodeAuthorizationEpoch(wire)
	return epoch, err == nil, err
}

func authorizationEpochStageName(name string) bool {
	return strings.HasPrefix(name, authorizationEpochStage) && len(name) > len(authorizationEpochStage)
}

// advanceAuthorizationEpoch durably publishes current+1. The caller holds the
// exclusive lock and has read current from the same descriptor. The staged
// candidate carries a protocol name, so an interruption leaves recovery state
// that every reader refuses until scoped recovery advances the epoch again.
// The epoch never wraps and is never decreased or reset.
func advanceAuthorizationEpoch(version *os.Root, current uint64, hooks publicationHooks) (uint64, error) {
	if current == math.MaxUint64 {
		return current, ErrConflict
	}
	next := current + 1
	stage, err := temporaryName(authorizationEpochStage)
	if err != nil {
		return current, err
	}
	if err := writePrivateFile(version, stage, encodeAuthorizationEpoch(next)); err != nil {
		_ = version.Remove(stage)
		return current, ErrRecoveryRequired
	}
	if err := hooks.at(faultEpoch); err != nil {
		if !errors.Is(err, ErrSimulatedInterruption) {
			_ = version.Remove(stage)
		}
		return current, ErrRecoveryRequired
	}
	if err := version.Rename(stage, authorizationEpochObject); err != nil {
		_ = version.Remove(stage)
		return current, ErrRecoveryRequired
	}
	if err := hooks.syncRoot(version); err != nil {
		return next, ErrRecoveryRequired
	}
	if observed, _, err := readAuthorizationEpoch(version); err != nil || observed != next {
		return next, ErrRecoveryRequired
	}
	return next, nil
}

func authorizationEpochFact(epoch uint64) string {
	return "\x00authorization-epoch:" + strconv.FormatUint(epoch, 10)
}
