package app

import "review-desk/terminal/internal/server"

type pullsRead struct {
	pulls server.Pulls
	err   error
}
