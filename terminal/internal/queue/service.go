package queue

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/commands/github"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/review"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// Service changes a review's queue, and sends a review to GitHub from it.
// Every change names build, the build id of the page its window was drawn
// from, and is refused once the review was loaded again since: that
// window's line numbers may name other code now.
type Service struct {
	tr     *i18n.TranslationSet
	store  *store.ReviewStore
	github GitHubFor
	now    func() string
}

// NewService changes the queues of store's reviews, talking to GitHub
// through github and stamping a review sent with now.
func NewService(tr *i18n.TranslationSet, reviews *store.ReviewStore, gh GitHubFor, now func() string) *Service {
	return &Service{tr: tr, store: reviews, github: gh, now: now}
}

// Where says where a comment on lines the app shows would go, as Target
// does, or why none can, for the page build the app drew from.
func (s *Service) Where(ref models.ReviewRef, at models.Lines, pull int, build string) (models.Where, error) {
	rev, err := s.store.Read(ref)
	if err != nil || rev == nil {
		return models.Where{}, s.noReview(ref, err)
	}
	drawn, err := s.store.ReadPage(ref)
	if err != nil {
		return models.Where{}, err
	}
	current := ""
	if drawn != nil {
		current = drawn.Build
	}
	if problem := store.BuildProblem(s.tr, current, build); problem != "" {
		return models.Where{Refused: capital(problem) + "."}, nil
	}
	return Target(s.tr, *rev, drawn.Page, at, pull), nil
}

// Add queues a comment on lines as the app shows them, once GitHub confirms
// the line is in that pull request's diff and its head has not moved,
// quoting the code of its lines as the page shows them.
func (s *Service) Add(ref models.ReviewRef, at models.Lines, pull int, body, build string) error {
	return s.change(ref, build, s.tr.NotQueued, true, func(rev *models.Review, gh *github.GitHub) (*Posting, error) {
		drawn, err := s.store.ReadPage(ref)
		if err != nil {
			return nil, err
		}
		where := Target(s.tr, *rev, drawn.Page, at, pull)
		if where.Goes == nil {
			why := where.Refused
			if why == "" {
				why = where.Words
			}
			return nil, s.refused(s.tr.NotQueued, why)
		}
		target := *where.Goes
		member, err := LiveMember(s.tr, rev, target.Pull)
		if err != nil {
			return nil, err
		}
		if problems := BodyProblems(s.tr, body); len(problems) > 0 {
			return nil, s.refused(s.tr.NotQueued, problems...)
		}
		now, err := gh.Pull(rev.Repo, target.Pull)
		if err != nil {
			return nil, err
		}
		if now.Head.Sha != member.Head {
			return nil, s.refused(s.tr.NotQueued, utils.ResolvePlaceholderString(s.tr.HeadMoved, map[string]string{"pull": strconv.Itoa(target.Pull), "from": prefix(member.Head, 8), "to": prefix(now.Head.Sha, 8)}))
		}
		places, err := filePlaces(gh, rev.Repo, target.Pull)
		if err != nil {
			return nil, err
		}
		comment := models.Comment{ID: NextID(*rev), Path: target.Path, Line: target.Line, Side: target.Side, Body: strings.TrimSpace(body), Code: review.QuotedCode(drawn.Page, at)}
		if target.StartLine != 0 {
			comment.StartLine, comment.StartSide = target.StartLine, target.StartSide
		}
		onPull := 0
		if drawn.Page.Stack != nil {
			onPull = target.Pull
		}
		if problem := PlaceProblem(s.tr, placesOf(places, target.Path), comment); problem != "" {
			placed := utils.ResolvePlaceholderString(s.tr.Placed, map[string]string{"at": commentAt(s.tr, target.Path, target.StartLine, target.Line, onPull), "problem": problem})
			return nil, s.refused(s.tr.NotQueued, placed)
		}
		if drawn.Page.Stack != nil {
			shown := models.Lines{Path: at.Path, Line: at.Line, Side: at.Side}
			if target.StartLine != 0 {
				shown.StartLine, shown.StartSide = at.StartLine, target.StartSide
			}
			comment.Shown = &shown
		}
		member.Comments = append(member.Comments, comment)
		return nil, nil
	})
}

