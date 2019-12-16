package signalfx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/signalfx/signalfx-go/datalink"
)

// DataLinkAPIURL is the base URL for interacting with data link.
const DataLinkAPIURL = "/v2/crosslink"

// CreateDataLink creates a data link.
func (c *Client) CreateDataLink(dataLinkRequest *datalink.CreateUpdateDataLinkRequest) (*datalink.DataLink, error) {
	payload, err := json.Marshal(dataLinkRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("POST", DataLinkAPIURL, nil, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalDataLink := &datalink.DataLink{}

	err = json.NewDecoder(resp.Body).Decode(finalDataLink)

	return finalDataLink, err
}

// DeleteDataLink deletes a data link.
func (c *Client) DeleteDataLink(id string) error {
	resp, err := c.doRequest("DELETE", DataLinkAPIURL+"/"+id, nil, nil)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// The API returns a 200 here, which I think is a mistake so covering for
	// future changes.
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	return nil
}

// GetDataLink gets a data link.
func (c *Client) GetDataLink(id string) (*datalink.DataLink, error) {
	resp, err := c.doRequest("GET", DataLinkAPIURL+"/"+id, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalDataLink := &datalink.DataLink{}

	err = json.NewDecoder(resp.Body).Decode(finalDataLink)

	return finalDataLink, err
}

// UpdateDataLink updates a data link.
func (c *Client) UpdateDataLink(id string, dataLinkRequest *datalink.CreateUpdateDataLinkRequest) (*datalink.DataLink, error) {
	payload, err := json.Marshal(dataLinkRequest)
	if err != nil {
		return nil, err
	}

	encodedName := url.PathEscape(id)
	resp, err := c.doRequest("PUT", DataLinkAPIURL+"/"+encodedName, nil, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalDataLink := &datalink.DataLink{}

	err = json.NewDecoder(resp.Body).Decode(finalDataLink)

	return finalDataLink, err
}

// SearchDataLinks searches for data links given a query string in `name`.
func (c *Client) SearchDataLinks(limit int, context string, offset int) (*datalink.SearchResults, error) {
	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("context", context)
	params.Add("offset", strconv.Itoa(offset))

	resp, err := c.doRequest("GET", DataLinkAPIURL, params, nil)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	finalDataLinks := &datalink.SearchResults{}

	err = json.NewDecoder(resp.Body).Decode(finalDataLinks)

	return finalDataLinks, err
}
