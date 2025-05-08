package sandwich

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var UserAgent = fmt.Sprintf("Sandwich/%s (https://github.com/WelcomerTeam/Sandwich-Daemon)", Version)

// NewProxyClient creates an HTTP client that redirects all requests through a specified host.
// This is useful when using a proxy such as twilight or nirn.
func NewProxyClient(client http.Client, host url.URL) *http.Client {
	if client.Transport == nil {
		client.Transport = http.DefaultTransport
	}

	client.Transport = &proxyTransport{
		host:      host,
		transport: client.Transport,
	}

	return &client
}

type proxyTransport struct {
	host      url.URL
	transport http.RoundTripper
}

func (t *proxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a copy of the request to modify
	proxyReq := req.Clone(req.Context())

	// Set the new host while keeping the original path and query
	proxyReq.URL.Host = t.host.Host
	proxyReq.URL.Scheme = t.host.Scheme
	proxyReq.Host = t.host.Host

	if !strings.HasPrefix(proxyReq.URL.String(), "/api") {
		proxyReq.URL.Path = "/api/v10" + proxyReq.URL.Path
	}

	proxyReq.Header.Set("User-Agent", UserAgent)

	// Perform the request using the underlying transport
	resp, err := t.transport.RoundTrip(proxyReq)
	if err != nil {
		return nil, fmt.Errorf("failed to round trip: %w", err)
	}

	return resp, nil
}
