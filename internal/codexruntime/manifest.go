package codexruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

// skillSetRevision is one skill set as an Axiom release published it: the
// skill-set and binary compatibility versions and each SKILL.md digest.
type skillSetRevision struct {
	skillSetVersion     string
	binaryCompatibility string
	skills              map[string]string
}

// sharedSkillHistory lists, oldest first, every earlier skill set that Axiom
// published through the shared Runtime integration (S9/T40 onward). Every
// Runtime installer recognizes these revisions, and the receipts derived from
// them, as Axiom-owned, so a binary whose skill text changed converges each
// Runtime root it finds instead of only the root `axiom upgrade` publishes.
// When the embedded skill text changes, append the revision being replaced
// here; never extend one Runtime's own history instead.
var sharedSkillHistory = []skillSetRevision{
	// First shared skill set (S9/T40), published through v0.1.2-rc.1; replaced
	// when axiom-work-item-run began selecting the Execution Runtime explicitly.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project-configure": "b0d16b482ca3a08c20f8c5d7154573db825a563f458b3bf5284570fa263e4ad3",
		"axiom-project-show":      "048975860d658e8134eb498c7cd5158104338a2629dd8a8cda4a2ecb0bb99c09",
		"axiom-work-item-create":  "85e4f4badff647a337acf59e96d07909c9a3c47ce5e8df65fac08395c8aac8c4",
		"axiom-work-item-run":     "f0fd487fbbfcfb87bee9906244fb1fedbc0dd5bcbd1fb01594dc3df0c8e5e897",
		"axiom-work-item-status":  "6a139090ff66759b64e630181373c32ff51ad2672d90d26184b6f81e9879b7d9",
	}},
	// Last five-skill revision, replaced when Project listing added the first
	// additive shared Runtime entrypoint.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project-configure": "b0d16b482ca3a08c20f8c5d7154573db825a563f458b3bf5284570fa263e4ad3",
		"axiom-project-show":      "048975860d658e8134eb498c7cd5158104338a2629dd8a8cda4a2ecb0bb99c09",
		"axiom-work-item-create":  "85e4f4badff647a337acf59e96d07909c9a3c47ce5e8df65fac08395c8aac8c4",
		"axiom-work-item-run":     "3856374198506e8d6628dc76dda6058229382346115dca3a838a9bfb439f60d9",
		"axiom-work-item-status":  "6a139090ff66759b64e630181373c32ff51ad2672d90d26184b6f81e9879b7d9",
	}},
	// Replaced when issue #136 added creation classification and story value.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project-configure": "b0d16b482ca3a08c20f8c5d7154573db825a563f458b3bf5284570fa263e4ad3",
		"axiom-project-list":      "d1537221016ddb634a44b3baa16e321aad6c826f07e50bba61cbc1f355de6ea5",
		"axiom-project-show":      "048975860d658e8134eb498c7cd5158104338a2629dd8a8cda4a2ecb0bb99c09",
		"axiom-work-item-create":  "85e4f4badff647a337acf59e96d07909c9a3c47ce5e8df65fac08395c8aac8c4",
		"axiom-work-item-run":     "3856374198506e8d6628dc76dda6058229382346115dca3a838a9bfb439f60d9",
		"axiom-work-item-status":  "6a139090ff66759b64e630181373c32ff51ad2672d90d26184b6f81e9879b7d9",
	}},
	// Shared revision before the minimal-intent interview (#134).
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project-configure": "b0d16b482ca3a08c20f8c5d7154573db825a563f458b3bf5284570fa263e4ad3",
		"axiom-project-list":      "d1537221016ddb634a44b3baa16e321aad6c826f07e50bba61cbc1f355de6ea5",
		"axiom-project-show":      "048975860d658e8134eb498c7cd5158104338a2629dd8a8cda4a2ecb0bb99c09",
		"axiom-work-item-create":  "4db778e4b4a3c56ecb654b425c5e3685ae2a7fa920b29eb4fc6e71cfede9229b",
		"axiom-work-item-run":     "3856374198506e8d6628dc76dda6058229382346115dca3a838a9bfb439f60d9",
		"axiom-work-item-status":  "6a139090ff66759b64e630181373c32ff51ad2672d90d26184b6f81e9879b7d9",
	}},
	// Shared revision before read-only skill argument discovery (#131).
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project-configure": "b0d16b482ca3a08c20f8c5d7154573db825a563f458b3bf5284570fa263e4ad3",
		"axiom-project-list":      "d1537221016ddb634a44b3baa16e321aad6c826f07e50bba61cbc1f355de6ea5",
		"axiom-project-show":      "048975860d658e8134eb498c7cd5158104338a2629dd8a8cda4a2ecb0bb99c09",
		"axiom-work-item-create":  "49f2dbf1e3d629bc240f66d1363912929b382111cee46ec30677999c4aca4e2b",
		"axiom-work-item-run":     "3856374198506e8d6628dc76dda6058229382346115dca3a838a9bfb439f60d9",
		"axiom-work-item-status":  "6a139090ff66759b64e630181373c32ff51ad2672d90d26184b6f81e9879b7d9",
	}},
	// Earlier gate-action branch revision, retained for owned upgrades.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project-configure": "ec1a0556f930eb05cd52d1902c0d8cb5a30c1890b34e6f375015879f0a949dea",
		"axiom-project-list":      "46960de54c14f904a915ccd7d3ab74284f70a37516a4d87648ec323d96d62045",
		"axiom-project-show":      "4c7ef9c089696d35e455ffe964d37c58e71c34bac1f28d0c464dc5d5989cbbf2",
		"axiom-work-item-create":  "c64fb39d06fa59876f0f0cdc487bbcfac91fb7748c77d3e38fa59fb455dab2c0",
		"axiom-work-item-run":     "2f3eb6e35794af4a17b7040a49b2ec462cc0d29ec6f4eeb760d1cfc6cf7aa6e8",
		"axiom-work-item-status":  "d7316123e5ec5aec11fdc5f1a6c12da0974d18047c96f8b1195941a0bb6956bc",
	}},

	// Six-skill revision immediately before domain-oriented Runtime surfaces
	// (#229). The operation-specific skills stay compatible in the next
	// revision, but this receipt must remain recognized during upgrade.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project-configure": "ec1a0556f930eb05cd52d1902c0d8cb5a30c1890b34e6f375015879f0a949dea",
		"axiom-project-list":      "46960de54c14f904a915ccd7d3ab74284f70a37516a4d87648ec323d96d62045",
		"axiom-project-show":      "4c7ef9c089696d35e455ffe964d37c58e71c34bac1f28d0c464dc5d5989cbbf2",
		"axiom-work-item-create":  "c64fb39d06fa59876f0f0cdc487bbcfac91fb7748c77d3e38fa59fb455dab2c0",
		"axiom-work-item-run":     "2e402473610b4945ca075892587cbf879f259410e62493ce72e2ad3bd638b3e3",
		"axiom-work-item-status":  "d7316123e5ec5aec11fdc5f1a6c12da0974d18047c96f8b1195941a0bb6956bc",
	}},
	// Eight-skill revision before the reviewed Runtime/Profile preview start
	// (#140) changed axiom-work-item-run and axiom-work-item.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":           "0ce8ca0f82a468286b6937e5bd3a7ba2972e016722c0d6842eb7568ef0cba022",
		"axiom-project-configure": "ec1a0556f930eb05cd52d1902c0d8cb5a30c1890b34e6f375015879f0a949dea",
		"axiom-project-list":      "46960de54c14f904a915ccd7d3ab74284f70a37516a4d87648ec323d96d62045",
		"axiom-project-show":      "4c7ef9c089696d35e455ffe964d37c58e71c34bac1f28d0c464dc5d5989cbbf2",
		"axiom-work-item":         "19c74b98697452f40133b803ad7f326753cd518c2fd4177c68835ec5d92a2a8e",
		"axiom-work-item-create":  "c64fb39d06fa59876f0f0cdc487bbcfac91fb7748c77d3e38fa59fb455dab2c0",
		"axiom-work-item-run":     "2e402473610b4945ca075892587cbf879f259410e62493ce72e2ad3bd638b3e3",
		"axiom-work-item-status":  "d7316123e5ec5aec11fdc5f1a6c12da0974d18047c96f8b1195941a0bb6956bc",
	}},
	// Domain skill revision before gate-action integration.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":           "0ce8ca0f82a468286b6937e5bd3a7ba2972e016722c0d6842eb7568ef0cba022",
		"axiom-work-item":         "fbba0785a36981950f0cf32718da693e790f9594fc2570e242b12880b488d44d",
		"axiom-project-configure": "ec1a0556f930eb05cd52d1902c0d8cb5a30c1890b34e6f375015879f0a949dea",
		"axiom-project-list":      "46960de54c14f904a915ccd7d3ab74284f70a37516a4d87648ec323d96d62045",
		"axiom-project-show":      "4c7ef9c089696d35e455ffe964d37c58e71c34bac1f28d0c464dc5d5989cbbf2",
		"axiom-work-item-create":  "c64fb39d06fa59876f0f0cdc487bbcfac91fb7748c77d3e38fa59fb455dab2c0",
		"axiom-work-item-run":     "df41a67f314560cd628b3d0aa50bbfad7f4164b2075368a895a5297549532cc7",
		"axiom-work-item-status":  "d7316123e5ec5aec11fdc5f1a6c12da0974d18047c96f8b1195941a0bb6956bc",
	}},
	// Eight-skill revision published by v0.9.0, replaced when the #230
	// resource lifecycle operations changed axiom-project, axiom-work-item
	// and axiom-work-item-status.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":           "0ce8ca0f82a468286b6937e5bd3a7ba2972e016722c0d6842eb7568ef0cba022",
		"axiom-project-configure": "ec1a0556f930eb05cd52d1902c0d8cb5a30c1890b34e6f375015879f0a949dea",
		"axiom-project-list":      "46960de54c14f904a915ccd7d3ab74284f70a37516a4d87648ec323d96d62045",
		"axiom-project-show":      "4c7ef9c089696d35e455ffe964d37c58e71c34bac1f28d0c464dc5d5989cbbf2",
		"axiom-work-item":         "68eb9081dc33302f12f961d93b876f97cb407d46661ad2477e20f4595f961de0",
		"axiom-work-item-create":  "c64fb39d06fa59876f0f0cdc487bbcfac91fb7748c77d3e38fa59fb455dab2c0",
		"axiom-work-item-run":     "2afb4b64b4bda2042a3ab1dbad64d052e21f7535c7f110cf0254ce3e0f9f15a2",
		"axiom-work-item-status":  "d7316123e5ec5aec11fdc5f1a6c12da0974d18047c96f8b1195941a0bb6956bc",
	}},
	// Complete execution targeting branch revision, retained for owned upgrades.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project-configure": "ec1a0556f930eb05cd52d1902c0d8cb5a30c1890b34e6f375015879f0a949dea",
		"axiom-project-list":      "46960de54c14f904a915ccd7d3ab74284f70a37516a4d87648ec323d96d62045",
		"axiom-project-show":      "4c7ef9c089696d35e455ffe964d37c58e71c34bac1f28d0c464dc5d5989cbbf2",
		"axiom-project":           "0ce8ca0f82a468286b6937e5bd3a7ba2972e016722c0d6842eb7568ef0cba022",
		"axiom-work-item-create":  "c64fb39d06fa59876f0f0cdc487bbcfac91fb7748c77d3e38fa59fb455dab2c0",
		"axiom-work-item-run":     "862bad903caec3048ea82e8096d48323ca98106464319605ee921b83b572bfbf",
		"axiom-work-item-status":  "d7316123e5ec5aec11fdc5f1a6c12da0974d18047c96f8b1195941a0bb6956bc",
		"axiom-work-item":         "0c657d63a6bbd31956b3fa85113a3da7dc50ef633b66d16696621b29e7d36374",
	}},
	// Complete resource lifecycle branch revision, retained for owned upgrades.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project-configure": "ec1a0556f930eb05cd52d1902c0d8cb5a30c1890b34e6f375015879f0a949dea",
		"axiom-project-list":      "46960de54c14f904a915ccd7d3ab74284f70a37516a4d87648ec323d96d62045",
		"axiom-project-show":      "4c7ef9c089696d35e455ffe964d37c58e71c34bac1f28d0c464dc5d5989cbbf2",
		"axiom-project":           "5c9b14767e39800e030dedddd115a0281dafc01161e8914ff53eaeb53d0ee3f1",
		"axiom-work-item-create":  "c64fb39d06fa59876f0f0cdc487bbcfac91fb7748c77d3e38fa59fb455dab2c0",
		"axiom-work-item-run":     "2afb4b64b4bda2042a3ab1dbad64d052e21f7535c7f110cf0254ce3e0f9f15a2",
		"axiom-work-item-status":  "a370f1b6c48c822b0003679264bee581484cfd4b8a7fd94263e93b280937e223",
		"axiom-work-item":         "5c471c81c8e2cf5103ab551f07de8238616937fe9d9f6e00472792a82c2ee878",
	}},
}

