package internal

import (
	"context"
	"fmt"
	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	"golang.org/x/xerrors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client represents the REST client.
type Client struct {
	mu sync.Mutex

	Token string

	HTTP *http.Client

	// We will manually add the API version
	APIVersion string

	// Used to safely create URLs and is filled if empty
	URLHost   string
	URLScheme string
	UserAgent string
}

// NewClient makes a new client.
func NewClient(baseURL url.URL, token string) *Client {
	return &Client{
		mu:         sync.Mutex{},
		Token:      token,
		HTTP:       http.DefaultClient,
		APIVersion: "9",
		URLHost:    baseURL.Host,
		URLScheme:  baseURL.Scheme,
		UserAgent:  "Sandwich/" + VERSION + " (github.com/WelcomerTeam/Sandwich-Daemon)",
	}
}

// Fetch returns the response. Passing any headers will be sent to the request however
// Authorization will be overwrote.
func (c *Client) Fetch(ctx context.Context, method string, url string,
	body io.Reader, headers map[string]string) (_body []byte, status int, err error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := c.HandleRequest(req, false)
	if err != nil {
		return
	}

	defer res.Body.Close()

	_body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, res.StatusCode, xerrors.Errorf("failed to read request body: %w", err)
	}

	return _body, res.StatusCode, nil
}

// FetchJSON attempts to convert the response into a JSON structure. Passing any headers
// will be sent to the request however Authorization will be overwrote.
func (c *Client) FetchJSON(ctx context.Context, method string, url string, body io.Reader,
	headers map[string]string, structure interface{}) (status int, err error) {
	responseBody, status, err := c.Fetch(ctx, method, url, body, headers)
	if err != nil {
		return
	}

	err = json.Unmarshal(responseBody, &structure)
	if err != nil {
		return -1, fmt.Errorf("failed to unmarshal body: %w", err)
	}

	return
}

// HandleRequest makes a request to the Discord API.
func (c *Client) HandleRequest(req *http.Request, retry bool) (res *http.Response, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Add the /api and version prefix when we did not include one
	if !strings.HasPrefix(req.URL.Path, "/api") {
		req.URL.Path = "/api/v" + c.APIVersion + req.URL.Path
	}

	req.URL.Host = replaceIfEmpty(req.URL.Host, c.URLHost)
	req.URL.Scheme = replaceIfEmpty(req.URL.Scheme, c.URLScheme)

	req.Header.Set("User-Agent", replaceIfEmpty(req.Header.Get("User-Agent"), c.UserAgent))

	if c.Token != "" {
		req.Header.Set("Authorization", replaceIfEmpty(req.Header.Get("Authorization"), "Bot "+c.Token))
	}

	if res, err = c.HTTP.Do(req); err != nil {
		return res, fmt.Errorf("failed to do HTTP request: %w", err)
	}

	if res.StatusCode == http.StatusTooManyRequests {
		var resp discord.TooManyRequests
		err = json.NewDecoder(res.Body).Decode(&resp)

		if err != nil {
			return res, fmt.Errorf("failed to decode body: %w", err)
		}

		<-time.After(time.Duration(resp.RetryAfter) * time.Millisecond)

		return c.HandleRequest(req, true)
	}

	if res.StatusCode == http.StatusUnauthorized {
		return res, ErrInvalidToken
	}

	return res, nil
}
