package operationaldocs

var (
	compiledGeneration   string
	compiledIdentity     string
	compiledCommit       string
	compiledPlatform     string
	compiledArchitecture string
	compiledSkills       string
)

// Compiled returns the descriptor linked at build time; an unlinked build yields
// an invalid zero-value descriptor, which Resolve reports as unqualified.
func Compiled() Descriptor {
	return Descriptor{Generation: compiledGeneration, Identity: compiledIdentity, SourceCommit: compiledCommit, Platform: compiledPlatform, Architecture: compiledArchitecture, SkillsSHA256: compiledSkills}
}
