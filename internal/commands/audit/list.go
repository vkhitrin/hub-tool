package audit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/go-units"
	"github.com/spf13/cobra"

	"github.com/docker/hub-tool/internal/commands/commandutil"
	"github.com/docker/hub-tool/internal/format"
	"github.com/docker/hub-tool/pkg/hub"
)

const (
	listName = "ls"
)

var (
	defaultColumns = []commandutil.Column[hub.AuditLog]{
		{
			Header: "TIME",
			Value: func(log hub.AuditLog) (string, int) {
				if log.Timestamp.IsZero() {
					return "", 0
				}
				s := fmt.Sprintf("%s ago", units.HumanDuration(time.Since(log.Timestamp)))
				return s, len(s)
			},
		},
		commandutil.TextColumn("ACTOR", func(log hub.AuditLog) string { return log.Actor }),
		commandutil.TextColumn("ACTION", func(log hub.AuditLog) string { return log.Action }),
		commandutil.TextColumn("NAME", func(log hub.AuditLog) string { return log.Name }),
		commandutil.TextColumn("DESCRIPTION", func(log hub.AuditLog) string { return log.ActionDescription }),
	}
)

type listOptions struct {
	format.Option
	action   string
	name     string
	actor    string
	from     string
	to       string
	page     int
	pageSize int
}

func newListCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	var opts listOptions
	cmd := commandutil.NewCommand(commandutil.CommandConfig{
		Use:     listName + " [OPTIONS] ORGANIZATION",
		Aliases: []string{"list"},
		Short:   "List audit log events",
		Args:    cli.ExactArgs(1),
		Parent:  parent,
		Name:    listName,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), streams, hubClient, opts, args)
		},
	})
	cmd.Flags().StringVar(&opts.action, "action", "", "Filter by audit log action")
	cmd.Flags().StringVar(&opts.name, "name", "", "Filter by audited resource name")
	cmd.Flags().StringVar(&opts.actor, "actor", "", "Filter by actor")
	cmd.Flags().StringVar(&opts.from, "from", "", "Filter from timestamp (RFC3339)")
	cmd.Flags().StringVar(&opts.to, "to", "", "Filter to timestamp (RFC3339)")
	cmd.Flags().IntVar(&opts.page, "page", 1, "Page number to fetch")
	cmd.Flags().IntVar(&opts.pageSize, "page-size", 25, "Number of events to fetch")
	opts.AddFormatFlag(cmd.Flags())
	return cmd
}

func runList(ctx context.Context, streams command.Streams, hubClient *hub.Client, opts listOptions, args []string) error {
	auditOpts, err := opts.auditLogOptions()
	if err != nil {
		return err
	}
	logs, err := hubClient.GetAuditLogs(ctx, args[0], auditOpts)
	if err != nil {
		return auditLogsError(args[0], err)
	}
	return opts.Print(streams.Out(), logs, commandutil.PrettyTable(defaultColumns))
}

func auditLogsError(account string, err error) error {
	var statusErr *hub.StatusError
	if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusInternalServerError {
		return fmt.Errorf("failed to list audit logs for %q: audit log events require an organization namespace", account)
	}
	return err
}

func (opts listOptions) auditLogOptions() (hub.AuditLogOptions, error) {
	from, err := parseTimestamp(opts.from)
	if err != nil {
		return hub.AuditLogOptions{}, err
	}
	to, err := parseTimestamp(opts.to)
	if err != nil {
		return hub.AuditLogOptions{}, err
	}
	if !from.IsZero() && !to.IsZero() && !to.After(from) {
		return hub.AuditLogOptions{}, fmt.Errorf("--to must be newer than --from")
	}
	if !from.IsZero() && to.IsZero() && from.After(time.Now()) {
		return hub.AuditLogOptions{}, fmt.Errorf("--from cannot be in the future unless --to is also set after --from")
	}
	if opts.page < 1 {
		return hub.AuditLogOptions{}, fmt.Errorf("--page must be greater than 0")
	}
	if opts.pageSize < 1 {
		return hub.AuditLogOptions{}, fmt.Errorf("--page-size must be greater than 0")
	}
	return hub.AuditLogOptions{
		Action:   opts.action,
		Name:     opts.name,
		Actor:    opts.actor,
		From:     from,
		To:       to,
		Page:     opts.page,
		PageSize: opts.pageSize,
	}, nil
}

func parseTimestamp(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp %q: use RFC3339 format", value)
	}
	return timestamp, nil
}
