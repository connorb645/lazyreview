// Package server reads and changes reviews through the Review Desk server on
// this machine, the one back half the web app and the terminal app share.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const readWait = 20 * time.Second

// Client talks to the Review Desk server. It reads the server's address and
// token from the server's own record at each call, so a server started again
// is found again.
type Client struct {
	file string
	http *http.Client
}

// NewClient returns a client of the server that records itself in file, the
// data home's server.json.
func NewClient(file string) *Client {
	return &Client{file: file, http: &http.Client{}}
}

// Address is where the server answers, such as http://127.0.0.1:4747.
func (c *Client) Address() (string, error) {
	record, err := c.record()
	if err != nil {
		return "", err
	}
	return record.Address, nil
}

// Reviews lists every review the server keeps, newest load first.
func (c *Client) Reviews() ([]Summary, error) {
	var reviews []Summary
	return reviews, c.get("/api/reviews", nil, &reviews)
}

// State reads a review's state: its pull requests, its queue, its
// conversations, the threads on GitHub and its written review.
func (c *Client) State(id string) (State, error) {
	var state State
	return state, c.get(ReviewPath(id, ""), nil, &state)
}

// Page reads what a review draws: its build, its model and its layout.
func (c *Client) Page(id string) (Page, error) {
	var page Page
	return page, c.get(ReviewPath(id, "/page"), nil, &page)
}

// File reads one file of a review ready to draw, for the page build the app
// drew from: its hunks of rows, the unchanged runs between them and its
// whole text.
func (c *Client) File(id, path, build string) (File, error) {
	var file File
	return file, c.get(ReviewPath(id, "/file"), url.Values{"path": {path}, "build": {build}}, &file)
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
		return Record{}, fmt.Errorf("the Review Desk server is not running; review-desk start starts it")
	}
	if err != nil {
		return Record{}, fmt.Errorf("could not read %s: %w", c.file, err)
	}
	var record Record
	if err := json.Unmarshal(text, &record); err != nil {
		return Record{}, fmt.Errorf("could not read %s: %w", c.file, err)
	}
	return record, nil
}

func (c *Client) get(path string, query url.Values, into any) error {
	record, err := c.record()
	if err != nil {
		return err
	}
	address := record.Address + path
	if len(query) > 0 {
		address += "?" + query.Encode()
	}
	ctx, cancel := context.WithTimeout(context.Background(), readWait)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return fmt.Errorf("could not ask the server for %s: %w", path, err)
	}
	request.Header.Set("Accept", "application/json")
	return c.do(request, into)
}

func (c *Client) do(request *http.Request, into any) error {
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("the Review Desk server did not answer: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("could not read the server's answer to %s: %w", request.URL.Path, err)
	}
	if response.StatusCode != http.StatusOK {
		var refused struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &refused) == nil && refused.Error != "" {
			return Refusal{Status: response.StatusCode, Message: refused.Error}
		}
		return Refusal{Status: response.StatusCode, Message: fmt.Sprintf("the server answered %d", response.StatusCode)}
	}
	if err := json.Unmarshal(body, into); err != nil {
		return fmt.Errorf("could not read the server's answer to %s: %w", request.URL.Path, err)
	}
	return nil
}
