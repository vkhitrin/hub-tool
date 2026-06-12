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

package token

import (
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/docker/hub-tool/internal/ansi"
	"github.com/docker/hub-tool/internal/commands/commandutil"
	"github.com/docker/hub-tool/pkg/hub"
)

const (
	tokenName      = "token"
	activateName   = "activate"
	deactivateName = "deactivate"
)

// NewTokenCmd configures the token manage command
func NewTokenCmd(streams command.Streams, hubClient *hub.Client) *cobra.Command {
	cmd := commandutil.NewParentCommand(streams, tokenName, "Manage Personal Access Tokens", commandutil.SudoAnnotation())
	cmd.AddCommand(
		newCreateCmd(streams, hubClient, tokenName),
		newInspectCmd(streams, hubClient, tokenName),
		newListCmd(streams, hubClient, tokenName),
		newActivateCmd(streams, hubClient, tokenName),
		newDeactivateCmd(streams, hubClient, tokenName),
		newRmCmd(streams, hubClient, tokenName),
	)
	return cmd
}

type tokenStatusOptions struct {
	name    string
	short   string
	active  bool
	message string
}

func newActivateCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	return newTokenStatusCmd(streams, hubClient, parent, tokenStatusOptions{
		name:    activateName,
		short:   "Activate a Personal Access Token",
		active:  true,
		message: "active",
	})
}

func newDeactivateCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	return newTokenStatusCmd(streams, hubClient, parent, tokenStatusOptions{
		name:    deactivateName,
		short:   "Deactivate a Personal Access Token",
		active:  false,
		message: "inactive",
	})
}

func newTokenStatusCmd(streams command.Streams, hubClient *hub.Client, parent string, opts tokenStatusOptions) *cobra.Command {
	return commandutil.NewCommand(commandutil.CommandConfig{
		Use:         opts.name + " TOKEN_UUID",
		Short:       opts.short,
		Args:        cli.ExactArgs(1),
		Annotations: commandutil.SudoAnnotation(),
		Parent:      parent,
		Name:        opts.name,
		RunE: func(_ *cobra.Command, args []string) error {
			return runTokenStatus(streams, hubClient, args[0], opts.active, opts.message)
		},
	})
}

func runTokenStatus(streams command.Streams, hubClient *hub.Client, tokenUUID string, active bool, message string) error {
	u, err := uuid.Parse(tokenUUID)
	if err != nil {
		return err
	}
	if _, err := hubClient.UpdateToken(u.String(), "", active); err != nil {
		return err
	}
	_, err = fmt.Fprintf(streams.Out(), ansi.Emphasise("%s is %s\n"), u.String(), message)
	return err
}
