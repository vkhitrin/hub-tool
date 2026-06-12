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
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/go-units"
	"github.com/spf13/cobra"

	"github.com/docker/hub-tool/internal/commands/commandutil"
	"github.com/docker/hub-tool/pkg/hub"
)

const (
	lsName = "ls"
)

var (
	defaultColumns = []commandutil.Column[hub.Token]{
		commandutil.TextColumn("DESCRIPTION", func(t hub.Token) string { return t.Description }),
		commandutil.TextColumn("UUID", func(t hub.Token) string { return t.UUID.String() }),
		{
			Header: "LAST USED",
			Value: func(t hub.Token) (string, int) {
				s := "Never"
				if !t.LastUsed.IsZero() {
					s = fmt.Sprintf("%s ago", units.HumanDuration(time.Since(t.LastUsed)))
				}
				return s, len(s)
			},
		},
		{
			Header: "CREATED",
			Value: func(t hub.Token) (string, int) {
				s := units.HumanDuration(time.Since(t.CreatedAt))
				return s, len(s)
			},
		},
		commandutil.BoolColumn("ACTIVE", func(t hub.Token) bool { return t.IsActive }),
	}
)

func newListCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	var opts commandutil.ListOptions
	cmd := commandutil.NewCommand(commandutil.CommandConfig{
		Use:         lsName + " [OPTION]",
		Aliases:     []string{"list"},
		Short:       "List all the Personal Access Tokens",
		Args:        cli.NoArgs,
		Annotations: commandutil.SudoAnnotation(),
		Parent:      parent,
		Name:        lsName,
		RunE: func(_ *cobra.Command, args []string) error {
			return runList(streams, hubClient, opts)
		},
	})
	opts.AddListFlags(cmd, "Fetch all available tokens")
	return cmd
}

func runList(streams command.Streams, hubClient *hub.Client, opts commandutil.ListOptions) error {
	if err := commandutil.UpdateAllElements(hubClient, opts.All); err != nil {
		return err
	}
	tokens, total, err := hubClient.GetTokens()
	if err != nil {
		return err
	}
	return opts.Print(streams.Out(), tokens, commandutil.PrettyTable(defaultColumns, total))
}
