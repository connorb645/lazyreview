package server

// Model is the change as the apps draw it: its heading, its components and
// the files they hold.
type Model struct {
	Heading      Heading     `json:"heading"`
	Components   []Component `json:"components"`
	Files        []FileEntry `json:"files"`
	ContextFiles []FileEntry `json:"contextFiles"`
}
