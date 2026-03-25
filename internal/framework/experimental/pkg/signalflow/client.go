// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package flow

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	net    *http.Client
	domain *url.URL
	token  string
}

func NewClient(domain *url.URL, token string, opts ...func(*Client)) *Client {
	c := &Client{
		net:    http.DefaultClient,
		domain: domain,
		token:  token,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) GetExecutionGraph(ctx context.Context, programText string) (ExecutionGraph, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.domain.JoinPath("/v2/signalflow/_/getSignalFlowModel").String(),
		strings.NewReader(programText),
	)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("parseDetectors", "true")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("X-Sf-Token", c.token)
	req.Header.Set("User-Agent", "terraform-provider-signalfx")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.net.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		content, _ := io.ReadAll(resp.Body)
		return nil, &url.Error{
			Op:  "GetExecutionGraph",
			URL: req.URL.EscapedPath(),
			Err: fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, bytes.TrimSpace(content)),
		}
	}

	return NewExecutionGraphFromJSON(resp.Body)
}
