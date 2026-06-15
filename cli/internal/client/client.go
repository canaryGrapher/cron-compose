// Package client is the CLI's thin REST wrapper. It signs every request with the
// saved session cookie and offers a few convenience helpers.
package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client wraps an http.Client with the API base URL and the persisted session id.
type Client struct {
	Base      string
	SessionID string
	HTTP      *http.Client
}

// New returns a Client.
func New(base, sessionID string) *Client {
	return &Client{
		Base:      strings.TrimRight(base, "/"),
		SessionID: sessionID,
		HTTP:      &http.Client{Timeout: 30 * time.Second},
	}
}

// Do executes the request, attaching the session cookie when set.
func (c *Client) Do(method, path string, body any) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.Base+path, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.SessionID != "" {
		req.AddCookie(&http.Cookie{Name: "cc_session", Value: c.SessionID})
	}
	return c.HTTP.Do(req)
}

// JSON does Do + decodes a successful body into out.
func (c *Client) JSON(method, path string, body, out any) error {
	res, err := c.Do(method, path, body)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("%s %s: %d %s", method, path, res.StatusCode, string(b))
	}
	if out == nil || res.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// Login posts credentials, captures the cc_session cookie, and returns its value.
func (c *Client) Login(email, password string) (string, error) {
	res, err := c.Do(http.MethodPost, "/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		b, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("login: %d %s", res.StatusCode, string(b))
	}
	for _, cookie := range res.Cookies() {
		if cookie.Name == "cc_session" {
			return cookie.Value, nil
		}
	}
	return "", errors.New("login: no session cookie returned")
}

// StreamSSE opens an EventSource-style stream. The callback receives the "event" and
// "data" of each event; return io.EOF to stop early.
func (c *Client) StreamSSE(path string, onEvent func(event, data string) error) error {
	req, err := http.NewRequest(http.MethodGet, c.Base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.SessionID != "" {
		req.AddCookie(&http.Cookie{Name: "cc_session", Value: c.SessionID})
	}
	// SSE may be long-lived; reuse a no-timeout client just for this call.
	streamClient := &http.Client{}
	res, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("stream: %d %s", res.StatusCode, string(b))
	}
	return parseSSE(res.Body, onEvent)
}

func parseSSE(r io.Reader, onEvent func(event, data string) error) error {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 1024)
	var event, data string
	flush := func() error {
		if event == "" && data == "" {
			return nil
		}
		err := onEvent(event, data)
		event, data = "", ""
		return err
	}
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			for {
				idx := bytes.IndexByte(buf, '\n')
				if idx < 0 {
					break
				}
				line := strings.TrimRight(string(buf[:idx]), "\r")
				buf = buf[idx+1:]
				if line == "" {
					if e := flush(); e != nil {
						if errors.Is(e, io.EOF) {
							return nil
						}
						return e
					}
					continue
				}
				if strings.HasPrefix(line, "event: ") {
					event = line[len("event: "):]
				} else if strings.HasPrefix(line, "data: ") {
					data = line[len("data: "):]
				}
			}
		}
		if err == io.EOF {
			return flush()
		}
		if err != nil {
			return err
		}
	}
}
