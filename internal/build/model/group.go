package model

type group struct {
	id          string
	title       string
	description string
	members     []string
	target      string
	kind        string
	component   string
	pkg         string
}
