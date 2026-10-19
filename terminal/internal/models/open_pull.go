package models

// OpenPull is one open pull request on the home screen. Asked names whom
// its review is asked of, an account or one of its teams; Yours says an
// account opened it, and Commented that one commented on it or reviewed it.
// Review is the review holding it, or empty.
type OpenPull struct {
	Number      int      `json:"number"`
	Title       string   `json:"title"`
	Author      string   `json:"author"`
	URL         string   `json:"url"`
	Draft       bool     `json:"draft"`
	Base        string   `json:"base"`
	Head        string   `json:"head"`
	Description string   `json:"description"`
	Asked       []string `json:"asked"`
	Yours       bool     `json:"yours"`
	Commented   bool     `json:"commented"`
	Review      string   `json:"review"`
}
