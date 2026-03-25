// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Metadata struct {
	Arguments  []Argument `json:"arguments,omitempty"`
	VizOptions VizOptions `json:"vizOptions,omitempty"`
}

type Argument struct {
	Name         string   `json:"argName,omitempty"`
	Type         string   `json:"argType,omitempty"`
	Label        string   `json:"label,omitempty"`
	Description  string   `json:"description,omitempty"`
	DefaultValue any      `json:"defaultValue,omitempty"`
	Unit         string   `json:"unit,omitempty"`
	Metric       string   `json:"metric,omitempty"`
	Dimensions   []string `json:"dimensions,omitempty"`
}

type VizOptions struct {
	Prefix string `json:"valuePrefix,omitempty"`
	Suffix string `json:"valueSuffix,omitempty"`
	Unit   string `json:"valueUnit,omitempty"`
}

type Client struct {
	net    *http.Client
	token  string
	domain *url.URL
}

func NewClient(domain *url.URL, token string, opts ...func(*Client)) *Client {
	c := &Client{
		token:  token,
		domain: domain,
		net:    &http.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) GetModuleFunctionMetadata(ctx context.Context, modulePath string, moduleName string, functionName string) (*Metadata, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.domain.JoinPath("/v2/signalflow/_/extractMetadata").String(),
		http.NoBody,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Sf-Token", c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "text/plain")

	q := req.URL.Query()
	if modulePath != "" {
		q.Set("modulePath", modulePath)
	}
	if moduleName != "" {
		q.Set("moduleName", moduleName)
	}
	if functionName != "" {
		q.Set("functionName", functionName)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.net.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		content, err := io.ReadAll(resp.Body)
		if err != nil {
			content = []byte("unable to dump response")
		}
		return nil, &url.Error{
			Op:  "GetModuleFunctionMetadata",
			URL: req.URL.EscapedPath(),
			Err: fmt.Errorf("response: %s", bytes.TrimSpace(content)),
		}
	}

	var metadata Metadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, &url.Error{
			Op:  "GetModuleFunctionMetadata",
			URL: req.URL.EscapedPath(),
			Err: fmt.Errorf("json parsing: %w", err),
		}
	}

	return &metadata, nil
}