// currentRevision is the skill set embedded in this binary.
func currentRevision() (skillSetRevision, error) {
	revision := skillSetRevision{skillSetVersion: SkillSetVersion, binaryCompatibility: BinaryCompatibility, skills: make(map[string]string, len(skillNames))}
	for _, name := range skillNames {
		content, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if err != nil {
			return skillSetRevision{}, err
		}
		revision.skills[name] = digestOf(content)
	}
	return revision, nil
}

// names lists the revision's own skills in canonical (sorted) order, which is
// the order every release wrote them in.
func (r skillSetRevision) names() []string {
	names := make([]string, 0, len(r.skills))
	for name := range r.skills {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r skillSetRevision) nameSet() map[string]bool {
	names := make(map[string]bool, len(r.skills))
	for name := range r.skills {
		names[name] = true
	}
	return names
}

func (r skillSetRevision) manifest() Manifest {
	names := r.names()
	manifest := Manifest{FormatVersion: 1, SkillSetVersion: r.skillSetVersion, BinaryCompatibility: r.binaryCompatibility, Skills: make([]SkillDigest, 0, len(names))}
	for _, name := range names {
		manifest.Skills = append(manifest.Skills, SkillDigest{Name: name, SHA256: r.skills[name]})
	}
	return manifest
}

func (r skillSetRevision) manifestDigest() (string, error) {
	if len(r.skills) == 0 {
		return "", errors.New("incomplete skill set revision")
	}
	for _, digest := range r.skills {
		if digest == "" {
			return "", errors.New("incomplete skill set revision")
		}
	}
	wire, err := json.Marshal(r.manifest())
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:]), nil
}

func CurrentManifest() (Manifest, error) {
	revision, err := currentRevision()
	if err != nil {
		return Manifest{}, err
	}
	return revision.manifest(), nil
}

func ManifestJSON() ([]byte, error) {
	manifest, err := CurrentManifest()
	if err != nil {
		return nil, err
	}
	return json.Marshal(manifest)
}

func receiptBytes() ([]byte, error) {
	revision, err := currentRevision()
	if err != nil {
		return nil, err
	}
	return codexReceiptBytes("", revision)
}

// codexReceiptBytes is the Codex skill-set receipt of one revision; the
// Codex receipt does not bind its root.
func codexReceiptBytes(_ string, revision skillSetRevision) ([]byte, error) {
	digest, err := revision.manifestDigest()
	if err != nil {
		return nil, err
	}
	return []byte("formatVersion=1\nskillSetVersion=" + revision.skillSetVersion + "\nbinaryCompatibility=" + revision.binaryCompatibility + "\nmanifestSha256=" + digest + "\n"), nil
}
