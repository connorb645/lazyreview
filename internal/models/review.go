package models

import (
	"bytes"
	"encoding/json"
)

// Review is a review as its review.json keeps it. A pull request review
// holds its pull requests, bottom first, in Members, and those that left its
// stack in Earlier, with the account it acts as and the login GitHub
// answered as. A branch review, Kind "branch", holds the folder it was read
// from, its base and its head, and takes no comments. A stack of branches
// is a branch review too: Branches holds its branches, bottom first, and
// Head is its top's.
type Review struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"`
	Host       string         `json:"host"`
	Repo       string         `json:"repo"`
	Path       string         `json:"path"`
	Account    *string        `json:"account"`
	Me         string         `json:"me"`
	Title      string         `json:"title"`
	GivenTitle *string        `json:"given_title"`
	Base       Ref            `json:"base"`
	Head       Ref            `json:"head"`
	Start      string         `json:"start"`
	Context    bool           `json:"context"`
	LoadedAt   string         `json:"loaded_at"`
	Notes      []string       `json:"notes"`
	Moved      *Moved         `json:"moved"`
	Members    []ReviewMember `json:"members"`
	Earlier    []ReviewMember `json:"earlier"`
	Branches   []BranchMember `json:"branches"`
}

// IsBranch says the review is of a branch with no pull request, or of a
// stack of them.
func (r Review) IsBranch() bool {
	return r.Kind == "branch"
}

// BranchNames are the branches a branch review reads, bottom first: its
// stack's, or its one branch.
func (r Review) BranchNames() []string {
	if len(r.Branches) == 0 {
		return []string{r.Head.Ref}
	}
	names := make([]string, len(r.Branches))
	for i, b := range r.Branches {
		names[i] = b.Branch
	}
	return names
}

// MarshalJSON writes the review in its kind's own shape, each field in the
// place every earlier version wrote it. A branch review writes its
// branches only when it is a stack, so one branch's review reads as it
// always has.
func (r Review) MarshalJSON() ([]byte, error) {
	if r.IsBranch() {
		return plainJSON(struct {
			ID         string         `json:"id"`
			Kind       string         `json:"kind"`
			Repo       string         `json:"repo"`
			Path       string         `json:"path"`
			Title      string         `json:"title"`
			GivenTitle *string        `json:"given_title"`
			Base       Ref            `json:"base"`
			Head       Ref            `json:"head"`
			Start      string         `json:"start"`
			Context    bool           `json:"context"`
			LoadedAt   string         `json:"loaded_at"`
			Notes      []string       `json:"notes"`
			Branches   []BranchMember `json:"branches,omitempty"`
		}{r.ID, r.Kind, r.Repo, r.Path, r.Title, r.GivenTitle, r.Base, r.Head, r.Start, r.Context, r.LoadedAt, orNone(r.Notes), r.Branches})
	}
	return plainJSON(struct {
		ID       string         `json:"id"`
		Host     string         `json:"host"`
		Repo     string         `json:"repo"`
		Account  *string        `json:"account"`
		Me       string         `json:"me"`
		Title    string         `json:"title"`
		Base     Ref            `json:"base"`
		Context  bool           `json:"context"`
		LoadedAt string         `json:"loaded_at"`
		Notes    []string       `json:"notes"`
		Moved    *Moved         `json:"moved"`
		Members  []ReviewMember `json:"members"`
		Earlier  []ReviewMember `json:"earlier"`
	}{r.ID, r.Host, r.Repo, r.Account, r.Me, r.Title, r.Base, r.Context, r.LoadedAt, orNone(r.Notes), r.Moved, orNone(r.Members), orNone(r.Earlier)})
}

func orNone[T any](list []T) []T {
	if list == nil {
		return []T{}
	}
	return list
}

// plainJSON is v as JSON with its text as it reads, < > and & left as they
// are, so a record stays readable.
func plainJSON(v any) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(out.Bytes(), "\n"), nil
}
