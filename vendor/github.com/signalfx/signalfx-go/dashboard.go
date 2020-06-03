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

	"github.com/signalfx/signalfx-go/dashboard"
)

// TODO Create simple dashboard

// DashboardAPIURL is the base URL for interacting with dashboard.
const DashboardAPIURL = "/v2/dashboard"

// CreateDashboard creates a dashboard.
func (c *Client) CreateDashboard(ctx context.Context, dashboardRequest *dashboard.CreateUpdateDashboardRequest) (*dashboard.Dashboard, error) {
	payload, err := json.Marshal(dashboardRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "POST", DashboardAPIURL, nil, bytes.NewReader(payload))
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

	finalDashboard := &dashboard.Dashboard{}

	err = json.NewDecoder(resp.Body).Decode(finalDashboard)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDashboard, err
}

// DeleteDashboard deletes a dashboard.
func (c *Client) DeleteDashboard(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, "DELETE", DashboardAPIURL+"/"+id, nil, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		message, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Unexpected status code: %d: %s", resp.StatusCode, message)
	}
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return nil
}

// GetDashboard gets a dashboard.
func (c *Client) GetDashboard(ctx context.Context, id string) (*dashboard.Dashboard, error) {
	resp, err := c.doRequest(ctx, "GET", DashboardAPIURL+"/"+id, nil, nil)
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

	finalDashboard := &dashboard.Dashboard{}

	err = json.NewDecoder(resp.Body).Decode(finalDashboard)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDashboard, err
}

// UpdateDashboard updates a dashboard.
func (c *Client) UpdateDashboard(ctx context.Context, id string, dashboardRequest *dashboard.CreateUpdateDashboardRequest) (*dashboard.Dashboard, error) {
	payload, err := json.Marshal(dashboardRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "PUT", DashboardAPIURL+"/"+id, nil, bytes.NewReader(payload))
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

	finalDashboard := &dashboard.Dashboard{}

	err = json.NewDecoder(resp.Body).Decode(finalDashboard)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDashboard, err
}

// SearchDashboard searches for dashboards, given a query string in `name`.
func (c *Client) SearchDashboard(ctx context.Context, limit int, name string, offset int, tags string) (*dashboard.SearchResult, error) {
	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("name", name)
	params.Add("offset", strconv.Itoa(offset))
	params.Add("tags", tags)

	resp, err := c.doRequest(ctx, "GET", DashboardAPIURL, params, nil)
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

	finalDashboards := &dashboard.SearchResult{}

	err = json.NewDecoder(resp.Body).Decode(finalDashboards)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalDashboards, err
}
