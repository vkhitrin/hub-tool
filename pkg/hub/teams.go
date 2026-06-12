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
	"context"
	"fmt"
	"sort"
	"sync"

	hubapi "github.com/docker/hub-tool/pkg/hub/api"
	"golang.org/x/sync/errgroup"
)

const (
	//GroupsURL path to the Hub API listing the groups in an organization
	GroupsURL = "/v2/orgs/%s/groups/"
)

// Team represents a hub group in an organization
type Team struct {
	Name        string
	Description string
	Members     []Member
}

// GetTeams lists all the teams in an organization
func (c *Client) GetTeams(organization string) ([]Team, error) {
	rawURL, err := paginatedURL(c.domain+fmt.Sprintf(GroupsURL, organization), itemsPerPage)
	if err != nil {
		return nil, err
	}

	teams, next, err := c.getTeamsPage(rawURL, organization)
	if err != nil {
		return nil, err
	}

	for next != "" {
		pageTeams, n, err := c.getTeamsPage(next, organization)
		if err != nil {
			return nil, err
		}
		next = n
		teams = append(teams, pageTeams...)
	}

	return teams, nil
}

// GetTeamsCount returns the number of teams in an organization
func (c *Client) GetTeamsCount(organization string) (int, error) {
	rawURL, err := paginatedURL(c.domain+fmt.Sprintf(GroupsURL, organization), 1)
	if err != nil {
		return 0, err
	}
	var hubResponse hubapi.OrgGroupPaginated
	if err := c.getJSON(rawURL, &hubResponse); err != nil {
		return 0, err
	}
	return ptrValue(hubResponse.Count), nil
}

func (c *Client) getTeamsPage(url, organization string) ([]Team, string, error) {
	var hubResponse hubapi.OrgGroupPaginated
	if err := c.getJSON(url, &hubResponse); err != nil {
		return nil, "", err
	}
	if hubResponse.Results == nil {
		return nil, ptrValue(hubResponse.Next), nil
	}
	var teams []Team
	var mu sync.Mutex
	eg, _ := errgroup.WithContext(context.Background())
	for _, result := range *hubResponse.Results {
		result := result
		eg.Go(func() error {
			members, err := c.GetMembersPerTeam(organization, ptrValue(result.Name))
			if err != nil {
				return err
			}
			team := Team{
				Name:        ptrValue(result.Name),
				Description: ptrValue(result.Description),
				Members:     members,
			}
			mu.Lock()
			teams = append(teams, team)
			mu.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return []Team{}, "", err
	}

	sort.Slice(teams, func(i, j int) bool {
		return teams[i].Name < teams[j].Name
	})

	return teams, ptrValue(hubResponse.Next), nil
}
