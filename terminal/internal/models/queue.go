package models

// Queue is a review's comments waiting to go to GitHub, one group per pull
// request, and the reviews sent from them. A branch's review has none.
type Queue struct {
	Repo    string        `json:"repo"`
	Members []QueueMember `json:"members"`
	Earlier []QueueMember `json:"earlier"`
	Moved   *Moved        `json:"moved"`
}
