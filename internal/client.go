package gateway

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/structs"
	jsoniter "github.com/json-iterator/go"
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

	isBot bool

	// Will use RestTunnel if not empty
	restTunnelURL string
	reverse       bool
}

// NewClient makes a new client.
func NewClient(token string, restTunnelURL string, reverse bool, isBot bool) *Client {
	return &Client{
		mu:            sync.RWMutex{},
		Token:         token,
		HTTP:          http.DefaultClient,
		APIVersion:    "6",
		URLHost:       "discord.com",
		URLScheme:     "https",
		isBot:         isBot,
		restTunnelURL: restTunnelURL,
		reverse:       reverse,
	}
}

// Fetch returns the response. Passing any headers will be sent to the request however
// Authorization will be overwrote.
func (c *Client) Fetch(ctx context.Context, method string, url string,
	body io.Reader, headers map[string]string) (_body []byte, err error, status int) {
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

	return _body, err, res.StatusCode
}

// FetchJSON attempts to convert the response into a JSON structure. Passing any headers
// will be sent to the request however Authorization will be overwrote.
func (c *Client) FetchJSON(ctx context.Context, method string, url string,
	body io.Reader, headers map[string]string, structure interface{}) (err error, status int) {

	_body, err, status := c.Fetch(ctx, method, url, body, headers)
	if err != nil {
		return
	}

	err = json.Unmarshal(_body, &structure)
	if err != nil {
		return fmt.Errorf("failed to unmarshal body: %w", err), -1
	}

	return
}

// HandleRequest makes a request to the Discord API.
func (c *Client) HandleRequest(req *http.Request, retry bool) (res *http.Response, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Add the /api and version prefix when we did not include one
	if !strings.HasPrefix(req.URL.Path, "/api") {
		req.URL.Path = "/api/v" + c.APIVersion + req.URL.Path
	}

	req.URL.Host = replaceIfEmpty(req.URL.Host, c.URLHost)
	req.URL.Scheme = replaceIfEmpty(req.URL.Scheme, c.URLScheme)

	req.Header.Set("User-Agent", replaceIfEmpty(req.Header.Get("User-Agent"), c.UserAgent))

	if c.Token != "" {
		if c.isBot {
			req.Header.Set("Authorization", replaceIfEmpty(req.Header.Get("Authorization"), ("Bot "+c.Token)))
		} else {
			req.Header.Set("Authorization", replaceIfEmpty(req.Header.Get("Authorization"), c.Token))
		}
	}

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

// Sending webhook example
// c := NewClient("", sg.Configuration.RestTunnel.URL, sg.RestTunnelReverse.IsSet(), false)
// ctx := context.Background()

// res, err := json.Marshal(structs.Message{
// 	Content: "Hello World!",
// 	Embeds: []structs.Embed{
// 		{
// 			Title:       "Test Embed",
// 			Description: "Welcomer 7.0 webhook testing",
// 		},
// 	},
// })
// if err != nil {
// 	sg.Logger.Error().Err(err).Msg("Failed to marshal event")
// }

// headers := map[string]string{
// 	"content-type": "application/json",
// }

// b, err := c.Fetch(ctx, "POST", "/api/webhooks/.../...", bytes.NewBuffer(res), headers)
// if err != nil {
// 	sg.Logger.Error().Err(err).Msg("Failed to send webhook")
// }
