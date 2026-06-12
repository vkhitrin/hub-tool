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
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"

	"github.com/docker/hub-tool/internal/commands/commandutil"
	"github.com/docker/hub-tool/internal/format"
	"github.com/docker/hub-tool/pkg/hub"
)

const (
	membersName = "members"
)

var (
	memberColumns = []commandutil.Column[hub.Member]{
		commandutil.TextColumn("USERNAME", func(m hub.Member) string { return m.Username }),
		commandutil.TextColumn("FULL NAME", func(m hub.Member) string { return m.FullName }),
		commandutil.TextColumn("EMAIL", func(m hub.Member) string { return m.Email }),
	}
)

type memberOptions struct {
	format.Option
}

func newMembersCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	var opts memberOptions
	cmd := commandutil.NewCommand(commandutil.CommandConfig{
		Use:    membersName + " ORGANIZATION",
		Short:  "List all the members in an organization",
		Args:   cli.ExactArgs(1),
		Parent: parent,
		Name:   membersName,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMembers(streams, hubClient, opts, args[0])
		},
	})
	opts.AddFormatFlag(cmd.Flags())
	return cmd
}

func runMembers(streams command.Streams, hubClient *hub.Client, opts memberOptions, organization string) error {
	members, err := hubClient.GetMembers(organization)
	if err != nil {
		return err
	}
	return opts.Print(streams.Out(), members, commandutil.PrettyTable(memberColumns))
}
