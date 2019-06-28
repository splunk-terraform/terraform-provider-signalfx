package signalfx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/signalfx/signalfx-go/dashboard_group"
)

// DashboardGroupAPIURL is the base URL for interacting with dashboard.
const DashboardGroupAPIURL = "/v2/dashboardgroup"

// TODO Clone dashboard to group

// CreateDashboardGroup creates a dashboard.
func (c *Client) CreateDashboardGroup(dashboardGroupRequest *dashboard_group.CreateUpdateDashboardGroupRequest) (*dashboard_group.DashboardGroup, error) {
	payload, err := json.Marshal(dashboardGroupRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("POST", DashboardGroupAPIURL, nil, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	finalDashboardGroup := &dashboard_group.DashboardGroup{}

	err = json.NewDecoder(resp.Body).Decode(finalDashboardGroup)

	return finalDashboardGroup, err
}

// DeleteDashboardGroup deletes a dashboard.
func (c *Client) DeleteDashboardGroup(id string) error {
	resp, err := c.doRequest("DELETE", DashboardGroupAPIURL+"/"+id, nil, nil)

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

// GetDashboardGroup gets a dashboard group.
func (c *Client) GetDashboardGroup(id string) (*dashboard_group.DashboardGroup, error) {
	resp, err := c.doRequest("GET", DashboardGroupAPIURL+"/"+id, nil, nil)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	finalDashboardGroup := &dashboard_group.DashboardGroup{}

	err = json.NewDecoder(resp.Body).Decode(finalDashboardGroup)

	return finalDashboardGroup, err
}

// UpdateDashboardGroup updates a dashboard group.
func (c *Client) UpdateDashboardGroup(id string, dashboardGroupRequest *dashboard_group.CreateUpdateDashboardGroupRequest) (*dashboard_group.DashboardGroup, error) {
	payload, err := json.Marshal(dashboardGroupRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("PUT", DashboardGroupAPIURL+"/"+id, nil, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}

	finalDashboardGroup := &dashboard_group.DashboardGroup{}

	err = json.NewDecoder(resp.Body).Decode(finalDashboardGroup)

	return finalDashboardGroup, err
}

// SearchDashboardGroup searches for dashboard groups, given a query string in `name`.
func (c *Client) SearchDashboardGroups(limit int, name string, offset int) (*dashboard_group.SearchResult, error) {
	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("name", name)
	params.Add("offset", strconv.Itoa(offset))

	resp, err := c.doRequest("GET", DashboardGroupAPIURL, params, nil)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	finalDashboardGroups := &dashboard_group.SearchResult{}

	err = json.NewDecoder(resp.Body).Decode(finalDashboardGroups)

	return finalDashboardGroups, err
}
