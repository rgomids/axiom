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
	SkillSetVersion     = "3"
	BinaryCompatibility = "3"
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
	// Eight-skill v0.10.0 revision, retired by the pre-MVP two-skill decision.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":           "5c9b14767e39800e030dedddd115a0281dafc01161e8914ff53eaeb53d0ee3f1",
		"axiom-project-configure": "ec1a0556f930eb05cd52d1902c0d8cb5a30c1890b34e6f375015879f0a949dea",
		"axiom-project-list":      "46960de54c14f904a915ccd7d3ab74284f70a37516a4d87648ec323d96d62045",
		"axiom-project-show":      "4c7ef9c089696d35e455ffe964d37c58e71c34bac1f28d0c464dc5d5989cbbf2",
		"axiom-work-item":         "084c35caf16a4a0e3f1f84209c808454a26474b6a2507103008830de05ce0a8a",
		"axiom-work-item-create":  "c64fb39d06fa59876f0f0cdc487bbcfac91fb7748c77d3e38fa59fb455dab2c0",
		"axiom-work-item-run":     "862bad903caec3048ea82e8096d48323ca98106464319605ee921b83b572bfbf",
		"axiom-work-item-status":  "a370f1b6c48c822b0003679264bee581484cfd4b8a7fd94263e93b280937e223",
	}},
	// Two-skill revision published through v0.12.0, replaced when the Runtime
	// skills began presenting Lingo's human-first Markdown result (#232).
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "64054953abe8bcdac1226b6a1a711879e4cfe7d83379c82f18cd10a445635041",
		"axiom-work-item": "09028588aec1447ec51842ffd7c9ae2b41830b2dd093e087e18c35f61f499a9e",
	}},
	// Workflow authoring revision before human-first integration.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "14cd046e386b7f683386d607b53b64d1a520c54429110ffae7bae7db2566c498",
		"axiom-work-item": "09028588aec1447ec51842ffd7c9ae2b41830b2dd093e087e18c35f61f499a9e",
	}},
	// Human-first presentation revision before workflow authoring integration.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "48d3d7381230e18ce1614ea45ce1109bcf99674b43a6d4348b64514551cb874a",
		"axiom-work-item": "26c26672524258c1a9bdafc2b64fbe06fbe6b2e4558f1dfb6ba98f4baaa7e3aa",
	}},
	// v0.13.0 authoring revision before configured Execution binding (#274).
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "d80cd1d1bca8e88ceb2c61b8ff92d4c050b83835a78ba96e92346c9f2f15b06c",
		"axiom-work-item": "26c26672524258c1a9bdafc2b64fbe06fbe6b2e4558f1dfb6ba98f4baaa7e3aa",
	}},
	// Initial #274 revision, before refusing packet-free final acceptance.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "d80cd1d1bca8e88ceb2c61b8ff92d4c050b83835a78ba96e92346c9f2f15b06c",
		"axiom-work-item": "c6f22b938d196a542f868405db9ad4cd66055e5f1c9f81cdca93c58b1c51d661",
	}},
	// v0.14.0 revision, before the read-only stage planning route (#275).
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "d80cd1d1bca8e88ceb2c61b8ff92d4c050b83835a78ba96e92346c9f2f15b06c",
		"axiom-work-item": "8f3a24324d83ad60a5e786fc8b2f8ca0695b0f420d3af123f38e04817756d0e5",
	}},
	// Published v0.15.0 two-skill set, replaced by the dedicated Workflow
	// conversational surface (#303). Official asset bytes pinned in testdata.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "d80cd1d1bca8e88ceb2c61b8ff92d4c050b83835a78ba96e92346c9f2f15b06c",
		"axiom-work-item": "e47c9256b40975bbcb96015cb681d3d27a6cfed0d1820c51ff98deee8a30cc9f",
	}},
	// Three-skill revision before local Runtime/Profile authoring (#306).
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "14bd0419ff6b050bd2f8bd46a2b5ee2c6a23147e73bd6838f775bc577ab34800",
		"axiom-work-item": "7775bdcad4e01018ab43f06df568a9cb73aeef2e1d5e2737fa894ad4ab0db1d5",
		"axiom-workflow":  "c562d6c4f74601099d71e4090f779ceca7b9a9e073da3ae4108b582d611af77b",
	}},
	// Four-skill revision (#306) before the #276 stage dispatch and
	// coordination routes of axiom-work-item.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "14bd0419ff6b050bd2f8bd46a2b5ee2c6a23147e73bd6838f775bc577ab34800",
		"axiom-runtime":   "d89778accb90d961327108516f3d0fb085e6e870907a4a7540cabc8a20d8594f",
		"axiom-work-item": "7775bdcad4e01018ab43f06df568a9cb73aeef2e1d5e2737fa894ad4ab0db1d5",
		"axiom-workflow":  "bfc1593560df9ad027df8bfc96483ec324c1e05bd74e969c9c4ebe33afff5cf2",
	}},
	// Four-skill revision before canonical v3-only Runtime/Profile activation
	// removed `runtime binding|profile migrate` (Specification 007 PM-11).
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "14bd0419ff6b050bd2f8bd46a2b5ee2c6a23147e73bd6838f775bc577ab34800",
		"axiom-runtime":   "d89778accb90d961327108516f3d0fb085e6e870907a4a7540cabc8a20d8594f",
		"axiom-work-item": "b0ccef7e3e262aeb4ec71fef6a6ba5b868e9df8d1adb214cc3520ef5500934ed",
		"axiom-workflow":  "bfc1593560df9ad027df8bfc96483ec324c1e05bd74e969c9c4ebe33afff5cf2",
	}},
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "14bd0419ff6b050bd2f8bd46a2b5ee2c6a23147e73bd6838f775bc577ab34800",
		"axiom-runtime":   "7a3d90f74a9e5eb3f6580ca0bb9f55b8983db2fda9ea69feca2bae72499fef35",
		"axiom-work-item": "b0ccef7e3e262aeb4ec71fef6a6ba5b868e9df8d1adb214cc3520ef5500934ed",
		"axiom-workflow":  "bfc1593560df9ad027df8bfc96483ec324c1e05bd74e969c9c4ebe33afff5cf2",
	}},
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "14bd0419ff6b050bd2f8bd46a2b5ee2c6a23147e73bd6838f775bc577ab34800",
		"axiom-work-item": "b25efa74ae5ace22d9d9da7b21f9d41c7781cd842a382d0898739ce337900699",
		"axiom-workflow":  "bfc1593560df9ad027df8bfc96483ec324c1e05bd74e969c9c4ebe33afff5cf2",
		"axiom-runtime":   "7a3d90f74a9e5eb3f6580ca0bb9f55b8983db2fda9ea69feca2bae72499fef35",
	}},
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "14bd0419ff6b050bd2f8bd46a2b5ee2c6a23147e73bd6838f775bc577ab34800",
		"axiom-work-item": "982f5a5ae9f21599faf30a24c49debc15edf6d455a108982674ae57f601be7e0",
		"axiom-workflow":  "bfc1593560df9ad027df8bfc96483ec324c1e05bd74e969c9c4ebe33afff5cf2",
		"axiom-runtime":   "7a3d90f74a9e5eb3f6580ca0bb9f55b8983db2fda9ea69feca2bae72499fef35",
	}},
	// AXM-31: retain ownership of the replaced instruction set for both roots.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "14bd0419ff6b050bd2f8bd46a2b5ee2c6a23147e73bd6838f775bc577ab34800",
		"axiom-work-item": "c59e6193ae89a4a4f615728b4a78a79816f13db6d6de6f7716e81ff53df304f0",
		"axiom-workflow":  "bfc1593560df9ad027df8bfc96483ec324c1e05bd74e969c9c4ebe33afff5cf2",
		"axiom-runtime":   "7a3d90f74a9e5eb3f6580ca0bb9f55b8983db2fda9ea69feca2bae72499fef35",
	}},
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "d676740525a254105521e1bf426284cc0a457830183da6ed6f8057dc7a8dfc26",
		"axiom-work-item": "965df8b3435c4afa3fd674ceae86ce98ecae405440a00854478610c74ef18c39",
		"axiom-workflow":  "c7c5dec9abcf5dd133036ef97c69203b52ca7d6aee1aa50dbf9d3616680f4773",
		"axiom-runtime":   "7a3d90f74a9e5eb3f6580ca0bb9f55b8983db2fda9ea69feca2bae72499fef35",
	}},
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "b5ec7bbce1c10109ea4c8ffc902fc04bd7788e8b486d2a3fd5506b3f518a03c5",
		"axiom-runtime":   "7a3d90f74a9e5eb3f6580ca0bb9f55b8983db2fda9ea69feca2bae72499fef35",
		"axiom-work-item": "965df8b3435c4afa3fd674ceae86ce98ecae405440a00854478610c74ef18c39",
		"axiom-workflow":  "c7c5dec9abcf5dd133036ef97c69203b52ca7d6aee1aa50dbf9d3616680f4773",
	}},
	// Retain the origin/main skill set replaced by this merge.
	{skillSetVersion: "2", binaryCompatibility: "2", skills: map[string]string{
		"axiom-project":   "71346c28efa3f0acdd1bebc55d8d803af347d7fbf40c82f5dbc97253cf06f712",
		"axiom-runtime":   "3e563db2e8c02765cbafdda19fd284e81a343849aa44b9ce30f5650c9a947f98",
		"axiom-work-item": "06467f4c06598dcb872cc8710521e350f5435f1161c07079e98a612d4676adaf",
		"axiom-workflow":  "223b0debe6ab24a2c37bcc7ac0c99ab24322d3d24e25e865fb6db80ce41167a9",
	}},
	// Retain the HEAD skill set replaced by this merge.
	{skillSetVersion: "3", binaryCompatibility: "3", skills: map[string]string{
		"axiom-project":   "5c282cedb7ad38f0cc2997ca3a28fa1c996ba7f8e3fe3b0d2efc9d570042961f",
		"axiom-runtime":   "f9ede6421e27df73dd68d477d776c622c56b8a91eac8eeb78a9b11ddf9392930",
		"axiom-work-item": "59f23649a793bb8a793be3753f0e1d25f057b9c94edc62c08f6018c7a03bed91",
		"axiom-workflow":  "ab9192a3c94b03c94431bc8d506569212ab17ba1ded03a4a5c64aa439f9028ab",
	}},
	{skillSetVersion: "3", binaryCompatibility: "3", skills: map[string]string{
		"axiom-project":   "95bbbbb7190759a7c12f262f826f048db50d000fd5480c1764753e8a5345bdfe",
		"axiom-runtime":   "596876d7ad5d36a2feac834475dd7487ef2d72e2aa3bbd88eb0ac9be53ec184a",
		"axiom-work-item": "99e4063d95ec80dccea30efe4c3d8abce8f8302c9d373cee34e124fc8f339b2e",
		"axiom-workflow":  "5148e83308b8927aeb98cc027a6d8cbd0c2c5539b3a335f39cb6b72f2781b36e",
	}},
	// Skill set before the installed-documentation section (AXM-37).
	{skillSetVersion: "3", binaryCompatibility: "3", skills: map[string]string{
		"axiom-project":   "c77779f5e9ee98d824395bbbe586713670516fa96ca26deff23ca764d0693d1a",
		"axiom-runtime":   "f6cb9eb1c86725602e4b2d831c9c2a9753f2016e1444cd283c0fcfb7edfd0497",
		"axiom-work-item": "6c44a83adec8d21e1d2ba776aeda3008ffbe12b9f104634070b0ae925a0d44bd",
		"axiom-workflow":  "40e888a35af2fdd93175ef15cad1c0155eb475a156e3e70584da501d801ca8b3",
	}},
	// Shared revision before AXM-38 rendered Workflow presentation. Both Runtime
	// consumers recognize exactly these historical bytes; compatibility remains 3.
	{skillSetVersion: "3", binaryCompatibility: "3", skills: map[string]string{
		"axiom-project":   "a249ea87133eeefd53f7e6bccf8b5d083c2bf87802f8141ec85183d52d9a892d",
		"axiom-runtime":   "90a725ed0ac96aa3378b3c9860f5357c8ef63b090662a19d362c24c8cc0aad0b",
		"axiom-work-item": "ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3",
		"axiom-workflow":  "580ca950da69046632543523d41447544e979f9f2237278ac39377772fc4f5e5",
	}},
	// Full shared revision replaced by AXM-39 conversational presentation (012).
	{skillSetVersion: "3", binaryCompatibility: "3", skills: map[string]string{
		"axiom-project":   "a249ea87133eeefd53f7e6bccf8b5d083c2bf87802f8141ec85183d52d9a892d",
		"axiom-runtime":   "90a725ed0ac96aa3378b3c9860f5357c8ef63b090662a19d362c24c8cc0aad0b",
		"axiom-work-item": "ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3",
		"axiom-workflow":  "8b47aa65611d28395eab3695ce4ca90773e541800f72d4189838565268a7f235",
	}},
	// Complete four-skill revision replaced by AXM-39 attempt-13 corrections.
	{skillSetVersion: "3", binaryCompatibility: "3", skills: map[string]string{
		"axiom-project":   "775ab63f1c428156cce595cfe587039462d8e0a713b156fcce90f61c479d158d",
		"axiom-runtime":   "165fa61ab665ea819805b2343a00e61b24bce50a32252ee85446039f7b31636d",
		"axiom-work-item": "ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3",
		"axiom-workflow":  "efdfbcd65cdffd3b3e3003e86caa8eb81f904ed9501c8f7bf22139a908154a76",
	}},
	// Complete shared revision replaced by AXM-39 attempt-15 corrections.
	{skillSetVersion: "3", binaryCompatibility: "3", skills: map[string]string{
		"axiom-project":   "52cde467850d0f51f10a219e5e8b0ae69a74bc22b65413501d0ef628a0e63050",
		"axiom-runtime":   "39b21768a263f5702afd7f637e46e333d3cc87b60400bb24371e8cf2035fc02e",
		"axiom-work-item": "ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3",
		"axiom-workflow":  "4df6fc93426cd9532a9f2a7cc2df6d58962c59eb9b0f91f9d032c73140f640b7",
	}},
	// Complete four-skill revision replaced by AXM-39 attempt-16 answer audits.
	{skillSetVersion: "3", binaryCompatibility: "3", skills: map[string]string{
		"axiom-project":   "42b86e7c5cd6a9929545382d640ee1957e60d2ec6b8076551f7ed60b7fc95604",
		"axiom-runtime":   "b35c1b2781f9b2b7939f8a86bf0aa8d6a9741ef9b9ee580b66b2f3573e9252db",
		"axiom-work-item": "ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3",
		"axiom-workflow":  "68e9d8d5b8b5f303e99b7fd7a6c18c7bf964aea6b3e9b1346a1705c6f64f2639",
	}},
	// Complete shared revision published in v0.22.0, replaced by the
	// Specification 013 guided setup and refusal diagnostics.
	{skillSetVersion: "3", binaryCompatibility: "3", skills: map[string]string{
		"axiom-project":   "fddb0c49313c98f5b726acb9eb6356cc6c47c4cebf7a521079758fb1f0b8f647",
		"axiom-runtime":   "aeed4e7cfc8c81c37935d630086ab7920b422421d5876119b72841ec06931bd9",
		"axiom-work-item": "ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3",
		"axiom-workflow":  "20b6d727b97505623df2cf3adb570bb042a5c02a74c05e6ffcf5d196f8d44270",
	}},
	// Before response context headers; retain ownership of the published skills.
	{skillSetVersion: "3", binaryCompatibility: "3", skills: map[string]string{
		"axiom-project":   "b4754fcce56c4af1225664804783ef1044e65bdea3781c2d66cca97afd8f87c5",
		"axiom-runtime":   "8d2e9568fe383417e87e7f1985984cf1f77f43e78b38d5dc2ce07c61e8924c64",
		"axiom-work-item": "ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3",
		"axiom-workflow":  "20b6d727b97505623df2cf3adb570bb042a5c02a74c05e6ffcf5d196f8d44270",
	}},
	// PR #360 initial header guidance, before CR-360-01 qualification refinements.
	{skillSetVersion: "3", binaryCompatibility: "3", skills: map[string]string{
		"axiom-project":   "bed0c51da9b0f8148aa1e14189e5cc28b6903287887f9df9cfe88a94a06f22b9",
		"axiom-runtime":   "23a42d1c29de8811f401531b64d8ffd20b5be659b60c47ddd80155e34b3567cc",
		"axiom-work-item": "044125bd0b9c168bb004c6e7b6b622f02e2065be8f83ffc4accf4cd3a1da3cd1",
		"axiom-workflow":  "f214b4b5836b139214065f61cd3a9e00bd42ba31789137a261134fc3c35ec8c9",
	}},
	// Shared revision before environment-name authentication classification was removed.
	{skillSetVersion: "3", binaryCompatibility: "3", skills: map[string]string{
		"axiom-project":   "4c6e5602212420e9cc23d10a299d6b494c08a1417f4747fc93ca507cd38c257f",
		"axiom-runtime":   "3649c248b036a7864e418b308edd4e83ae86514924738fc57cc9e34e0639b7c1",
		"axiom-work-item": "dfa0a34033dbbf4109f724085e1fed7f813d194362262e68cec683d8ffa6532a",
		"axiom-workflow":  "899abc6a29c6ece448535f844ddaefb7a2419a9f7cfb0533b534b47af89236b7",
	}},
	// Shared revision before operation-scoped inspection and refusal details.
	{skillSetVersion: "3", binaryCompatibility: "3", skills: map[string]string{
		"axiom-project":   "4c6e5602212420e9cc23d10a299d6b494c08a1417f4747fc93ca507cd38c257f",
		"axiom-runtime":   "6ffde6e410088649d59fefff37859aadf0209939907e51d216fef6b0f50ebc8e",
		"axiom-work-item": "dfa0a34033dbbf4109f724085e1fed7f813d194362262e68cec683d8ffa6532a",
		"axiom-workflow":  "899abc6a29c6ece448535f844ddaefb7a2419a9f7cfb0533b534b47af89236b7",
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
