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

package repo

import (
	"fmt"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/go-units"
	"github.com/spf13/cobra"

	"github.com/docker/hub-tool/internal/ansi"
	"github.com/docker/hub-tool/internal/commands/commandutil"
	"github.com/docker/hub-tool/pkg/hub"
)

const (
	listName = "ls"
)

var (
	defaultColumns = []commandutil.Column[hub.Repository]{
		{
			Header: "REPOSITORY",
			Value: func(r hub.Repository) (string, int) {
				return ansi.Link(fmt.Sprintf("https://hub.docker.com/repository/docker/%s", r.Name), r.Name), len(r.Name)
			},
		},
		commandutil.TextColumn("DESCRIPTION", func(r hub.Repository) string { return r.Description }),
		{
			Header: "LAST UPDATE",
			Value: func(r hub.Repository) (string, int) {
				if r.LastUpdated.Nanosecond() == 0 {
					return "", 0
				}
				s := fmt.Sprintf("%s ago", units.HumanDuration(time.Since(r.LastUpdated)))
				return s, len(s)
			},
		},
		commandutil.IntColumn("PULLS", func(r hub.Repository) int { return r.PullCount }),
		commandutil.IntColumn("STARS", func(r hub.Repository) int { return r.StarCount }),
		commandutil.BoolColumn("PRIVATE", func(r hub.Repository) bool { return r.IsPrivate }),
	}
)

func newListCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	var opts commandutil.ListOptions
	cmd := commandutil.NewCommand(commandutil.CommandConfig{
		Use:     listName + " [OPTIONS] [ORGANIZATION]",
		Aliases: []string{"list"},
		Short:   "List repositories from your account or an organization",
		Args:    cli.RequiresMaxArgs(1),
		Parent:  parent,
		Name:    listName,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(streams, hubClient, opts, args)
		},
	})
	opts.AddListFlags(cmd, "Fetch all available repositories")
	return cmd
}

func runList(streams command.Streams, hubClient *hub.Client, opts commandutil.ListOptions, args []string) error {
	account := hubClient.AuthConfig.Username
	if err := commandutil.UpdateAllElements(hubClient, opts.All); err != nil {
		return err
	}
	if len(args) > 0 {
		account = args[0]
	}
	repositories, total, err := hubClient.GetRepositories(account)
	if err != nil {
		return err
	}

	return opts.Print(streams.Out(), repositories, commandutil.PrettyTable(defaultColumns, total))
}
