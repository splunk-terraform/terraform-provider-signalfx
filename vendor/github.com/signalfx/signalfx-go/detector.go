package signalfx

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/signalfx/signalfx-go/detector"
)

// DetectorAPIURL is the base URL for interacting with detectors.
const DetectorAPIURL = "/v2/detector"

// CreateDetector creates a detector.
func (c *Client) CreateDetector(detectorRequest *detector.CreateUpdateDetectorRequest) (*detector.Detector, error) {
	payload, err := json.Marshal(detectorRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("POST", DetectorAPIURL, nil, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	finalDetector := &detector.Detector{}

	err = json.NewDecoder(resp.Body).Decode(finalDetector)

	return finalDetector, err
}

// DeleteDetector deletes a detector.
func (c *Client) DeleteDetector(id string) error {
	resp, err := c.doRequest("DELETE", DetectorAPIURL+"/"+id, nil, nil)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("Unexpected status code: " + resp.Status)
	}

	return nil
}

// DisableDetector disables a detector.
func (c *Client) DisableDetector(id string) error {
	resp, err := c.doRequest("PUT", DetectorAPIURL+"/"+id+"/disable", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// EnableDetector enables a detector.
func (c *Client) EnableDetector(id string) error {
	resp, err := c.doRequest("PUT", DetectorAPIURL+"/"+id+"/enable", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetDetector gets a detector.
func (c *Client) GetDetector(id string) (*detector.Detector, error) {
	resp, err := c.doRequest("GET", DetectorAPIURL+"/"+id, nil, nil)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	finalDetector := &detector.Detector{}

	err = json.NewDecoder(resp.Body).Decode(finalDetector)

	return finalDetector, err
}

// UpdateDetector updates a detector.
func (c *Client) UpdateDetector(id string, detectorRequest *detector.CreateUpdateDetectorRequest) (*detector.Detector, error) {
	payload, err := json.Marshal(detectorRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("PUT", DetectorAPIURL+"/"+id, nil, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	finalDetector := &detector.Detector{}

	err = json.NewDecoder(resp.Body).Decode(finalDetector)

	return finalDetector, err
}

// SearchDetector searches for detectors, given a query string in `name`.
func (c *Client) SearchDetectors(limit int, name string, offset int, tags string) (*detector.SearchResults, error) {
	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("name", name)
	params.Add("offset", strconv.Itoa(offset))
	params.Add("tags", tags)

	resp, err := c.doRequest("GET", DetectorAPIURL, params, nil)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	finalDetectors := &detector.SearchResults{}

	err = json.NewDecoder(resp.Body).Decode(finalDetectors)

	return finalDetectors, err
}
