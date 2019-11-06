package signalfx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/signalfx/signalfx-go/alertmuting"
)

// AlertMutingRuleAPIURL is the base URL for interacting with alert muting rules.
const AlertMutingRuleAPIURL = "/v2/alertmuting"

// CreateAlertMutingRule creates an alert muting rule.
func (c *Client) CreateAlertMutingRule(muteRequest *alertmuting.CreateUpdateAlertMutingRuleRequest) (*alertmuting.AlertMutingRule, error) {
	payload, err := json.Marshal(muteRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("POST", AlertMutingRuleAPIURL, nil, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalRule := &alertmuting.AlertMutingRule{}

	err = json.NewDecoder(resp.Body).Decode(finalRule)

	return finalRule, err
}

// DeleteAlertMutingRule deletes an alert muting rule.
func (c *Client) DeleteAlertMutingRule(name string) error {
	resp, err := c.doRequest("DELETE", AlertMutingRuleAPIURL+"/"+name, nil, nil)

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

// GetAlertMutingRule gets an alert muting rule.
func (c *Client) GetAlertMutingRule(id string) (*alertmuting.AlertMutingRule, error) {
	resp, err := c.doRequest("GET", AlertMutingRuleAPIURL+"/"+id, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalRule := &alertmuting.AlertMutingRule{}

	err = json.NewDecoder(resp.Body).Decode(finalRule)

	return finalRule, err
}

// UpdateAlertMutingRule updates an alert muting rule.
func (c *Client) UpdateAlertMutingRule(id string, muteRequest *alertmuting.CreateUpdateAlertMutingRuleRequest) (*alertmuting.AlertMutingRule, error) {
	payload, err := json.Marshal(muteRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("PUT", AlertMutingRuleAPIURL+"/"+id, nil, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bad status %d: %s", resp.StatusCode, message)
	}

	finalRule := &alertmuting.AlertMutingRule{}

	err = json.NewDecoder(resp.Body).Decode(finalRule)

	return finalRule, err
}

// SearchAlertMutingRules searches for alert muting rules given a query string in `name`.
func (c *Client) SearchAlertMutingRules(include string, limit int, name string, offset int) (*alertmuting.SearchResult, error) {
	params := url.Values{}
	params.Add("include", include)
	params.Add("limit", strconv.Itoa(limit))
	params.Add("name", name)
	params.Add("offset", strconv.Itoa(offset))

	resp, err := c.doRequest("GET", AlertMutingRuleAPIURL, params, nil)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	finalRules := &alertmuting.SearchResult{}

	err = json.NewDecoder(resp.Body).Decode(finalRules)

	return finalRules, err
}
