package operationaldocs

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxMetadata = 16 << 10
	MaxReceipt  = 64 << 10
	MaxBinding  = 16 << 10
)

var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

var metadataKeys = []string{"formatVersion", "product", "version", "revision", "sourceState", "release", "platform", "goos", "architecture", "skillSetVersion", "sourceCommit", "archiveGeneration", "docsInventorySha256", "docsProvenanceSha256", "docsManMapSha256", "binarySha256"}

var receiptKeys = []string{"formatVersion", "destination", "sha256", "product", "version", "revision", "sourceState", "release", "platform", "goos", "architecture", "skillSetVersion", "archiveSha256", "skillManifestSha256", "installedAt", "sourceCommit", "docsGeneration", "docsBindingSha256", "docsInventorySha256", "docsProvenanceSha256", "docsManMapSha256"}

var bindingFields = []string{"schemaVersion", "binaryPath", "parentDevice", "parentInode", "binaryDevice", "binaryInode", "binarySha256", "sourceCommit", "version", "archiveSha256", "metadataSha256", "generation", "provenanceSha256", "manMapSha256", "skillManifestSha256"}

func ordered(wire []byte, keys []string, limit int) (map[string]string, error) {
	if len(wire) > limit {
		return nil, errors.New("quota_exceeded")
	}
	if !utf8.Valid(wire) || len(wire) == 0 || wire[len(wire)-1] != '\n' || strings.ContainsAny(string(wire), "\r\x00") {
		return nil, errors.New("malformed_record")
	}
	lines := strings.Split(string(wire[:len(wire)-1]), "\n")
	if len(lines) != len(keys) {
		return nil, errors.New("malformed_record")
	}
	values := make(map[string]string, len(keys))
	for i, line := range lines {
		key, value, ok := strings.Cut(line, "=")
		if !ok || key != keys[i] || value == "" || strings.Contains(value, "=") || strings.ContainsFunc(value, func(r rune) bool { return r < 32 || r == 127 }) {
			return nil, errors.New("malformed_record")
		}
		values[key] = value
	}
	return values, nil
}

func platformTuple(platform, goos, arch string) bool {
	switch platform + "/" + goos + "/" + arch {
	case "macos/darwin/arm64", "linux/linux/amd64", "linux/linux/arm64":
		return true
	}
	return false
}

