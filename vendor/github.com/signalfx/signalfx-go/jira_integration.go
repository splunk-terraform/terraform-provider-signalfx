package signalfx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/signalfx/signalfx-go/integration"
)

// CreateJiraIntegration creates an Jira integration.
func (c *Client) CreateJiraIntegration(ji *integration.JiraIntegration) (*integration.JiraIntegration, error) {
	payload, err := json.Marshal(ji)
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

	finalIntegration := integration.JiraIntegration{}

	err = json.NewDecoder(resp.Body).Decode(&finalIntegration)

	return &finalIntegration, err
}

// GetJiraIntegration retrieves an Jira integration.
func (c *Client) GetJiraIntegration(id string) (*integration.JiraIntegration, error) {
	resp, err := c.doRequest("GET", IntegrationAPIURL+"/"+id, nil, nil)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	finalIntegration := integration.JiraIntegration{}

	err = json.NewDecoder(resp.Body).Decode(&finalIntegration)

	return &finalIntegration, err
}

// UpdateJiraIntegration updates an Jira integration.
func (c *Client) UpdateJiraIntegration(id string, ji *integration.JiraIntegration) (*integration.JiraIntegration, error) {
	payload, err := json.Marshal(ji)
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

	finalIntegration := integration.JiraIntegration{}

	err = json.NewDecoder(resp.Body).Decode(&finalIntegration)

	return &finalIntegration, err
}

// DeleteJiraIntegration deletes an Jira integration.
func (c *Client) DeleteJiraIntegration(id string) error {
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
