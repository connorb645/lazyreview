package server

// Record is the server's own record of itself, in server.json: where it
// answers, and the token a client sends with each change.
type Record struct {
	Address string `json:"address"`
	Token   string `json:"token"`
}
