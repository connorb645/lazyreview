package home

type accountFound struct {
	login string
	err   error
	repos []*foundRepo
}
