package hub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	hubapi "github.com/docker/hub-tool/pkg/hub/api"
)

const (
	// InvitesURL path to the Hub API listing pending invites in an organization
	InvitesURL = "/v2/orgs/%s/invites"
	// InviteURL path to the Hub API managing a pending invite
	InviteURL = "/v2/invites/%s"
	// ResendInviteURL path to the Hub API resending a pending invite
	ResendInviteURL = "/v2/invites/%s/resend"
	// BulkInvitesURL path to the Hub API creating invites in bulk
	BulkInvitesURL = "/v2/invites/bulk"
)

// Invite is a pending invitation to join an organization team.
type Invite struct {
	ID              string `json:"id"`
	InviterUsername string `json:"inviter_username"`
	Invitee         string `json:"invitee"`
	Org             string `json:"org"`
	Team            string `json:"team"`
	CreatedAt       string `json:"created_at"`
}

type orgInvitesResponse struct {
	Data []hubapi.Invite `json:"data"`
}

// BulkInviteOptions describes a bulk invite request.
type BulkInviteOptions struct {
	Org      string   `json:"org"`
	Team     string   `json:"team,omitempty"`
	Role     string   `json:"role,omitempty"`
	Invitees []string `json:"invitees"`
	DryRun   bool     `json:"dry_run,omitempty"`
}

// BulkInviteResult is the validation or creation result for an invitee.
type BulkInviteResult struct {
	Invitee string  `json:"invitee"`
	Status  string  `json:"status"`
	Invite  *Invite `json:"invite,omitempty"`
}

type bulkInviteAPIResponse struct {
	Invitees []bulkInviteResult `json:"invitees,omitempty"`
}

type nestedBulkInviteAPIResponse struct {
	Invitees bulkInviteAPIResponse `json:"invitees,omitempty"`
}

type bulkInviteResult struct {
	Invitee *string        `json:"invitee,omitempty"`
	Status  *string        `json:"status,omitempty"`
	Invite  *hubapi.Invite `json:"invite,omitempty"`
}

// GetInvites lists all pending invites in an organization.
func (c *Client) GetInvites(organization string) ([]Invite, error) {
	var hubResponse orgInvitesResponse
	if err := c.getJSON(c.domain+fmt.Sprintf(InvitesURL, organization), &hubResponse); err != nil {
		return nil, err
	}

	invites := make([]Invite, 0, len(hubResponse.Data))
	for _, result := range hubResponse.Data {
		invites = append(invites, mapInvite(result))
	}
	return invites, nil
}

// RemoveInvite cancels a pending invite.
func (c *Client) RemoveInvite(inviteID string) error {
	return c.deleteHubResource(c.domain + fmt.Sprintf(InviteURL, inviteID))
}

// ResendInvite resends a pending invite.
func (c *Client) ResendInvite(inviteID string) error {
	req, err := http.NewRequest(http.MethodPatch, c.domain+fmt.Sprintf(ResendInviteURL, inviteID), nil)
	if err != nil {
		return err
	}
	_, err = c.doRequest(req, withHubToken(c.token))
	return err
}

// BulkInvite creates or validates multiple organization invites.
func (c *Client) BulkInvite(opts BulkInviteOptions) ([]BulkInviteResult, error) {
	data, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.domain+BulkInvitesURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	response, err := c.doRequest(req, withHubToken(c.token))
	if err != nil {
		return nil, err
	}

	var nested nestedBulkInviteAPIResponse
	if err := json.Unmarshal(response, &nested); err != nil {
		return nil, err
	}
	results := nested.Invitees.Invitees
	if len(results) == 0 {
		var flat bulkInviteAPIResponse
		if err := json.Unmarshal(response, &flat); err != nil {
			return nil, err
		}
		results = flat.Invitees
	}

	invitees := make([]BulkInviteResult, 0, len(results))
	for _, result := range results {
		invite := mapOptionalInvite(result.Invite)
		invitees = append(invitees, BulkInviteResult{
			Invitee: ptrValue(result.Invitee),
			Status:  ptrValue(result.Status),
			Invite:  invite,
		})
	}
	return invitees, nil
}

func mapOptionalInvite(result *hubapi.Invite) *Invite {
	if result == nil {
		return nil
	}
	invite := mapInvite(*result)
	return &invite
}

func mapInvite(result hubapi.Invite) Invite {
	return Invite{
		ID:              ptrValue(result.Id),
		InviterUsername: ptrValue(result.InviterUsername),
		Invitee:         ptrValue(result.Invitee),
		Org:             ptrValue(result.Org),
		Team:            ptrValue(result.Team),
		CreatedAt:       ptrValue(result.CreatedAt),
	}
}
