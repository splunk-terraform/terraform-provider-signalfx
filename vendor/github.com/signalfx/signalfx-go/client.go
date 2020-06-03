package signalfx

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/signalfx/signalfx-go/signalflow"
)

// DefaultAPIURL is the default URL for making API requests
const DefaultAPIURL = "https://api.signalfx.com"

// AuthHeaderKey is the HTTP header used to pass along the auth token
// Note that while HTTP headers are case insensitive this header is case
// sensitive on the tests for convenience.
const AuthHeaderKey = "X-Sf-Token"

// Client is a SignalFx API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
	userAgent  string
}

// ClientParam is an option for NewClient. Its implementation borrows
// from Dave Cheney's functional options API
// (https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis).
type ClientParam func(*Client) error

// NewClient creates a new SignalFx client using the specified token.
func NewClient(token string, options ...ClientParam) (*Client, error) {
	client := &Client{
		baseURL: DefaultAPIURL,
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		authToken: token,
		userAgent: "signalfx-go",
	}

	for _, option := range options {
		option(client)
	}

	return client, nil
}

// APIUrl sets the URL that our client will communicate with, allowing
// it to be adjusted to another URL for testing or communication with other
// SignalFx clusters. Example `"https://api.signalfx.com"`.
func APIUrl(apiURL string) ClientParam {
	return func(client *Client) error {
		client.baseURL = apiURL
		return nil
	}
}

// UserAgent sets the UserAgent string to include with the request.
func UserAgent(userAgent string) ClientParam {
	return func(client *Client) error {
		client.userAgent = userAgent
		return nil
	}
}

// HTTPClient sets the `http.Client` that this API client will use to
// to communicate. This allows you to replace the client or tune it to your
// needs.
func HTTPClient(httpClient *http.Client) ClientParam {
	return func(client *Client) error {
		client.httpClient = httpClient
		return nil
	}
}

func (c *Client) doRequest(ctx context.Context, method string, path string, params url.Values, body io.Reader) (*http.Response, error) {
	return c.doRequestWithToken(ctx, method, path, params, body, c.authToken)
}

func (c *Client) doRequestWithToken(ctx context.Context, method string, path string, params url.Values, body io.Reader, token string) (*http.Response, error) {
	destURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	destURL.Path = path

	if params != nil {
		destURL.RawQuery = params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, destURL.String(), body)
	if token != "" {
		req.Header.Set(AuthHeaderKey, token)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}

	return c.httpClient.Do(req)
}

// SignalFlow creates and returns a SignalFlow client that can be used to
// execute streaming jobs.
func (c *Client) SignalFlow(options ...signalflow.ClientParam) (*signalflow.Client, error) {
	options = append(options, signalflow.AccessToken(c.authToken))
	return signalflow.NewClient(options...)
}
