package signalfx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/signalfx/signalfx-go/integration"
)

// CreatePagerDutyIntegration creates a PagerDuty integration.
func (c *Client) CreatePagerDutyIntegration(pdi *integration.PagerDutyIntegration) (*integration.PagerDutyIntegration, error) {
	payload, err := json.Marshal(pdi)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("POST", IntegrationAPIURL, nil, bytes.NewReader(payload))

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	finalIntegration := integration.PagerDutyIntegration{}

	err = json.NewDecoder(resp.Body).Decode(&finalIntegration)

	return &finalIntegration, err
}

// GetPagerDutyIntegration retrieves a PagerDuty integration.
func (c *Client) GetPagerDutyIntegration(id string) (*integration.PagerDutyIntegration, error) {
	resp, err := c.doRequest("GET", IntegrationAPIURL+"/"+id, nil, nil)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	finalIntegration := integration.PagerDutyIntegration{}

	err = json.NewDecoder(resp.Body).Decode(&finalIntegration)

	return &finalIntegration, err
}

// UpdatePagerDutyIntegration updates a PagerDuty integration.
func (c *Client) UpdatePagerDutyIntegration(id string, pdi *integration.PagerDutyIntegration) (*integration.PagerDutyIntegration, error) {
	payload, err := json.Marshal(pdi)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("PUT", IntegrationAPIURL+"/"+id, nil, bytes.NewReader(payload))

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	finalIntegration := integration.PagerDutyIntegration{}

	err = json.NewDecoder(resp.Body).Decode(&finalIntegration)

	return &finalIntegration, err
}

// DeletePagerDutyIntegration deletes a PagerDuty integration.
func (c *Client) DeletePagerDutyIntegration(id string) error {
	resp, err := c.doRequest("DELETE", IntegrationAPIURL+"/"+id, nil, nil)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		message, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	return err
}
