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
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/go-units"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/docker/hub-tool/internal/ansi"
	"github.com/docker/hub-tool/internal/format"
	"github.com/docker/hub-tool/internal/metrics"
	"github.com/docker/hub-tool/pkg/hub"
)

const (
	infoName = "info"
)

type infoOptions struct {
	format.Option
}

func newInfoCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	var opts infoOptions
	cmd := &cobra.Command{
		Use:                   infoName + " [OPTIONS] [ORGANIZATION]",
		Short:                 "Print the account information",
		Args:                  cli.RequiresMaxArgs(1),
		DisableFlagsInUseLine: true,
		Annotations: map[string]string{
			"sudo": "true",
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			metrics.Send(parent, infoName)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return runOrgInfo(streams, hubClient, opts, args[0])
			}
			return runUserInfo(streams, hubClient, opts)
		},
	}
	opts.AddFormatFlag(cmd.Flags())
	return cmd
}

func runOrgInfo(streams command.Streams, hubClient *hub.Client, opts infoOptions, orgName string) error {
	var (
		org         *hub.Account
		consumption *hub.Consumption
	)

	g := errgroup.Group{}
	g.Go(func() error {
		var err error
		org, err = hubClient.GetOrganizationInfo(orgName)
		return checkForbiddenError(err)
	})
	g.Go(func() error {
		var err error
		consumption, err = hubClient.GetOrgConsumption(orgName)
		return checkForbiddenError(err)
	})
	if err := g.Wait(); err != nil {
		return err
	}

	return opts.Print(streams.Out(), account{Account: org, Consumption: consumption}, printAccount)
}

func runUserInfo(streams command.Streams, hubClient *hub.Client, opts infoOptions) error {
	var (
		user          *hub.Account
		organizations []hub.Organization
	)

	g := errgroup.Group{}
	g.Go(func() error {
		var err error
		user, err = hubClient.GetUserInfo()
		return checkForbiddenError(err)
	})
	g.Go(func() error {
		var err error
		organizations, err = hubClient.GetOrganizations(context.Background())
		return checkForbiddenError(err)
	})
	if err := g.Wait(); err != nil {
		return err
	}

	consumption, err := hubClient.GetUserConsumption(user.Name)
	if err != nil {
		return checkForbiddenError(err)
	}
	return opts.Print(streams.Out(), account{Account: user, Consumption: consumption, Organizations: organizations}, printAccount)
}

func checkForbiddenError(err error) error {
	if hub.IsForbiddenError(err) {
		return errors.New(ansi.Error("failed to get organization information, you need to be the organization Owner"))
	}
	return err
}

func printAccount(out io.Writer, value interface{}) error {
	account := value.(account)

	// print user info
	if err := writeLine(out, ansi.Key("Name:")+"\t\t%s\n", account.Account.Name); err != nil {
		return err
	}
	if err := writeLine(out, ansi.Key("ID:")+"\t\t%s\n", account.Account.ID); err != nil {
		return err
	}
	if err := printProfileFields(out, account.Account); err != nil {
		return err
	}
	if err := writeLine(out, ansi.Key("Joined:")+"\t\t%s ago\n", units.HumanDuration(time.Since(account.Account.Joined))); err != nil {
		return err
	}
	if err := writeString(out, ansi.Key("Usage:")+"\n"); err != nil {
		return err
	}
	if err := printUsage(out, account.Account, account.Consumption); err != nil {
		return err
	}

	if len(account.Organizations) > 0 {
		if err := writeString(out, ansi.Key("Organizations:")+"\n"); err != nil {
			return err
		}
		if err := printOrganizations(out, account.Organizations); err != nil {
			return err
		}
	}

	return nil
}

func printProfileFields(out io.Writer, account *hub.Account) error {
	fields := []struct {
		label string
		value string
	}{
		{ansi.Key("Type:") + "\t\t%s\n", account.Type},
		{ansi.Key("Full name:") + "\t%s\n", account.FullName},
		{ansi.Key("Company:") + "\t%s\n", account.Company},
		{ansi.Key("Location:") + "\t%s\n", account.Location},
		{ansi.Key("Profile URL:") + "\t%s\n", account.ProfileURL},
	}
	for _, field := range fields {
		if field.value == "" {
			continue
		}
		if err := writeLine(out, field.label, field.value); err != nil {
			return err
		}
	}
	return nil
}

func printUsage(out io.Writer, account *hub.Account, consumption *hub.Consumption) error {
	if isOrganization(account) {
		if err := writeLine(out, ansi.Key("  Members:")+"\t\t%v\n", consumption.Seats); err != nil {
			return err
		}
		if err := writeLine(out, ansi.Key("  Teams:")+"\t\t%v\n", consumption.Teams); err != nil {
			return err
		}
	}
	if err := writeLine(out, ansi.Key("  Repositories:")+"\t\t%v\n", consumption.Repositories); err != nil {
		return err
	}
	if err := writeLine(out, ansi.Key("  Public repositories:")+"\t%v\n", consumption.Repositories-consumption.PrivateRepositories); err != nil {
		return err
	}
	return writeLine(out, ansi.Key("  Private repositories:")+"\t%v\n", consumption.PrivateRepositories)
}

func isOrganization(account *hub.Account) bool {
	return strings.EqualFold(account.Type, "org") || strings.EqualFold(account.Type, "organization")
}

func printOrganizations(out io.Writer, organizations []hub.Organization) error {
	for _, org := range organizations {
		name := org.Namespace
		if org.FullName != "" {
			name = fmt.Sprintf("%s (%s)", org.Namespace, org.FullName)
		}
		if err := writeLine(out, "  %-24s %-8s repos=%-4d public=%-4d private=%-4d teams=%-4d members=%d\n",
			name, org.Role, org.Repositories, org.PublicRepos, org.PrivateRepos, len(org.Teams), len(org.Members)); err != nil {
			return err
		}
	}

	return nil
}

func writeLine(out io.Writer, format string, args ...interface{}) error {
	_, err := fmt.Fprintf(out, format, args...)
	return err
}

func writeString(out io.Writer, value string) error {
	_, err := fmt.Fprint(out, value)
	return err
}

type account struct {
	Account       *hub.Account
	Consumption   *hub.Consumption
	Organizations []hub.Organization
}
