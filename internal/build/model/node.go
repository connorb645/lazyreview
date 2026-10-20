// Package model puts a change's diagram together: the types each changed
// file declares, the links between them, the unchanged types drawn around
// them, the components and packages they sit in, and the groups and arrows
// that keep a large change readable.
package model

import "github.com/connorb645/lazyreview/internal/models"

const unplaced = int(^uint(0) >> 1)

// Node is a type to draw, or a box for a file with no types, or for a
// file's extensions: what its code declares, its masked code, which links
// are read from, its language family, so a name only resolves within one,
// and once placed, where it sits and what state the change leaves it in.
type Node struct {
	ID         string
	Name       string
	Kind       string
	Label      string
	File       string
	Line       int
	End        int
	Family     string
	Inherits   []string
	Properties []models.Property
	Nested     []string
	View       bool
	Test       bool
	Code       string
	ExtFiles   []string
	Extends    []string
	State      string
	Group      string
	Component  string
	Package    string
	Pulls      []int
	Stereotype string
}
