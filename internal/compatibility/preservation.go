package compatibility

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/rgomids/axiom/internal/local"
)

// Issue #153 / I153-T02: the RecognizedPOC transition preserve -> clean
// rebuild -> supported reconfiguration.
//
// Preservation copies every object of the revalidated source inventory into
// a dedicated Axiom-owned machine-local archive, content-addressed under
// objects/<sha256>, verifies each copy from the destination, writes the final
// manifest last, and proves exact correspondence between the manifest and the
// policy-required inventory before any legacy object may be retired. The
// archive never becomes active state and the transition never deletes it.
//
// Rebuild retires, from the active state root, exactly the historical POC
// workflow material (workflow records and the Work Item links the POC
// workflow created). What stays active is what the current contract
// already validates: unrelated Work Item links, installation records
// (DecodeRecord) and portable manifests (manifest.Decode), so the rebuilt root
// is canonical v1 state that
// holds supported Project intent, associations and unrelated current Work Item
// links, without POC workflow truth.

// PreservationPolicy names the exact object-set rule this release applies.
const PreservationPolicy = "recognized-poc-preservation/v1"

const (
	preservationKind     = "recognized_poc_preservation"
	preservationManifest = "manifest.json"
	preservationObjects  = "objects"
	preservationStage    = ".axiom-archive-stage-"
	maxManifestBytes     = 4 << 20
	archivePrefix        = "recognized-poc-"
)

var archiveNamePattern = regexp.MustCompile(`^recognized-poc-[0-9a-f]{64}$`)

var (
	// ErrPreservationConflict reports an archive that is not exactly this
	// transition's verified preservation: unknown, foreign, changed, or
	// unverifiable content. Nothing is retired while it holds.
	ErrPreservationConflict = errors.New("preservation archive does not correspond to the source inventory")
	// ErrPreservationSource reports source state that no longer matches the
	// inventory the transition was authorized for.
	ErrPreservationSource = errors.New("preservation source changed")
	// ErrPreservationTarget reports an archive destination that is unsafe,
	// not Axiom-owned, or overlapping an active root.
	ErrPreservationTarget = errors.New("preservation archive destination is unsafe or overlapping")
)

// PreservationManifest is written last into the archive. Its presence with
// verified objects marks a complete preservation; it is never active state.
type PreservationManifest struct {
	FormatVersion int      `json:"formatVersion"`
	Kind          string   `json:"kind"`
	Policy        string   `json:"policy"`
	POCTag        string   `json:"pocTag"`
	POCRevision   string   `json:"pocRevision"`
	SourceDigest  string   `json:"sourceDigest"`
	Objects       []Object `json:"objects"`
	Retired       []string `json:"retired"`
	Kept          []string `json:"kept"`
}

// TransitionPlan is the read-only plan for one RecognizedPOC transition. It
// carries exact objects only; it grants no authority.
type TransitionPlan struct {
	SourceDigest         string
	ConfirmedRetirements int
	ArchiveRoot          string
	Archive              string
	Preserve             []Object
	Copy                 []Object
	ManifestPresent      bool
	Manifest             []byte
	Retire               []Object
	Keep                 []Object
	Leftovers            []string
	RequiredBytes        int64
	historical           []Object // complete original retirement set, including confirmed objects
}

// ArchivePath is the operation's archive directory.
func (p TransitionPlan) ArchivePath() string { return filepath.Join(p.ArchiveRoot, p.Archive) }

// ObjectPath is where object's preserved copy lives.
func (p TransitionPlan) ObjectPath(object Object) string {
	return filepath.Join(p.ArchivePath(), preservationObjects, object.Digest)
}

// ManifestPath is the final manifest's location.
func (p TransitionPlan) ManifestPath() string {
	return filepath.Join(p.ArchivePath(), preservationManifest)
}

// ArchiveName is the archive identity of a RecognizedPOC source digest.
func ArchiveName(sourceDigest string) string { return archivePrefix + sourceDigest }

