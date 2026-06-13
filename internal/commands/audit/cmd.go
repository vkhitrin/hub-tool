package audit

import (
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"

	"github.com/docker/hub-tool/internal/commands/commandutil"
	"github.com/docker/hub-tool/pkg/hub"
)

const (
	auditName = "audit"
)

// NewAuditCmd configures the audit log command.
func NewAuditCmd(streams command.Streams, hubClient *hub.Client) *cobra.Command {
	cmd := commandutil.NewParentCommand(streams, auditName, "View audit logs", nil)
	cmd.AddCommand(
		newListCmd(streams, hubClient, auditName),
		newActionsCmd(streams, hubClient, auditName),
	)
	return cmd
}
