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

package tag

import (
	"fmt"
	"strings"
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
	defaultColumns = []commandutil.Column[hub.Tag]{
		commandutil.TextColumn("TAG", func(t hub.Tag) string { return t.Name }),
		{
			Header: "DIGEST",
			Value: func(t hub.Tag) (string, int) {
				if len(t.Images) > 0 {
					return t.Images[0].Digest, len(t.Images[0].Digest)
				}
				return "", 0
			},
		},
		commandutil.TextColumn("STATUS", func(t hub.Tag) string { return t.Status }),
		{
			Header: "LAST UPDATE",
			Value: func(t hub.Tag) (string, int) {
				if t.LastUpdated.Nanosecond() == 0 {
					return "", 0
				}
				s := fmt.Sprintf("%s ago", units.HumanDuration(time.Since(t.LastUpdated)))
				return s, len(s)
			},
		},
		{
			Header: "LAST PUSHED",
			Value: func(t hub.Tag) (string, int) {
				if t.LastPushed.Nanosecond() == 0 {
					return "", 0
				}
				s := units.HumanDuration(time.Since(t.LastPushed))
				return s, len(s)
			},
		},
		{
			Header: "LAST PULLED",
			Value: func(t hub.Tag) (string, int) {
				if t.LastPulled.Nanosecond() == 0 {
					return "", 0
				}
				s := units.HumanDuration(time.Since(t.LastPulled))
				return s, len(s)
			},
		},
		{
			Header: "SIZE",
			Value: func(t hub.Tag) (string, int) {
				size := t.FullSize
				if len(t.Images) > 0 {
					size = 0
					for _, image := range t.Images {
						size += image.Size
					}
				}
				s := units.HumanSize(float64(size))
				return s, len(s)
			},
		},
	}
	platformColumn = commandutil.Column[hub.Tag]{
		Header: "OS/ARCH",
		Value: func(t hub.Tag) (string, int) {
			var platforms []string
			for _, image := range t.Images {
				platform := fmt.Sprintf("%s/%s", image.Os, image.Architecture)
				if image.Variant != "" {
					platform += "/" + image.Variant
				}
				platforms = append(platforms, platform)
			}
			s := strings.Join(platforms, ",")
			return s, len(s)
		},
	}
)

type listOptions struct {
	commandutil.ListOptions
	platforms bool
	sort      string
}

func newListCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	var opts listOptions
	cmd := commandutil.NewCommand(commandutil.CommandConfig{
		Use:     lsName + " [OPTION] REPOSITORY",
		Aliases: []string{"list"},
		Short:   "List all the images in a repository",
		Args:    cli.ExactArgs(1),
		Parent:  parent,
		Name:    lsName,
		RunE: func(_ *cobra.Command, args []string) error {
			return runList(streams, hubClient, opts, args[0])
		},
	})
	cmd.Flags().BoolVar(&opts.platforms, "platforms", false, "List all available platforms per tag")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Fetch all available tags")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort tags by (updated|name)[=(asc|desc)] (e.g.: --sort updated or --sort name=desc)")
	opts.AddFormatFlag(cmd.Flags())
	return cmd
}

func runList(streams command.Streams, hubClient *hub.Client, opts listOptions, repository string) error {
	ordering, err := mapOrdering(opts.sort)
	if err != nil {
		return err
	}
	if err := commandutil.UpdateAllElements(hubClient, opts.All); err != nil {
		return err
	}

	var reqOps []hub.RequestOp
	if ordering != "" {
		reqOps = append(reqOps, hub.WithSortingOrder(ordering))
	}
	tags, total, err := hubClient.GetTags(repository, reqOps...)
	if err != nil {
		return err
	}

	if opts.platforms {
		columns := append([]commandutil.Column[hub.Tag]{}, defaultColumns...)
		columns = append(columns, platformColumn)
		return opts.Print(streams.Out(), tags, commandutil.PrettyTable(columns, total))
	}

	return opts.Print(streams.Out(), tags, commandutil.PrettyTable(defaultColumns, total))
}

const (
	sortAsc  = "asc"
	sortDesc = "desc"
)

func mapOrdering(order string) (string, error) {
	if order == "" {
		return "", nil
	}
	name := "-name"
	update := "last_updated"
	fields := strings.SplitN(order, "=", 2)
	if len(fields) == 2 {
		switch fields[1] {
		case sortDesc:
			name = "name"
			update = "-last_updated"
		case sortAsc:
		default:
			return "", fmt.Errorf(`invalid sorting direction %q: should be either "asc" or "desc"`, fields[1])
		}
	}
	switch fields[0] {
	case "updated":
		return update, nil
	case "name":
		return name, nil
	default:
		return "", fmt.Errorf(`unknown sorting column %q: should be either "name" or "updated"`, fields[0])
	}
}
