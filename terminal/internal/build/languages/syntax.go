package languages

type syntax struct {
	line          []string
	block         [][2]string
	nestedBlocks  bool
	literals      []literalSyntax
	rawHashes     bool
	regex         bool
	prefixLetters string
}