func cleanPath(p string) bool {
	if !strings.HasPrefix(p, "/") || p == "/" || len(p) > 4096 || strings.Contains(p, "//") || strings.HasSuffix(p, "/") || strings.ContainsFunc(p, func(r rune) bool { return r < 32 || r == 127 || r == '=' }) {
		return false
	}
	for _, part := range strings.Split(p[1:], "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}

type Metadata struct {
	Version, Revision, SourceState, Platform, GOOS, Architecture, SourceCommit string
	Release                                                                     bool
	Inventory, Provenance, ManMap, Binary                                       string
}

func ParseMetadata(wire []byte) (Metadata, error) {
	v, err := ordered(wire, metadataKeys, MaxMetadata)
	if err != nil {
		return Metadata{}, err
	}
	bad := errors.New("malformed_record")
	if v["formatVersion"] != "2" || v["product"] != "Axiom" || !versionPattern.MatchString(v["version"]) || len(v["version"]) > 128 || !fullCommit.MatchString(v["sourceCommit"]) || v["revision"] != v["sourceCommit"][:12] || (v["sourceState"] != "clean" && v["sourceState"] != "dirty") || (v["release"] != "true" && v["release"] != "false") || (v["release"] == "true" && v["sourceState"] != "clean") || !platformTuple(v["platform"], v["goos"], v["architecture"]) || v["skillSetVersion"] != "1" || v["archiveGeneration"] != "2" {
		return Metadata{}, bad
	}
	for _, k := range []string{"docsInventorySha256", "docsProvenanceSha256", "docsManMapSha256", "binarySha256"} {
		if !hexDigest.MatchString(v[k]) {
			return Metadata{}, bad
		}
	}
	return Metadata{v["version"], v["revision"], v["sourceState"], v["platform"], v["goos"], v["architecture"], v["sourceCommit"], v["release"] == "true", v["docsInventorySha256"], v["docsProvenanceSha256"], v["docsManMapSha256"], v["binarySha256"]}, nil
}

type Receipt struct {
	Destination, Binary, Version, Revision, SourceCommit, Generation, Binding, InstalledAt string
	Inventory, Provenance, ManMap, Archive, SkillManifest                                 string
}

func ParseReceipt(wire []byte) (Receipt, error) {
	v, err := ordered(wire, receiptKeys, MaxReceipt)
	if err != nil {
		return Receipt{}, err
	}
	bad := errors.New("malformed_record")
	if _, err := time.Parse("2006-01-02T15:04:05Z", v["installedAt"]); err != nil || v["formatVersion"] != "2" || v["product"] != "Axiom" || !cleanPath(v["destination"]) || !versionPattern.MatchString(v["version"]) || len(v["version"]) > 128 || !fullCommit.MatchString(v["sourceCommit"]) || v["revision"] != v["sourceCommit"][:12] || (v["sourceState"] != "clean" && v["sourceState"] != "dirty") || (v["release"] != "true" && v["release"] != "false") || (v["release"] == "true" && v["sourceState"] != "clean") || !platformTuple(v["platform"], v["goos"], v["architecture"]) || v["skillSetVersion"] != "1" {
		return Receipt{}, bad
	}
	for _, k := range []string{"sha256", "archiveSha256", "skillManifestSha256", "docsGeneration", "docsBindingSha256", "docsInventorySha256", "docsProvenanceSha256", "docsManMapSha256"} {
		if !hexDigest.MatchString(v[k]) {
			return Receipt{}, bad
		}
	}
	if v["docsGeneration"] != Generation(v["docsInventorySha256"], v["docsProvenanceSha256"], v["docsManMapSha256"]) {
		return Receipt{}, bad
	}
	return Receipt{v["destination"], v["sha256"], v["version"], v["revision"], v["sourceCommit"], v["docsGeneration"], v["docsBindingSha256"], v["installedAt"], v["docsInventorySha256"], v["docsProvenanceSha256"], v["docsManMapSha256"], v["archiveSha256"], v["skillManifestSha256"]}, nil
}

type Binding struct {
	SchemaVersion       int    `json:"schemaVersion"`
	BinaryPath          string `json:"binaryPath"`
	ParentDevice        uint64 `json:"parentDevice"`
	ParentInode         uint64 `json:"parentInode"`
	BinaryDevice        uint64 `json:"binaryDevice"`
	BinaryInode         uint64 `json:"binaryInode"`
	BinarySHA256        string `json:"binarySha256"`
	SourceCommit        string `json:"sourceCommit"`
	Version             string `json:"version"`
	ArchiveSHA256       string `json:"archiveSha256"`
	MetadataSHA256      string `json:"metadataSha256"`
	Generation          string `json:"generation"`
	ProvenanceSHA256    string `json:"provenanceSha256"`
	ManMapSHA256        string `json:"manMapSha256"`
	SkillManifestSHA256 string `json:"skillManifestSha256"`
}

func ParseBinding(wire []byte) (Binding, error) {
	var b Binding
	if len(wire) > MaxBinding {
		return b, errors.New("quota_exceeded")
	}
	if err := DecodeClosed(wire, &b, bindingFields); err != nil {
		return Binding{}, err
	}
	for _, n := range []uint64{b.ParentDevice, b.ParentInode, b.BinaryDevice, b.BinaryInode} {
		if n > 1<<53-1 {
			return Binding{}, errors.New("malformed_record")
		}
	}
	if b.SchemaVersion != 1 || !cleanPath(b.BinaryPath) || !versionPattern.MatchString(b.Version) || len(b.Version) > 128 || !fullCommit.MatchString(b.SourceCommit) {
		return Binding{}, errors.New("malformed_record")
	}
	for _, h := range []string{b.BinarySHA256, b.ArchiveSHA256, b.MetadataSHA256, b.Generation, b.ProvenanceSHA256, b.ManMapSHA256, b.SkillManifestSHA256} {
		if !hexDigest.MatchString(h) {
			return Binding{}, errors.New("malformed_record")
		}
	}
	return b, nil
}

func (b Binding) Wire() ([]byte, error) {
	wire, err := json.Marshal(b)
	return append(wire, '\n'), err
}

func (b Binding) Hash() (string, error) {
	wire, err := b.Wire()
	return Hash(wire), err
}
