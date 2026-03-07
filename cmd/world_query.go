package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"efctl/pkg/graphql"
	"efctl/pkg/ui"
	"efctl/pkg/validate"
	"github.com/jedib0t/go-pretty/v6/list"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var NetworkEndpoints = map[string]string{
	"devnet":   "https://sui-devnet.hub.astria.org/graphql",
	"testnet":  "https://sui-testnet.hub.astria.org/graphql",
	"mainnet":  "https://sui-mainnet.hub.astria.org/graphql",
	"localnet": "http://localhost:9125/graphql",
}

var worldQueryCmd = &cobra.Command{
	Use:   "query [object_id]",
	Short: "Query an EVE Frontier Smart Assembly",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

		if err := validate.SuiAddress(id); err != nil {
			ui.Error.Println("Invalid object ID: " + err.Error())
			os.Exit(1)
		}

		// Handle network-to-endpoint mapping
		endpoint := GraphqlEndpoint

		// If endpoint is at default and network is specified, override with network defaults
		if endpoint == "http://localhost:9125/graphql" && Network != "localnet" {
			if url, ok := NetworkEndpoints[strings.ToLower(Network)]; ok {
				endpoint = url
			}
		}

		ui.Info.Printf("Querying world object %s (%s) at %s...\n", id, Network, endpoint)

		if err := queryWorldObject(endpoint, id); err != nil {
			ui.Error.Println("Query failed: " + err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	worldCmd.AddCommand(worldQueryCmd)
}

func queryWorldObject(endpoint, id string) error {
	query := `query ($address: SuiAddress!) {
	  object(address: $address) {
	    address
	    owner {
	      __typename
	    }
	    asMoveObject {
	      contents {
	        type {
	          repr
	        }
	        json
	      }
	    }
	    dynamicFields(first: 50) {
	      nodes {
	        name {
	          type { repr }
	          json
	        }
	        value {
	          ... on MoveValue { json }
	          ... on MoveObject {
	            contents { json }
	          }
	        }
	      }
	    }
	  }
	}`

	variables := map[string]interface{}{"address": id}

	ui.Debug.Println("Executing GraphQL Query:")
	ui.Debug.Println(query)

	if varBytes, err := json.MarshalIndent(variables, "", "  "); err == nil {
		ui.Debug.Println("Variables:")
		ui.Debug.Println(string(varBytes))
	}
	resp, err := graphql.RunQuery(endpoint, query, variables)
	if err != nil {
		return err
	}

	if respBytes, err := json.MarshalIndent(resp, "", "  "); err == nil {
		ui.Debug.Println("GraphQL Response:")
		ui.Debug.Println(string(respBytes))
	}

	objMap, ok := resp.Data["object"].(map[string]interface{})
	if !ok || objMap == nil {
		return fmt.Errorf("object not found or invalid response")
	}

	asMoveObj, ok := objMap["asMoveObject"].(map[string]interface{})
	if !ok || asMoveObj == nil {
		return fmt.Errorf("object is not a MoveObject")
	}

	contents, ok := asMoveObj["contents"].(map[string]interface{})
	if !ok || contents == nil {
		return fmt.Errorf("object contents not found")
	}

	typMap, ok := contents["type"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("object type not found")
	}

	repr, _ := typMap["repr"].(string)
	jsonMap, _ := contents["json"].(map[string]interface{})

	l := list.NewWriter()
	l.SetStyle(list.StyleConnectedRounded)

	address, _ := objMap["address"].(string)

	renderWorldObject(l, address, repr, jsonMap, objMap)

	output := l.Render()
	// Make the lines a lighter grey
	grey := pterm.NewStyle(pterm.FgGray).Sprint
	output = strings.ReplaceAll(output, "├─", grey("├─"))
	output = strings.ReplaceAll(output, "╰─", grey("╰─"))
	output = strings.ReplaceAll(output, "│ ", grey("│ "))

	fmt.Println(output)
	return nil
}

func renderWorldObject(l list.Writer, address string, repr string, jsonMap map[string]interface{}, objMap map[string]interface{}) {
	reprLower := strings.ToLower(repr)
	if strings.Contains(reprLower, "owner_cap") || strings.Contains(reprLower, "ownercap") {
		renderOwnerCap(l, address, jsonMap, repr)
	} else if strings.Contains(reprLower, "storage_unit") || strings.Contains(reprLower, "storageunit") {
		renderSSU(l, address, jsonMap, objMap, repr)
	} else if strings.Contains(reprLower, "network_node") || strings.Contains(reprLower, "networknode") {
		renderNetworkNode(l, address, jsonMap, objMap, repr)
	} else if strings.Contains(reprLower, "gate") {
		renderGate(l, address, jsonMap, objMap, repr)
	} else if strings.Contains(reprLower, "turret") {
		renderTurret(l, address, jsonMap, objMap, repr)
	} else if strings.Contains(reprLower, "character") {
		renderCharacter(l, address, jsonMap, repr)
	} else {
		displayName := deriveDisplayName(repr)
		l.AppendItem(fmt.Sprintf("📦 %s (%s)", displayName, address))
		l.Indent()
		l.AppendItem(fmt.Sprintf("%s %s", styledKey("Type"), styledValue(repr)))
		l.UnIndent()
	}
}

func deriveDisplayName(repr string) string {
	// Extract the last part of the type (e.g., GovernorCap from ...::world::GovernorCap)
	parts := strings.Split(repr, "::")
	name := parts[len(parts)-1]

	// Specific naming overrides
	overrides := map[string]string{
		"AdminACL": "Admin Access Control List",
	}
	if override, ok := overrides[name]; ok {
		return override
	}

	// Simple camelCase/PascalCase to Space Separated (e.g., GovernorCap -> Governor Cap)
	var result strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}
	return result.String()
}

func styledKey(key string) string {
	return pterm.NewStyle(pterm.FgWhite, pterm.Bold).Sprint(key + ":")
}

func styledValue(val string) string {
	return pterm.NewStyle(pterm.FgCyan).Sprint(val)
}

func styledStatus(status string) string {
	statusUpper := strings.ToUpper(status)
	style := pterm.NewStyle(pterm.FgYellow) // Default
	if statusUpper == "ONLINE" {
		style = pterm.NewStyle(pterm.FgLightGreen)
	} else if statusUpper == "OFFLINE" || statusUpper == "DESTROYED" {
		style = pterm.NewStyle(pterm.FgRed)
	}
	return style.Sprint(status)
}

func renderTenant(l list.Writer, jsonMap map[string]interface{}) {
	if key, ok := jsonMap["key"].(map[string]interface{}); ok {
		if tenant, ok := key["tenant"].(string); ok && tenant != "" {
			l.AppendItem(fmt.Sprintf("%s %s", styledKey("Tenant"), styledValue(tenant)))
		}
	}
}

func getMapValueString(m map[string]interface{}, key string) string {
	val, ok := m[key]
	if !ok || val == nil {
		return "N/A"
	}

	// Handle nested status structure: status -> status -> @variant
	if key == "status" {
		if inner, ok := val.(map[string]interface{}); ok {
			if s2, ok := inner["status"].(map[string]interface{}); ok {
				if variant, ok := s2["@variant"].(string); ok {
					return variant
				}
			}
		}
	}

	// Handle nested metadata structure: metadata -> name
	if key == "metadata" {
		if inner, ok := val.(map[string]interface{}); ok {
			if name, ok := inner["name"].(string); ok && name != "" {
				return name
			}
			if desc, ok := inner["description"].(string); ok && desc != "" {
				return desc
			}
		}
	}

	return fmt.Sprintf("%v", val)
}

func renderSSU(l list.Writer, address string, jsonMap map[string]interface{}, objMap map[string]interface{}, repr string) {
	l.AppendItem(fmt.Sprintf("📦 Smart Storage Unit (%s)", address))
	l.Indent()
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Type"), styledValue(repr)))

	renderTenant(l, jsonMap)

	status := getMapValueString(jsonMap, "status")
	owner := getMapValueWithFallback(jsonMap, "owner", "owner_cap_id")
	networkNode := getMapValueWithFallback(jsonMap, "network_node", "energy_source_id")

	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Status"), styledStatus(status)))
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Owner"), styledValue(owner)))
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Connected Node"), styledValue(networkNode)))

	if metadata, ok := jsonMap["metadata"]; ok {
		renderMetadata(l, metadata)
	}

	renderDynamicFieldsAsInventories(l, objMap)
	l.UnIndent()
}

func renderGate(l list.Writer, address string, jsonMap map[string]interface{}, objMap map[string]interface{}, repr string) {
	l.AppendItem(fmt.Sprintf("📦 Smart Gate (%s)", address))
	l.Indent()
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Type"), styledValue(repr)))

	renderTenant(l, jsonMap)

	status := getMapValueString(jsonMap, "status")
	networkNode := getMapValueWithFallback(jsonMap, "energy_source_id", "network_node")
	pairedGateId := getMapValueWithFallback(jsonMap, "linked_gate_id", "paired_gate_id", "paired_gate")

	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Status"), styledStatus(status)))
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Connected Node"), styledValue(networkNode)))
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Paired Gate ID"), styledValue(pairedGateId)))

	if metadata, ok := jsonMap["metadata"]; ok {
		renderMetadata(l, metadata)
	}

	l.UnIndent()
}

func renderTurret(l list.Writer, address string, jsonMap map[string]interface{}, objMap map[string]interface{}, repr string) {
	l.AppendItem(fmt.Sprintf("📦 Smart Turret (%s)", address))
	l.Indent()
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Type"), styledValue(repr)))

	renderTenant(l, jsonMap)

	status := getMapValueString(jsonMap, "status")
	networkNode := getMapValueWithFallback(jsonMap, "energy_source_id", "network_node")

	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Status"), styledStatus(status)))
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Connected Node"), styledValue(networkNode)))

	if metadata, ok := jsonMap["metadata"]; ok {
		renderMetadata(l, metadata)
	}

	l.UnIndent()
}

func renderNetworkNode(l list.Writer, address string, jsonMap map[string]interface{}, objMap map[string]interface{}, repr string) {
	l.AppendItem(fmt.Sprintf("📦 Network Node (%s)", address))
	l.Indent()
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Type"), styledValue(repr)))

	renderTenant(l, jsonMap)

	status := getMapValueString(jsonMap, "status")
	owner := getMapValueWithFallback(jsonMap, "owner", "owner_cap_id")

	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Status"), styledStatus(status)))
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Owner"), styledValue(owner)))

	// Render Fuel
	if fuel, ok := jsonMap["fuel"].(map[string]interface{}); ok {
		qty := getMapValueString(fuel, "quantity")
		max := getMapValueString(fuel, "max_capacity")
		burning := "No"
		if b, ok := fuel["is_burning"].(bool); ok && b {
			burning = "Yes"
		}
		l.AppendItem(fmt.Sprintf("%s %s", styledKey("Fuel"), styledValue(fmt.Sprintf("%s / %s (Burning: %s)", qty, max, burning))))
	}

	// Render Energy
	if energy, ok := jsonMap["energy_source"].(map[string]interface{}); ok {
		curr := getMapValueString(energy, "current_energy_production")
		max := getMapValueString(energy, "max_energy_production")
		reserved := getMapValueString(energy, "total_reserved_energy")
		l.AppendItem(fmt.Sprintf("%s %s", styledKey("Energy Production"), styledValue(fmt.Sprintf("%s / %s (Reserved: %s)", curr, max, reserved))))
	}

	// Render Connected Assemblies
	if conn, ok := jsonMap["connected_assembly_ids"].([]interface{}); ok {
		l.AppendItem(fmt.Sprintf("%s %s", styledKey("Connected Assemblies"), styledValue(fmt.Sprintf("%d", len(conn)))))
		if len(conn) > 0 {
			l.Indent()
			for _, id := range conn {
				l.AppendItem(fmt.Sprintf("%v", id))
			}
			l.UnIndent()
		}
	}

	if metadata, ok := jsonMap["metadata"]; ok {
		renderMetadata(l, metadata)
	}

	l.UnIndent()
}

func renderOwnerCap(l list.Writer, address string, jsonMap map[string]interface{}, repr string) {
	l.AppendItem(fmt.Sprintf("🔑 Owner Capability (%s)", address))
	l.Indent()
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Type"), styledValue(repr)))
	target := getMapValueString(jsonMap, "authorized_object_id")
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Authorized Object"), styledValue(target)))
	l.UnIndent()
}

func renderCharacter(l list.Writer, address string, jsonMap map[string]interface{}, repr string) {
	l.AppendItem(fmt.Sprintf("👤 Character (%s)", address))
	l.Indent()
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Type"), styledValue(repr)))

	renderTenant(l, jsonMap)

	tribe := getMapValueString(jsonMap, "tribe_id")
	charAddr := getMapValueString(jsonMap, "character_address")
	ownerCap := getMapValueString(jsonMap, "owner_cap_id")

	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Tribe ID"), styledValue(tribe)))
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Character Address"), styledValue(charAddr)))
	l.AppendItem(fmt.Sprintf("%s %s", styledKey("Owner Capability"), styledValue(ownerCap)))

	if metadata, ok := jsonMap["metadata"]; ok {
		renderMetadata(l, metadata)
	}

	l.UnIndent()
}

func renderMetadata(l list.Writer, metadata interface{}) {
	m, ok := metadata.(map[string]interface{})
	if !ok || len(m) == 0 {
		l.AppendItem(fmt.Sprintf("Metadata: %v", metadata))
		return
	}

	l.AppendItem(styledKey("Metadata"))
	l.Indent()
	for k, v := range m {
		l.AppendItem(fmt.Sprintf("%s %s", styledKey(k), styledValue(fmt.Sprintf("%v", v))))
	}
	l.UnIndent()
}

func getMapValueWithFallback(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		val := getMapValueString(m, key)
		if val != "N/A" && val != "" {
			return val
		}
	}
	return "N/A"
}

