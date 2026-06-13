package hub

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"time"

	hubapi "github.com/docker/hub-tool/pkg/hub/api"
)

const (
	// AuditLogsURL path to the Hub API listing audit log events.
	AuditLogsURL = "/v2/auditlogs/%s"
	// AuditLogActionsURL path to the Hub API listing audit log actions.
	AuditLogActionsURL = "/v2/auditlogs/%s/actions"
)

// AuditLog is an audit log event.
type AuditLog struct {
	Account           string
	Action            string
	Name              string
	Actor             string
	Data              map[string]string
	Timestamp         time.Time
	ActionDescription string
}

// AuditLogAction is an action that can be used to filter audit logs.
type AuditLogAction struct {
	Group       string
	GroupLabel  string
	Name        string
	Description string
	Label       string
}

// AuditLogOptions contains filters for querying audit log events.
type AuditLogOptions struct {
	Action   string
	Name     string
	Actor    string
	From     time.Time
	To       time.Time
	Page     int
	PageSize int
}

// GetAuditLogs lists audit log events for an account.
func (c *Client) GetAuditLogs(ctx context.Context, account string, opts AuditLogOptions) ([]AuditLog, error) {
	if account == "" {
		account = c.account
	}
	u, err := url.Parse(c.domain + fmt.Sprintf(AuditLogsURL, account))
	if err != nil {
		return nil, err
	}
	values := u.Query()
	addStringQuery(values, "action", opts.Action)
	addStringQuery(values, "name", opts.Name)
	addStringQuery(values, "actor", opts.Actor)
	addTimeQuery(values, "from", opts.From)
	addTimeQuery(values, "to", opts.To)
	addIntQuery(values, "page", opts.Page)
	addIntQuery(values, "page_size", opts.PageSize)
	u.RawQuery = values.Encode()

	var hubResponse hubapi.GetAuditLogsResponse
	if err := c.getJSONContext(ctx, u.String(), &hubResponse); err != nil {
		return nil, err
	}
	if hubResponse.Logs == nil {
		return nil, nil
	}
	logs := make([]AuditLog, 0, len(*hubResponse.Logs))
	for _, result := range *hubResponse.Logs {
		logs = append(logs, convertAuditLog(result))
	}
	return logs, nil
}

// GetAuditLogActions lists audit log actions for an account.
func (c *Client) GetAuditLogActions(ctx context.Context, account string) ([]AuditLogAction, error) {
	if account == "" {
		account = c.account
	}
	var hubResponse hubapi.GetAuditActionsResponse
	if err := c.getJSONContext(ctx, c.domain+fmt.Sprintf(AuditLogActionsURL, account), &hubResponse); err != nil {
		return nil, err
	}
	if hubResponse.Actions == nil {
		return nil, nil
	}
	actions := make([]AuditLogAction, 0)
	for group, actionGroup := range *hubResponse.Actions {
		if actionGroup.Actions == nil {
			continue
		}
		for _, action := range *actionGroup.Actions {
			actions = append(actions, AuditLogAction{
				Group:       group,
				GroupLabel:  ptrValue(actionGroup.Label),
				Name:        ptrValue(action.Name),
				Description: ptrValue(action.Description),
				Label:       ptrValue(action.Label),
			})
		}
	}
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Group == actions[j].Group {
			return actions[i].Name < actions[j].Name
		}
		return actions[i].Group < actions[j].Group
	})
	return actions, nil
}

func convertAuditLog(response hubapi.AuditLog) AuditLog {
	return AuditLog{
		Account:           ptrValue(response.Account),
		Action:            ptrValue(response.Action),
		Name:              ptrValue(response.Name),
		Actor:             ptrValue(response.Actor),
		Data:              ptrValue(response.Data),
		Timestamp:         ptrValue(response.Timestamp),
		ActionDescription: ptrValue(response.ActionDescription),
	}
}

func addStringQuery(values url.Values, name, value string) {
	if value != "" {
		values.Add(name, value)
	}
}

func addTimeQuery(values url.Values, name string, value time.Time) {
	if !value.IsZero() {
		values.Add(name, value.Format(time.RFC3339))
	}
}

func addIntQuery(values url.Values, name string, value int) {
	if value > 0 {
		values.Add(name, fmt.Sprintf("%d", value))
	}
}
