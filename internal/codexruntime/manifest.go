package codexruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"sort"
)

const (
	SkillSetVersion     = "2"
	BinaryCompatibility = "2"
	receiptName         = ".axiom-skill-set.receipt"
)

type SkillDigest struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	FormatVersion       int           `json:"formatVersion"`
	SkillSetVersion     string        `json:"skillSetVersion"`
	BinaryCompatibility string        `json:"binaryCompatibility"`
	Skills              []SkillDigest `json:"skills"`
}

func CurrentManifest() (Manifest, error) {
	manifest := Manifest{FormatVersion: 1, SkillSetVersion: SkillSetVersion, BinaryCompatibility: BinaryCompatibility, Skills: make([]SkillDigest, 0, len(skillNames))}
	for _, name := range skillNames {
		content, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if err != nil {
			return Manifest{}, err
		}
		digest := sha256.Sum256(content)
		manifest.Skills = append(manifest.Skills, SkillDigest{Name: name, SHA256: hex.EncodeToString(digest[:])})
	}
	sort.Slice(manifest.Skills, func(i, j int) bool { return manifest.Skills[i].Name < manifest.Skills[j].Name })
	return manifest, nil
}

func ManifestJSON() ([]byte, error) {
	manifest, err := CurrentManifest()
	if err != nil {
		return nil, err
	}
	return json.Marshal(manifest)
}

func manifestDigest() (string, error) {
	wire, err := ManifestJSON()
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:]), nil
}

func receiptBytes() ([]byte, error) {
	digest, err := manifestDigest()
	if err != nil {
		return nil, err
	}
	return []byte("formatVersion=1\nskillSetVersion=" + SkillSetVersion + "\nbinaryCompatibility=" + BinaryCompatibility + "\nmanifestSha256=" + digest + "\n"), nil
}
