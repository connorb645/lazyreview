package languages

var (
	swiftSyntax = syntax{
		line:         []string{"//"},
		block:        [][2]string{{"/*", "*/"}},
		nestedBlocks: true,
		rawHashes:    true,
		literals: []literalSyntax{
			{open: `"""`, close: `"""`, escape: `\`, multiline: true, interpolation: `\(`},
			{open: `"`, close: `"`, escape: `\`, interpolation: `\(`},
		},
	}
	kotlinSyntax = syntax{
		line:         []string{"//"},
		block:        [][2]string{{"/*", "*/"}},
		nestedBlocks: true,
		literals: []literalSyntax{
			{open: `"""`, close: `"""`, multiline: true, interpolation: "${"},
			{open: `"`, close: `"`, escape: `\`, interpolation: "${"},
			{open: `'`, close: `'`, escape: `\`},
		},
	}
	javaSyntax = syntax{
		line:  []string{"//"},
		block: [][2]string{{"/*", "*/"}},
		literals: []literalSyntax{
			{open: `"""`, close: `"""`, escape: `\`, multiline: true},
			{open: `"`, close: `"`, escape: `\`},
			{open: `'`, close: `'`, escape: `\`},
		},
	}
	typescriptSyntax = syntax{
		line:  []string{"//"},
		block: [][2]string{{"/*", "*/"}},
		regex: true,
		literals: []literalSyntax{
			{open: "`", close: "`", escape: `\`, multiline: true, interpolation: "${"},
			{open: `"`, close: `"`, escape: `\`},
			{open: `'`, close: `'`, escape: `\`},
		},
	}
	pythonSyntax = syntax{
		line:          []string{"#"},
		prefixLetters: "rRbBuUfF",
		literals: []literalSyntax{
			{open: `"""`, close: `"""`, escape: `\`, multiline: true},
			{open: `'''`, close: `'''`, escape: `\`, multiline: true},
			{open: `"`, close: `"`, escape: `\`},
			{open: `'`, close: `'`, escape: `\`},
		},
	}
)
