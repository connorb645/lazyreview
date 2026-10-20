package gui

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/internal/models"
)

// standInReviews gives the app its reviews from a handler in the test's
// own process, which answers each call as the recorded answers do, keyed
// by the method and address the lazyreview server answered them at.
type standInReviews struct {
	handler http.Handler
}

func (s standInReviews) State(id string) (models.State, error) {
	var state models.State
	return state, s.call(http.MethodGet, reviewPath(id, ""), nil, nil, &state)
}

func (s standInReviews) Page(id string) (models.Page, error) {
	var page models.Page
	return page, s.call(http.MethodGet, reviewPath(id, "/page"), nil, nil, &page)
}

func (s standInReviews) File(id, path, build string) (models.File, error) {
	var file models.File
	return file, s.call(http.MethodGet, reviewPath(id, "/file"), url.Values{"path": {path}, "build": {build}}, nil, &file)
}

func (s standInReviews) Where(id string, at models.Lines, pull int, build string) (models.Where, error) {
	query := url.Values{"path": {at.Path}, "side": {at.Side}, "line": {strconv.Itoa(at.Line)}, "build": {build}}
	if at.StartLine != 0 {
		query.Set("start_line", strconv.Itoa(at.StartLine))
		query.Set("start_side", at.StartSide)
	}
	if pull != 0 {
		query.Set("pull", strconv.Itoa(pull))
	}
	var where models.Where
	return where, s.call(http.MethodGet, reviewPath(id, "/where"), query, nil, &where)
}

func (s standInReviews) Queue(id string, at models.Lines, pull int, body, build string) (models.State, error) {
	fields := linesFields(at)
	fields["body"], fields["build"] = body, build
	if pull != 0 {
		fields["pull"] = pull
	}
	var answer struct{ State models.State }
	return answer.State, s.call(http.MethodPost, reviewPath(id, "/comments"), nil, fields, &answer)
}

func (s standInReviews) Edit(id string, comment int, body, version, build string) (models.State, error) {
	var answer struct{ State models.State }
	fields := map[string]any{"body": body, "version": version, "build": build}
	return answer.State, s.call(http.MethodPatch, reviewPath(id, fmt.Sprintf("/comments/%d", comment)), nil, fields, &answer)
}

func (s standInReviews) Drop(id string, comment int, version, build string) (models.State, error) {
	var answer struct{ State models.State }
	query := url.Values{"version": {version}, "build": {build}}
	return answer.State, s.call(http.MethodDelete, reviewPath(id, fmt.Sprintf("/comments/%d", comment)), query, nil, &answer)
}

func (s standInReviews) Ask(id string, at models.Lines, text, build string) (models.Conversation, error) {
	fields := linesFields(at)
	fields["text"], fields["build"] = text, build
	var answer struct{ Thread models.Conversation }
	return answer.Thread, s.call(http.MethodPost, reviewPath(id, "/threads"), nil, fields, &answer)
}

func (s standInReviews) Reply(id string, conversation int, text, build string) (models.Conversation, error) {
	var answer struct{ Thread models.Conversation }
	fields := map[string]any{"text": text, "build": build}
	return answer.Thread, s.call(http.MethodPost, reviewPath(id, fmt.Sprintf("/threads/%d/messages", conversation)), nil, fields, &answer)
}

func (s standInReviews) Stop(id string, conversation int) (models.Conversation, error) {
	var answer struct{ Thread models.Conversation }
	return answer.Thread, s.call(http.MethodPost, reviewPath(id, fmt.Sprintf("/threads/%d/stop", conversation)), nil, map[string]any{}, &answer)
}

func (s standInReviews) Post(id string, pull int, action, set, build string) (string, models.State, error) {
	var answer struct {
		Message string
		State   models.State
	}
	fields := map[string]any{"pull": pull, "action": action, "set": set, "build": build}
	err := s.call(http.MethodPost, reviewPath(id, "/post"), nil, fields, &answer)
	return answer.Message, answer.State, err
}

func (s standInReviews) Refresh(id string) ([]string, error) {
	var answer struct{ Notes []string }
	return answer.Notes, s.call(http.MethodPost, reviewPath(id, "/refresh"), nil, map[string]any{}, &answer)
}

func (s standInReviews) Pulls() (models.Pulls, error) {
	var pulls models.Pulls
	return pulls, s.call(http.MethodGet, "/api/pulls", nil, nil, &pulls)
}

func (s standInReviews) ChangeHidden(hide string, show, repos []string) (models.Hidden, error) {
	var hidden models.Hidden
	fields := map[string]any{"repos": repos}
	if hide != "" {
		fields["hide"] = hide
	}
	if len(show) > 0 {
		fields["show"] = show
	}
	return hidden, s.call(http.MethodPost, "/api/hidden", nil, fields, &hidden)
}

func (s standInReviews) Load(links []string, account string) (string, []string, error) {
	var answer struct {
		ID    string
		Notes []string
	}
	err := s.call(http.MethodPost, "/api/load", nil, map[string]any{"links": links, "account": account}, &answer)
	return answer.ID, answer.Notes, err
}

func (s standInReviews) Events(ctx context.Context, id string, notices chan<- models.Notice) error {
	request := httptest.NewRequestWithContext(ctx, http.MethodGet, reviewPath(id, "/events"), nil)
	answer := httptest.NewRecorder()
	s.handler.ServeHTTP(answer, request)
	if answer.Code != http.StatusOK {
		return models.Refused(fmt.Sprintf("the stand-in answered %d to listening for changes", answer.Code))
	}
	lines := bufio.NewScanner(answer.Body)
	for lines.Scan() {
		data, ok := strings.CutPrefix(lines.Text(), "data: ")
		if !ok {
			continue
		}
		var notice models.Notice
		if err := json.Unmarshal([]byte(data), &notice); err != nil {
			return err
		}
		select {
		case notices <- notice:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return errors.New("the stand-in ended the stream of changes")
}

func (s standInReviews) call(method, path string, query url.Values, fields map[string]any, into any) error {
	address := path
	if len(query) > 0 {
		address += "?" + query.Encode()
	}
	var body *bytes.Reader
	if fields != nil {
		text, err := json.Marshal(fields)
		if err != nil {
			return err
		}
		body = bytes.NewReader(text)
	} else {
		body = bytes.NewReader(nil)
	}
	request := httptest.NewRequest(method, address, body)
	answer := httptest.NewRecorder()
	s.handler.ServeHTTP(answer, request)
	if answer.Code != http.StatusOK {
		var refused struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(answer.Body.Bytes(), &refused) == nil && refused.Error != "" {
			return models.Refused(refused.Error)
		}
		return models.Refused(fmt.Sprintf("the stand-in answered %d", answer.Code))
	}
	return json.Unmarshal(answer.Body.Bytes(), into)
}

func linesFields(at models.Lines) map[string]any {
	fields := map[string]any{"path": at.Path, "side": at.Side, "line": at.Line}
	if at.StartLine != 0 {
		fields["start_line"], fields["start_side"] = at.StartLine, at.StartSide
	}
	return fields
}

func reviewPath(id, rest string) string {
	parts := strings.Split(id, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return "/api/reviews/" + strings.Join(parts, "/") + rest
}