// ValidArchiveName reports a well-formed archive identity.
func ValidArchiveName(name string) bool { return archiveNamePattern.MatchString(name) }

// retirementCandidate reports kinds that may hold historical material.
// Work Item membership is proved separately from persisted identity.
func retirementCandidate(kind local.InventoryKind) bool {
	return kind == local.InventoryPOCWorkflow || kind == local.InventoryWorkItem
}

// keptKind reports the supported Project intent and associations that the
// current contract already validated while inventorying them.
func keptKind(kind local.InventoryKind) bool {
	return kind == local.InventoryInstallation || kind == local.InventoryPortableManifest
}

// skillKind reports Axiom skill files inventoried in a Runtime skills root.
// That root belongs to the Runtime and its Axiom files are published content
// that the owned skill upgrade converges under its own ownership rules, so
// they are neither preserved nor retired here.
func skillKind(kind local.InventoryKind) bool {
	return strings.HasPrefix(string(kind), "skill_")
}

// preservedObjects is the policy-required set: every inventoried object of
// the Projects and State roots.
func preservedObjects(objects []Object) []Object {
	preserved := []Object{}
	for _, object := range objects {
		if object.Category != CategorySkills {
			preserved = append(preserved, object)
		}
	}
	return preserved
}

// PlanPOCTransition resolves the exact transition for report. resumeArchive
// is the archive an interrupted operation recorded ("" for a new operation):
// the transition then continues from that archive's verified truth instead
// of starting a second preservation.
//
// resumeManifest is the digest of the final manifest the interrupted
// operation verified and recorded before its first retirement ("" if it
// retired nothing): a present manifest must be byte-for-byte that one.
func PlanPOCTransition(ctx context.Context, roots Roots, report Report, archiveRoot, resumeArchive, resumeManifest string, confirmedRetirements int) (TransitionPlan, error) {
	if err := ctx.Err(); err != nil {
		return TransitionPlan{}, err
	}
	if err := checkArchiveRoot(archiveRoot, roots); err != nil {
		return TransitionPlan{}, err
	}
	// Resolve only trusted system aliases (such as macOS /var); any other
	// symlink on the archive path is refused when the archive is opened.
	canonical, err := local.CanonicalPath(archiveRoot)
	if err != nil {
		return TransitionPlan{}, ErrPreservationTarget
	}
	// An existing namespace must already be a safe Axiom-owned directory.
	if _, err := os.Lstat(canonical); err == nil {
		if local.CheckPrivateDirectory(canonical) != nil {
			return TransitionPlan{}, ErrPreservationTarget
		}
	} else if !os.IsNotExist(err) {
		return TransitionPlan{}, ErrPreservationTarget
	}
	plan := TransitionPlan{ConfirmedRetirements: confirmedRetirements, ArchiveRoot: canonical, Copy: []Object{}, Retire: []Object{}, Keep: []Object{}, Leftovers: []string{}}
	switch {
	case resumeArchive == "":
		if report.Classification != RecognizedPOC || !historicalPOCPreconditions(report) {
			return TransitionPlan{}, ErrPreservationSource
		}
		plan.SourceDigest, plan.Archive = report.Digest, ArchiveName(report.Digest)
	case ValidArchiveName(resumeArchive):
		plan.SourceDigest, plan.Archive = strings.TrimPrefix(resumeArchive, archivePrefix), resumeArchive
	default:
		return TransitionPlan{}, ErrPreservationConflict
	}
	archive, err := readArchive(plan.ArchivePath())
	if err != nil {
		return TransitionPlan{}, err
	}
	if archive.manifest != nil {
		document := *archive.manifest
		if document.SourceDigest != plan.SourceDigest || document.Policy != PreservationPolicy {
			return TransitionPlan{}, ErrPreservationConflict
		}
		plan.Preserve, plan.ManifestPresent, plan.Manifest = document.Objects, true, archive.manifestWire
		plan.historical, err = historicalRetirementSet(plan.Preserve, archiveObjectReader(plan.ArchivePath()))
		if err != nil {
			return TransitionPlan{}, ErrPreservationConflict
		}
		expected, err := plan.manifestWire()
		if err != nil || !bytes.Equal(expected, archive.manifestWire) {
			return TransitionPlan{}, ErrPreservationConflict
		}
		if resumeManifest != "" && digestHex(archive.manifestWire) != resumeManifest {
			return TransitionPlan{}, ErrPreservationConflict
		}
		if err := verifyArchiveObjects(plan.ArchivePath(), plan.Preserve, archive); err != nil {
			return TransitionPlan{}, err
		}
		if len(archive.leftovers) != 0 {
			return TransitionPlan{}, ErrPreservationConflict
		}
		// While nothing is retired, the manifest must list exactly the
		// policy-required inventory, whatever the copied objects' digests say.
		if report.Digest == plan.SourceDigest {
			if !sameObjects(plan.Preserve, preservedObjects(report.objects)) {
				return TransitionPlan{}, ErrPreservationConflict
			}
		} else if resumeManifest == "" {
			// Retirement started only under a recorded verified manifest.
			return TransitionPlan{}, ErrPreservationConflict
		}
		if err := verifyActiveInventory(plan, report.objects); err != nil {
			return TransitionPlan{}, err
		}

	} else {
		// Nothing may have been retired before the final manifest, so the
		// source must still be exactly the inventory the archive is named for.
		if confirmedRetirements != 0 || report.Digest != plan.SourceDigest || report.Classification != RecognizedPOC || !historicalPOCPreconditions(report) {
			return TransitionPlan{}, ErrPreservationSource
		}
		plan.Preserve = preservedObjects(report.objects)
		plan.historical, err = historicalRetirementSet(plan.Preserve, func(object Object) ([]byte, error) {
			return readSourceObject(roots, object)
		})
		if err != nil {
			return TransitionPlan{}, ErrPreservationSource
		}
		if len(plan.Preserve) == 0 {
			return TransitionPlan{}, ErrPreservationSource
		}
		if len(archive.leftovers) != 0 && resumeArchive == "" {
			return TransitionPlan{}, ErrPreservationConflict
		}
		plan.Leftovers = archive.leftovers
		wanted := map[string]Object{}
		for _, object := range plan.Preserve {
			wanted[object.Digest] = object
		}
		for digest := range archive.objects {
			if _, ok := wanted[digest]; !ok {
				return TransitionPlan{}, ErrPreservationConflict
			}
		}
		copied := map[string]bool{}
		for _, object := range plan.Preserve {
			if copied[object.Digest] {
				continue
			}
			copied[object.Digest] = true
			if _, present := archive.objects[object.Digest]; present {
				if err := verifyArchiveObject(plan.ArchivePath(), object); err != nil {
					return TransitionPlan{}, err
				}
				continue
			}
			plan.Copy = append(plan.Copy, object)
			plan.RequiredBytes += object.Bytes
		}
		wire, err := plan.manifestWire()
		if err != nil {
			return TransitionPlan{}, err
		}
		plan.Manifest = wire
		plan.RequiredBytes += int64(len(wire))
	}
	historical := objectSet(plan.historical)
	for _, object := range report.objects {
		switch {
		case historical[objectKey(object)]:
			plan.Retire = append(plan.Retire, object)
		case keptKind(object.Kind), object.Kind == local.InventoryWorkItem, skillKind(object.Kind):
			plan.Keep = append(plan.Keep, object)
		default:
			return TransitionPlan{}, ErrPreservationSource
		}
	}
	sort.Slice(plan.Retire, func(i, j int) bool { return retireOrder(plan.Retire[i]) < retireOrder(plan.Retire[j]) })
	return plan, nil
}

