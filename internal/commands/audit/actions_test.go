package audit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/docker/hub-tool/pkg/hub"
)

func TestRunActionsUsesAuthenticatedAccountByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/v2/auditlogs/example-user/actions")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"actions":{"repo":{"label":"Repository","actions":[{"name":"tag.push","label":"Tag Pushed","description":"Tag pushed"}]}}}`)
	}))
	defer server.Close()

	t.Setenv("DOCKER_HUB_API_URL", server.URL)
	t.Setenv("DOCKER_REGISTRY_URL", "registry.example.test")
	client, err := hub.NewClient(hub.WithHubAccount("example-user"), hub.WithHubToken("token"))
	assert.NilError(t, err)

	out := bytes.NewBuffer(nil)
	err = runActions(context.Background(), newTestStreams(out, io.Discard), client, actionsOptions{}, nil)
	assert.NilError(t, err)
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("Repository")))
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("tag.push")))
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("Tag pushed")))
}

func TestRunActionsUsesExplicitAccount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/v2/auditlogs/my-org/actions")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"actions":{}}`)
	}))
	defer server.Close()

	t.Setenv("DOCKER_HUB_API_URL", server.URL)
	t.Setenv("DOCKER_REGISTRY_URL", "registry.example.test")
	client, err := hub.NewClient(hub.WithHubAccount("example-user"), hub.WithHubToken("token"))
	assert.NilError(t, err)

	err = runActions(context.Background(), newTestStreams(io.Discard, io.Discard), client, actionsOptions{}, []string{"my-org"})
	assert.NilError(t, err)
}
