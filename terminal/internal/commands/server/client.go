// Package server reads and changes reviews through the lazyreview server on
// this machine, the one back half the web app and the terminal app share.
package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/common"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const (
	readWait    = 20 * time.Second
	changeWait  = 60 * time.Second
	askWait     = 200 * time.Second
	refreshWait = 120 * time.Second
	pullsWait   = 180 * time.Second
	loadWait    = 20 * time.Minute
)

// Client talks to the lazyreview server. It reads the server's address and
// token from the server's own record at each call, so a server started again
// is found again.
type Client struct {
	tr   *i18n.TranslationSet
	file string
	http *http.Client
}

// NewClient returns a client of the server that records itself in file, the
// data home's server.json.
func NewClient(cmn *common.Common, file string) *Client {
	return NewClientThrough(cmn, file, http.DefaultTransport)
}

// NewClientThrough returns a client of the server that records itself in
// file, which reaches it through transport, such as one that answers in the
// same process.
func NewClientThrough(cmn *common.Common, file string, transport http.RoundTripper) *Client {
	return &Client{tr: cmn.Tr, file: file, http: &http.Client{Transport: transport}}
}

// Address is where the server answers, such as http://127.0.0.1:4747.
func (c *Client) Address() (string, error) {
	record, err := c.record()
	if err != nil {
		return "", err
	}
	return record.Address, nil
}

// State reads a review's state: its pull requests, its queue, its
// conversations, the threads on GitHub and its written review.
func (c *Client) State(id string) (State, error) {
	var state State
	return state, c.get(ReviewPath(id, ""), nil, readWait, &state)
}

// Page reads what a review draws: its build, its model and its layout.
func (c *Client) Page(id string) (Page, error) {
	var page Page
	return page, c.get(ReviewPath(id, "/page"), nil, readWait, &page)
}

// File reads one file of a review ready to draw, for the page build the app
// drew from: its hunks of rows, the unchanged runs between them and its
// whole text.
func (c *Client) File(id, path, build string) (File, error) {
	var file File
	return file, c.get(ReviewPath(id, "/file"), url.Values{"path": {path}, "build": {build}}, readWait, &file)
}

// Where says where a comment on lines as the app shows them would go, for
// the page build the app drew from; pull names one pull request of a stack
// where several can take it, or is 0.
func (c *Client) Where(id string, at Lines, pull int, build string) (Where, error) {
	query := url.Values{"path": {at.Path}, "side": {at.Side}, "line": {strconv.Itoa(at.Line)}, "build": {build}}
	if at.StartLine != 0 {
		query.Set("start_line", strconv.Itoa(at.StartLine))
		query.Set("start_side", at.StartSide)
	}
	if pull != 0 {
		query.Set("pull", strconv.Itoa(pull))
	}
	var where Where
	return where, c.get(ReviewPath(id, "/where"), query, readWait, &where)
}

// Queue queues a comment on lines as the app shows them, and returns the
// state as it then stands.
func (c *Client) Queue(id string, at Lines, pull int, body, build string) (State, error) {
	fields := linesFields(at)
	fields["body"], fields["build"] = body, build
	if pull != 0 {
		fields["pull"] = pull
	}
	var answer struct{ State State }
	return answer.State, c.change(http.MethodPost, ReviewPath(id, "/comments"), nil, fields, changeWait, &answer)
}

// Edit replaces a queued comment's text, naming the version it was shown
// at, and returns the state as it then stands.
func (c *Client) Edit(id string, comment int, body, version, build string) (State, error) {
	var answer struct{ State State }
	fields := map[string]any{"body": body, "version": version, "build": build}
	return answer.State, c.change(http.MethodPatch, ReviewPath(id, fmt.Sprintf("/comments/%d", comment)), nil, fields, changeWait, &answer)
}

// Drop drops a queued comment, naming the version it was shown at, and
// returns the state as it then stands.
func (c *Client) Drop(id string, comment int, version, build string) (State, error) {
	var answer struct{ State State }
	query := url.Values{"version": {version}, "build": {build}}
	return answer.State, c.change(http.MethodDelete, ReviewPath(id, fmt.Sprintf("/comments/%d", comment)), query, nil, changeWait, &answer)
}

// Ask starts a conversation with Claude on lines as the app shows them, and
// returns it once Claude has answered, or said why not.
func (c *Client) Ask(id string, at Lines, text, build string) (Conversation, error) {
	fields := linesFields(at)
	fields["text"], fields["build"] = text, build
	var answer struct{ Thread Conversation }
	return answer.Thread, c.change(http.MethodPost, ReviewPath(id, "/threads"), nil, fields, askWait, &answer)
}

// Reply asks a follow-up in a conversation, and returns it once Claude has
// answered, or said why not.
func (c *Client) Reply(id string, conversation int, text, build string) (Conversation, error) {
	var answer struct{ Thread Conversation }
	fields := map[string]any{"text": text, "build": build}
	return answer.Thread, c.change(http.MethodPost, ReviewPath(id, fmt.Sprintf("/threads/%d/messages", conversation)), nil, fields, askWait, &answer)
}

// Post sends one action offered for a pull request, naming the set of
// comments it was shown with, and returns the server's words for what it
// did and the state as it then stands.
func (c *Client) Post(id string, pull int, action, set, build string) (string, State, error) {
	var answer struct {
		Message string
		State   State
	}
	fields := map[string]any{"pull": pull, "action": action, "set": set, "build": build}
	err := c.change(http.MethodPost, ReviewPath(id, "/post"), nil, fields, changeWait, &answer)
	return answer.Message, answer.State, err
}

