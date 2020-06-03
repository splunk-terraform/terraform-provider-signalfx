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

	"github.com/signalfx/signalfx-go/detector"
)

// DetectorAPIURL is the base URL for interacting with detectors.
const DetectorAPIURL = "/v2/detector"

// CreateDetector creates a detector.
func (c *Client) CreateDetector(ctx context.Context, detectorRequest *detector.CreateUpdateDetectorRequest) (*detector.Detector, error) {
	payload, err := json.Marshal(detectorRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "POST", DetectorAPIURL, nil, bytes.NewReader(payload))
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

	finalDetector := &detector.Detector{}

	err = json.NewDecoder(resp.Body).Decode(finalDetector)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDetector, err
}

// DeleteDetector deletes a detector.
func (c *Client) DeleteDetector(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, "DELETE", DetectorAPIURL+"/"+id, nil, nil)
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

// DisableDetector disables a detector.
func (c *Client) DisableDetector(ctx context.Context, id string, labels []string) error {
	payload, err := json.Marshal(labels)
	if err != nil {
		return err
	}

	resp, err := c.doRequest(ctx, "PUT", DetectorAPIURL+"/"+id+"/disable", nil, bytes.NewReader(payload))
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

// EnableDetector enables a detector.
func (c *Client) EnableDetector(ctx context.Context, id string, labels []string) error {
	payload, err := json.Marshal(labels)
	if err != nil {
		return err
	}

	resp, err := c.doRequest(ctx, "PUT", DetectorAPIURL+"/"+id+"/enable", nil, bytes.NewReader(payload))
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

// GetDetector gets a detector.
func (c *Client) GetDetector(ctx context.Context, id string) (*detector.Detector, error) {
	resp, err := c.doRequest(ctx, "GET", DetectorAPIURL+"/"+id, nil, nil)
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

	finalDetector := &detector.Detector{}

	err = json.NewDecoder(resp.Body).Decode(finalDetector)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDetector, err
}

// UpdateDetector updates a detector.
func (c *Client) UpdateDetector(ctx context.Context, id string, detectorRequest *detector.CreateUpdateDetectorRequest) (*detector.Detector, error) {
	payload, err := json.Marshal(detectorRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "PUT", DetectorAPIURL+"/"+id, nil, bytes.NewReader(payload))
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

	finalDetector := &detector.Detector{}

	err = json.NewDecoder(resp.Body).Decode(finalDetector)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDetector, err
}

// SearchDetectors searches for detectors, given a query string in `name`.
func (c *Client) SearchDetectors(ctx context.Context, limit int, name string, offset int, tags string) (*detector.SearchResults, error) {
	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("name", name)
	params.Add("offset", strconv.Itoa(offset))
	if tags != "" {
		params.Add("tags", tags)
	}

	resp, err := c.doRequest(ctx, "GET", DetectorAPIURL, params, nil)
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

	finalDetectors := &detector.SearchResults{}

	err = json.NewDecoder(resp.Body).Decode(finalDetectors)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDetectors, err
}

// GetDetectorEvents gets a detector's events.
func (c *Client) GetDetectorEvents(ctx context.Context, id string, from int, to int, offset int, limit int) ([]*detector.Event, error) {
	params := url.Values{}
	params.Add("from", strconv.Itoa(from))
	params.Add("to", strconv.Itoa(to))
	params.Add("offset", strconv.Itoa(offset))
	params.Add("limit", strconv.Itoa(limit))
	resp, err := c.doRequest(ctx, "GET", DetectorAPIURL+"/"+id+"/events", params, nil)
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

	var events []*detector.Event

	err = json.NewDecoder(resp.Body).Decode(&events)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return events, err
}

// GetDetectorIncidents gets a detector's incidents.
func (c *Client) GetDetectorIncidents(ctx context.Context, id string, offset int, limit int) ([]*detector.Incident, error) {
	params := url.Values{}
	params.Add("offset", strconv.Itoa(offset))
	params.Add("limit", strconv.Itoa(limit))
	resp, err := c.doRequest(ctx, "GET", DetectorAPIURL+"/"+id+"/incidents", params, nil)
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

	var incidents []*detector.Incident

	err = json.NewDecoder(resp.Body).Decode(&incidents)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return incidents, err
}
