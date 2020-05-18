package signalfx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
func (c *Client) CreateDashboardGroup(dashboardGroupRequest *dashboard_group.CreateUpdateDashboardGroupRequest, skipImplicitDashboard bool) (*dashboard_group.DashboardGroup, error) {
	payload, err := json.Marshal(dashboardGroupRequest)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	if skipImplicitDashboard {
		params.Add("empty", "true")
	}

	resp, err := c.doRequest("POST", DashboardGroupAPIURL, params, bytes.NewReader(payload))
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

	finalDashboardGroup := &dashboard_group.DashboardGroup{}

	err = json.NewDecoder(resp.Body).Decode(finalDashboardGroup)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDashboardGroup, err
}

// DeleteDashboardGroup deletes a dashboard.
func (c *Client) DeleteDashboardGroup(id string) error {
	resp, err := c.doRequest("DELETE", DashboardGroupAPIURL+"/"+id, nil, nil)
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

// GetDashboardGroup gets a dashboard group.
func (c *Client) GetDashboardGroup(id string) (*dashboard_group.DashboardGroup, error) {
	resp, err := c.doRequest("GET", DashboardGroupAPIURL+"/"+id, nil, nil)
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

	finalDashboardGroup := &dashboard_group.DashboardGroup{}

	err = json.NewDecoder(resp.Body).Decode(finalDashboardGroup)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDashboardGroup, err
}

// UpdateDashboardGroup updates a dashboard group.
func (c *Client) UpdateDashboardGroup(id string, dashboardGroupRequest *dashboard_group.CreateUpdateDashboardGroupRequest) (*dashboard_group.DashboardGroup, error) {
	payload, err := json.Marshal(dashboardGroupRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("PUT", DashboardGroupAPIURL+"/"+id, nil, bytes.NewReader(payload))
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

	finalDashboardGroup := &dashboard_group.DashboardGroup{}

	err = json.NewDecoder(resp.Body).Decode(finalDashboardGroup)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDashboardGroup, err
}

// SearchDashboardGroup searches for dashboard groups, given a query string in `name`.
func (c *Client) SearchDashboardGroups(limit int, name string, offset int) (*dashboard_group.SearchResult, error) {
	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("name", name)
	params.Add("offset", strconv.Itoa(offset))

	resp, err := c.doRequest("GET", DashboardGroupAPIURL, params, nil)
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

	finalDashboardGroups := &dashboard_group.SearchResult{}

	err = json.NewDecoder(resp.Body).Decode(finalDashboardGroups)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDashboardGroups, err
}
