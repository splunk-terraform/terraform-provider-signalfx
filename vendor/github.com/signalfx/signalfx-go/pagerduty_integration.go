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

	"github.com/signalfx/signalfx-go/integration"
)

// CreatePagerDutyIntegration creates a PagerDuty integration.
func (c *Client) CreatePagerDutyIntegration(ctx context.Context, pdi *integration.PagerDutyIntegration) (*integration.PagerDutyIntegration, error) {
	payload, err := json.Marshal(pdi)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "POST", IntegrationAPIURL, nil, bytes.NewReader(payload))
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

	finalIntegration := integration.PagerDutyIntegration{}

	err = json.NewDecoder(resp.Body).Decode(&finalIntegration)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return &finalIntegration, err
}

// GetPagerDutyIntegration retrieves a PagerDuty integration.
func (c *Client) GetPagerDutyIntegration(ctx context.Context, id string) (*integration.PagerDutyIntegration, error) {
	resp, err := c.doRequest(ctx, "GET", IntegrationAPIURL+"/"+id, nil, nil)
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

	finalIntegration := integration.PagerDutyIntegration{}

	err = json.NewDecoder(resp.Body).Decode(&finalIntegration)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return &finalIntegration, err
}

// GetPagerDutyIntegrationByName retrieves a PagerDuty integration by name.
func (c *Client) GetPagerDutyIntegrationByName(ctx context.Context, name string) (*integration.PagerDutyIntegration, error) {
	params := url.Values{}
	params.Add("type", "PagerDuty")
	params.Add("name", name)

	resp, err := c.doRequest(ctx, "GET", IntegrationAPIURL, params, nil)
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

	integrationList := integration.PagerDutyIntegrationList{}

	err = json.NewDecoder(resp.Body).Decode(&integrationList)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	if integrationList.Count == 0 {
		return nil, nil
	}

	for _, integration := range integrationList.Results {
		if integration.Name == name {
			return &integration, nil
		}
	}

	return nil, err
}

// UpdatePagerDutyIntegration updates a PagerDuty integration.
func (c *Client) UpdatePagerDutyIntegration(ctx context.Context, id string, pdi *integration.PagerDutyIntegration) (*integration.PagerDutyIntegration, error) {
	payload, err := json.Marshal(pdi)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "PUT", IntegrationAPIURL+"/"+id, nil, bytes.NewReader(payload))
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

	finalIntegration := integration.PagerDutyIntegration{}

	err = json.NewDecoder(resp.Body).Decode(&finalIntegration)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return &finalIntegration, err
}

// DeletePagerDutyIntegration deletes a PagerDuty integration.
func (c *Client) DeletePagerDutyIntegration(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, "DELETE", IntegrationAPIURL+"/"+id, nil, nil)
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

	return err
}