// verifyActiveInventory requires positive progress truth: kept and pending
// objects remain exact; only a deterministic confirmed retirement prefix is absent.
func verifyActiveInventory(plan TransitionPlan, objects []Object) error {
	retired := append([]Object(nil), plan.historical...)
	sort.Slice(retired, func(i, j int) bool { return retireOrder(retired[i]) < retireOrder(retired[j]) })
	if plan.ConfirmedRetirements < 0 || plan.ConfirmedRetirements > len(retired) {
		return ErrPreservationConflict
	}
	confirmed := objectSet(retired[:plan.ConfirmedRetirements])
	expected := []Object{}
	for _, object := range plan.Preserve {
		if !confirmed[objectKey(object)] {
			expected = append(expected, object)
		}
	}
	if !sameObjects(expected, preservedObjects(objects)) {
		return ErrPreservationSource
	}
	return nil
}

// retireOrder retires Work Item links before workflow records, so an
// interrupted retirement keeps the complete POC signature until the last
// workflow record goes and the root never looks like v1 state with a
// partially retired history.
func retireOrder(object Object) string {
	rank := "1"
	if object.Kind == local.InventoryPOCWorkflow {
		rank = "2"
	}
	return rank + object.Category + "/" + object.Relative
}

func (p TransitionPlan) manifestWire() ([]byte, error) {
	if !digestName.MatchString(p.SourceDigest) || len(p.Preserve) == 0 {
		return nil, ErrPreservationConflict
	}
	seen := map[string]bool{}
	for _, object := range p.Preserve {
		key := object.Category + "/" + object.Relative
		_, clean := cleanRelative(object.Relative)
		validKind := object.Category == CategoryState && (retirementCandidate(object.Kind) || object.Kind == local.InventoryInstallation) || object.Category == CategoryProjects && object.Kind == local.InventoryPortableManifest
		if !clean || !validKind || !digestName.MatchString(object.Digest) || object.Bytes <= 0 || object.Bytes > maxTransferFile || seen[key] {
			return nil, ErrPreservationConflict
		}
		seen[key] = true
	}

	document := PreservationManifest{FormatVersion: 1, Kind: preservationKind, Policy: PreservationPolicy, POCTag: HistoricalPOCTag, POCRevision: HistoricalPOCRevision, SourceDigest: p.SourceDigest, Objects: sortedObjects(p.Preserve), Retired: []string{}, Kept: []string{}}
	historical := objectSet(p.historical)
	for _, object := range p.Preserve {
		key := object.Category + "/" + object.Relative
		if historical[objectKey(object)] {
			document.Retired = append(document.Retired, key)
		} else {
			document.Kept = append(document.Kept, key)
		}
	}
	sort.Strings(document.Retired)
	sort.Strings(document.Kept)
	wire, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(wire, '\n'), nil
}

