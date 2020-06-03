package signalfx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/signalfx/signalfx-go/orgtoken"
)

// TokenAPIURL is the base URL for interacting with org tokens.
const TokenAPIURL = "/v2/token"

// CreateOrgToken creates a org token.
func (c *Client) CreateOrgToken(ctx context.Context, tokenRequest *orgtoken.CreateUpdateTokenRequest) (*orgtoken.Token, error) {
	payload, err := json.Marshal(tokenRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "POST", TokenAPIURL, nil, bytes.NewReader(payload))
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalToken := &orgtoken.Token{}

	err = json.NewDecoder(resp.Body).Decode(finalToken)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalToken, err
}

// DeleteOrgToken deletes a token.
func (c *Client) DeleteOrgToken(ctx context.Context, name string) error {
	encodedName := url.PathEscape(name)
	resp, err := c.doRequest(ctx, "DELETE", TokenAPIURL+"/"+encodedName, nil, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		message, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return nil
}

// GetOrgToken gets a token.
func (c *Client) GetOrgToken(ctx context.Context, id string) (*orgtoken.Token, error) {
	encodedName := url.PathEscape(id)
	resp, err := c.doRequest(ctx, "GET", TokenAPIURL+"/"+encodedName, nil, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalToken := &orgtoken.Token{}

	err = json.NewDecoder(resp.Body).Decode(finalToken)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalToken, err
}

// UpdateOrgToken updates a token.
func (c *Client) UpdateOrgToken(ctx context.Context, id string, tokenRequest *orgtoken.CreateUpdateTokenRequest) (*orgtoken.Token, error) {
	payload, err := json.Marshal(tokenRequest)
	if err != nil {
		return nil, err
	}

	encodedName := url.PathEscape(id)
	resp, err := c.doRequest(ctx, "PUT", TokenAPIURL+"/"+encodedName, nil, bytes.NewReader(payload))
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalToken := &orgtoken.Token{}

	err = json.NewDecoder(resp.Body).Decode(finalToken)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalToken, err
}

// SearchOrgTokens searches for tokens given a query string in `name`.
func (c *Client) SearchOrgTokens(ctx context.Context, limit int, name string, offset int) (*orgtoken.SearchResults, error) {
	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("name", url.PathEscape(name))
	params.Add("offset", strconv.Itoa(offset))

	resp, err := c.doRequest(ctx, "GET", TokenAPIURL, params, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	finalTokens := &orgtoken.SearchResults{}

	err = json.NewDecoder(resp.Body).Decode(finalTokens)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalTokens, err
}
