package discord

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/sandwichjson"
)

const (
	APIVersion      = "v10"
	EndpointDiscord = "https://discord.com/api"
	UserAgent       = "Sandwich (github.com/WelcomerTeam/Discord)"
)

type RESTInterface interface {
	// Fetch constructs a request. It will return a response body along with any errors.
	// Errors can include ErrInvalidToken, ErrRateLimited,
	Fetch(s *Session, method, endpoint, contentType string, body []byte, headers http.Header) ([]byte, error)
	FetchBJ(s *Session, method, endpoint, contentType string, body []byte, headers http.Header, response interface{}) error
	FetchJJ(s *Session, method, endpoint string, payload interface{}, headers http.Header, response interface{}) error

	SetDebug(value bool)
}

// Session contains the context for the discord rest interface.
type Session struct {
	Context   context.Context
	Interface RESTInterface
	Token     string
}

func NewSession(context context.Context, token string, httpInterface RESTInterface) *Session {
	return &Session{
		Context:   context,
		Token:     token,
		Interface: httpInterface,
	}
}

// BaseInterface is the default HTTP Interface and simply handles routing to discord. Careful,
// this does not handle rate limiting.
type BaseInterface struct {
	HTTP       *http.Client
	APIVersion string
	URLHost    string
	URLScheme  string
	UserAgent  string

	Debug bool
}

func NewBaseInterface() RESTInterface {
	return NewInterface(&http.Client{
		Timeout: 20 * time.Second,
	}, EndpointDiscord, APIVersion, UserAgent)
}

func NewInterface(httpClient *http.Client, endpoint string, version string, useragent string) RESTInterface {
	url, _ := url.Parse(endpoint)

	return &BaseInterface{
		HTTP:       httpClient,
		APIVersion: version,
		URLHost:    url.Host,
		URLScheme:  url.Scheme,
		UserAgent:  useragent,
	}
}

func (bi *BaseInterface) Fetch(session *Session, method, endpoint, contentType string, body []byte, headers http.Header) ([]byte, error) {
	req, err := http.NewRequestWithContext(session.Context, method, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	req.URL.Host = bi.URLHost
	req.URL.Scheme = bi.URLScheme

	if strings.Contains(endpoint, "?") {
		req.URL.RawQuery = strings.SplitN(endpoint, "?", 2)[1]
		endpoint = strings.SplitN(endpoint, "?", 2)[0]
	}

	if bi.APIVersion != "" && !strings.HasPrefix(req.URL.Path, "/api") {
		req.URL.Path = "/api/" + bi.APIVersion + endpoint
	}

	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	if body != nil && len(req.Header.Get("Content-Type")) == 0 {
		req.Header.Set("Content-Type", contentType)
	}

	if session.Token != "" {
		req.Header.Set("Authorization", session.Token)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := bi.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}

	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	if bi.Debug {
		println(method, req.URL.String(), resp.StatusCode, contentType, string(body), string(response))
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusCreated:
	case http.StatusNoContent:
	case http.StatusUnauthorized:
		return response, ErrUnauthorized
	default:
		return response, NewRestError(req, resp, body)
	}

	return response, nil
}

func (bi *BaseInterface) FetchBJ(session *Session, method, endpoint, contentType string, body []byte, headers http.Header, response interface{}) error {
	resp, err := bi.Fetch(session, method, endpoint, contentType, body, headers)
	if err != nil {
		return err
	}

	if response != nil {
		err = sandwichjson.Unmarshal(resp, response)
		if err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

func (bi *BaseInterface) FetchJJ(session *Session, method, endpoint string, payload interface{}, headers http.Header, response interface{}) error {
	var body []byte
	var err error

	if payload != nil {
		body, err = sandwichjson.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	} else {
		body = make([]byte, 0)
	}

	return bi.FetchBJ(session, method, endpoint, "application/json", body, headers, response)
}

func (bi *BaseInterface) SetDebug(value bool) {
	bi.Debug = value
}

// TwilightProxy is a proxy that requests are sent through, instead of directly to discord that will handle
// distributed requests and ratelimits automatically. See more at: https://github.com/twilight-rs/http-proxy
type TwilightProxy struct {
	HTTP       *http.Client
	APIVersion string
	URLHost    string
	URLScheme  string
	UserAgent  string

	Debug bool
}

func NewTwilightProxy(url url.URL) RESTInterface {
	return &TwilightProxy{
		HTTP: &http.Client{
			Timeout: 20 * time.Second,
		},
		APIVersion: APIVersion,
		URLHost:    url.Host,
		URLScheme:  url.Scheme,
		UserAgent:  "Sandwich (github.com/WelcomerTeam/Discord)",
	}
}

func (tl *TwilightProxy) Fetch(session *Session, method, endpoint, contentType string, body []byte, headers http.Header) ([]byte, error) {
	req, err := http.NewRequestWithContext(session.Context, method, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	req.URL.Host = tl.URLHost
	req.URL.Scheme = tl.URLScheme

	if strings.Contains(endpoint, "?") {
		req.URL.RawQuery = strings.SplitN(endpoint, "?", 2)[1]
		endpoint = strings.SplitN(endpoint, "?", 2)[0]
	}

	if tl.APIVersion != "" && !strings.HasPrefix(req.URL.Path, "/api") {
		req.URL.Path = "/api/" + tl.APIVersion + endpoint
	}

	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	if body != nil && len(req.Header.Get("Content-Type")) == 0 {
		req.Header.Set("Content-Type", contentType)
	}

	if session.Token != "" {
		req.Header.Set("Authorization", session.Token)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := tl.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}

	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	if tl.Debug {
		println(method, req.URL.String(), resp.StatusCode, contentType, string(body), string(response))
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusCreated:
	case http.StatusNoContent:
	case http.StatusUnauthorized:
		return response, ErrUnauthorized
	default:
		return response, NewRestError(req, resp, body)
	}

	return response, nil
}

func (tl *TwilightProxy) FetchBJ(session *Session, method, endpoint, contentType string, body []byte, headers http.Header, response interface{}) error {
	resp, err := tl.Fetch(session, method, endpoint, contentType, body, headers)
	if err != nil {
		return err
	}

	if response != nil {
		err = sandwichjson.Unmarshal(resp, response)
		if err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

func (tl *TwilightProxy) FetchJJ(session *Session, method, endpoint string, payload interface{}, headers http.Header, response interface{}) error {
	var body []byte
	var err error

	if payload != nil {
		body, err = sandwichjson.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	} else {
		body = make([]byte, 0)
	}

	return tl.FetchBJ(session, method, endpoint, "application/json", body, headers, response)
}

func (tl *TwilightProxy) SetDebug(value bool) {
	tl.Debug = value
}
