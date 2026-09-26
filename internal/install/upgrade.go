// Package install owns the exact, authorized upgrade of an Axiom-owned
// release installation (binary, receipt) with explicit partial truth.
package install

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rgomids/axiom/internal/codexruntime"
	"github.com/rgomids/axiom/internal/compatibility"
	"github.com/rgomids/axiom/internal/local"
	"golang.org/x/sys/unix"
)

const (
	binaryName      = "lingo"
	receiptName     = "installation.receipt"
	lockName        = ".axiom-install.lock"
	markerName      = ".axiom-install-operation"
	binaryStage     = ".axiom-lingo-stage."
	receiptStage    = ".axiom-install-receipt."
	maxReceiptBytes = 64 << 10
)

const (
	SkillsNotConfigured  = "not_configured"
	SkillsMatch          = "matches"
	SkillsRequireInstall = "requires_runtime_install"
)

var (
	receiptFields    = []string{"formatVersion", "destination", "sha256", "product", "version", "revision", "sourceState", "release", "platform", "goos", "architecture", "skillSetVersion", "archiveSha256", "skillManifestSha256", "installedAt"}
	installedPattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$`)
)

// Error carries a fixed presentation-safe category; it never wraps paths or
// operating-system text for display.
type Error struct{ Category string }

func (e *Error) Error() string { return "upgrade: " + e.Category }

// Target names the explicit owned roots. SkillsRoot and State are inspected
// read-only for compatibility; they are never written by the upgrade.
type Target struct {
	BinaryDir  string
	ReceiptDir string
	SkillsRoot string
	State      compatibility.Roots
}

type Effect struct {
	Kind     string `json:"kind"`
	Target   string `json:"target"`
	Expected string `json:"expected"`
	Next     string `json:"next"`
}

type Preview struct {
	SourceVersion  string                       `json:"sourceVersion"`
	TargetVersion  string                       `json:"targetVersion"`
	ArchiveSHA256  string                       `json:"archiveSha256"`
	Resume         bool                         `json:"resume"`
	Effects        []Effect                     `json:"effects"`
	Leftovers      []string                     `json:"leftovers"`
	State          compatibility.Classification `json:"stateClassification"`
	StateDigest    string                       `json:"stateDigest"`
	Skills         string                       `json:"skills"`
	RequiredBytes  int64                        `json:"requiredBytes"`
	AvailableBytes uint64                       `json:"availableBytes"`
	Digest         string                       `json:"digest"`
	target         Target
	candidate      Candidate
	currentReceipt []byte
	nextReceipt    []byte
}

type Authority struct{ digest string }

type LedgerEntry struct {
	Kind      string `json:"kind"`
	Target    string `json:"target"`
	Revision  string `json:"revision"`
	Confirmed bool   `json:"confirmed"`
}

type Result struct {
	Status string        `json:"status"`
	Ledger []LedgerEntry `json:"ledger"`
	Skills string        `json:"skills"`
}

// Service holds deterministic seams; production uses NewService.
type Service struct {
	afterEffect    func(string) error
	availableSpace func(string) (uint64, error)
}

func NewService() Service { return Service{} }

func (s Service) Preview(ctx context.Context, target Target, candidate Candidate) (Preview, error) {
	return s.preview(ctx, target, candidate, false)
}

func (s Service) preview(ctx context.Context, target Target, candidate Candidate, lockHeld bool) (Preview, error) {
	if err := ctx.Err(); err != nil {
		return Preview{}, err
	}
	for _, directory := range []string{target.BinaryDir, target.ReceiptDir} {
		if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || directory == string(filepath.Separator) {
			return Preview{}, &Error{Category: "invalid_target"}
		}
		if local.CheckPrivateDirectory(directory) != nil {
			return Preview{}, &Error{Category: "unsafe_target"}
		}
	}
	if len(candidate.Binary) == 0 || candidate.ArchiveSHA256 == "" || candidate.Values == nil {
		return Preview{}, &Error{Category: "invalid_candidate"}
	}
	row := candidate.Values["platform"] + ":" + candidate.Values["goos"] + ":" + candidate.Values["architecture"]
	if hostRow() != row {
		return Preview{}, &Error{Category: "unsupported_host"}
	}
	if !lockHeld {
		if _, err := os.Lstat(filepath.Join(target.ReceiptDir, lockName)); !os.IsNotExist(err) {
			return Preview{}, &Error{Category: "installation_busy_or_interrupted"}
		}
	}
	preview := Preview{TargetVersion: candidate.Version, ArchiveSHA256: candidate.ArchiveSHA256, Effects: []Effect{}, Leftovers: []string{}, target: target, candidate: candidate}
	if marker, err := readMarker(target.ReceiptDir); err == nil {
		if marker["archiveSha256"] != candidate.ArchiveSHA256 || marker["operation"] != "upgrade" {
			return Preview{}, &Error{Category: "recovery_required"}
		}
		preview.Resume = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return Preview{}, &Error{Category: "recovery_required"}
	}
	currentReceipt, err := local.ReadOwnedFile(target.ReceiptDir, receiptName, maxReceiptBytes)
	if err != nil {
		return Preview{}, &Error{Category: "owned_receipt_required"}
	}
	values, err := parseReceipt(currentReceipt)
	destination := filepath.Join(target.BinaryDir, binaryName)
	if err != nil || values["destination"] != destination {
		return Preview{}, &Error{Category: "receipt_invalid"}
	}
	binary, err := local.ReadOwnedFile(target.BinaryDir, binaryName, maxBinaryBytes)
	if err != nil {
		return Preview{}, &Error{Category: "unsafe_binary"}
	}
	currentDigest, candidateDigest := digest(binary), digest(candidate.Binary)
	if currentDigest != values["sha256"] && !(preview.Resume && currentDigest == candidateDigest) {
		return Preview{}, &Error{Category: "binary_modified"}
	}
	preview.currentReceipt = currentReceipt
	preview.nextReceipt = expectedReceipt(destination, candidate, values["installedAt"])
	receiptAlreadyNext := bytes.Equal(currentReceipt, preview.nextReceipt)
	preview.SourceVersion = values["version"]
	comparison := compareSemver(candidate.Version, values["version"])
	switch {
	case receiptAlreadyNext:
	case comparison < 0:
		return Preview{}, &Error{Category: "downgrade_refused"}
	case comparison == 0 && candidateDigest != values["sha256"]:
		return Preview{}, &Error{Category: "divergent_equivalent_version"}
	}
	state, err := compatibility.Inspect(ctx, target.State)
	if err != nil {
		return Preview{}, &Error{Category: "state_inspection_failed"}
	}
	preview.State, preview.StateDigest = state.Classification, state.Digest
	if state.Classification != compatibility.AbsentV1 && state.Classification != compatibility.ValidV1 {
		return preview, &Error{Category: "state_incompatible"}
	}
	if currentDigest != candidateDigest {
		preview.Effects = append(preview.Effects, Effect{Kind: "binary", Target: destination, Expected: currentDigest, Next: candidateDigest})
		preview.RequiredBytes += int64(len(candidate.Binary))
	}
	if !receiptAlreadyNext {
		preview.Effects = append(preview.Effects, Effect{Kind: "receipt", Target: filepath.Join(target.ReceiptDir, receiptName), Expected: digest(currentReceipt), Next: digest(preview.nextReceipt)})
		preview.RequiredBytes += int64(len(preview.nextReceipt))
	}
	if preview.Resume {
		preview.Leftovers = stageLeftovers(target)
	}
	space := s.availableSpace
	if space == nil {
		space = statfsAvailable
	}
	if len(preview.Effects) != 0 {
		available, err := space(target.BinaryDir)
		if err != nil || uint64(preview.RequiredBytes)+maxReceiptBytes > available {
			return preview, &Error{Category: "insufficient_space"}
		}
		preview.AvailableBytes = available
	}
	preview.Skills, err = skillCompatibility(ctx, target.SkillsRoot, candidate)
	if err != nil {
		return preview, &Error{Category: "skill_inspection_failed"}
	}
	preview.Digest = previewDigest(preview)
	return preview, nil
}

func previewDigest(preview Preview) string {
	wire, _ := json.Marshal(struct {
		Source, Target, Archive string
		Resume                  bool
		Effects                 []Effect
		Leftovers               []string
		State                   compatibility.Classification
		StateDigest, Skills     string
	}{preview.SourceVersion, preview.TargetVersion, preview.ArchiveSHA256, preview.Resume, preview.Effects, preview.Leftovers, preview.State, preview.StateDigest, preview.Skills})
	return digest(wire)
}

func Authorize(preview Preview, reviewedDigest string) (Authority, error) {
	if len(preview.Effects) == 0 && len(preview.Leftovers) == 0 || preview.Digest == "" || preview.Digest != reviewedDigest {
		return Authority{}, &Error{Category: "authority_denied"}
	}
	return Authority{digest: preview.Digest}, nil
}

// Apply publishes each effect separately; there is no cross-root transaction.
// A later failure is reported as partial with every confirmed effect listed,
// and the owned operation marker keeps the installation resumable.
func (s Service) Apply(ctx context.Context, preview Preview, authority Authority) (Result, error) {
	result := Result{Status: "denied_authority", Ledger: []LedgerEntry{}}
	if authority.digest == "" || authority.digest != preview.Digest {
		return result, &Error{Category: "authority_denied"}
	}
	lock := filepath.Join(preview.target.ReceiptDir, lockName)
	if err := os.Mkdir(lock, 0o700); err != nil {
		result.Status = "failure"
		return result, &Error{Category: "installation_busy_or_interrupted"}
	}
	defer os.Remove(lock)
	current, err := s.preview(ctx, preview.target, preview.candidate, true)
	if err != nil || current.Digest != preview.Digest {
		return result, &Error{Category: "authority_denied"}
	}
	result.Status = "failure"
	if !current.Resume {
		if err := writeMarker(current.target.ReceiptDir, current.candidate.ArchiveSHA256, "prepare", true); err != nil {
			return result, &Error{Category: "marker_unavailable"}
		}
	}
	for _, effect := range current.Effects {
		if err := ctx.Err(); err != nil {
			return partial(result), err
		}
		var err error
		switch effect.Kind {
		case "binary":
			err = publishEffect(current.target.BinaryDir, binaryName, binaryStage, current.candidate.Binary, effect, 0o700, maxBinaryBytes)
		case "receipt":
			err = publishEffect(current.target.ReceiptDir, receiptName, receiptStage, current.nextReceipt, effect, 0o600, maxReceiptBytes)
		default:
			err = &Error{Category: "invalid_effect"}
		}
		if err != nil {
			return partial(result), err
		}
		result.Ledger = append(result.Ledger, LedgerEntry{Kind: effect.Kind, Target: effect.Target, Revision: effect.Next, Confirmed: true})
		if effect.Kind == "binary" {
			if err := writeMarker(current.target.ReceiptDir, current.candidate.ArchiveSHA256, "binary_committed", false); err != nil {
				return partial(result), &Error{Category: "marker_unavailable"}
			}
		}
		if s.afterEffect != nil {
			if err := s.afterEffect(effect.Kind); err != nil {
				return partial(result), err
			}
		}
	}
	for _, leftover := range current.Leftovers {
		_ = os.Remove(leftover)
	}
	if err := os.Remove(filepath.Join(current.target.ReceiptDir, markerName)); err != nil && !os.IsNotExist(err) {
		return partial(result), &Error{Category: "marker_cleanup_failed"}
	}
	syncDirectory(current.target.ReceiptDir)
	final, err := compatibility.Inspect(ctx, current.target.State)
	if err != nil || final.Classification != compatibility.AbsentV1 && final.Classification != compatibility.ValidV1 {
		return partial(result), &Error{Category: "final_verification_failed"}
	}
	result.Skills, err = skillCompatibility(ctx, current.target.SkillsRoot, current.candidate)
	if err != nil {
		return partial(result), &Error{Category: "final_verification_failed"}
	}
	result.Status = "success"
	if result.Skills == SkillsRequireInstall {
		result.Status = "partial"
	}
	return result, nil
}

func partial(result Result) Result {
	if len(result.Ledger) > 0 {
		result.Status = "partial"
	}
	return result
}

// publishEffect stages private bytes beside the target, rechecks the exact
// expected current revision, renames, and confirms the published revision.
func publishEffect(directory, name, prefix string, wire []byte, effect Effect, mode os.FileMode, limit int) error {
	var entropy [8]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return err
	}
	stage := filepath.Join(directory, prefix+hex.EncodeToString(entropy[:]))
	file, err := os.OpenFile(stage, os.O_WRONLY|os.O_CREATE|os.O_EXCL|unix.O_NOFOLLOW, mode)
	if err != nil {
		return &Error{Category: "stage_unavailable"}
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(stage)
		}
	}()
	written, writeErr := file.Write(wire)
	chmodErr := file.Chmod(mode)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || chmodErr != nil || syncErr != nil || closeErr != nil || written != len(wire) {
		return &Error{Category: "stage_write_failed"}
	}
	staged, err := local.ReadOwnedFile(directory, filepath.Base(stage), limit)
	if err != nil || digest(staged) != effect.Next {
		return &Error{Category: "stage_verification_failed"}
	}
	current, err := local.ReadOwnedFile(directory, name, limit)
	if err != nil || digest(current) != effect.Expected {
		return &Error{Category: "target_changed"}
	}
	if err := os.Rename(stage, filepath.Join(directory, name)); err != nil {
		return &Error{Category: "publication_failed"}
	}
	committed = true
	syncDirectory(directory)
	confirmed, err := local.ReadOwnedFile(directory, name, limit)
	if err != nil || digest(confirmed) != effect.Next {
		return &Error{Category: "publication_uncertain"}
	}
	return nil
}

func skillCompatibility(ctx context.Context, root string, candidate Candidate) (string, error) {
	if root == "" {
		return SkillsNotConfigured, nil
	}
	service, err := codexruntime.New(root)
	if err != nil {
		return "", err
	}
	inventory, err := service.Inventory(ctx)
	if err != nil {
		return "", err
	}
	if inventory.State == codexruntime.SkillSetAbsent {
		return SkillsNotConfigured, nil
	}
	for _, skill := range inventory.Skills {
		if skill.SHA256 == "" || skill.SHA256 != candidate.Skills[skill.Name] {
			return SkillsRequireInstall, nil
		}
	}
	return SkillsMatch, nil
}

func expectedReceipt(destination string, candidate Candidate, installedAt string) []byte {
	var builder strings.Builder
	builder.WriteString("formatVersion=1\n")
	builder.WriteString("destination=" + destination + "\n")
	builder.WriteString("sha256=" + digest(candidate.Binary) + "\n")
	for _, line := range candidate.Metadata {
		if !strings.HasPrefix(line, "formatVersion=") {
			builder.WriteString(line + "\n")
		}
	}
	builder.WriteString("archiveSha256=" + candidate.ArchiveSHA256 + "\n")
	builder.WriteString("skillManifestSha256=" + candidate.SkillManifestSHA256 + "\n")
	builder.WriteString("installedAt=" + installedAt + "\n")
	return []byte(builder.String())
}

func parseReceipt(wire []byte) (map[string]string, error) {
	lines := strings.Split(strings.TrimSuffix(string(wire), "\n"), "\n")
	if len(lines) != len(receiptFields) || !strings.HasSuffix(string(wire), "\n") {
		return nil, errors.New("receipt schema")
	}
	allowed := map[string]bool{}
	for _, field := range receiptFields {
		allowed[field] = true
	}
	values := map[string]string{}
	for _, line := range lines {
		key, value, ok := strings.Cut(line, "=")
		if !ok || !allowed[key] || value == "" || values[key] != "" {
			return nil, errors.New("receipt schema")
		}
		values[key] = value
	}
	if values["formatVersion"] != "1" || values["product"] != "Axiom" || !digestPattern.MatchString(values["sha256"]) || !semverPattern.MatchString(values["version"]) || !installedPattern.MatchString(values["installedAt"]) {
		return nil, errors.New("unsupported receipt")
	}
	return values, nil
}

func readMarker(directory string) (map[string]string, error) {
	if _, err := os.Lstat(filepath.Join(directory, markerName)); os.IsNotExist(err) {
		return nil, os.ErrNotExist
	}
	wire, err := local.ReadOwnedFile(directory, markerName, 4096)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, line := range strings.Split(strings.TrimSuffix(string(wire), "\n"), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || values[key] != "" {
			return nil, errors.New("marker schema")
		}
		values[key] = value
	}
	if values["formatVersion"] != "1" || !digestPattern.MatchString(values["archiveSha256"]) || values["stage"] != "prepare" && values["stage"] != "binary_committed" {
		return nil, errors.New("marker schema")
	}
	return values, nil
}

// writeMarker uses the release installer's marker name and fields so the
// installer and the upgrade path refuse each other's interrupted state.
func writeMarker(directory, archive, stage string, create bool) error {
	wire := []byte(fmt.Sprintf("formatVersion=1\nstage=%s\narchiveSha256=%s\noperation=upgrade\n", stage, archive))
	path := filepath.Join(directory, markerName)
	if create {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|unix.O_NOFOLLOW, 0o600)
		if err != nil {
			return err
		}
		_, writeErr := file.Write(wire)
		syncErr := file.Sync()
		if err := errors.Join(writeErr, syncErr, file.Close()); err != nil {
			return err
		}
		syncDirectory(directory)
		return nil
	}
	temporary := path + ".next"
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(wire)
	syncErr := file.Sync()
	if err := errors.Join(writeErr, syncErr, file.Close()); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	syncDirectory(directory)
	return nil
}

func stageLeftovers(target Target) []string {
	leftovers := []string{}
	for _, check := range []struct{ directory, prefix string }{{target.BinaryDir, binaryStage}, {target.ReceiptDir, receiptStage}} {
		entries, err := os.ReadDir(check.directory)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), check.prefix) && entry.Type().IsRegular() {
				leftovers = append(leftovers, filepath.Join(check.directory, entry.Name()))
			}
		}
	}
	sort.Strings(leftovers)
	return leftovers
}

func syncDirectory(path string) {
	if directory, err := os.Open(path); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
}

func statfsAvailable(path string) (uint64, error) {
	var filesystem unix.Statfs_t
	if err := unix.Statfs(path, &filesystem); err != nil {
		return 0, err
	}
	return uint64(filesystem.Bavail) * uint64(filesystem.Bsize), nil
}

// compareSemver implements SemVer 2.0.0 precedence; build metadata is ignored.
func compareSemver(left, right string) int {
	split := func(value string) ([]string, []string) {
		value, _, _ = strings.Cut(value, "+")
		core, pre, _ := strings.Cut(value, "-")
		identifiers := []string(nil)
		if pre != "" {
			identifiers = strings.Split(pre, ".")
		}
		return strings.Split(core, "."), identifiers
	}
	numeric := func(value string) (int, bool) {
		number, err := strconv.Atoi(value)
		return number, err == nil
	}
	leftCore, leftPre := split(left)
	rightCore, rightPre := split(right)
	for index := 0; index < 3; index++ {
		a, _ := numeric(leftCore[index])
		b, _ := numeric(rightCore[index])
		if a != b {
			if a < b {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(leftPre) == 0 && len(rightPre) == 0:
		return 0
	case len(leftPre) == 0:
		return 1
	case len(rightPre) == 0:
		return -1
	}
	for index := 0; index < len(leftPre) && index < len(rightPre); index++ {
		a, aNumeric := numeric(leftPre[index])
		b, bNumeric := numeric(rightPre[index])
		switch {
		case aNumeric && bNumeric && a != b:
			if a < b {
				return -1
			}
			return 1
		case aNumeric != bNumeric:
			if aNumeric {
				return -1
			}
			return 1
		case !aNumeric && leftPre[index] != rightPre[index]:
			if leftPre[index] < rightPre[index] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(leftPre) < len(rightPre):
		return -1
	case len(leftPre) > len(rightPre):
		return 1
	}
	return 0
}
