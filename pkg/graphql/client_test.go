package graphql

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── RunQuery ───────────────────────────────────────────────────────

func TestRunQuery_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req GraphQLRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Contains(t, req.Query, "{ hello }")

		resp := GraphQLResponse{
			Data: map[string]interface{}{"hello": "world"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	resp, err := RunQuery(srv.URL, "{ hello }", nil)
	require.NoError(t, err)
	assert.Equal(t, "world", resp.Data["hello"])
}

func TestRunQuery_WithVariables(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GraphQLRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "0x123", req.Variables["id"])

		json.NewEncoder(w).Encode(GraphQLResponse{
			Data: map[string]interface{}{"object": "found"},
		})
	}))
	defer srv.Close()

	resp, err := RunQuery(srv.URL, "query ($id: String!) { object(id: $id) }", map[string]interface{}{"id": "0x123"})
	require.NoError(t, err)
	assert.Equal(t, "found", resp.Data["object"])
}

func TestRunQuery_GraphQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":   nil,
			"errors": []map[string]string{{"message": "object not found"}},
		})
	}))
	defer srv.Close()

	_, err := RunQuery(srv.URL, "{ broken }", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "object not found")
}

func TestRunQuery_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := RunQuery(srv.URL, "{ q }", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response JSON")
}

func TestRunQuery_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	// Should still parse the body (empty data, no errors field)
	resp, err := RunQuery(srv.URL, "{ q }", nil)
	require.NoError(t, err)
	assert.Empty(t, resp.Data)
}

// ── URL validation ─────────────────────────────────────────────────

func TestRunQuery_RejectsNonHTTPScheme(t *testing.T) {
	_, err := RunQuery("ftp://example.com/graphql", "{ q }", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid endpoint URL scheme")
}

func TestRunQuery_RejectsFileScheme(t *testing.T) {
	_, err := RunQuery("file:///etc/passwd", "{ q }", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid endpoint URL scheme")
}

func TestRunQuery_RejectsEmptyScheme(t *testing.T) {
	_, err := RunQuery("example.com/graphql", "{ q }", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid endpoint URL scheme")
}

func TestRunQuery_AcceptsHTTPS(t *testing.T) {
	// Will fail to connect, but the URL validation should pass
	_, err := RunQuery("https://localhost:99999/graphql", "{ q }", nil)
	assert.Error(t, err)
	// The error should be a connection error, not a URL validation error
	assert.NotContains(t, err.Error(), "invalid endpoint URL scheme")
}

func TestRunQuery_ConnectionRefused(t *testing.T) {
	_, err := RunQuery("http://localhost:1/graphql", "{ q }", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute request")
}
