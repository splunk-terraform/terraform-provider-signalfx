package signalfx

import (
	"io"
	"net/http"
	"net/url"
	"time"
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
	}

	for _, option := range options {
		option(client)
	}

	return client, nil
}

// APIUrl sets the URL that our client will communicate with, allowing
// it to be adjusted to another URL for testing or communication with other
// SignalFx clusters.
func APIUrl(apiURL string) ClientParam {
	return func(client *Client) error {
		client.baseURL = apiURL
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

func (c *Client) doRequest(method string, path string, params url.Values, body io.Reader) (*http.Response, error) {
	destURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	destURL.Path = path

	if params != nil {
		destURL.RawQuery = params.Encode()
	}
	req, err := http.NewRequest(method, destURL.String(), body)
	req.Header.Set(AuthHeaderKey, c.authToken)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Do(req)
}
