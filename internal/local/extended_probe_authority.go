//go:build unix

package local

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/executiongraph"
)

const extendedRootPermitName = ".axiom-recovery-probe-extended"

type extendedRootPermit struct {
	Version        int                               `json:"version"`
	ProbeID        string                            `json:"probeId"`
	ManifestDigest string                            `json:"manifestDigest"`
	Owner          executiongraph.ProcessCorrelation `json:"owner"`
	IssuedAt       time.Time                         `json:"issuedAt"`
	ExpiresAt      time.Time                         `json:"expiresAt"`
	Preview        capabilityproof.ExtendedPreview   `json:"preview"`
}

func ExtendedProbeService(kinds []string) (capabilityproof.ExtendedService, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return capabilityproof.ExtendedService{}, err
	}
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		return capabilityproof.ExtendedService{}, err
	}
	temp, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return capabilityproof.ExtendedService{}, err
	}
	system, err := filepath.EvalSymlinks("/tmp")
	if err != nil {
		return capabilityproof.ExtendedService{}, err
	}
	policy := capabilityproof.ExtendedPolicy{TempDir: temp, SystemTemp: system, Home: home, VendorHomes: []string{filepath.Join(home, ".codex"), filepath.Join(home, ".claude")}}
	for _, key := range []string{"CODEX_HOME", "CLAUDE_CONFIG_DIR"} {
		if path := os.Getenv(key); path != "" {
			policy.VendorHomes = append(policy.VendorHomes, path)
		}
	}
	for i, path := range policy.VendorHomes {
		if physical, err := filepath.EvalSymlinks(path); err == nil {
			policy.VendorHomes[i] = physical
		} else if !os.IsNotExist(err) {
			return capabilityproof.ExtendedService{}, err
		}
	}
	observer := ExtendedRootObserver{}
	for _, kind := range kinds {
		switch kind {
		case "tmpdir":
			observer.TempDir = temp
		case "system-temp":
			observer.SystemTemp = system
		default:
			return capabilityproof.ExtendedService{}, capabilityproof.ErrExtendedRoot
		}
	}
	return capabilityproof.ExtendedService{Policy: policy, Inspector: observer}, nil
}

func validateExtendedPreview(ctx context.Context, preview capabilityproof.ExtendedPreview) error {
	if len(preview.Roots) == 0 || len(preview.Roots) > 2 {
		return capabilityproof.ErrExtendedRoot
	}
	kinds := make([]string, len(preview.Roots))
	for i, root := range preview.Roots {
		kinds[i] = root.Kind
	}
	service, err := ExtendedProbeService(kinds)
	if err != nil {
		return err
	}
	fresh, err := service.Preview(ctx, preview.ProbeID, kinds)
	if err != nil || !reflect.DeepEqual(fresh, preview) {
		return capabilityproof.ErrStalePreview
	}
	return nil
}

func validateLiveProbeSession(session *probeSession, probeID, manifestDigest string) error {
	if session == nil || session.envelope == nil || session.listener == nil || session.done == nil {
		return ErrUnsafeEnvelope
	}
	select {
	case <-session.done:
		return ErrUnsafeEnvelope
	default:
	}
	e := session.envelope
	e.mu.Lock()
	claimed := e.prepared && e.claimed && e.manifest.Phase == "ready" && e.manifest.ProbeID == probeID && e.digest == manifestDigest
	e.mu.Unlock()
	if !claimed || session.launch.ProbeID != probeID || session.launch.ManifestDigest != manifestDigest {
		return ErrUnsafeEnvelope
	}
	if err := e.InspectReady(); err != nil {
		return err
	}
	directory, err := OpenOwnedDirectory(e.path)
	if err != nil {
		return err
	}
	defer directory.Close()
	claim, err := directory.ReadFile(probeClaimName, 128)
	if err != nil || string(claim) != manifestDigest {
		return ErrUnsafeEnvelope
	}
	inbox, err := OpenOwnedDirectory(filepath.Join(e.path, "inbox"))
	if err != nil {
		return ErrUnsafeEnvelope
	}
	defer inbox.Close()
	wire, err := inbox.ReadFile("probe-run.json", 64<<10)
	if err != nil {
		return ErrUnsafeEnvelope
	}
	var launch probeCanaryLaunch
	if strictEnvelopeJSON(wire, &launch) != nil || !reflect.DeepEqual(launch, session.launch) {
		return ErrUnsafeEnvelope
	}
	return nil
}

