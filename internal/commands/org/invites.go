package org

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"

	"github.com/docker/hub-tool/internal/ansi"
	"github.com/docker/hub-tool/internal/commands/commandutil"
	"github.com/docker/hub-tool/internal/format"
	"github.com/docker/hub-tool/pkg/hub"
)

const (
	invitesName = "invites"
	lsName      = "ls"
	resendName  = "resend"
	bulkName    = "bulk-invite"
	rmName      = "rm"
)

var (
	inviteColumns = []commandutil.Column[hub.Invite]{
		commandutil.TextColumn("INVITEE", func(i hub.Invite) string { return i.Invitee }),
		commandutil.TextColumn("TEAM", func(i hub.Invite) string { return i.Team }),
		commandutil.TextColumn("INVITED BY", func(i hub.Invite) string { return i.InviterUsername }),
		commandutil.TextColumn("CREATED", func(i hub.Invite) string { return i.CreatedAt }),
		commandutil.TextColumn("ID", func(i hub.Invite) string { return i.ID }),
	}
	bulkInviteColumns = []commandutil.Column[hub.BulkInviteResult]{
		commandutil.TextColumn("INVITEE", func(i hub.BulkInviteResult) string { return i.Invitee }),
		commandutil.TextColumn("STATUS", func(i hub.BulkInviteResult) string { return i.Status }),
		commandutil.TextColumn("TEAM", func(i hub.BulkInviteResult) string {
			if i.Invite == nil {
				return ""
			}
			return i.Invite.Team
		}),
		commandutil.TextColumn("ID", func(i hub.BulkInviteResult) string {
			if i.Invite == nil {
				return ""
			}
			return i.Invite.ID
		}),
	}
)

type inviteOptions struct {
	format.Option
}

type bulkInviteOptions struct {
	format.Option
	file string
}

func newInvitesCmd(streams command.Streams, hubClient *hub.Client) *cobra.Command {
	cmd := commandutil.NewParentCommand(streams, invitesName, "Manage organization invites", nil)
	cmd.AddCommand(
		newInvitesListCmd(streams, hubClient, invitesName),
		newInvitesResendCmd(streams, hubClient, invitesName),
		newBulkInviteCmd(streams, hubClient, invitesName),
		newInvitesRmCmd(streams, hubClient, invitesName),
	)
	return cmd
}

func newInvitesListCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	var opts inviteOptions
	cmd := commandutil.NewCommand(commandutil.CommandConfig{
		Use:    lsName + " ORGANIZATION",
		Short:  "List all pending invites in an organization",
		Args:   cli.ExactArgs(1),
		Parent: parent,
		Name:   lsName,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvites(streams, hubClient, opts, args[0])
		},
	})
	opts.AddFormatFlag(cmd.Flags())
	return cmd
}

func newInvitesResendCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	return commandutil.NewCommand(commandutil.CommandConfig{
		Use:    resendName + " INVITE_ID",
		Short:  "Resend a pending invite",
		Args:   cli.ExactArgs(1),
		Parent: parent,
		Name:   resendName,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResendInvite(streams, hubClient, args[0])
		},
	})
}

func newInvitesRmCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	return commandutil.NewCommand(commandutil.CommandConfig{
		Use:    rmName + " INVITE_ID",
		Short:  "Cancel a pending invite",
		Args:   cli.ExactArgs(1),
		Parent: parent,
		Name:   rmName,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemoveInvite(streams, hubClient, args[0])
		},
	})
}

func newBulkInviteCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	var opts bulkInviteOptions
	cmd := commandutil.NewCommand(commandutil.CommandConfig{
		Use:   bulkName + " --file FILE",
		Short: "Create multiple organization invites",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("%q accepts input from --file only", bulkName)
			}
			if opts.file == "" {
				return fmt.Errorf("--file is required")
			}
			return nil
		},
		Parent: parent,
		Name:   bulkName,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBulkInvite(streams, hubClient, opts)
		},
	})
	opts.AddFormatFlag(cmd.Flags())
	cmd.Flags().StringVar(&opts.file, "file", "", "Read bulk invite request JSON from file, or - for stdin")
	return cmd
}

func runInvites(streams command.Streams, hubClient *hub.Client, opts inviteOptions, organization string) error {
	invites, err := hubClient.GetInvites(organization)
	if err != nil {
		return err
	}
	return opts.Print(streams.Out(), invites, commandutil.PrettyTable(inviteColumns))
}

func runResendInvite(streams command.Streams, hubClient *hub.Client, inviteID string) error {
	if err := hubClient.ResendInvite(inviteID); err != nil {
		return err
	}
	_, err := fmt.Fprintf(streams.Out(), ansi.Emphasise("Invite resent")+" %s\n", inviteID)
	return err
}

func runRemoveInvite(streams command.Streams, hubClient *hub.Client, inviteID string) error {
	if err := hubClient.RemoveInvite(inviteID); err != nil {
		return err
	}
	_, err := fmt.Fprintf(streams.Out(), ansi.Emphasise("Invite cancelled")+" %s\n", inviteID)
	return err
}

func runBulkInvite(streams command.Streams, hubClient *hub.Client, opts bulkInviteOptions) error {
	request, err := bulkInviteRequest(streams, opts)
	if err != nil {
		return err
	}
	results, err := hubClient.BulkInvite(request)
	if err != nil {
		return err
	}
	return opts.Print(streams.Out(), results, commandutil.PrettyTable(bulkInviteColumns))
}

func bulkInviteRequest(streams command.Streams, opts bulkInviteOptions) (hub.BulkInviteOptions, error) {
	data, err := readBulkInviteFile(streams, opts.file)
	if err != nil {
		return hub.BulkInviteOptions{}, err
	}
	var request hub.BulkInviteOptions
	if err := json.Unmarshal(data, &request); err != nil {
		return hub.BulkInviteOptions{}, err
	}
	if request.Org == "" {
		return hub.BulkInviteOptions{}, fmt.Errorf("bulk invite file must include org")
	}
	if len(request.Invitees) == 0 {
		return hub.BulkInviteOptions{}, fmt.Errorf("bulk invite file must include at least one invitee")
	}
	return request, nil
}

func readBulkInviteFile(streams command.Streams, file string) ([]byte, error) {
	if file == "-" {
		return io.ReadAll(streams.In())
	}
	return os.ReadFile(file)
}