func renderDynamicFieldsAsInventories(l list.Writer, objMap map[string]interface{}) {
	dfMap, ok := objMap["dynamicFields"].(map[string]interface{})
	if !ok {
		return
	}
	nodes, ok := dfMap["nodes"].([]interface{})
	if !ok || len(nodes) == 0 {
		return
	}

	l.AppendItem("Inventories")
	l.Indent()

	for _, n := range nodes {
		nodeMap, ok := n.(map[string]interface{})
		if !ok {
			continue
		}

		nameMap, ok := nodeMap["name"].(map[string]interface{})
		if !ok {
			continue
		}

		nameJson := ""
		if js, ok := nameMap["json"].(string); ok {
			nameJson = js
		} else if nameMap["json"] != nil {
			nameJson = fmt.Sprintf("%v", nameMap["json"])
		}

		if nameJson == "" {
			nameJson = "Unknown Inventory" // fallback
		}

		l.AppendItem(nameJson)
		l.Indent()

		valueMap, ok := nodeMap["value"].(map[string]interface{})
		if ok {
			if valJson, ok := valueMap["json"]; ok {
				renderItems(l, valJson)
			} else if contents, ok := valueMap["contents"].(map[string]interface{}); ok {
				if valJson, ok := contents["json"]; ok {
					renderItems(l, valJson)
				}
			}
		}

		l.UnIndent()
	}

	l.UnIndent()
}