// Edit replaces a queued comment's text, refused once it changed since
// version.
func (s *Service) Edit(ref models.ReviewRef, id int, body, version, build string) error {
	if problems := BodyProblems(s.tr, body); len(problems) > 0 {
		return s.refused(s.tr.NotChanged, problems...)
	}
	return s.change(ref, build, s.tr.NotChanged, false, func(rev *models.Review, _ *github.GitHub) (*Posting, error) {
		member, k, err := FindComment(s.tr, rev, id)
		if err != nil {
			return nil, err
		}
		if _, err := LiveMember(s.tr, rev, member.Pull); err != nil {
			return nil, err
		}
		if err := CheckVersion(s.tr, member.Comments[k], version, s.tr.NotChanged); err != nil {
			return nil, err
		}
		member.Comments[k].Body = strings.TrimSpace(body)
		return nil, nil
	})
}

// Drop takes a queued comment off, refused once it changed since version.
func (s *Service) Drop(ref models.ReviewRef, id int, version, build string) error {
	return s.change(ref, build, s.tr.NotDropped, false, func(rev *models.Review, _ *github.GitHub) (*Posting, error) {
		member, k, err := FindComment(s.tr, rev, id)
		if err != nil {
			return nil, err
		}
		if err := CheckVersion(s.tr, member.Comments[k], version, s.tr.NotDropped); err != nil {
			return nil, err
		}
		member.Comments = append(member.Comments[:k:k], member.Comments[k+1:]...)
		return nil, nil
	})
}

// Post is the only way a review reaches GitHub. It refuses, and posts
// nothing, unless the queue still gives exactly set for action, the pull
// request is open, its head has not moved, every comment sits on a line of
// its diff and none is stale, and the account is the one the review was
// loaded as. Before posting it looks for a review of the same account at
// this head that already does the same thing, so a second press, or one
// whose answer was lost, never posts twice. GitHub creates a review and
// its comments in one request, so a failure there posts nothing.
func (s *Service) Post(ref models.ReviewRef, pull int, action, set, build string) (Posting, error) {
	var done Posting
	err := s.change(ref, build, s.tr.NothingWasPosted, true, func(rev *models.Review, gh *github.GitHub) (*Posting, error) {
		posting, err := s.post(rev, gh, pull, action, set)
		if posting != nil {
			done = *posting
		}
		return posting, err
	})
	return done, err
}

func (s *Service) post(rev *models.Review, gh *github.GitHub, pull int, action, set string) (*Posting, error) {
	if _, ok := ActionOf[action]; !ok {
		return nil, models.Refused(utils.ResolvePlaceholderString(s.tr.NotAnAction, map[string]string{"action": action}))
	}
	member, err := LiveMember(s.tr, rev, pull)
	if err != nil {
		return nil, err
	}
	if !contains(Offered(*rev, *member), action) {
		return nil, models.Refused(utils.ResolvePlaceholderString(s.tr.NotOffered, map[string]string{"action": action, "pull": strconv.Itoa(pull)}))
	}
	if current := SetID(*rev, *member, action); set != current {
		return nil, models.Refused(utils.ResolvePlaceholderString(s.tr.OtherSet, map[string]string{"shown": set, "action": action, "now": current}))
	}
	me, err := gh.Login()
	if err != nil {
		return nil, err
	}
	if me != rev.Me {
		return nil, models.Refused(utils.ResolvePlaceholderString(s.tr.OtherAccount, map[string]string{"now": me, "loaded": rev.Me}))
	}
	earlier, err := alreadyPosted(gh, *rev, *member, action, me)
	if err != nil {
		return nil, err
	}
	if earlier != nil {
		if !postedAs(*member, earlier.HTMLURL) {
			record(member, *rev, action, *earlier, s.now())
		}
		return &Posting{Message: utils.ResolvePlaceholderString(s.tr.AlreadyDone, map[string]string{"url": earlier.HTMLURL}), Review: earlier.HTMLURL}, nil
	}
	comments := Sent(*member, action)
	problems, now, err := s.problemsIn(gh, *rev, *member, comments)
	if err != nil {
		return nil, err
	}
	if ActionOf[action].Event == "APPROVE" && now.User.Login == me {
		problems = append(problems, s.tr.OwnPull)
	}
	if len(problems) > 0 {
		return nil, s.refused(s.tr.NothingWasPosted, problems...)
	}
	payload := map[string]any{"commit_id": member.Head, "event": ActionOf[action].Event}
	if len(comments) > 0 {
		fields := make([]PostedFields, len(comments))
		for i, c := range comments {
			fields[i] = Posted(c)
		}
		payload["comments"] = fields
	}
	posted, err := gh.CreateReview(rev.Repo, pull, payload)
	if err != nil {
		return nil, err
	}
	record(member, *rev, action, posted, s.now())
	back, err := gh.ReviewComments(rev.Repo, pull)
	if err != nil {
		return nil, err
	}
	count := 0
	for _, c := range back {
		if c.ReviewID == posted.ID {
			count++
		}
	}
	words := map[string]string{"comments": utils.Plural(len(comments), s.tr.CommentNoun), "url": posted.HTMLURL, "state": posted.State, "back": strconv.Itoa(count)}
	message := utils.ResolvePlaceholderString(postedWords(s.tr, action), words)
	if posted.State != ActionOf[action].State || count != len(comments) {
		message = utils.ResolvePlaceholderString(s.tr.ReadBack, words)
	}
	return &Posting{Message: message, Review: posted.HTMLURL}, nil
}

