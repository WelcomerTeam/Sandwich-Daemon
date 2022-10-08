package internal

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	jsoniter "github.com/json-iterator/go"
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
	body io.Reader, headers map[string]string,
) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := c.HandleRequest(req, false)
	if err != nil {
		return nil, 0, err
	}

	defer res.Body.Close()

	resultBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, res.StatusCode, fmt.Errorf("failed to read request body: %w", err)
	}

	return resultBody, res.StatusCode, nil
}

// FetchJSON attempts to convert the response into a JSON structure. Passing any headers
// will be sent to the request however Authorization will be overwrote.
func (c *Client) FetchJSON(ctx context.Context, method string, url string, body io.Reader,
	headers map[string]string, structure interface{},
) (int, error) {
	responseBody, status, err := c.Fetch(ctx, method, url, body, headers)
	if err != nil {
		return status, err
	}

	err = jsoniter.Unmarshal(responseBody, &structure)
	if err != nil {
		return -1, fmt.Errorf("failed to unmarshal body: %w", err)
	}

	return status, nil
}

// HandleRequest makes a request to the Discord API.
func (c *Client) HandleRequest(req *http.Request, retry bool) (*http.Response, error) {
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

	res, err := c.HTTP.Do(req)
	if err != nil {
		return res, fmt.Errorf("failed to do HTTP request: %w", err)
	}

	if res.StatusCode == http.StatusTooManyRequests {
		return res, fmt.Errorf("failed to do HTTP request: %w", err)
	}

	if res.StatusCode == http.StatusUnauthorized {
		return res, ErrInvalidToken
	}

	return res, nil
}
