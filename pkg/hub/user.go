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
	"time"

	hubapi "github.com/docker/hub-tool/pkg/hub/api"
)

const (
	//UserURL path to user informations
	UserURL = "/v2/user/"
)

// Account represents a user or organization information
type Account struct {
	ID         string
	Name       string
	FullName   string
	Location   string
	Company    string
	Joined     time.Time
	Type       string
	ProfileURL string
}

// GetUserInfo returns the information on the user retrieved from Hub
func (c *Client) GetUserInfo() (*Account, error) {
	var hubResponse hubapi.User
	if err := c.getJSON(c.domain+UserURL, &hubResponse); err != nil {
		return nil, err
	}
	joined, err := parseAPITime(ptrValue(hubResponse.DateJoined))
	if err != nil {
		return nil, err
	}
	return &Account{
		ID:         ptrValue(hubResponse.Id),
		Name:       ptrValue(hubResponse.Username),
		FullName:   ptrValue(hubResponse.FullName),
		Location:   ptrValue(hubResponse.Location),
		Company:    ptrValue(hubResponse.Company),
		Joined:     joined,
		Type:       string(ptrValue(hubResponse.Type)),
		ProfileURL: ptrValue(hubResponse.ProfileUrl),
	}, nil
}
