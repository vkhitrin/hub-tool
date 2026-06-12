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

package account

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/docker/hub-tool/pkg/hub"
)

func TestInfoOutput(t *testing.T) {
	account := account{
		Account: &hub.Account{
			ID:         "id",
			Name:       "my-user-name",
			FullName:   "My Full Name",
			Location:   "MyLocation",
			Company:    "My Company",
			Joined:     time.Now(),
			Type:       "User",
			ProfileURL: "https://hub.docker.com/u/my-user-name",
		},
		Consumption: &hub.Consumption{
			Seats:               0,
			Repositories:        4,
			PrivateRepositories: 1,
			Teams:               2,
		},
		Organizations: []hub.Organization{
			{
				Namespace:    "my-org",
				FullName:     "My Org",
				Role:         "Owner",
				Repositories: 7,
				PublicRepos:  5,
				PrivateRepos: 2,
				Repos:        []hub.Repository{{Name: "my-org/my-repo", IsPrivate: false}},
				Teams:        []hub.Team{{Name: "owners"}},
				Members:      []hub.Member{{Username: "my-user-name", Email: "me@example.com"}},
			},
		},
	}
	buf := bytes.NewBuffer(nil)
	err := printAccount(buf, account)
	assert.NilError(t, err)
	golden.Assert(t, buf.String(), "info.golden")
}

func TestInfoJSONIncludesMemberEmail(t *testing.T) {
	value := account{
		Account:     &hub.Account{Name: "my-user-name"},
		Consumption: &hub.Consumption{},
		Organizations: []hub.Organization{
			{
				Namespace: "my-org",
				Members: []hub.Member{
					{Username: "my-user-name", Email: "me@example.com"},
				},
			},
		},
	}

	data, err := json.Marshal(value)
	assert.NilError(t, err)
	assert.Check(t, bytes.Contains(data, []byte(`"email":"me@example.com"`)))
}

func TestInfoOrgUsageOutput(t *testing.T) {
	value := account{
		Account: &hub.Account{
			Name:   "my-org",
			Joined: time.Now(),
			Type:   "Organization",
		},
		Consumption: &hub.Consumption{
			Seats:               3,
			Teams:               2,
			Repositories:        4,
			PrivateRepositories: 1,
		},
	}

	buf := bytes.NewBuffer(nil)
	err := printAccount(buf, value)
	assert.NilError(t, err)
	assert.Check(t, bytes.Contains(buf.Bytes(), []byte("  Members:")))
	assert.Check(t, bytes.Contains(buf.Bytes(), []byte("  Teams:")))
	assert.Check(t, !bytes.Contains(buf.Bytes(), []byte("  Seats:")))
}
