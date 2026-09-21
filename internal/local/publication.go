package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type FaultStage string

const (
	FaultF0 FaultStage = "F0"
	FaultF1 FaultStage = "F1"
	FaultF2 FaultStage = "F2"
	FaultF3 FaultStage = "F3"
	FaultF4 FaultStage = "F4"
	FaultF5 FaultStage = "F5"
	FaultF6 FaultStage = "F6"
	FaultF7 FaultStage = "F7"
	FaultF8 FaultStage = "F8"
)

var ErrSimulatedInterruption = errors.New("simulated process interruption")
var ErrCapacity = errors.New("local capacity exhausted")

type PublicationError struct {
	Stage     FaultStage
	Committed bool
	Err       error
}

func (e *PublicationError) Error() string {
	return fmt.Sprintf("publication %s committed=%t: %v", e.Stage, e.Committed, e.Err)
}

func (e *PublicationError) Unwrap() error { return e.Err }

type protocolMarker struct {
	FormatVersion int        `json:"formatVersion"`
	OperationID   string     `json:"operationId"`
	Object        string     `json:"object"`
	Stage         FaultStage `json:"stage"`
	PriorRevision string     `json:"priorRevision"`
	NewRevision   string     `json:"newRevision"`
	Staging       string     `json:"staging"`
}

type publicationHooks struct {
	fault func(FaultStage) error
}

func (h publicationHooks) at(stage FaultStage) error {
	if h.fault == nil {
		return nil
	}
	return h.fault(stage)
}

func protocolStatePresent(root *os.Root) (bool, error) {
	directory, err := root.Open(".")
	if err != nil {
		return false, err
	}
	names, err := directory.Readdirnames(-1)
	directory.Close()
	if err != nil {
		return false, err
	}
	for _, name := range names {
		if strings.HasPrefix(name, ".axiom-stage-") || strings.HasPrefix(name, ".axiom-recovery-") {
			return true, nil
		}
	}
	return false, nil
}

func writeProtocolMarker(root *os.Root, object, staging string, prior, next [32]byte) (string, error) {
	operation, err := temporaryName("")
	if err != nil {
		return "", err
	}
	name := ".axiom-recovery-" + operation
	marker := protocolMarker{1, operation, object, FaultF3, hex.EncodeToString(prior[:]), hex.EncodeToString(next[:]), staging}
	wire, err := json.Marshal(marker)
	if err != nil {
		return "", err
	}
	wire = append(wire, '\n')
	if len(wire) > 64<<10 {
		return "", ErrUnsafe
	}
	if err := writePrivateFile(root, name, wire); err != nil {
		return "", err
	}
	if err := syncRoot(root); err != nil {
		return name, ErrRecoveryRequired
	}
	return name, nil
}

func updateProtocolStage(root *os.Root, name string, marker protocolMarker, stage FaultStage) error {
	marker.Stage = stage
	wire, err := json.Marshal(marker)
	if err != nil {
		return err
	}
	wire = append(wire, '\n')
	temporary, err := temporaryName(".axiom-stage-marker-")
	if err != nil {
		return err
	}
	defer root.Remove(temporary)
	if err := writePrivateFile(root, temporary, wire); err != nil {
		return err
	}
	if err := root.Rename(temporary, name); err != nil {
		return err
	}
	return syncRoot(root)
}

func readProtocolMarker(root *os.Root, name string) (protocolMarker, error) {
	wire, err := readPrivateFileBounded(root, name, 64<<10)
	if err != nil {
		return protocolMarker{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(wire)))
	decoder.DisallowUnknownFields()
	var marker protocolMarker
	if err := decoder.Decode(&marker); err != nil || marker.FormatVersion != 1 || marker.OperationID == "" || marker.Object == "" || marker.Staging == "" {
		return protocolMarker{}, ErrRecoveryRequired
	}
	return marker, nil
}

func digestBytes(value []byte) [32]byte { return sha256.Sum256(value) }

func publicationFailure(stage FaultStage, committed bool, err error) error {
	if err == nil {
		return nil
	}
	return &PublicationError{Stage: stage, Committed: committed, Err: err}
}