func BindExtendedRootPermit(ctx context.Context, envelope capabilityproof.ProbeEnvelope, preview capabilityproof.ExtendedPreview) error {
	session, ok := envelope.(*probeSession)
	if !ok || session == nil {
		return ErrUnsafeEnvelope
	}
	if err := validateLiveProbeSession(session, preview.ProbeID, session.launch.ManifestDigest); err != nil {
		return err
	}
	if err := validateExtendedPreview(ctx, preview); err != nil {
		return err
	}
	owner, err := executiongraph.ObserveProcessCorrelation(int64(os.Getpid()))
	if err != nil || owner == nil {
		return ErrUnsafeEnvelope
	}
	issuedAt := time.Now().UTC()
	permit := extendedRootPermit{Version: 1, ProbeID: preview.ProbeID, ManifestDigest: session.launch.ManifestDigest, Owner: *owner, IssuedAt: issuedAt, ExpiresAt: issuedAt.Add(preview.Timeout), Preview: preview}
	wire, err := json.Marshal(permit)
	if err != nil || len(wire) > 16<<10 {
		return capabilityproof.ErrExtendedRoot
	}
	directory, err := OpenOwnedDirectory(session.envelope.path)
	if err != nil {
		return err
	}
	defer directory.Close()
	if _, err := directory.Lstat(extendedRootPermitName); !os.IsNotExist(err) {
		return ErrUnsafeEnvelope
	}
	if err := directory.CreateExclusive(extendedRootPermitName, wire, 0600); err != nil {
		return errors.Join(ErrUnsafeEnvelope, err)
	}
	if err := directory.Sync(); err != nil {
		return err
	}
	return validateLiveProbeSession(session, preview.ProbeID, session.launch.ManifestDigest)
}

func readExtendedRootPermit(cwd, id, digest, path string) (capabilityproof.ExtendedRoot, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return capabilityproof.ExtendedRoot{}, capabilityproof.ErrCanaryContext
	}
	if _, _, err := (ProbeCanaryLoader{WorkingDirectory: cwd}).Load(context.Background(), id, digest); err != nil {
		return capabilityproof.ExtendedRoot{}, capabilityproof.ErrCanaryContext
	}
	directory, err := OpenOwnedDirectory(filepath.Dir(cwd))
	if err != nil {
		return capabilityproof.ExtendedRoot{}, capabilityproof.ErrCanaryContext
	}
	defer directory.Close()
	permitWire, err := directory.ReadFile(extendedRootPermitName, 16<<10)
	var permit extendedRootPermit
	if err != nil || strictEnvelopeJSON(permitWire, &permit) != nil || permit.Version != 1 || permit.ProbeID != id || permit.ManifestDigest != digest || validateExtendedPreview(context.Background(), permit.Preview) != nil || permit.Preview.ProbeID != id || permit.Preview.Filename != ".axiom-probe-"+id || !validExtendedPermitOwner(permit) {
		return capabilityproof.ExtendedRoot{}, capabilityproof.ErrExtendedRoot
	}
	for _, root := range permit.Preview.Roots {
		if path == filepath.Join(root.Path, permit.Preview.Filename) {
			return root, nil
		}
	}
	return capabilityproof.ExtendedRoot{}, capabilityproof.ErrExtendedRoot
}

func validExtendedPermitOwner(permit extendedRootPermit) bool {
	now := time.Now()
	if now.Before(permit.IssuedAt) || permit.Owner.PID <= 0 || permit.Owner.StartIdentity == "" || permit.Owner.GroupIdentity == "" || permit.IssuedAt.IsZero() || permit.ExpiresAt.IsZero() || permit.Preview.Timeout <= 0 || !permit.ExpiresAt.Equal(permit.IssuedAt.Add(permit.Preview.Timeout)) || !now.Before(permit.ExpiresAt) {
		return false
	}
	current, err := executiongraph.ObserveRecordedProcess(permit.Owner)
	return err == nil && current != nil && *current == permit.Owner
}
