package config

// KeybindingConfig is the config's keybinding: the keys of each place, by
// action, as lazygit's keybinding is. Each action has a list of key
// sequences, named as keys.Reader names them, such as "g g", "space ?" or
// "ctrl+o"; the first is the one the help and the bar show, and an empty
// list leaves the action without a key.
type KeybindingConfig struct {
	Universal    KeybindingUniversalConfig    `json:"universal"`
	Home         KeybindingHomeConfig         `json:"home"`
	ReviewScreen KeybindingReviewScreenConfig `json:"reviewScreen"`
	Panels       KeybindingPanelsConfig       `json:"panels"`
	ReviewPanel  KeybindingReviewPanelConfig  `json:"reviewPanel"`
	FilesPanel   KeybindingFilesPanelConfig   `json:"filesPanel"`
	NotesPanel   KeybindingNotesPanelConfig   `json:"notesPanel"`
	DiagramPanel KeybindingDiagramPanelConfig `json:"diagramPanel"`
	ReviewView   KeybindingReviewViewConfig   `json:"reviewView"`
	Diff         KeybindingDiffConfig         `json:"diff"`
	Diagram      KeybindingDiagramConfig      `json:"diagram"`
	Menu         KeybindingMenuConfig         `json:"menu"`
	Finder       KeybindingFinderConfig       `json:"finder"`
	Posting      KeybindingPostingConfig      `json:"posting"`
	Confirmation KeybindingConfirmationConfig `json:"confirmation"`
}
