//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"efctl/pkg/graphql"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGraphQLRoundTrip validates a full request → parse → response cycle
// using a realistic GraphQL response shape from Sui's RPC.
func TestGraphQLRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphql.GraphQLRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		assert.Contains(t, req.Query, "object")
		assert.Equal(t, "0xaabb", req.Variables["id"])

		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"address": "0xaabb",
					"version": 42,
					"digest":  "CiPsE3...",
					"owner": map[string]interface{}{
						"asAddress": map[string]interface{}{
							"address": "0xowner",
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	query := `query ($id: SuiAddress!) {
		object(address: $id) {
			address
			version
			digest
			owner { ... on AddressOwner { owner { asAddress { address } } } }
		}
	}`
	vars := map[string]interface{}{"id": "0xaabb"}

	resp, err := graphql.RunQuery(srv.URL, query, vars)
	require.NoError(t, err)

	obj, ok := resp.Data["object"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "0xaabb", obj["address"])
}

// TestGraphQLError validates that GraphQL errors are properly surfaced.
func TestGraphQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data":   nil,
			"errors": []map[string]string{{"message": "Object not found: 0xdead"}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	_, err := graphql.RunQuery(srv.URL, "query { object }", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Object not found: 0xdead")
}
