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

package org

import (
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"

	"github.com/docker/hub-tool/internal/commands/commandutil"
	"github.com/docker/hub-tool/pkg/hub"
)

const (
	orgName = "org"
)

// NewOrgCmd configures the org manage command
func NewOrgCmd(streams command.Streams, hubClient *hub.Client) *cobra.Command {
	cmd := commandutil.NewParentCommand(streams, orgName, "Manage organizations", nil)
	cmd.AddCommand(
		newListCmd(streams, hubClient, orgName),
		newMembersCmd(streams, hubClient, orgName),
		newTeamsCmd(streams, hubClient, orgName),
	)
	return cmd
}
