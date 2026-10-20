package queue

import (
	"slices"
	"strconv"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

const colours = 8

// Regroup takes a review's queue to the pull requests a load names. A
// member keeps its queued comments, the reviews sent from it and its
// colour. A comment whose pull request's head moved is marked stale, as is
// one on a removed line, or whose range starts on one, whose pull
// request's diff now starts from another commit: its line may mean
// something else now, so no button sends it until it is queued again or
// dropped. A member the load no longer names leaves the stack, kept with
// what it had, unless it is still open with comments queued, which is
// refused. Each absorbed review's comments get new numbers here, and the
// review itself comes back in Moved, emptied and pointing at this one.
func Regroup(tr *i18n.TranslationSet, r Regrouping) (Regrouped, error) {
	var notes []string
	var order []int
	old := map[int]models.ReviewMember{}
	live := map[int]bool{}
	if r.Previous != nil && r.Previous.Moved == nil {
		for _, m := range append(append([]models.ReviewMember{}, r.Previous.Members...), r.Previous.Earlier...) {
			if _, held := old[m.Pull]; !held {
				order = append(order, m.Pull)
			}
			old[m.Pull] = m
		}
		for _, m := range r.Previous.Members {
			live[m.Pull] = true
		}
	}

	members := make([]models.ReviewMember, len(r.Fresh))
	for i, member := range r.Fresh {
		before, held := old[member.Pull]
		delete(old, member.Pull)
		member.Color, member.Comments, member.Posted = 0, []models.Comment{}, []models.PostedReview{}
		if held {
			member.Color, member.Posted = before.Color, before.Posted
			member.Comments = staled(tr, before.Comments, before, member, &notes)
		}
		members[i] = member
	}
	taken := map[int]bool{}
	for _, m := range members {
		if m.Color != 0 {
			taken[m.Color] = true
		}
	}
	for i := range members {
		if members[i].Color != 0 {
			continue
		}
		colour := len(taken)%colours + 1
		for c := 1; c <= colours; c++ {
			if !taken[c] {
				colour = c
				break
			}
		}
		members[i].Color = colour
		taken[colour] = true
	}

	earlier := []models.ReviewMember{}
	for _, pull := range order {
		member, held := old[pull]
		if !held {
			continue
		}
		if !live[pull] {
			earlier = append(earlier, member)
			continue
		}
		state := member.State
		if state == "" {
			state = "open"
		}
		var ended *string
		if leaving, held := r.Leaving[pull]; held {
			state, ended = leaving.State, leaving.EndedAt
		}
		if state == "open" && len(member.Comments) > 0 {
			return Regrouped{}, models.Refused(utils.ResolvePlaceholderString(tr.QueueStillOpen, map[string]string{
				"pull": strconv.Itoa(pull), "count": utils.Plural(len(member.Comments), tr.QueuedCommentNoun),
			}))
		}
		member.State, member.EndedAt = state, ended
		earlier = append(earlier, member)
		notes = append(notes, utils.ResolvePlaceholderString(tr.QueueLeftTheStack, map[string]string{"pull": strconv.Itoa(pull), "state": state}))
	}

	next := NextID(models.Review{Members: members, Earlier: earlier})
	stack := make([]int, len(r.Fresh))
	for i, m := range r.Fresh {
		stack[i] = m.Pull
	}
	moved := []Absorbed{}
	for _, a := range r.Absorbed {
		for _, from := range a.Review.Members {
			i := slices.IndexFunc(members, func(m models.ReviewMember) bool { return m.Pull == from.Pull })
			target := &members[i]
			for _, comment := range staled(tr, from.Comments, from, *target, &notes) {
				was := comment.ID
				comment.ID = next
				target.Comments = append(target.Comments, comment)
				notes = append(notes, utils.ResolvePlaceholderString(tr.QueueRenumbered, map[string]string{
					"pull": strconv.Itoa(from.Pull), "id": strconv.Itoa(was), "next": strconv.Itoa(next),
				}))
				next++
			}
			for _, p := range from.Posted {
				if !slices.ContainsFunc(target.Posted, func(q models.PostedReview) bool { return q.Review == p.Review }) {
					target.Posted = append(target.Posted, p)
				}
			}
		}
		emptied := a.Review
		emptied.Members = make([]models.ReviewMember, len(a.Review.Members))
		for i, m := range a.Review.Members {
			m.Comments = []models.Comment{}
			emptied.Members[i] = m
		}
		emptied.Moved = &models.Moved{To: r.Anchor, Stack: stack, At: r.Now}
		moved = append(moved, Absorbed{Ref: a.Ref, Review: emptied})
	}
	if notes == nil {
		notes = []string{}
	}
	return Regrouped{Members: members, Earlier: earlier, Moved: moved, Notes: notes}, nil
}

func staled(tr *i18n.TranslationSet, comments []models.Comment, before, now models.ReviewMember, notes *[]string) []models.Comment {
	var why string
	var sides []string
	switch {
	case before.Head != now.Head:
		why = utils.ResolvePlaceholderString(tr.QueueStaleHead, map[string]string{"from": utils.Short(before.Head), "pull": strconv.Itoa(now.Pull), "to": utils.Short(now.Head)})
		sides = []string{"RIGHT", "LEFT"}
	case before.Start != "" && now.Start != "" && before.Start != now.Start:
		why = utils.ResolvePlaceholderString(tr.QueueStaleStart, map[string]string{"pull": strconv.Itoa(now.Pull), "now": utils.Short(now.Start), "before": utils.Short(before.Start)})
		sides = []string{"LEFT"}
	default:
		return comments
	}
	touched := func(c models.Comment) bool {
		startSide := c.StartSide
		if startSide == "" {
			startSide = c.Side
		}
		return slices.Contains(sides, c.Side) || (c.StartLine != 0 && slices.Contains(sides, startSide))
	}
	out := make([]models.Comment, len(comments))
	count := 0
	for i, c := range comments {
		if touched(c) {
			count++
			if c.Stale == "" {
				c.Stale = why
			}
		}
		out[i] = c
	}
	if count > 0 {
		*notes = append(*notes, utils.ResolvePlaceholderString(tr.QueueMarkedStale, map[string]string{
			"pull": strconv.Itoa(now.Pull), "count": utils.Plural(count, tr.QueuedCommentNoun), "why": why,
		}))
	}
	return out
}
