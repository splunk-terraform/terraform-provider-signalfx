package signalfx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/signalfx/signalfx-go/integration"
)

// CreateWebhookIntegration creates an Webhook integration.
func (c *Client) CreateWebhookIntegration(oi *integration.WebhookIntegration) (*integration.WebhookIntegration, error) {
	payload, err := json.Marshal(oi)
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

	finalIntegration := integration.WebhookIntegration{}

	err = json.NewDecoder(resp.Body).Decode(&finalIntegration)

	return &finalIntegration, err
}

// GetWebhookIntegration retrieves an Webhook integration.
func (c *Client) GetWebhookIntegration(id string) (*integration.WebhookIntegration, error) {
	resp, err := c.doRequest("GET", IntegrationAPIURL+"/"+id, nil, nil)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	finalIntegration := integration.WebhookIntegration{}

	err = json.NewDecoder(resp.Body).Decode(&finalIntegration)

	return &finalIntegration, err
}

// UpdateWebhookIntegration updates an Webhook integration.
func (c *Client) UpdateWebhookIntegration(id string, oi *integration.WebhookIntegration) (*integration.WebhookIntegration, error) {
	payload, err := json.Marshal(oi)
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

	finalIntegration := integration.WebhookIntegration{}

	err = json.NewDecoder(resp.Body).Decode(&finalIntegration)

	return &finalIntegration, err
}

// DeleteWebhookIntegration deletes an Webhook integration.
func (c *Client) DeleteWebhookIntegration(id string) error {
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
