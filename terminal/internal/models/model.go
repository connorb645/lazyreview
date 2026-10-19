package models

// Model is the change as the apps draw it: its heading, its components and
// areas, the files they hold, the types in them, the links between the
// types, and groups of look-alike types.
type Model struct {
	Heading      Heading     `json:"heading"`
	Components   []Component `json:"components"`
	Areas        []Area      `json:"areas"`
	Files        []FileEntry `json:"files"`
	ContextFiles []FileEntry `json:"contextFiles"`
	Nodes        []Node      `json:"nodes"`
	Edges        []Edge      `json:"edges"`
	Groups       []Group     `json:"groups"`
}
