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
	"net/http"
	"net/url"
	"time"

	hubapi "github.com/docker/hub-tool/pkg/hub/api"
)

const (
	// RepositoriesURL is the Hub API base URL
	RepositoriesURL = "/v2/namespaces/%s/repositories"
	// RepositoryURL is the Hub API repository URL
	RepositoryURL = "/v2/namespaces/%s/repositories/%s"
)

// Repository represents a Docker Hub repository
type Repository struct {
	Name        string
	Description string
	LastUpdated time.Time
	PullCount   int
	StarCount   int
	IsPrivate   bool
}

// GetRepositories lists all the repositories a user can access
func (c *Client) GetRepositories(account string) ([]Repository, int, error) {
	return c.getRepositories(account, c.fetchAllElements)
}

func (c *Client) getRepositories(account string, fetchAll bool) ([]Repository, int, error) {
	if account == "" {
		account = c.account
	}
	rawURL, err := paginatedURL(c.domain+fmt.Sprintf(RepositoriesURL, account), itemsPerPage)
	if err != nil {
		return nil, 0, err
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, 0, err
	}
	values := u.Query()
	values.Add("ordering", "last_updated")
	u.RawQuery = values.Encode()

	repos, total, next, err := c.getRepositoriesPage(u.String(), account)
	if err != nil {
		return nil, 0, err
	}

	if fetchAll {
		for next != "" {
			pageRepos, _, n, err := c.getRepositoriesPage(next, account)
			if err != nil {
				return nil, 0, err
			}
			next = n
			repos = append(repos, pageRepos...)
		}
	}

	return repos, total, nil
}

// GetAllRepositories lists all repositories for an account, following every page.
func (c *Client) GetAllRepositories(account string) ([]Repository, int, error) {
	return c.getRepositories(account, true)
}

// GetRepositoryStats returns total and private repository counts for an account.
func (c *Client) GetRepositoryStats(account string) (int, int, error) {
	if account == "" {
		account = c.account
	}
	rawURL, err := paginatedURL(c.domain+fmt.Sprintf(RepositoriesURL, account), itemsPerPage)
	if err != nil {
		return 0, 0, err
	}

	repos, total, next, err := c.getRepositoriesPage(rawURL, account)
	if err != nil {
		return 0, 0, err
	}

	privateRepos := countPrivateRepositories(repos)
	for next != "" {
		pageRepos, _, n, err := c.getRepositoriesPage(next, account)
		if err != nil {
			return 0, 0, err
		}
		next = n
		privateRepos += countPrivateRepositories(pageRepos)
	}

	return total, privateRepos, nil
}

// RemoveRepository removes a repository on Hub
func (c *Client) RemoveRepository(repository string) error {
	namespace, name, err := getNamespaceRepository(repository)
	if err != nil {
		return err
	}
	repositoryURL := c.domain + fmt.Sprintf(RepositoryURL, namespace, name)
	req, err := http.NewRequest(http.MethodDelete, repositoryURL, nil)
	if err != nil {
		return err
	}
	_, err = c.doRequest(req, withHubToken(c.token))
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) getRepositoriesPage(url, account string) ([]Repository, int, string, error) {
	var hubResponse hubapi.ListRepositoriesResponse
	if err := c.getJSON(url, &hubResponse); err != nil {
		return nil, 0, "", err
	}
	var repos []Repository
	if hubResponse.Results == nil {
		return nil, ptrValue(hubResponse.Count), ptrValue(hubResponse.Next), nil
	}
	for _, result := range *hubResponse.Results {
		repo := Repository{
			Name:        fmt.Sprintf("%s/%s", account, ptrValue(result.Name)),
			Description: ptrValue(result.Description),
			LastUpdated: ptrValue(result.LastUpdated),
			PullCount:   ptrValue(result.PullCount),
			StarCount:   ptrValue(result.StarCount),
			IsPrivate:   ptrValue(result.IsPrivate),
		}
		repos = append(repos, repo)
	}
	return repos, ptrValue(hubResponse.Count), ptrValue(hubResponse.Next), nil
}

func countPrivateRepositories(repos []Repository) int {
	privateRepos := 0
	for _, repo := range repos {
		if repo.IsPrivate {
			privateRepos++
		}
	}
	return privateRepos
}

func ptrValue[T any](value *T) T {
	if value == nil {
		var zero T
		return zero
	}
	return *value
}
