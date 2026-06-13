package audit

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"

	"github.com/docker/hub-tool/internal/commands/commandutil"
	"github.com/docker/hub-tool/internal/format"
	"github.com/docker/hub-tool/pkg/hub"
)

const (
	actionsName = "actions"
)

var (
	actionColumns = []commandutil.Column[hub.AuditLogAction]{
		commandutil.TextColumn("GROUP", func(action hub.AuditLogAction) string { return action.GroupLabel }),
		commandutil.TextColumn("ACTION", func(action hub.AuditLogAction) string { return action.Name }),
		commandutil.TextColumn("LABEL", func(action hub.AuditLogAction) string { return action.Label }),
		commandutil.TextColumn("DESCRIPTION", func(action hub.AuditLogAction) string { return action.Description }),
	}
)

type actionsOptions struct {
	format.Option
}

func newActionsCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	var opts actionsOptions
	cmd := commandutil.NewCommand(commandutil.CommandConfig{
		Use:    actionsName + " [ACCOUNT]",
		Short:  "List audit log actions available to a namespace",
		Args:   cli.RequiresMaxArgs(1),
		Parent: parent,
		Name:   actionsName,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runActions(cmd.Context(), streams, hubClient, opts, args)
		},
	})
	opts.AddFormatFlag(cmd.Flags())
	return cmd
}

func runActions(ctx context.Context, streams command.Streams, hubClient *hub.Client, opts actionsOptions, args []string) error {
	account := hubClient.AuthConfig.Username
	if len(args) > 0 {
		account = args[0]
	}
	actions, err := hubClient.GetAuditLogActions(ctx, account)
	if err != nil {
		return err
	}
	return opts.Print(streams.Out(), actions, commandutil.PrettyTable(actionColumns))
}
