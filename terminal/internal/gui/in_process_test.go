package gui

import (
	"net/http"
	"net/http/httptest"
)

// inProcess answers the server client's requests with a handler in the
// test's own process, so no test waits on a socket.
type inProcess struct {
	handler http.Handler
}

func (p inProcess) RoundTrip(r *http.Request) (*http.Response, error) {
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	w := httptest.NewRecorder()
	p.handler.ServeHTTP(w, r)
	answer := w.Result()
	answer.Request = r
	return answer, nil
}
