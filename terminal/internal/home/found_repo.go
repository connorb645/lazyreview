package home

import "github.com/connorb645/lazyreview/terminal/internal/models"

type foundRepo struct {
	name     string
	account  string
	pulls    []models.GitHubPull
	numbers  []int
	involved map[int]involved
}
