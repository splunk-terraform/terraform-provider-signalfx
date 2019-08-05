package signalfx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/signalfx/signalfx-go/orgtoken"
)

// TokenAPIURL is the base URL for interacting with org tokens.
const TokenAPIURL = "/v2/token"

// CreateOrgToken creates a org token.
func (c *Client) CreateOrgToken(tokenRequest *orgtoken.CreateUpdateTokenRequest) (*orgtoken.Token, error) {
	payload, err := json.Marshal(tokenRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("POST", TokenAPIURL, nil, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalToken := &orgtoken.Token{}

	err = json.NewDecoder(resp.Body).Decode(finalToken)

	return finalToken, err
}

// DeleteOrgToken deletes a token.
func (c *Client) DeleteOrgToken(name string) error {
	resp, err := c.doRequest("DELETE", TokenAPIURL+"/"+name, nil, nil)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		message, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	return nil
}

// GetToken gets a token.
func (c *Client) GetOrgToken(id string) (*orgtoken.Token, error) {
	resp, err := c.doRequest("GET", TokenAPIURL+"/"+id, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalToken := &orgtoken.Token{}

	err = json.NewDecoder(resp.Body).Decode(finalToken)

	return finalToken, err
}

// UpdateToken updates a token.
func (c *Client) UpdateOrgToken(id string, tokenRequest *orgtoken.CreateUpdateTokenRequest) (*orgtoken.Token, error) {
	payload, err := json.Marshal(tokenRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("PUT", TokenAPIURL+"/"+id, nil, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalToken := &orgtoken.Token{}

	err = json.NewDecoder(resp.Body).Decode(finalToken)

	return finalToken, err
}

// SearchToken searches for tokens given a query string in `name`.
func (c *Client) SearchOrgTokens(limit int, name string, offset int) (*orgtoken.SearchResults, error) {
	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("name", name)
	params.Add("offset", strconv.Itoa(offset))

	resp, err := c.doRequest("GET", TokenAPIURL, params, nil)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	finalTokens := &orgtoken.SearchResults{}

	err = json.NewDecoder(resp.Body).Decode(finalTokens)

	return finalTokens, err
}
