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
	// OrganizationsURL path to the Hub API listing the organizations
	OrganizationsURL = "/v2/user/orgs/"
	// OrganizationInfoURL path to the Hub API returning organization info
	OrganizationInfoURL = "/v2/orgs/%s"
)

// Organization represents a Docker Hub organization
type Organization struct {
	Namespace    string
	FullName     string
	Role         string
	Repositories int
	PublicRepos  int
	PrivateRepos int
	Repos        []Repository
	Teams        []Team
	Members      []Member
}

// GetOrganizations lists all the organizations a user has joined
func (c *Client) GetOrganizations(ctx context.Context) ([]Organization, error) {
	rawURL, err := paginatedURL(c.domain+OrganizationsURL, itemsPerPage)
	if err != nil {
		return nil, err
	}

	organizations, next, err := c.getOrganizationsPage(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	for next != "" {
		pageOrganizations, n, err := c.getOrganizationsPage(ctx, next)
		if err != nil {
			return nil, err
		}
		next = n
		organizations = append(organizations, pageOrganizations...)
	}

	return organizations, nil
}

// GetOrganizationInfo returns organization info
func (c *Client) GetOrganizationInfo(orgname string) (*Account, error) {
	var hubResponse hubapi.UserOrg
	if err := c.getJSON(c.domain+fmt.Sprintf(OrganizationInfoURL, orgname), &hubResponse); err != nil {
		return nil, err
	}

	return &Account{
		ID:         ptrValue(hubResponse.Id),
		Name:       ptrValue(hubResponse.Orgname),
		FullName:   ptrValue(hubResponse.FullName),
		Location:   ptrValue(hubResponse.Location),
		Company:    ptrValue(hubResponse.Company),
		Joined:     ptrValue(hubResponse.DateJoined),
		Type:       ptrValue(hubResponse.Type),
		ProfileURL: ptrValue(hubResponse.ProfileUrl),
	}, nil
}

func (c *Client) getOrganizationsPage(ctx context.Context, url string) ([]Organization, string, error) {
	var hubResponse hubapi.UserOrgPaginated
	if err := c.getJSONContext(ctx, url, &hubResponse); err != nil {
		return nil, "", err
	}
	if hubResponse.Results == nil {
		return nil, ptrValue(hubResponse.Next), nil
	}

	var organizations []Organization
	var mu sync.Mutex
	eg, _ := errgroup.WithContext(ctx)

	for _, result := range *hubResponse.Results {
		result := result
		eg.Go(func() error {
			orgName := ptrValue(result.Orgname)
			var (
				teams        []Team
				members      []Member
				repos        []Repository
				repositories int
				privateRepos int
			)
			subeg, _ := errgroup.WithContext(ctx)

			subeg.Go(func() error {
				var err error
				teams, err = c.GetTeams(orgName)
				return err
			})
			subeg.Go(func() error {
				var err error
				members, err = c.GetMembers(orgName)
				return err
			})
			subeg.Go(func() error {
				var err error
				repos, repositories, err = c.GetAllRepositories(orgName)
				if err != nil {
					return err
				}
				privateRepos = countPrivateRepositories(repos)
				return nil
			})

			if err := subeg.Wait(); err != nil {
				return err
			}
			organization := Organization{
				Namespace:    orgName,
				FullName:     ptrValue(result.FullName),
				Role:         getRole(teams),
				Repositories: repositories,
				PublicRepos:  repositories - privateRepos,
				PrivateRepos: privateRepos,
				Repos:        repos,
				Teams:        teams,
				Members:      members,
			}
			mu.Lock()
			organizations = append(organizations, organization)
			mu.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return []Organization{}, "", err
	}

	sort.Slice(organizations, func(i, j int) bool {
		return organizations[i].Namespace < organizations[j].Namespace
	})
	return organizations, ptrValue(hubResponse.Next), nil
}

func getRole(teams []Team) string {
	for _, t := range teams {
		if t.Name == "owners" {
			return "Owner"
		}
	}
	return "Member"
}
