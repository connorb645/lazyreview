package models

// Row is one row of a hunk: Kind is add, del, same or note; Old and New are
// its line numbers on each side, 0 on the side it is not on. On a stack,
// Owner is the pull request that changed an added or removed row, and Words
// say who changed it and where a comment on it goes.
type Row struct {
	Kind  string `json:"kind"`
	Old   int    `json:"old"`
	New   int    `json:"new"`
	Text  string `json:"text"`
	Owner int    `json:"owner"`
	Words string `json:"words"`
}
