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
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"

	"github.com/docker/hub-tool/internal/ansi"
	"github.com/docker/hub-tool/internal/commands/commandutil"
	"github.com/docker/hub-tool/internal/format"
	"github.com/docker/hub-tool/pkg/hub"
)

const (
	listName = "ls"
)

var (
	defaultColumns = []commandutil.Column[hub.Organization]{
		{
			Header: "NAMESPACE",
			Value: func(o hub.Organization) (string, int) {
				return ansi.Link(fmt.Sprintf("https://hub.docker.com/orgs/%s", o.Namespace), o.Namespace), len(o.Namespace)
			},
		},
		commandutil.TextColumn("NAME", func(o hub.Organization) string { return o.FullName }),
		commandutil.TextColumn("MY ROLE", func(o hub.Organization) string { return o.Role }),
		commandutil.IntColumn("TEAMS", func(o hub.Organization) int { return len(o.Teams) }),
		commandutil.IntColumn("MEMBERS", func(o hub.Organization) int { return len(o.Members) }),
	}
)

type listOptions struct {
	format.Option
}

func newListCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	var opts listOptions
	cmd := commandutil.NewCommand(commandutil.CommandConfig{
		Use:     listName,
		Aliases: []string{"list"},
		Short:   "List all the organizations",
		Args:    cli.NoArgs,
		Parent:  parent,
		Name:    listName,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), streams, hubClient, opts)
		},
	})
	opts.AddFormatFlag(cmd.Flags())
	return cmd
}

func runList(ctx context.Context, streams command.Streams, hubClient *hub.Client, opts listOptions) error {
	organizations, err := hubClient.GetOrganizations(ctx)
	if err != nil {
		return err
	}
	return opts.Print(streams.Out(), organizations, commandutil.PrettyTable(defaultColumns))
}
