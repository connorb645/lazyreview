package app

import "review-desk/terminal/internal/server"

type checked struct {
	e     editing
	where server.Where
	err   error
}