func publishFile(ctx context.Context, root *os.Root, name string, expected, next []byte, create bool, hooks publicationHooks) error {
	if err := hooks.at(FaultF0); err != nil {
		return publicationFailure(FaultF0, false, err)
	}
	if pending, err := protocolStatePresent(root); err != nil || pending {
		if err != nil {
			return err
		}
		return ErrRecoveryRequired
	}
	current, readErr := readPrivateFile(root, name)
	if create {
		if readErr == nil {
			if bytes.Equal(current, next) {
				return nil
			}
			return ErrConflict
		}
		if !os.IsNotExist(readErr) {
			return readErr
		}
	} else {
		if readErr != nil {
			if os.IsNotExist(readErr) {
				return ErrNotFound
			}
			return readErr
		}
		if len(expected) == 0 || !bytes.Equal(current, expected) {
			return ErrConflict
		}
	}
	stage, err := temporaryName(".axiom-stage-file-")
	if err != nil {
		return err
	}
	removeStage := func() { _ = root.Remove(stage) }
	if err := hooks.at(FaultF1); err != nil {
		if !errors.Is(err, ErrSimulatedInterruption) {
			removeStage()
		}
		return publicationFailure(FaultF1, false, err)
	}
	if err := writePrivateFile(root, stage, next); err != nil {
		removeStage()
		return publicationFailure(FaultF1, false, err)
	}
	if err := hooks.at(FaultF2); err != nil {
		if !errors.Is(err, ErrSimulatedInterruption) {
			removeStage()
		}
		return publicationFailure(FaultF2, false, err)
	}
	if err := verifyPreparedFile(root, stage, next); err != nil {
		removeStage()
		return publicationFailure(FaultF2, false, err)
	}
	markerName, err := writeProtocolMarker(root, name, stage, digestBytes(current), digestBytes(next))
	if err != nil {
		return publicationFailure(FaultF3, false, err)
	}
	marker, err := readProtocolMarker(root, markerName)
	if err != nil {
		return publicationFailure(FaultF3, false, err)
	}
	cleanupPreCommit := func() { _ = root.Remove(markerName); removeStage(); _ = syncRoot(root) }
	if err := hooks.at(FaultF3); err != nil {
		if !errors.Is(err, ErrSimulatedInterruption) {
			cleanupPreCommit()
		}
		return publicationFailure(FaultF3, false, err)
	}
	if err := ctx.Err(); err != nil {
		cleanupPreCommit()
		return publicationFailure(FaultF3, false, err)
	}
	if !create {
		observed, err := readPrivateFile(root, name)
		if err != nil || !bytes.Equal(observed, expected) {
			cleanupPreCommit()
			return publicationFailure(FaultF3, false, ErrConflict)
		}
	}
	if err := hooks.at(FaultF4); err != nil {
		if !errors.Is(err, ErrSimulatedInterruption) {
			cleanupPreCommit()
		}
		return publicationFailure(FaultF4, false, err)
	}
	if err := updateProtocolStage(root, markerName, marker, FaultF5); err != nil {
		return publicationFailure(FaultF4, false, ErrRecoveryRequired)
	}
	if err := hooks.at(FaultF5); err != nil {
		if !errors.Is(err, ErrSimulatedInterruption) {
			cleanupPreCommit()
		}
		return publicationFailure(FaultF5, false, err)
	}
	if create {
		err = renameNoReplace(root, stage, name)
	} else {
		err = root.Rename(stage, name)
	}
	if err != nil {
		cleanupPreCommit()
		return publicationFailure(FaultF5, false, err)
	}
	if err := updateProtocolStage(root, markerName, marker, FaultF6); err != nil {
		return publicationFailure(FaultF6, true, ErrRecoveryRequired)
	}
	if err := hooks.at(FaultF6); err != nil {
		return publicationFailure(FaultF6, true, ErrRecoveryRequired)
	}
	confirmed, err := readPrivateFile(root, name)
	if err != nil || !bytes.Equal(confirmed, next) {
		return publicationFailure(FaultF6, true, ErrRecoveryRequired)
	}
	if err := syncRoot(root); err != nil {
		return publicationFailure(FaultF6, true, ErrRecoveryRequired)
	}
	if err := updateProtocolStage(root, markerName, marker, FaultF7); err != nil {
		return publicationFailure(FaultF7, true, ErrRecoveryRequired)
	}
	if err := hooks.at(FaultF7); err != nil {
		return publicationFailure(FaultF7, true, err)
	}
	if err := updateProtocolStage(root, markerName, marker, FaultF8); err != nil {
		return publicationFailure(FaultF8, true, ErrRecoveryRequired)
	}
	if err := hooks.at(FaultF8); err != nil {
		return publicationFailure(FaultF8, true, ErrRecoveryRequired)
	}
	if err := root.Remove(markerName); err != nil {
		return publicationFailure(FaultF8, true, ErrRecoveryRequired)
	}
	if err := syncRoot(root); err != nil {
		return publicationFailure(FaultF8, true, ErrRecoveryRequired)
	}
	return nil
}

func readPublishedFile(root *os.Root, name string) ([]byte, error) {
	pending, err := protocolStatePresent(root)
	if err != nil {
		return nil, err
	}
	if pending {
		return nil, ErrRecoveryRequired
	}
	return readPrivateFile(root, name)
}
