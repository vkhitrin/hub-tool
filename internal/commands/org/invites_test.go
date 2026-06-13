package org

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/docker/cli/cli/streams"
	"gotest.tools/v3/assert"

	"github.com/docker/hub-tool/pkg/hub"
)

const (
	inviteListResponse = `{"data":[{"id":"invite-id","inviter_username":"owner","invitee":"invitee@example.com","org":"example-org","team":"owners","created_at":"2026-01-02T03:04:05Z"}]}`
	bulkInviteResponse = `{"invitees":{"invitees":[{"invitee":"invitee@example.com","status":"invited"}]}}`
)

type testStreams struct {
	in  *streams.In
	out *streams.Out
	err io.Writer
}

func newTestStreams(out, err io.Writer) testStreams {
	return testStreams{
		in:  streams.NewIn(io.NopCloser(bytes.NewBuffer(nil))),
		out: streams.NewOut(out),
		err: err,
	}
}

func (s testStreams) In() *streams.In {
	return s.in
}

func (s testStreams) Out() *streams.Out {
	return s.out
}

func (s testStreams) Err() io.Writer {
	return s.err
}

func expectedBulkInviteRequest() hub.BulkInviteOptions {
	request := hub.BulkInviteOptions{}
	request.Org = "example-org"
	request.Team = "owners"
	request.Role = "member"
	request.Invitees = []string{"invitee@example.com"}
	request.DryRun = true
	return request
}

func newInviteTestClient(t *testing.T, handler http.HandlerFunc) *hub.Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	t.Setenv("DOCKER_HUB_API_URL", server.URL)
	t.Setenv("DOCKER_REGISTRY_URL", "registry.example.test")
	client, err := hub.NewClient(hub.WithHubToken("token"))
	assert.NilError(t, err)
	return client
}

func TestRunInvites(t *testing.T) {
	client := newInviteTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/v2/orgs/example-org/invites")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, inviteListResponse)
	})

	out := bytes.NewBuffer(nil)
	err := runInvites(newTestStreams(out, io.Discard), client, inviteOptions{}, "example-org")
	assert.NilError(t, err)
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("INVITEE")))
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("invitee@example.com")))
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("owners")))
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("owner")))
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("invite-id")))
}

func TestRunResendInvite(t *testing.T) {
	client := newInviteTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPatch)
		assert.Equal(t, r.URL.Path, "/v2/invites/invite-id/resend")
		w.WriteHeader(http.StatusNoContent)
	})

	out := bytes.NewBuffer(nil)
	err := runResendInvite(newTestStreams(out, io.Discard), client, "invite-id")
	assert.NilError(t, err)
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("Invite resent")))
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("invite-id")))
}

func TestRunRemoveInvite(t *testing.T) {
	client := newInviteTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodDelete)
		assert.Equal(t, r.URL.Path, "/v2/invites/invite-id")
		w.WriteHeader(http.StatusNoContent)
	})

	out := bytes.NewBuffer(nil)
	err := runRemoveInvite(newTestStreams(out, io.Discard), client, "invite-id")
	assert.NilError(t, err)
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("Invite cancelled")))
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("invite-id")))
}

func TestRunBulkInviteFromFile(t *testing.T) {
	file := t.TempDir() + "/invites.json"
	err := os.WriteFile(file, []byte(`{"org":"example-org","team":"owners","role":"member","invitees":["invitee@example.com"],"dry_run":true}`), 0o600)
	assert.NilError(t, err)

	client := newInviteTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPost)
		assert.Equal(t, r.URL.Path, "/v2/invites/bulk")

		var request hub.BulkInviteOptions
		assert.NilError(t, json.NewDecoder(r.Body).Decode(&request))
		assert.DeepEqual(t, request, expectedBulkInviteRequest())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprint(w, bulkInviteResponse)
	})

	out := bytes.NewBuffer(nil)
	err = runBulkInvite(newTestStreams(out, io.Discard), client, bulkInviteOptions{file: file})
	assert.NilError(t, err)
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("invitee@example.com")))
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("invited")))
}

func TestBulkInviteCommandRequiresFile(t *testing.T) {
	cmd := newBulkInviteCmd(newTestStreams(io.Discard, io.Discard), nil, invitesName)
	err := cmd.Args(cmd, nil)
	assert.Error(t, err, "--file is required")
}

func TestBulkInviteCommandRejectsPositionalArguments(t *testing.T) {
	cmd := newBulkInviteCmd(newTestStreams(io.Discard, io.Discard), nil, invitesName)
	assert.NilError(t, cmd.Flags().Set("file", "invites.json"))
	err := cmd.Args(cmd, []string{"example-org", "invitee@example.com"})
	assert.Error(t, err, `"bulk-invite" accepts input from --file only`)
}
