package server

// Hidden is the config's hidden repositories after a change: the config
// file's path, and each repository asked about with what now hides it.
type Hidden struct {
	Config string       `json:"config"`
	Repos  []HiddenRepo `json:"repos"`
}
