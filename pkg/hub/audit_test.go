package hub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
)

func TestGetAuditLogsDecodesEventData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/v2/auditlogs/example-org")
		assert.Equal(t, r.URL.Query().Get("action"), "repo.tag.push")
		assert.Equal(t, r.URL.Query().Get("page"), "2")
		assert.Equal(t, r.URL.Query().Get("page_size"), "10")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"logs":[{"account":"example-org","action":"repo.tag.push","name":"example-org/example-repo","actor":"example-ci-user - access token \"CI\"","data":{"access_token_id":"00000000-0000-0000-0000-000000000000","access_token_label":"CI","access_token_type":"TYPE_UNSPECIFIED","digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111","tag":"latest"},"timestamp":"2026-01-02T03:04:05.006Z","action_description":"pushed the tag latest"}]}`))
	}))
	defer server.Close()

	t.Setenv("DOCKER_HUB_API_URL", server.URL)
	t.Setenv("DOCKER_REGISTRY_URL", "registry.example.test")
	client, err := NewClient(WithHubToken("token"))
	assert.NilError(t, err)

	logs, err := client.GetAuditLogs(context.Background(), "example-org", AuditLogOptions{
		Action:   "repo.tag.push",
		Page:     2,
		PageSize: 10,
	})
	assert.NilError(t, err)
	assert.Equal(t, len(logs), 1)
	assert.Equal(t, logs[0].Account, "example-org")
	assert.Equal(t, logs[0].Action, "repo.tag.push")
	assert.Equal(t, logs[0].Name, "example-org/example-repo")
	assert.Equal(t, logs[0].Actor, `example-ci-user - access token "CI"`)
	assert.Equal(t, logs[0].Data["access_token_label"], "CI")
	assert.Equal(t, logs[0].Data["digest"], "sha256:1111111111111111111111111111111111111111111111111111111111111111")
	assert.Equal(t, logs[0].Data["tag"], "latest")
	assert.Equal(t, logs[0].Timestamp.Format("2006-01-02T15:04:05.000Z"), "2026-01-02T03:04:05.006Z")
	assert.Equal(t, logs[0].ActionDescription, "pushed the tag latest")
}

func TestGetAuditLogActionsSortsByGroupAndName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/v2/auditlogs/example-org/actions")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"actions":{"repo":{"label":"Repository","actions":[{"name":"tag.push","label":"Tag Pushed","description":"Tags pushed"},{"name":"create","label":"Repository Created","description":"Repository created"}]},"billing":{"label":"Billing","actions":[{"name":"plan.seat_add","label":"Seat Added","description":"Seat added"}]}}}`))
	}))
	defer server.Close()

	t.Setenv("DOCKER_HUB_API_URL", server.URL)
	t.Setenv("DOCKER_REGISTRY_URL", "registry.example.test")
	client, err := NewClient(WithHubToken("token"))
	assert.NilError(t, err)

	actions, err := client.GetAuditLogActions(context.Background(), "example-org")
	assert.NilError(t, err)
	assert.Equal(t, len(actions), 3)
	assert.Equal(t, actions[0].Group, "billing")
	assert.Equal(t, actions[0].Name, "plan.seat_add")
	assert.Equal(t, actions[1].Group, "repo")
	assert.Equal(t, actions[1].Name, "create")
	assert.Equal(t, actions[2].Group, "repo")
	assert.Equal(t, actions[2].Name, "tag.push")
}
