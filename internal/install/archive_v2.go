package install

import (
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rgomids/axiom/internal/operationaldocs"
)

const (
	maxArchiveEntriesV2 = 192
	maxFilesV2          = 77
	maxStreamV2         = 288 << 20
	maxPayloadV2        = 280 << 20
)

var (
	v2Fixed     = []string{binaryName, "LICENSE", "install.sh", "release-metadata.txt", "skills-manifest.txt", "MANIFEST.sha256", "docs/inventory.tsv", "docs/provenance.tsv", "docs/man-map.tsv"}
	v2Component = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
)

func v2Mode(name string) int64 {
	if name == binaryName || name == "install.sh" {
		return 0o700
	}
	return 0o600
}

// loadCandidateV2 verifies archive generation 2 completely, in memory, before
// any candidate execution: closed layout, modes, manifest, metadata, skills and
// documentary identity. The binary digest is the metadata's claim, not authenticity.
func loadCandidateV2(read bundleRead, archiveSHA string) (Candidate, error) {
	invalid := func(category string) (Candidate, error) { return Candidate{}, &Error{Category: category} }
	files := read.files
	if read.headers > maxArchiveEntriesV2 || len(files) > maxFilesV2 || read.stream > maxStreamV2 || !read.zeroTail || read.trailing {
		return invalid("archive_invalid")
	}
	payload := 0
	for name, wire := range files {
		payload += len(wire)
		entry := read.entries[name]
		if !entry.ustar || entry.mode != v2Mode(name) || len(name) > 240 {
			return invalid("archive_invalid")
		}
		for _, part := range strings.Split(name, "/") {
			if part != "LICENSE" && part != "MANIFEST.sha256" && part != "SKILL.md" && !v2Component.MatchString(part) {
				return invalid("archive_invalid")
			}
		}
	}
	if payload > maxPayloadV2 {
		return invalid("archive_invalid")
	}
	for _, dir := range read.dirs {
		if !dir.ustar || dir.mode != 0o700 {
			return invalid("archive_invalid")
		}
	}
	if err := verifyManifestV2(files); err != nil {
		return invalid("archive_invalid")
	}
	metadata, err := operationaldocs.ParseMetadata(files["release-metadata.txt"])
	if err != nil {
		return invalid("release_metadata_invalid")
	}
	values := map[string]string{"release": strconv.FormatBool(metadata.Release), "version": metadata.Version, "revision": metadata.Revision, "sourceState": metadata.SourceState, "platform": metadata.Platform, "goos": metadata.GOOS, "architecture": metadata.Architecture}
	if read.root != "axiom-"+metadata.Version+"-"+metadata.Platform+"-"+metadata.Architecture {
		return invalid("archive_invalid")
	}
	candidate := Candidate{ArchiveSHA256: archiveSHA, Version: metadata.Version, Binary: files[binaryName], Skills: map[string]string{}, SkillFiles: map[string][]byte{}, Values: values}
	candidate.Metadata = strings.Split(strings.TrimSuffix(string(files["release-metadata.txt"]), "\n"), "\n")
	if err := parseSkillManifest(files["skills-manifest.txt"], files, candidate.Skills); err != nil {
		return invalid("skill_manifest_invalid")
	}
	candidate.SkillManifestSHA256 = digest(files["skills-manifest.txt"])
	for _, name := range skillNames {
		candidate.SkillFiles[name] = files["skills/"+name+"/SKILL.md"]
	}
	docsFiles := map[string][]byte{}
	for name, wire := range files {
		if strings.HasPrefix(name, "docs/") {
			docsFiles[name] = wire
		}
	}
	bundle, err := operationaldocs.Validate(docsFiles, metadata.SourceCommit)
	if err != nil || bundle.InventorySHA256 != metadata.Inventory || bundle.ProvenanceSHA256 != metadata.Provenance || bundle.ManMapSHA256 != metadata.ManMap || digest(candidate.Binary) != metadata.Binary || len(candidate.Binary) == 0 {
		return invalid("archive_invalid")
	}
	if len(files) != len(v2Fixed)+len(skillNames)+len(bundle.Entries) {
		return invalid("archive_invalid")
	}
	for _, name := range v2Fixed {
		if _, ok := files[name]; !ok {
			return invalid("archive_invalid")
		}
	}
	candidate.Docs, candidate.DocsMetadata = &bundle, &metadata
	candidate.MetadataSHA256 = digest(files["release-metadata.txt"])
	return candidate, nil
}

func verifyManifestV2(files map[string][]byte) error {
	manifest, ok := files["MANIFEST.sha256"]
	if !ok || len(manifest) > 64<<10 || len(manifest) == 0 || manifest[len(manifest)-1] != '\n' {
		return errManifest
	}
	names := make([]string, 0, len(files))
	for name := range files {
		if name != "MANIFEST.sha256" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	lines := strings.Split(string(manifest[:len(manifest)-1]), "\n")
	if len(lines) != len(names) {
		return errManifest
	}
	for i, name := range names {
		if lines[i] != digest(files[name])+"  "+name {
			return errManifest
		}
	}
	return nil
}

var errManifest = errors.New("bundle manifest")