// VerifyArchive re-proves, after the rebuilt state was activated, that the
// operation's archive is still exactly the verified preservation recorded by
// manifestDigest: the manifest bytes, every listed object, and nothing else.
// Active state is not compared: once activated it is ordinary v1 state that
// may legitimately change.
func VerifyArchive(roots Roots, archiveRoot, archive, manifestDigest string, confirmedRetirements int) (string, error) {
	if err := checkArchiveRoot(archiveRoot, roots); err != nil {
		return "", err
	}
	canonical, err := local.CanonicalPath(archiveRoot)
	if err != nil || !ValidArchiveName(archive) || !digestName.MatchString(manifestDigest) {
		return "", ErrPreservationConflict
	}
	path := filepath.Join(canonical, archive)
	observed, err := readArchive(path)
	if err != nil {
		return "", err
	}
	if observed.manifest == nil || digestHex(observed.manifestWire) != manifestDigest || len(observed.leftovers) != 0 || observed.manifest.SourceDigest != strings.TrimPrefix(archive, archivePrefix) {
		return "", ErrPreservationConflict
	}
	plan := TransitionPlan{SourceDigest: observed.manifest.SourceDigest, Preserve: observed.manifest.Objects}
	plan.historical, err = historicalRetirementSet(plan.Preserve, archiveObjectReader(path))
	if err != nil {
		return "", ErrPreservationConflict
	}
	canonicalManifest, err := plan.manifestWire()
	if err != nil || !bytes.Equal(canonicalManifest, observed.manifestWire) {
		return "", ErrPreservationConflict
	}
	retired := len(plan.historical)
	if confirmedRetirements != retired {
		return "", ErrPreservationConflict
	}
	if err := verifyArchiveObjects(path, observed.manifest.Objects, observed); err != nil {
		return "", err
	}
	return path, nil
}

