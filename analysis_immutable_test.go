package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-github/v93/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newReleaseClient returns a client for a server that answers the
// latest-release request of repository o/r with the given status and body.
// Any other request fails the test.
func newReleaseClient(t *testing.T, status int, body string) *github.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !strings.HasSuffix(req.URL.Path, "/repos/o/r/releases/latest") {
			assert.Failf(t, "unexpected request", "%s %s", req.Method, req.URL.Path)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	base := srv.URL + "/"
	client, err := github.NewClient(github.WithURLs(&base, &base))
	require.NoError(t, err)
	return client
}

func TestImmutableReleasesAnalyzer(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   ImmutableResult
	}{
		{
			name: "immutable release", status: http.StatusOK,
			body: `{"id":1,"tag_name":"v1.0.0","immutable":true}`,
			want: ImmutableResult{Repository: "o/r", HasRelease: true, LatestReleaseImmutable: true},
		},
		{
			name: "mutable release", status: http.StatusOK,
			body: `{"id":1,"tag_name":"v1.0.0","immutable":false}`,
			want: ImmutableResult{Repository: "o/r", HasRelease: true},
		},
		{
			name: "old release without the field", status: http.StatusOK,
			body: `{"id":1,"tag_name":"v0.1.0"}`,
			want: ImmutableResult{Repository: "o/r", HasRelease: true},
		},
		{
			name: "no release at all", status: http.StatusNotFound,
			body: `{"message":"Not Found"}`,
			want: ImmutableResult{Repository: "o/r"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := NewImmutableReleasesAnalyzer(newReleaseClient(t, test.status, test.body))

			got, err := a.Analyze(context.Background(), "unused", "o/r")

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestImmutableReleasesAnalyzerErrors(t *testing.T) {
	t.Run("server error is not read as no release", func(t *testing.T) {
		a := NewImmutableReleasesAnalyzer(newReleaseClient(t, http.StatusInternalServerError, `{"message":"boom"}`))

		got, err := a.Analyze(context.Background(), "unused", "o/r")

		assert.ErrorContains(t, err, "fetching latest release for o/r")
		assert.Nil(t, got)
	})

	t.Run("malformed repository name", func(t *testing.T) {
		a := NewImmutableReleasesAnalyzer(newReleaseClient(t, http.StatusOK, `{}`))

		got, err := a.Analyze(context.Background(), "unused", "no-slash")

		assert.ErrorContains(t, err, "unexpected repo format")
		assert.Nil(t, got)
	})
}