// Refresh loads a review again: a pull request's from GitHub, or a branch's
// from its repository. It returns the server's notes on what it found.
func (c *Client) Refresh(id string) ([]string, error) {
	var answer struct{ Notes []string }
	return answer.Notes, c.change(http.MethodPost, ReviewPath(id, "/refresh"), nil, map[string]any{}, refreshWait, &answer)
}

// Pulls reads the open pull requests that involve the accounts in the
// settings file, as GitHub has them now.
func (c *Client) Pulls() (Pulls, error) {
	var pulls Pulls
	return pulls, c.get("/api/pulls", nil, pullsWait, &pulls)
}

// Load loads pull requests from GitHub into a review, a stack bottom
// first, as the GitHub account named, and returns the review's id and the
// server's notes on what it found.
func (c *Client) Load(links []string, account string) (string, []string, error) {
	var answer struct {
		ID    string
		Notes []string
	}
	err := c.change(http.MethodPost, "/api/load", nil, map[string]any{"links": links, "account": account}, loadWait, &answer)
	return answer.ID, answer.Notes, err
}

// Events sends each change notice of a review to notices, the first naming
// both the state's version and the page's build, until ctx ends or the
// stream breaks, and says why it stopped.
func (c *Client) Events(ctx context.Context, id string, notices chan<- Notice) error {
	record, err := c.record()
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, record.Address+ReviewPath(id, "/events"), nil)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ResolvePlaceholderString(c.tr.CouldNotListen, map[string]string{"review": id}), err)
	}
	request.Header.Set("Accept", "text/event-stream")
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("%s: %w", c.tr.ServerDidNotAnswer, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Refusal{Status: response.StatusCode, Message: utils.ResolvePlaceholderString(c.tr.ServerAnsweredListening, map[string]string{"status": strconv.Itoa(response.StatusCode)})}
	}
	lines := bufio.NewScanner(response.Body)
	for lines.Scan() {
		data, ok := strings.CutPrefix(lines.Text(), "data: ")
		if !ok {
			continue
		}
		var notice Notice
		if err := json.Unmarshal([]byte(data), &notice); err != nil {
			return fmt.Errorf("%s: %w", c.tr.CouldNotReadNotice, err)
		}
		select {
		case notices <- notice:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := lines.Err(); err != nil {
		return fmt.Errorf("%s: %w", c.tr.StreamBroke, err)
	}
	return errors.New(c.tr.StreamEnded)
}

func linesFields(at Lines) map[string]any {
	fields := map[string]any{"path": at.Path, "side": at.Side, "line": at.Line}
	if at.StartLine != 0 {
		fields["start_line"], fields["start_side"] = at.StartLine, at.StartSide
	}
	return fields
}

// ReviewPath is the address of a review's part of the API, from its id.
func ReviewPath(id, rest string) string {
	parts := strings.Split(id, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return "/api/reviews/" + strings.Join(parts, "/") + rest
}

func (c *Client) record() (Record, error) {
	text, err := os.ReadFile(c.file)
	if errors.Is(err, os.ErrNotExist) {
		return Record{}, errors.New(c.tr.ServerNotRunning)
	}
	couldNotRead := utils.ResolvePlaceholderString(c.tr.CouldNotRead, map[string]string{"path": c.file})
	if err != nil {
		return Record{}, fmt.Errorf("%s: %w", couldNotRead, err)
	}
	var record Record
	if err := json.Unmarshal(text, &record); err != nil {
		return Record{}, fmt.Errorf("%s: %w", couldNotRead, err)
	}
	return record, nil
}

func (c *Client) get(path string, query url.Values, wait time.Duration, into any) error {
	record, err := c.record()
	if err != nil {
		return err
	}
	address := record.Address + path
	if len(query) > 0 {
		address += "?" + query.Encode()
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ResolvePlaceholderString(c.tr.CouldNotAsk, map[string]string{"path": path}), err)
	}
	request.Header.Set("Accept", "application/json")
	return c.do(request, into)
}

func (c *Client) change(method, path string, query url.Values, fields map[string]any, wait time.Duration, into any) error {
	record, err := c.record()
	if err != nil {
		return err
	}
	address := record.Address + path
	if len(query) > 0 {
		address += "?" + query.Encode()
	}
	var body io.Reader
	if fields != nil {
		text, err := json.Marshal(fields)
		if err != nil {
			return fmt.Errorf("%s: %w", utils.ResolvePlaceholderString(c.tr.CouldNotWriteChange, map[string]string{"path": path}), err)
		}
		body = bytes.NewReader(text)
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, method, address, body)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ResolvePlaceholderString(c.tr.CouldNotSendChange, map[string]string{"path": path}), err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+record.Token)
	if fields != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	return c.do(request, into)
}

func (c *Client) do(request *http.Request, into any) error {
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("%s: %w", c.tr.ServerDidNotAnswer, err)
	}
	defer response.Body.Close()
	couldNotRead := utils.ResolvePlaceholderString(c.tr.CouldNotReadAnswer, map[string]string{"path": request.URL.Path})
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("%s: %w", couldNotRead, err)
	}
	if response.StatusCode != http.StatusOK {
		var refused struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &refused) == nil && refused.Error != "" {
			return Refusal{Status: response.StatusCode, Message: refused.Error}
		}
		return Refusal{Status: response.StatusCode, Message: utils.ResolvePlaceholderString(c.tr.ServerAnswered, map[string]string{"status": strconv.Itoa(response.StatusCode)})}
	}
	if err := json.Unmarshal(body, into); err != nil {
		return fmt.Errorf("%s: %w", couldNotRead, err)
	}
	return nil
}
