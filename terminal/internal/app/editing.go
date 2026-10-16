package app

import "review-desk/terminal/internal/server"

type editing struct {
	purpose      string
	at           server.Lines
	pull         int
	comment      int
	version      string
	conversation int
	text         string
	about        []string
	key          string
}
