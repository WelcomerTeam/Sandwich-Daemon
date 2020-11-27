package gateway

import (
	"context"
	"fmt"
	"github.com/TheRockettek/Sandwich-Daemon/structs"
	jsoniter "github.com/json-iterator/go"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func replaceIfEmpty(v string, s string) string {
	if v == "" {
		return s
	}

	return v
}

// Client represents the REST client.
type Client struct {
	mu sync.RWMutex

	Token string

	HTTP    *http.Client
	Buckets *sync.Map

	// We will manually add the API version
	APIVersion string

	// Used to safely create URLs and is filled if empty
	URLHost   string
	URLScheme string
	UserAgent string

	// Will use RestTunnel if not empty
	restTunnelURL string
	reverse       bool
}

// NewClient makes a new client.
func NewClient(token string, restTunnelURL string, reverse bool) *Client {
	return &Client{
		mu:            sync.RWMutex{},
		Token:         token,
		HTTP:          http.DefaultClient,
		APIVersion:    "6",
		URLHost:       "discord.com",
		URLScheme:     "https",
		restTunnelURL: restTunnelURL,
		reverse:       reverse,
	}
}

// FetchJSON attempts to convert the response into a JSON structure.
func (c *Client) FetchJSON(ctx context.Context, method string, url string,
	body io.Reader, structure interface{}) (err error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return
	}

	res, err := c.HandleRequest(req, false)
	if err != nil {
		return
	}

	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(structure)

	if err != nil {
		return fmt.Errorf("failed to unmarshal body: %w", err)
	}

	return
}

// HandleRequest makes a request to the Discord API.
func (c *Client) HandleRequest(req *http.Request, retry bool) (res *http.Response, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !retry {
		// If we are trying the request, do not add again
		req.URL.Path = "/api/v" + c.APIVersion + req.URL.Path
	}

	req.URL.Host = replaceIfEmpty(req.URL.Host, c.URLHost)
	req.URL.Scheme = replaceIfEmpty(req.URL.Scheme, c.URLScheme)

	req.Header.Set("User-Agent", replaceIfEmpty(req.Header.Get("User-Agent"), c.UserAgent))
	req.Header.Set("Authorization", replaceIfEmpty(req.Header.Get("Authorization"), ("Bot "+c.Token)))

	if c.restTunnelURL == "" {
		if res, err = c.HTTP.Do(req); err != nil {
			return res, fmt.Errorf("failed to do HTTP request: %w", err)
		}

		if res.StatusCode == http.StatusTooManyRequests {
			resp := structs.TooManyRequests{}
			err = json.NewDecoder(res.Body).Decode(&resp)

			if err != nil {
				return res, fmt.Errorf("failed to decode body: %w", err)
			}

			<-time.After(time.Duration(resp.RetryAfter) * time.Millisecond)

			return c.HandleRequest(req, true)
		}
	} else {
		req.Header.Set("Rt-Priority", "true")
		req.Header.Set("Rt-ResponseType", "RespondWithResponse")

		_url, _ := url.Parse(c.restTunnelURL)

		if c.reverse {
			req.URL.Host = _url.Host
			req.URL.Scheme = _url.Scheme
		} else {
			req.Header.Set("Rt-URL", req.URL.String())
			req.URL = _url
		}

		if res, err = c.HTTP.Do(req); err != nil {
			return res, fmt.Errorf("failed to do HTTP request: %w", err)
		}
	}

	if res.StatusCode == http.StatusUnauthorized {
		return res, ErrInvalidToken
	}

	return res, nil
}
