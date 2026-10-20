package testsupport

// FakeCall is one run of the stand-in Claude: its arguments, the question
// on its standard input, the folder it ran in, every file it could read
// there but .git, by path, and what git said of the folder.
type FakeCall struct {
	Args   []string          `json:"args"`
	Prompt string            `json:"prompt"`
	Folder string            `json:"folder"`
	Files  map[string]string `json:"files"`
	Git    FakeGit           `json:"git"`
}
