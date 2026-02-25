package graphql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"efctl/pkg/ui"
	"github.com/jedib0t/go-pretty/v6/table"
)

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   map[string]interface{} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// RunQuery executes a GraphQL query against the specified endpoint.
func RunQuery(endpoint, query string, variables map[string]interface{}) (*GraphQLResponse, error) {
	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var gqlResp GraphQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w (raw body: %s)", err, string(body))
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	return &gqlResp, nil
}

// QueryObject fetches basic info about an object.
func QueryObject(endpoint, id string) error {
	query := `query ($address: SuiAddress!) {
		object(address: $address) {
			address
			version
			digest
			owner {
				__typename
			}
		}
	}`

	variables := map[string]interface{}{"address": id}
	resp, err := RunQuery(endpoint, query, variables)
	if err != nil {
		return err
	}

	objData, ok := resp.Data["object"].(map[string]interface{})
	if !ok || objData == nil {
		return fmt.Errorf("object not found or invalid response")
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"Property", "Value"})

	t.AppendRow(table.Row{"Address", objData["address"]})
	t.AppendRow(table.Row{"Version", objData["version"]})
	t.AppendRow(table.Row{"Digest", objData["digest"]})

	// Safely get owner
	if owner, ok := objData["owner"].(map[string]interface{}); ok {
		t.AppendRow(table.Row{"Owner Type", owner["__typename"]})
	}

	ui.Info.Println("Object Details:")
	t.Render()
	return nil
}

// QueryPackage fetches modules from a user package.
func QueryPackage(endpoint, id string) error {
	query := `query ($address: SuiAddress!) {
		object(address: $address) {
			address
			version
			asMovePackage {
				modules {
					nodes {
						name
					}
				}
			}
		}
	}`

	variables := map[string]interface{}{"address": id}
	resp, err := RunQuery(endpoint, query, variables)
	if err != nil {
		return err
	}

	objData, ok := resp.Data["object"].(map[string]interface{})
	if !ok || objData == nil {
		return fmt.Errorf("package not found or invalid response")
	}

	pkgData, ok := objData["asMovePackage"].(map[string]interface{})
	if !ok || pkgData == nil {
		return fmt.Errorf("object is not a Move Package")
	}

	modulesRaw, ok := pkgData["modules"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("could not find modules field")
	}

	nodesRaw, ok := modulesRaw["nodes"].([]interface{})
	if !ok {
		return fmt.Errorf("could not find module nodes")
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"Module Name"})

	for _, node := range nodesRaw {
		if nodeMap, ok := node.(map[string]interface{}); ok {
			t.AppendRow(table.Row{nodeMap["name"]})
		}
	}

	ui.Info.Printf("Package Details (%s - Version %v):\n", objData["address"], objData["version"])
	t.Render()
	return nil
}
