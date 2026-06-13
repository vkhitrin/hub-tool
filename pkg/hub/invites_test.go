package hub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
)

const (
	inviteListResponse = `{"data":[{"id":"invite-id","inviter_username":"owner","invitee":"invitee@example.com","org":"example-org","team":"owners","created_at":"2026-01-02T03:04:05Z"}]}`
	bulkInviteResponse = `{"invitees":{"invitees":[{"invitee":"invitee@example.com","status":"invited","invite":{"id":"invite-id","inviter_username":"owner","invitee":"invitee@example.com","org":"example-org","team":"owners","created_at":"2026-01-02T03:04:05Z"}}]}}`
)

var bulkInviteRequest = BulkInviteOptions{
	Org:      "example-org",
	Team:     "owners",
	Role:     "member",
	Invitees: []string{"invitee@example.com"},
	DryRun:   true,
}

var expectedInvite = Invite{
	ID:              "invite-id",
	InviterUsername: "owner",
	Invitee:         "invitee@example.com",
	Org:             "example-org",
	Team:            "owners",
	CreatedAt:       "2026-01-02T03:04:05Z",
}

func newInviteTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	t.Setenv("DOCKER_HUB_API_URL", server.URL)
	t.Setenv("DOCKER_REGISTRY_URL", "registry.example.test")
	client, err := NewClient(WithHubToken("token"))
	assert.NilError(t, err)
	return client
}

func TestGetInvites(t *testing.T) {
	client := newInviteTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/v2/orgs/example-org/invites")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, inviteListResponse)
	})

	invites, err := client.GetInvites("example-org")
	assert.NilError(t, err)
	assert.DeepEqual(t, invites, []Invite{expectedInvite})
}

func TestResendInvite(t *testing.T) {
	client := newInviteTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPatch)
		assert.Equal(t, r.URL.Path, "/v2/invites/invite-id/resend")
		w.WriteHeader(http.StatusNoContent)
	})

	assert.NilError(t, client.ResendInvite("invite-id"))
}

func TestRemoveInvite(t *testing.T) {
	client := newInviteTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodDelete)
		assert.Equal(t, r.URL.Path, "/v2/invites/invite-id")
		w.WriteHeader(http.StatusNoContent)
	})

	assert.NilError(t, client.RemoveInvite("invite-id"))
}

func TestBulkInvite(t *testing.T) {
	client := newInviteTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPost)
		assert.Equal(t, r.URL.Path, "/v2/invites/bulk")

		var request BulkInviteOptions
		assert.NilError(t, json.NewDecoder(r.Body).Decode(&request))
		assert.DeepEqual(t, request, bulkInviteRequest)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprint(w, bulkInviteResponse)
	})

	results, err := client.BulkInvite(bulkInviteRequest)
	assert.NilError(t, err)
	assert.DeepEqual(t, results, []BulkInviteResult{
		{
			Invitee: "invitee@example.com",
			Status:  "invited",
			Invite:  &expectedInvite,
		},
	})
}
