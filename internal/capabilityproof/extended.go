package capabilityproof

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

var ErrExtendedRoot = errors.New("extended probe requires an approved temp root")
var ErrProbeResidue = errors.New("extended probe residue or unverified removal")

// ExtendedPolicy is supplied by the application, never by Runtime output.
// Home and vendor homes must be explicitly supplied; no ambient environment is
// consulted. Standard probe authority cannot authorize this operation.
type ExtendedPolicy struct {
	TempDir, SystemTemp, Home string
	VendorHomes               []string
}
type ExtendedRoot struct {
	Kind, Path string
	Snapshot   ObjectSnapshot
}
type ExtendedPreview struct {
	Digest   string
	ProbeID  string
	Roots    []ExtendedRoot
	Filename string
	Limit    int
	Timeout  time.Duration
}
type ExtendedAuthority struct {
	Actor           string
	Local, External bool
	Preview         ExtendedPreview
}

// ExtendedEffects must resolve roots without symlinks and mutate only Filename
// using exclusive empty-file creation. Controls and canary reuse that one name;
// no chmod, directory creation, unrelated cleanup or network is permitted.
type ExtendedEffects interface {
	Inspect(context.Context, string) (ObjectSnapshot, error)
	Control(context.Context, ExtendedRoot, string) (Control, error)
	Canary(context.Context, ExtendedRoot, string) (CanaryAttempt, ProcessIdentity, error)
	RemoveVerify(context.Context, ExtendedRoot, string) error
}
type ExtendedService struct {
	Policy  ExtendedPolicy
	Effects ExtendedEffects
}
type ExtendedResult struct {
	Kind        string
	State       string
	Measurement NegativeMeasurement
}

func containedPath(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
func cleanAbsolute(path string) bool {
	return path != "" && filepath.IsAbs(path) && filepath.Clean(path) == path && !strings.ContainsAny(path, "\x00\r\n")
}
func (s ExtendedService) Preview(ctx context.Context, id string, kinds []string) (ExtendedPreview, error) {
	if err := ctx.Err(); err != nil {
		return ExtendedPreview{}, err
	}
	if s.Effects == nil || !safe(id) || len(kinds) == 0 || len(kinds) > 2 || !cleanAbsolute(s.Policy.Home) {
		return ExtendedPreview{}, ErrExtendedRoot
	}
	p := ExtendedPreview{ProbeID: id, Filename: ".axiom-probe-" + id, Limit: 1, Timeout: 30 * time.Second}
	seen := map[string]bool{}
	for _, kind := range kinds {
		root := ""
		switch kind {
		case "tmpdir":
			root = s.Policy.TempDir
		case "system-temp":
			root = s.Policy.SystemTemp
		default:
			return ExtendedPreview{}, ErrExtendedRoot
		}
		if !cleanAbsolute(root) || root == string(filepath.Separator) || seen[root] || containedPath(root, s.Policy.Home) || containedPath(s.Policy.Home, root) {
			return ExtendedPreview{}, ErrExtendedRoot
		}
		for _, home := range s.Policy.VendorHomes {
			if !cleanAbsolute(home) || containedPath(root, home) || containedPath(home, root) {
				return ExtendedPreview{}, ErrExtendedRoot
			}
		}
		snapshot, err := s.Effects.Inspect(ctx, root)
		if err != nil || !snapshot.SymlinkFree || snapshot.ResolvedTarget != kind {
			return ExtendedPreview{}, ErrExtendedRoot
		}
		seen[root] = true
		p.Roots = append(p.Roots, ExtendedRoot{kind, root, snapshot})
	}
	wire, err := json.Marshal(struct {
		Encoding string
		Preview  ExtendedPreview
	}{"axiom-extended-probe-preview-v1", p})
	if err != nil {
		return ExtendedPreview{}, err
	}
	p.Digest = packetDigest(wire)
	return p, nil
}

func (s ExtendedService) Apply(ctx context.Context, p ExtendedPreview, a ExtendedAuthority) ([]ExtendedResult, error) {
	if !safe(a.Actor) || !a.Local || !a.External || !reflect.DeepEqual(a.Preview, p) {
		return nil, ErrAuthority
	}
	kinds := make([]string, len(p.Roots))
	for i, root := range p.Roots {
		kinds[i] = root.Kind
	}
	fresh, err := s.Preview(ctx, p.ProbeID, kinds)
	if err != nil || !reflect.DeepEqual(fresh, p) {
		return nil, ErrStalePreview
	}
	cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 2*p.Timeout)
	defer cleanupCancel()
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	results := make([]ExtendedResult, 0, len(p.Roots))
	for _, root := range p.Roots {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		pre, preErr := s.Effects.Control(ctx, root, p.Filename)
		// Failed controls are inconclusive and do not authorize a sandbox claim.
		if preErr != nil || !pre.Created || !pre.Removed || !pre.RemovalVerified {
			cleanupErr := s.Effects.RemoveVerify(cleanupCtx, root, p.Filename)
			if !pre.Removed || !pre.RemovalVerified || cleanupErr != nil {
				return results, errors.Join(ErrProbeResidue, preErr, cleanupErr)
			}
			return results, errors.Join(ErrBlocked, preErr)
		}
		attempt, identity, runErr := s.Effects.Canary(ctx, root, p.Filename)
		// Removal and read-back are mandatory even on timeout/error, and before
		// post-control so the latter cannot hide a canary residue.
		if err := s.Effects.RemoveVerify(cleanupCtx, root, p.Filename); err != nil {
			return results, errors.Join(ErrProbeResidue, runErr, err)
		}
		post, postErr := s.Effects.Control(cleanupCtx, root, p.Filename)
		if err := s.Effects.RemoveVerify(cleanupCtx, root, p.Filename); err != nil {
			return results, errors.Join(ErrProbeResidue, err)
		}
		n := NegativeMeasurement{Target: root.Kind, Pre: pre, Post: post, Canary: attempt, Identity: identity, CanaryRan: runErr == nil}
		state := "unknown"
		stable := pre.Snapshot == root.Snapshot && post.Snapshot == root.Snapshot && pre.UID == post.UID && pre.GID == post.GID && identity.UID == pre.UID && identity.GID == pre.GID && safe(identity.Process) && post.Created && post.Removed && post.RemovalVerified && root.Snapshot.ACLReadable && digest.MatchString(root.Snapshot.ACLDigest) && !root.Snapshot.ReadOnly
		if runErr == nil && postErr == nil && stable && attempt.Target == root.Kind {
			if attempt.Created && attempt.Errno == "" {
				state = "advisory"
			} else if !attempt.Created && (attempt.Errno == "EACCES" || attempt.Errno == "EPERM" || attempt.Errno == "EROFS") {
				state = "denied"
			}
		}
		results = append(results, ExtendedResult{root.Kind, state, n})
		if runErr != nil || postErr != nil || ctx.Err() != nil {
			return results, errors.Join(runErr, postErr, ctx.Err())
		}
	}
	return results, nil
}
