package config

// KeybindingDiffConfig is the keys of the diff in the main panel.
type KeybindingDiffConfig struct {
	Down          []string `json:"down"`
	Up            []string `json:"up"`
	HalfDown      []string `json:"halfDown"`
	HalfUp        []string `json:"halfUp"`
	PageDown      []string `json:"pageDown"`
	PageUp        []string `json:"pageUp"`
	Top           []string `json:"top"`
	Bottom        []string `json:"bottom"`
	NextHunk      []string `json:"nextHunk"`
	PreviousHunk  []string `json:"previousHunk"`
	NextFile      []string `json:"nextFile"`
	PreviousFile  []string `json:"previousFile"`
	Search        []string `json:"search"`
	SearchBack    []string `json:"searchBack"`
	NextMatch     []string `json:"nextMatch"`
	PreviousMatch []string `json:"previousMatch"`
	ToggleFold    []string `json:"toggleFold"`
	OpenFold      []string `json:"openFold"`
	CloseFold     []string `json:"closeFold"`
	OpenFolds     []string `json:"openFolds"`
	CloseFolds    []string `json:"closeFolds"`
	Open          []string `json:"open"`
	Back          []string `json:"back"`
	Split         []string `json:"split"`
	Wrap          []string `json:"wrap"`
	Whole         []string `json:"whole"`
	Rendered      []string `json:"rendered"`
	ShowType      []string `json:"showType"`
	NextNote      []string `json:"nextNote"`
	PreviousNote  []string `json:"previousNote"`
	Select        []string `json:"select"`
	OtherEnd      []string `json:"otherEnd"`
	Comment       []string `json:"comment"`
	Ask           []string `json:"ask"`
	Edit          []string `json:"edit"`
	Drop          []string `json:"drop"`
	Stop          []string `json:"stop"`
}