// ManifestDigest is the digest of the manifest the plan publishes or found.
func (p TransitionPlan) ManifestDigest() string { return digestHex(p.Manifest) }

// PreserveObject copies one source object into the archive: the source is
// re-read through the anchored owned-file path and must still have its
// inventoried digest and size; the copy is staged privately, published
// without replacing any name, and re-read from the destination before it is
// confirmed. An existing copy is accepted only when it verifies.
func PreserveObject(ctx context.Context, roots Roots, plan TransitionPlan, object Object) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := checkArchiveRoot(plan.ArchiveRoot, roots); err != nil {
		return err
	}
	wire, err := readSourceObject(roots, object)
	if err != nil {
		return err
	}
	objects, err := openArchiveDirectory(plan.ArchivePath(), true)
	if err != nil {
		return err
	}
	defer objects.Close()
	if _, err := objects.Lstat(object.Digest); err == nil {
		return verifyArchiveObject(plan.ArchivePath(), object)
	}
	if beforePreservationWrite != nil {
		if err := beforePreservationWrite(object); err != nil {
			return err
		}
	}
	staged, err := objects.Stage(preservationStage, wire, 0o600)
	if err != nil {
		return err
	}
	if err := objects.PublishNew(staged, object.Digest); err != nil {
		_ = objects.Remove(staged)
		return err
	}
	if err := errors.Join(objects.Sync(), objects.StillAtPath()); err != nil {
		return err
	}
	return verifyArchiveObject(plan.ArchivePath(), object)
}

// CompletePreservation publishes the final manifest (unless present) and then
// proves correspondence: the manifest lists exactly the policy-required
// inventory (category, relative path, kind, digest, bytes), every listed
// object verifies from the destination, the objects directory holds nothing
// else, and the source still holds no object outside the manifest. Only a nil
// result authorizes retirement.
func CompletePreservation(ctx context.Context, roots Roots, plan TransitionPlan) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := checkArchiveRoot(plan.ArchiveRoot, roots); err != nil {
		return err
	}
	if !plan.ManifestPresent {
		directory, err := openArchiveDirectory(plan.ArchivePath(), false)
		if err != nil {
			return err
		}
		staged, err := directory.Stage(preservationStage, plan.Manifest, 0o600)
		if err == nil {
			if err = directory.PublishNew(staged, preservationManifest); err != nil {
				_ = directory.Remove(staged)
			}
		}
		if err == nil {
			err = errors.Join(directory.Sync(), directory.StillAtPath())
		}
		directory.Close()
		if err != nil {
			return err
		}
	}
	return VerifyPreservation(ctx, roots, plan)
}

// VerifyPreservation is the read-only correspondence proof of
// CompletePreservation, for an archive whose manifest is present.
func VerifyPreservation(ctx context.Context, roots Roots, plan TransitionPlan) error {
	archive, err := readArchive(plan.ArchivePath())
	if err != nil {
		return err
	}
	if archive.manifest == nil || !bytes.Equal(archive.manifestWire, plan.Manifest) || len(archive.leftovers) != 0 {
		return ErrPreservationConflict
	}
	document := *archive.manifest
	if document.SourceDigest != plan.SourceDigest || document.Policy != PreservationPolicy || !sameObjects(document.Objects, plan.Preserve) {
		return ErrPreservationConflict
	}
	expected, err := plan.manifestWire()
	if err != nil || !bytes.Equal(expected, archive.manifestWire) {
		return ErrPreservationConflict
	}

	if err := verifyArchiveObjects(plan.ArchivePath(), document.Objects, archive); err != nil {
		return err
	}
	report, err := Inspect(ctx, roots)
	if err != nil {
		return ErrPreservationSource
	}
	return verifyActiveInventory(plan, report.objects)
}

