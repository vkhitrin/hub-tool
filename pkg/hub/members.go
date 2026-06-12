/*
   Copyright 2020 Docker Hub Tool authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package hub

import (
	"fmt"

	hubapi "github.com/docker/hub-tool/pkg/hub/api"
)

const (
	//MembersURL path to the Hub API listing the members in an organization
	MembersURL = "/v2/orgs/%s/members/"
	//MembersPerTeamURL path to the Hub API listing the members in a team
	MembersPerTeamURL = "/v2/orgs/%s/groups/%s/members/"
)

// Member is a user part of an organization
type Member struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

// GetMembers lists all the members in an organization
func (c *Client) GetMembers(organization string) ([]Member, error) {
	rawURL, err := paginatedURL(c.domain+fmt.Sprintf(MembersURL, organization), itemsPerPage)
	if err != nil {
		return nil, err
	}

	return c.getAllMembers(rawURL)
}

// GetMembersCount return the number of members in an organization
func (c *Client) GetMembersCount(organization string) (int, error) {
	rawURL, err := paginatedURL(c.domain+fmt.Sprintf(MembersURL, organization), 1)
	if err != nil {
		return 0, err
	}
	var hubResponse hubapi.OrgMemberPaginated
	if err := c.getJSON(rawURL, &hubResponse); err != nil {
		return 0, err
	}
	return int(ptrValue(hubResponse.Count)), nil
}

// GetMembersPerTeam returns the members of a team in an organization
func (c *Client) GetMembersPerTeam(organization, team string) ([]Member, error) {
	rawURL, err := paginatedURL(c.domain+fmt.Sprintf(MembersPerTeamURL, organization, team), itemsPerPage)
	if err != nil {
		return nil, err
	}

	return c.getAllMembers(rawURL)
}

func (c *Client) getAllMembers(rawURL string) ([]Member, error) {
	members, next, err := c.getMembersPage(rawURL)
	if err != nil {
		return nil, err
	}

	for next != "" {
		pageMembers, n, err := c.getMembersPage(next)
		if err != nil {
			return nil, err
		}
		next = n
		members = append(members, pageMembers...)
	}

	return members, nil
}

func (c *Client) getMembersPage(url string) ([]Member, string, error) {
	var hubResponse hubapi.OrgMemberPaginated
	if err := c.getJSON(url, &hubResponse); err != nil {
		return nil, "", err
	}
	var members []Member
	if hubResponse.Results == nil {
		return nil, ptrValue(hubResponse.Next), nil
	}
	for _, result := range *hubResponse.Results {
		member := Member{
			Username: ptrValue(result.Username),
			FullName: ptrValue(result.FullName),
			Email:    ptrValue(result.Email),
		}
		members = append(members, member)
	}
	return members, ptrValue(hubResponse.Next), nil
}