func renderItems(l list.Writer, itemsJson interface{}) {
	if m, ok := itemsJson.(map[string]interface{}); ok {
		if contents, ok := m["contents"]; ok {
			renderItemsSlice(l, contents)
			return
		}
		if items, ok := m["items"]; ok {
			renderItems(l, items)
			return
		}
		formatItem(l, m)
		return
	}
	renderItemsSlice(l, itemsJson)
}

func renderItemsSlice(l list.Writer, slice interface{}) {
	if s, ok := slice.([]interface{}); ok {
		for _, item := range s {
			formatItem(l, item)
		}
	} else {
		formatItem(l, slice)
	}
}

func formatItem(l list.Writer, item interface{}) {
	if m, ok := item.(map[string]interface{}); ok {
		// Handle nested value structure: { key: "...", value: { ... } }
		if val, ok := m["value"].(map[string]interface{}); ok {
			formatItem(l, val)
			return
		}

		name := "Item"
		if n, ok := m["name"]; ok && n != "" {
			name = fmt.Sprintf("%v", n)
		} else if tid, ok := m["type_id"]; ok {
			name = fmt.Sprintf("Type %v", tid)
		} else if t, ok := m["type"]; ok {
			name = fmt.Sprintf("%v", t)
		}

		qty := ""
		if q, ok := m["quantity"]; ok {
			qty = fmt.Sprintf("%vx ", q)
		} else if c, ok := m["count"]; ok {
			qty = fmt.Sprintf("%vx ", c)
		}

		l.AppendItem(fmt.Sprintf("Item: %s", styledValue(fmt.Sprintf("%s%s", qty, name))))
	} else {
		l.AppendItem(fmt.Sprintf("Item: %v", item))
	}
}