// RetireObject removes one preserved historical generation under the same
// broad-to-narrow coordination used by local writers. Empty containers are
// pruned while those locks remain held.
func RetireObject(ctx context.Context, roots Roots, object Object) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if object.Category != CategoryState || !retirementCandidate(object.Kind) || roots.State == "" {
		return ErrPreservationSource
	}
	relative, ok := cleanRelative(object.Relative)
	if !ok {
		return ErrPreservationSource
	}
	entry := local.InventoryEntry{Relative: filepath.ToSlash(relative), Kind: object.Kind, Digest: object.Digest, Bytes: object.Bytes}
	if err := local.RetireHistoricalStateObject(ctx, roots.State, entry); err != nil {
		return errors.Join(ErrPreservationSource, err)
	}
	return nil
}

// Deterministic fault seam for tests: called before each object copy.
var beforePreservationWrite func(Object) error

type archiveObservation struct {
	present      bool
	manifest     *PreservationManifest
	manifestWire []byte
	objects      map[string]bool
	leftovers    []string
}

// readArchive observes an archive directory read-only. Absent is empty; any
// entry the transition never writes is a conflict.
func readArchive(path string) (archiveObservation, error) {
	observed := archiveObservation{objects: map[string]bool{}, leftovers: []string{}}
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return observed, nil
	}
	directory, err := local.OpenOwnedDirectory(path)
	if err != nil {
		return observed, ErrPreservationTarget
	}
	defer directory.Close()
	observed.present = true
	entries, err := directory.ReadDir()
	if err != nil {
		return observed, ErrPreservationConflict
	}
	for _, entry := range entries {
		switch name := entry.Name(); {
		case name == preservationObjects:
		case name == preservationManifest:
			wire, err := directory.ReadFile(name, maxManifestBytes)
			if err != nil {
				return observed, ErrPreservationConflict
			}
			var document PreservationManifest
			decoder := json.NewDecoder(bytes.NewReader(wire))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&document) != nil || decoder.Decode(new(any)) != io.EOF || document.FormatVersion != 1 || document.Kind != preservationKind || len(document.Objects) == 0 {
				return observed, ErrPreservationConflict
			}
			observed.manifest, observed.manifestWire = &document, wire
		case strings.HasPrefix(name, preservationStage):
			observed.leftovers = append(observed.leftovers, filepath.Join(path, name))
		default:
			return observed, ErrPreservationConflict
		}
	}
	if _, err := directory.Lstat(preservationObjects); os.IsNotExist(err) {
		return observed, nil
	}
	objects, err := local.OpenOwnedDirectory(filepath.Join(path, preservationObjects))
	if err != nil {
		return observed, ErrPreservationConflict
	}
	defer objects.Close()
	entries, err = objects.ReadDir()
	if err != nil {
		return observed, ErrPreservationConflict
	}
	for _, entry := range entries {
		switch name := entry.Name(); {
		case digestName.MatchString(name):
			observed.objects[name] = true
		case strings.HasPrefix(name, preservationStage):
			observed.leftovers = append(observed.leftovers, filepath.Join(path, preservationObjects, name))
		default:
			return observed, ErrPreservationConflict
		}
	}
	sort.Strings(observed.leftovers)
	return observed, nil
}

var digestName = regexp.MustCompile(`^[0-9a-f]{64}$`)

func verifyArchiveObjects(archive string, objects []Object, observed archiveObservation) error {
	wanted := map[string]bool{}
	for _, object := range objects {
		wanted[object.Digest] = true
		if !observed.objects[object.Digest] {
			return ErrPreservationConflict
		}
		if err := verifyArchiveObject(archive, object); err != nil {
			return err
		}
	}
	for digest := range observed.objects {
		if !wanted[digest] {
			return ErrPreservationConflict
		}
	}
	return nil
}