func (s *Service) change(ref models.ReviewRef, build, refusedWords string, online bool, work func(*models.Review, *github.GitHub) (*Posting, error)) error {
	return s.store.Change([]models.ReviewRef{ref}, func() ([]store.FileWrite, error) {
		rev, err := s.store.Read(ref)
		if err != nil || rev == nil {
			return nil, s.noReview(ref, err)
		}
		current, err := s.store.PageBuild(ref)
		if err != nil {
			return nil, err
		}
		if problem := store.BuildProblem(s.tr, current, build); problem != "" {
			return nil, s.refused(refusedWords, problem)
		}
		var gh *github.GitHub
		if online {
			account := ""
			if rev.Account != nil {
				account = *rev.Account
			}
			gh = s.github(rev.Host, account)
		}
		if _, err := work(rev, gh); err != nil {
			return nil, err
		}
		write, err := s.store.RecordWrite(ref, *rev)
		if err != nil {
			return nil, err
		}
		return []store.FileWrite{write}, nil
	})
}

func (s *Service) problemsIn(gh *github.GitHub, rev models.Review, member models.ReviewMember, comments []models.Comment) ([]string, models.GitHubPull, error) {
	now, err := gh.Pull(rev.Repo, member.Pull)
	if err != nil {
		return nil, now, err
	}
	var problems []string
	if now.State != "open" {
		state := now.State
		if now.MergedAt != nil {
			state = "merged"
		}
		problems = append(problems, utils.ResolvePlaceholderString(s.tr.PullIsEnded, map[string]string{"pull": strconv.Itoa(member.Pull), "state": state}))
	}
	if now.Head.Sha != member.Head {
		problems = append(problems, utils.ResolvePlaceholderString(s.tr.HeadMovedFrom, map[string]string{"from": prefix(member.Head, 9), "to": prefix(now.Head.Sha, 9)}))
	}
	places := map[string]Places{}
	if len(comments) > 0 {
		if places, err = filePlaces(gh, rev.Repo, member.Pull); err != nil {
			return nil, now, err
		}
	}
	for _, c := range comments {
		where := utils.ResolvePlaceholderString(s.tr.CommentNamed, map[string]string{"id": strconv.Itoa(c.ID), "path": c.Path, "line": strconv.Itoa(c.Line)})
		if problem := PlaceProblem(s.tr, placesOf(places, c.Path), c); problem != "" {
			problems = append(problems, utils.ResolvePlaceholderString(s.tr.CommentProblem, map[string]string{"comment": where, "problem": problem}))
		}
		for _, p := range BodyProblems(s.tr, c.Body) {
			problems = append(problems, utils.ResolvePlaceholderString(s.tr.CommentText, map[string]string{"comment": where, "problem": p}))
		}
		if c.Stale != "" {
			problems = append(problems, utils.ResolvePlaceholderString(s.tr.CommentStale, map[string]string{"comment": where, "why": c.Stale}))
		}
	}
	return problems, now, nil
}

