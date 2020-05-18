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

	"github.com/signalfx/signalfx-go/organization"
)

// OrganizationAPIURL is the base URL for interacting with detectors.
const OrganizationAPIURL = "/v2/organization"
const OrganizationMemberAPIURL = "/v2/organization/member"
const OrganizationMembersAPIURL = "/v2/organization/members"

// GetOrganization gets an organization.
func (c *Client) GetOrganization(id string) (*organization.Organization, error) {
	resp, err := c.doRequest("GET", OrganizationAPIURL+"/"+id, nil, nil)
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

	finalOrganization := &organization.Organization{}

	err = json.NewDecoder(resp.Body).Decode(finalOrganization)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalOrganization, err
}

// GetMember gets a member.
func (c *Client) GetMember(id string) (*organization.Member, error) {
	resp, err := c.doRequest("GET", OrganizationMemberAPIURL+"/"+id, nil, nil)
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

	finalMember := &organization.Member{}

	err = json.NewDecoder(resp.Body).Decode(finalMember)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalMember, err
}

// DeleteMember deletes a detector.
func (c *Client) DeleteMember(id string) error {
	resp, err := c.doRequest("DELETE", OrganizationMemberAPIURL+"/"+id, nil, nil)
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

// InviteMember invites a member to the organization.
func (c *Client) InviteMember(inviteRequest *organization.CreateUpdateMemberRequest) (*organization.Member, error) {
	payload, err := json.Marshal(inviteRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("POST", OrganizationMemberAPIURL, nil, bytes.NewReader(payload))
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

	finalMember := &organization.Member{}

	err = json.NewDecoder(resp.Body).Decode(finalMember)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalMember, err
}

// InviteMembers invites many members to the organization.
func (c *Client) InviteMembers(inviteRequest *organization.InviteMembersRequest) (*organization.InviteMembersRequest, error) {
	payload, err := json.Marshal(inviteRequest)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("POST", OrganizationMembersAPIURL, nil, bytes.NewReader(payload))
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

	finalMembers := &organization.InviteMembersRequest{}

	err = json.NewDecoder(resp.Body).Decode(finalMembers)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalMembers, err
}

// GetOrganizationMembers gets members for an org, with an optional search.
func (c *Client) GetOrganizationMembers(limit int, query string, offset int, orderBy string) (*organization.MemberSearchResults, error) {
	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("query", query)
	params.Add("offset", strconv.Itoa(offset))
	params.Add("orderBy", orderBy)

	resp, err := c.doRequest("GET", OrganizationMemberAPIURL, params, nil)
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

	finalMembers := &organization.MemberSearchResults{}

	err = json.NewDecoder(resp.Body).Decode(finalMembers)
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	return finalMembers, err
}