func verifyArchiveObject(archive string, object Object) error {
	objects, err := local.OpenOwnedDirectory(filepath.Join(archive, preservationObjects))
	if err != nil {
		return ErrPreservationConflict
	}
	defer objects.Close()
	wire, err := objects.ReadFile(object.Digest, maxTransferFile)
	if err != nil || digestHex(wire) != object.Digest || int64(len(wire)) != object.Bytes {
		return ErrPreservationConflict
	}
	return nil
}

// RemoveArchiveLeftover discards one interrupted stage of this archive.
func RemoveArchiveLeftover(plan TransitionPlan, leftover string) error {
	parent, name := filepath.Dir(leftover), filepath.Base(leftover)
	if !strings.HasPrefix(name, preservationStage) || parent != plan.ArchivePath() && parent != filepath.Join(plan.ArchivePath(), preservationObjects) {
		return ErrPreservationConflict
	}
	directory, err := local.OpenOwnedDirectory(parent)
	if err != nil {
		return err
	}
	defer directory.Close()
	if err := directory.Remove(name); err != nil {
		return err
	}
	return directory.Sync()
}

func openArchiveDirectory(archive string, objects bool) (local.AnchoredDirectory, error) {
	directory, err := local.CreateOwnedDirectory(archive)
	if err != nil {
		return local.AnchoredDirectory{}, ErrPreservationTarget
	}
	if !objects {
		return directory, nil
	}
	directory.Close()
	directory, err = local.CreateOwnedDirectory(filepath.Join(archive, preservationObjects))
	if err != nil {
		return local.AnchoredDirectory{}, ErrPreservationTarget
	}
	return directory, nil
}

// checkArchiveRoot requires an explicit absolute archive namespace outside,
// and not containing, every active root.
func checkArchiveRoot(archiveRoot string, roots Roots) error {
	if !filepath.IsAbs(archiveRoot) || filepath.Clean(archiveRoot) == string(filepath.Separator) || filepath.Clean(archiveRoot) != archiveRoot || ValidArchiveName(filepath.Base(archiveRoot)) {
		return ErrPreservationTarget
	}
	for _, active := range []string{roots.Projects, roots.State, roots.Skills} {
		if active == "" {
			continue
		}
		overlap, err := local.RootsOverlap(archiveRoot, active)
		if err != nil || overlap {
			return ErrPreservationTarget
		}
	}
	return nil
}

func readSourceObject(roots Roots, object Object) ([]byte, error) {
	base := map[string]string{CategoryProjects: roots.Projects, CategoryState: roots.State, CategorySkills: roots.Skills}[object.Category]
	if base == "" {
		return nil, ErrPreservationSource
	}
	wire, err := local.ReadOwnedFile(base, object.Relative, maxTransferFile)
	if err != nil || digestHex(wire) != object.Digest || int64(len(wire)) != object.Bytes {
		return nil, ErrPreservationSource
	}
	return wire, nil
}

func cleanRelative(relative string) (string, bool) {
	clean := filepath.Clean(filepath.FromSlash(relative))
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.Dir(clean) == "." {
		return "", false
	}
	return clean, true
}

func objectKey(object Object) string {
	wire, _ := json.Marshal(object)
	return string(wire)
}

func objectSet(objects []Object) map[string]bool {
	set := make(map[string]bool, len(objects))
	for _, object := range objects {
		set[objectKey(object)] = true
	}
	return set
}

func sortedObjects(objects []Object) []Object {
	sorted := append([]Object(nil), objects...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Category != sorted[j].Category {
			return sorted[i].Category < sorted[j].Category
		}
		return sorted[i].Relative < sorted[j].Relative
	})
	return sorted
}

func sameObjects(left, right []Object) bool {
	a, _ := json.Marshal(sortedObjects(left))
	b, _ := json.Marshal(sortedObjects(right))
	return bytes.Equal(a, b)
}

func digestHex(wire []byte) string {
	sum := sha256.Sum256(wire)
	return hex.EncodeToString(sum[:])
}
