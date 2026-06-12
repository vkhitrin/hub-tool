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

	"golang.org/x/sync/errgroup"
)

// Consumption represents current user or org consumption
type Consumption struct {
	Seats               int
	Repositories        int
	PrivateRepositories int
	Teams               int
}

// GetOrgConsumption return the current organization consumption
func (c *Client) GetOrgConsumption(org string) (*Consumption, error) {
	var (
		members      int
		repositories int
		privateRepos int
		teams        int
	)
	c.fetchAllElements = true
	eg, _ := errgroup.WithContext(context.Background())
	eg.Go(func() error {
		count, err := c.GetMembersCount(org)
		if err != nil {
			return err
		}
		members = count
		return nil
	})
	eg.Go(func() error {
		count, err := c.GetTeamsCount(org)
		if err != nil {
			return err
		}
		teams = count
		return nil
	})
	eg.Go(func() error {
		total, private, err := c.GetRepositoryStats(org)
		if err != nil {
			return err
		}
		repositories = total
		privateRepos = private
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return &Consumption{
		Seats:               members,
		Repositories:        repositories,
		PrivateRepositories: privateRepos,
		Teams:               teams,
	}, nil
}

// GetUserConsumption return the current user consumption
func (c *Client) GetUserConsumption(user string) (*Consumption, error) {
	c.fetchAllElements = true
	repositories, privateRepos, err := c.GetRepositoryStats(user)
	if err != nil {
		return nil, err
	}
	return &Consumption{
		Seats:               1,
		Repositories:        repositories,
		PrivateRepositories: privateRepos,
		Teams:               0,
	}, nil
}
