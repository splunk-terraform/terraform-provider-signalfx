package signalfx

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

// TeamAPIURL is the base URL for interacting with teams.
const TeamAPIURL = "/v2/team"

// TODO Update team members

// Team is a team.
type Team struct {
	Description       string   `json:"description,omitempty"`
	ID                string   `json:"id,omitempty"`
	Members           []string `json:"members,omitempty"`
	Name              string   `json:"name,omitempty"`
	NotificationLists struct {
		Critical []string `json:"critical,omitempty"`
		Default  []string `json:"default,omitempty"`
		Info     []string `json:"info,omitempty"`
		Major    []string `json:"major,omitempty"`
		Minor    []string `json:"minor,omitempty"`
		Warning  []string `json:"warning,omitempty"`
	} `json:"notificationLists,omitempty"`
}

// TeamSearch is the result of a query for Team
type TeamSearch struct {
	Count   int64 `json:"count,omitempty"`
	Results []Team
}

// CreateTeam creates a team.
func (c *Client) CreateTeam(team *Team) (*Team, error) {
	payload, err := json.Marshal(team)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("POST", TeamAPIURL, nil, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	finalTeam := &Team{}

	err = json.NewDecoder(resp.Body).Decode(finalTeam)

	return finalTeam, err
}

// DeleteTeam deletes a team.
func (c *Client) DeleteTeam(id string) error {
	resp, err := c.doRequest("DELETE", TeamAPIURL+"/"+id, nil, nil)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("Unexpected status code: " + resp.Status)
	}

	return nil
}

// GetTeam gets a team.
func (c *Client) GetTeam(id string) (*Team, error) {
	resp, err := c.doRequest("GET", TeamAPIURL+"/"+id, nil, nil)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	finalTeam := &Team{}

	err = json.NewDecoder(resp.Body).Decode(finalTeam)

	return finalTeam, err
}

// UpdateTeam updates a team.
func (c *Client) UpdateTeam(id string, team *Team) (*Team, error) {
	payload, err := json.Marshal(team)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("PUT", TeamAPIURL+"/"+id, nil, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	finalTeam := &Team{}

	err = json.NewDecoder(resp.Body).Decode(finalTeam)

	return finalTeam, err
}

// SearchTeam searches for teams, given a query string in `name`.
func (c *Client) SearchTeam(limit int, name string, offset int, tags string) (*TeamSearch, error) {
	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("name", name)
	params.Add("offset", strconv.Itoa(offset))
	params.Add("tags", tags)

	resp, err := c.doRequest("GET", TeamAPIURL, params, nil)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	finalTeams := &TeamSearch{}

	err = json.NewDecoder(resp.Body).Decode(finalTeams)

	return finalTeams, err
}
