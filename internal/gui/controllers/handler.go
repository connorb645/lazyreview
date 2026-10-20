package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/context"
)

// Handler does an action: it takes the state and the count typed before
// the action's keys, and gives the next state and what to run.
type Handler func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd)
