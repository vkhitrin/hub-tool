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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	hubapi "github.com/docker/hub-tool/pkg/hub/api"
	"github.com/google/uuid"
)

const (
	// TokensURL path to the Hub API listing the Personal Access Tokens
	TokensURL = "/v2/access-tokens"
	// TokenURL path to the Hub API Personal Access Token
	TokenURL = "/v2/access-tokens/%s"
)

// Token is a personal access token. The token field will only be filled at creation and can never been accessed again.
type Token struct {
	UUID        uuid.UUID
	ClientID    string
	CreatorIP   string
	CreatorUA   string
	CreatedAt   time.Time
	LastUsed    time.Time
	GeneratedBy string
	IsActive    bool
	Token       string
	Description string
}

// CreateToken creates a Personal Access Token and returns the token field only once
func (c *Client) CreateToken(description string) (*Token, error) {
	data, err := json.Marshal(hubapi.CreateAccessTokenRequest{
		TokenLabel: description,
		Scopes:     []string{"repo:write"},
	})
	if err != nil {
		return nil, err
	}
	return c.doTokenRequest(http.MethodPost, c.domain+TokensURL, bytes.NewBuffer(data))
}

// GetTokens calls the hub repo API and returns all the information on all tokens
func (c *Client) GetTokens() ([]Token, int, error) {
	rawURL, err := paginatedURL(c.domain+TokensURL, itemsPerPage)
	if err != nil {
		return nil, 0, err
	}

	tokens, total, next, err := c.getTokensPage(rawURL)
	if err != nil {
		return nil, 0, err
	}
	if c.fetchAllElements {
		for next != "" {
			pageTokens, _, n, err := c.getTokensPage(next)
			if err != nil {
				return nil, 0, err
			}
			next = n
			tokens = append(tokens, pageTokens...)
		}
	}

	return tokens, total, nil
}

// GetToken calls the hub repo API and returns the information on one token
func (c *Client) GetToken(tokenUUID string) (*Token, error) {
	return c.doTokenRequest(http.MethodGet, c.domain+fmt.Sprintf(TokenURL, tokenUUID), nil)
}

// UpdateToken updates a token's description and activeness
func (c *Client) UpdateToken(tokenUUID, description string, isActive bool) (*Token, error) {
	tokenRequest := hubapi.PatchAccessTokenRequest{IsActive: &isActive}
	if description != "" {
		tokenRequest.TokenLabel = &description
	}
	data, err := json.Marshal(tokenRequest)
	if err != nil {
		return nil, err
	}
	return c.doTokenRequest(http.MethodPatch, c.domain+fmt.Sprintf(TokenURL, tokenUUID), bytes.NewBuffer(data))
}

func (c *Client) doTokenRequest(method, url string, body io.Reader) (*Token, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	response, err := c.doRequest(req, withHubToken(c.token))
	if err != nil {
		return nil, err
	}
	return decodeToken(response)
}

// RemoveToken deletes a token from personal access token
func (c *Client) RemoveToken(tokenUUID string) error {
	//DELETE https://hub.docker.com/v2/api_tokens/8208674e-d08a-426f-b6f4-e3aba7058459 => 202
	return c.deleteHubResource(c.domain + fmt.Sprintf(TokenURL, tokenUUID))
}

func (c *Client) getTokensPage(url string) ([]Token, int, string, error) {
	var hubResponse hubapi.GetAccessTokensResponse
	if err := c.getJSON(url, &hubResponse); err != nil {
		return nil, 0, "", err
	}
	var tokens []Token
	if hubResponse.Results == nil {
		return nil, int(ptrValue(hubResponse.Count)), ptrValue(hubResponse.Next), nil
	}
	for _, result := range *hubResponse.Results {
		token, err := convertToken(result)
		if err != nil {
			return nil, 0, "", err
		}
		tokens = append(tokens, token)
	}
	return tokens, int(ptrValue(hubResponse.Count)), ptrValue(hubResponse.Next), nil
}

func decodeToken(response []byte) (*Token, error) {
	var tokenResponse hubapi.AccessToken
	if err := json.Unmarshal(response, &tokenResponse); err != nil {
		return nil, err
	}
	token, err := convertToken(tokenResponse)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func convertToken(response hubapi.AccessToken) (Token, error) {
	u, err := uuid.Parse(ptrValue(response.Uuid))
	if err != nil {
		return Token{}, err
	}
	createdAt, err := parseAPITime(ptrValue(response.CreatedAt))
	if err != nil {
		return Token{}, err
	}
	lastUsed, err := parseAPITime(ptrValue(response.LastUsed))
	if err != nil {
		return Token{}, err
	}
	return Token{
		UUID:        u,
		ClientID:    ptrValue(response.ClientId),
		CreatorIP:   ptrValue(response.CreatorIp),
		CreatorUA:   ptrValue(response.CreatorUa),
		CreatedAt:   createdAt,
		LastUsed:    lastUsed,
		GeneratedBy: ptrValue(response.GeneratedBy),
		IsActive:    ptrValue(response.IsActive),
		Token:       ptrValue(response.Token),
		Description: ptrValue(response.TokenLabel),
	}, nil
}

func parseAPITime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, value)
}
