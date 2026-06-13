package audit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/cli/cli/streams"
	"gotest.tools/v3/assert"

	"github.com/docker/hub-tool/pkg/hub"
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

func TestAuditLogOptionsValidation(t *testing.T) {
	testCases := []struct {
		name          string
		opts          listOptions
		expectedError string
	}{
		{
			name:          "page must be positive",
			opts:          listOptions{page: 0, pageSize: 25},
			expectedError: "--page must be greater than 0",
		},
		{
			name:          "page size must be positive",
			opts:          listOptions{page: 1, pageSize: 0},
			expectedError: "--page-size must be greater than 0",
		},
		{
			name:          "to must be after from",
			opts:          listOptions{page: 1, pageSize: 25, from: "2024-01-02T00:00:00Z", to: "2024-01-01T00:00:00Z"},
			expectedError: "--to must be newer than --from",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.opts.auditLogOptions()
			assert.Error(t, err, testCase.expectedError)
		})
	}
}

func TestRunListFetchesAuditLogsWithFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/v2/auditlogs/example-org")
		assert.Equal(t, r.URL.Query().Get("action"), "repo.tag.push")
		assert.Equal(t, r.URL.Query().Get("name"), "example-org/example-repo")
		assert.Equal(t, r.URL.Query().Get("actor"), "example-ci-user")
		assert.Equal(t, r.URL.Query().Get("from"), "2024-01-01T00:00:00Z")
		assert.Equal(t, r.URL.Query().Get("to"), "2024-01-02T00:00:00Z")
		assert.Equal(t, r.URL.Query().Get("page"), "3")
		assert.Equal(t, r.URL.Query().Get("page_size"), "2")

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"logs":[{"account":"example-org","action":"repo.tag.push","name":"example-org/example-repo","actor":"example-ci-user","timestamp":"2024-01-01T12:00:00Z","action_description":"pushed the tag latest"}]}`)
	}))
	defer server.Close()

	t.Setenv("DOCKER_HUB_API_URL", server.URL)
	t.Setenv("DOCKER_REGISTRY_URL", "registry.example.test")
	client, err := hub.NewClient(hub.WithHubToken("token"))
	assert.NilError(t, err)

	out := bytes.NewBuffer(nil)
	streams := newTestStreams(out, io.Discard)
	err = runList(context.Background(), streams, client, listOptions{
		action:   "repo.tag.push",
		name:     "example-org/example-repo",
		actor:    "example-ci-user",
		from:     "2024-01-01T00:00:00Z",
		to:       "2024-01-02T00:00:00Z",
		page:     3,
		pageSize: 2,
	}, []string{"example-org"})
	assert.NilError(t, err)
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("ACTION")))
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("repo.tag.push")))
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("example-org/example-repo")))
	assert.Assert(t, bytes.Contains(out.Bytes(), []byte("pushed the tag latest")))
}

func TestRunListMapsInternalServerErrorForAccountNamespace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/v2/auditlogs/example-user")
		http.Error(w, `{"message":"internal server error"}`, http.StatusInternalServerError)
	}))
	defer server.Close()

	t.Setenv("DOCKER_HUB_API_URL", server.URL)
	t.Setenv("DOCKER_REGISTRY_URL", "registry.example.test")
	client, err := hub.NewClient(hub.WithHubToken("token"))
	assert.NilError(t, err)

	err = runList(context.Background(), newTestStreams(io.Discard, io.Discard), client, listOptions{page: 1, pageSize: 25}, []string{"example-user"})
	assert.Error(t, err, `failed to list audit logs for "example-user": audit log events require an organization namespace`)
}