func alreadyPosted(gh *github.GitHub, rev models.Review, member models.ReviewMember, action, me string) (*models.GitHubReview, error) {
	wanted := sentKey(Sent(member, action))
	reviews, err := gh.Reviews(rev.Repo, member.Pull)
	if err != nil {
		return nil, err
	}
	var mine []models.GitHubReview
	for _, r := range reviews {
		if r.User.Login == me && r.CommitID == member.Head && r.State == ActionOf[action].State {
			mine = append(mine, r)
		}
	}
	if len(mine) == 0 {
		return nil, nil
	}
	if wanted == "[]" {
		return &mine[0], nil
	}
	comments, err := gh.ReviewComments(rev.Repo, member.Pull)
	if err != nil {
		return nil, err
	}
	for i, r := range mine {
		var theirs [][2]string
		for _, c := range comments {
			if c.ReviewID == r.ID {
				theirs = append(theirs, [2]string{c.Path, strings.TrimSpace(c.Body)})
			}
		}
		if pairsKey(theirs) == wanted {
			return &mine[i], nil
		}
	}
	return nil, nil
}

func sentKey(comments []models.Comment) string {
	pairs := make([][2]string, len(comments))
	for i, c := range comments {
		pairs[i] = [2]string{c.Path, strings.TrimSpace(c.Body)}
	}
	return pairsKey(pairs)
}

func pairsKey(pairs [][2]string) string {
	sorted := append([][2]string{}, pairs...)
	sort.Slice(sorted, func(a, b int) bool {
		return sorted[a][0]+","+sorted[a][1] < sorted[b][0]+","+sorted[b][1]
	})
	text, _ := json.Marshal(sorted)
	if len(sorted) == 0 {
		return "[]"
	}
	return string(text)
}

func record(member *models.ReviewMember, rev models.Review, action string, posted models.GitHubReview, at string) {
	member.Posted = append(member.Posted, models.PostedReview{Set: SetID(rev, *member, action), Action: action, Review: posted.HTMLURL, At: at,
		Comments: append([]models.Comment{}, Sent(*member, action)...)})
	if ActionOf[action].Comments {
		member.Comments = []models.Comment{}
	}
}

func postedAs(member models.ReviewMember, url string) bool {
	for _, p := range member.Posted {
		if p.Review == url {
			return true
		}
	}
	return false
}

func filePlaces(gh *github.GitHub, repo string, pull int) (map[string]Places, error) {
	files, err := gh.Files(repo, pull)
	if err != nil {
		return nil, err
	}
	places := map[string]Places{}
	for _, f := range files {
		patch := ""
		if f.Patch != nil {
			patch = *f.Patch
		}
		places[f.Filename] = DiffPlaces(patch)
	}
	return places, nil
}

func placesOf(places map[string]Places, path string) Places {
	if p, ok := places[path]; ok {
		return p
	}
	return DiffPlaces("")
}

func postedWords(tr *i18n.TranslationSet, action string) string {
	switch action {
	case "comment":
		return tr.PostedComment
	case "approve-with-comments":
		return tr.PostedApproveWithComments
	}
	return tr.PostedApprove
}

func commentAt(tr *i18n.TranslationSet, path string, start, line, pull int) string {
	lines := strconv.Itoa(line)
	if start != 0 {
		lines = strconv.Itoa(start) + "-" + lines
	}
	words := map[string]string{"path": path, "lines": lines, "pull": strconv.Itoa(pull)}
	if pull != 0 {
		return utils.ResolvePlaceholderString(tr.CommentAtOf, words)
	}
	return utils.ResolvePlaceholderString(tr.CommentAt, words)
}

func (s *Service) refused(words string, why ...string) error {
	return models.Refused(utils.ResolvePlaceholderString(words, map[string]string{"why": strings.Join(why, "; ")}))
}

func (s *Service) noReview(ref models.ReviewRef, err error) error {
	if err != nil {
		return err
	}
	return models.Refused(utils.ResolvePlaceholderString(s.tr.ReviewNone, map[string]string{"id": ref.ID()}))
}

func prefix(sha string, n int) string {
	if len(sha) > n {
		return sha[:n]
	}
	return sha
}

func contains(list []string, item string) bool {
	for _, x := range list {
		if x == item {
			return true
		}
	}
	return false
}
