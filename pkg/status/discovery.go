package status

import (
	"fmt"
	"strings"

	"efctl/pkg/graphql"
)

type DiscoveredPackage struct {
	ID      string
	Version string
	Owner   string
}

func DiscoverAssemblies(endpoint, worldPkgID string) ([]DiscoveredObject, error) {
	if worldPkgID == "" {
		return nil, nil
	}

	types := []string{
		fmt.Sprintf("%s::gate::Gate", worldPkgID),
		fmt.Sprintf("%s::turret::Turret", worldPkgID),
		fmt.Sprintf("%s::storage_unit::StorageUnit", worldPkgID),
	}

	var all []DiscoveredObject
	for _, t := range types {
		objs, err := queryObjectsByType(endpoint, t)
		if err != nil {
			return nil, err
		}
		all = append(all, objs...)
	}

	return all, nil
}

func DiscoverExtensions(endpoint string, packageIDs []string) ([]DiscoveredObject, error) {
	if len(packageIDs) == 0 {
		return nil, nil
	}

	var all []DiscoveredObject
	for _, pkgID := range packageIDs {
		modules, err := queryPackageModules(endpoint, pkgID)
		if err != nil {
			// Fallback to searching common modules if package lookup fails
			modules = []string{"extension", "config"}
		}

		for _, mod := range modules {
			extType := fmt.Sprintf("%s::%s::ExtensionConfig", pkgID, mod)
			objs, err := queryObjectsByType(endpoint, extType)
			if err != nil {
				return nil, err
			}
			all = append(all, objs...)
		}
	}
	return all, nil
}

func DiscoverPackages(endpoint string, owners []string) ([]DiscoveredPackage, error) {
	if len(owners) == 0 {
		return nil, nil
	}

	capType := "0x2::package::UpgradeCap"
	var allPkgs []DiscoveredPackage
	pkgMap := make(map[string]bool)

	for _, owner := range owners {
		query := `query ($owner: SuiAddress!, $type: String!) {
			objects(filter: { owner: $owner, type: $type }) {
				nodes {
					asMoveObject {
						contents {
							json
						}
					}
				}
			}
		}`
		variables := map[string]interface{}{
			"owner": owner,
			"type":  capType,
		}
		resp, err := graphql.RunQuery(endpoint, query, variables)
		if err != nil {
			continue // Skip this owner if query fails
		}

		objs, ok := resp.Data["objects"].(map[string]interface{})
		if !ok {
			continue
		}
		nodes, ok := objs["nodes"].([]interface{})
		if !ok {
			continue
		}

		for _, n := range nodes {
			node, ok := n.(map[string]interface{})
			if !ok {
				continue
			}
			moveObj, _ := node["asMoveObject"].(map[string]interface{})
			if moveObj == nil {
				continue
			}
			contents, _ := moveObj["contents"].(map[string]interface{})
			if contents == nil {
				continue
			}
			jsonMap, _ := contents["json"].(map[string]interface{})
			if jsonMap == nil {
				continue
			}
			pkgID, _ := jsonMap["package"].(string)
			version := "-"
			if v, ok := jsonMap["version"]; ok {
				version = fmt.Sprintf("%v", v)
			}

			if pkgID != "" && !pkgMap[pkgID] {
				pkgMap[pkgID] = true
				allPkgs = append(allPkgs, DiscoveredPackage{
					ID:      pkgID,
					Version: version,
					Owner:   owner,
				})
			}
		}
	}

	return allPkgs, nil
}

func queryObjectsByType(endpoint, objectType string) ([]DiscoveredObject, error) {
	query := `query ($type: String!) {
		objects(filter: { type: $type }) {
			nodes {
				address
				asMoveObject {
					contents {
						type {
							repr
						}
						json
					}
				}
			}
		}
	}`

	variables := map[string]interface{}{"type": objectType}
	resp, err := graphql.RunQuery(endpoint, query, variables)
	if err != nil {
		return nil, err
	}

	return parseObjectNodes(resp.Data, objectType), nil
}

func queryPackageModules(endpoint, packageID string) ([]string, error) {
	query := `query ($address: SuiAddress!) {
		object(address: $address) {
			asMovePackage {
				modules {
					nodes {
						name
					}
				}
			}
		}
	}`

	variables := map[string]interface{}{"address": packageID}
	resp, err := graphql.RunQuery(endpoint, query, variables)
	if err != nil {
		return nil, err
	}

	obj, _ := resp.Data["object"].(map[string]interface{})
	if obj == nil {
		return nil, fmt.Errorf("package not found: %s", packageID)
	}
	pkg, _ := obj["asMovePackage"].(map[string]interface{})
	if pkg == nil {
		return nil, fmt.Errorf("object %s is not a Move package", packageID)
	}
	modulesRoot, _ := pkg["modules"].(map[string]interface{})
	if modulesRoot == nil {
		return nil, fmt.Errorf("could not find modules field")
	}
	nodes, _ := modulesRoot["nodes"].([]interface{})
	if nodes == nil {
		return nil, fmt.Errorf("could not find module nodes")
	}

	var result []string
	for _, n := range nodes {
		node, ok := n.(map[string]interface{})
		if ok {
			name, _ := node["name"].(string)
			if name != "" {
				result = append(result, name)
			}
		}
	}
	return result, nil
}

func queryObjectsBySuffix(endpoint, suffix string) []DiscoveredObject {
	// If we can't filter by suffix in GraphQL easily, we might need to query all and filter.
	// But in a local dev env, the number of objects is small.
	// Alternatively, if we have the builder package ID we could use it.

	// Let's try to query for 'ExtensionConfig' if we can find a way to filter.
	// Sui GraphQL doesn't support suffix matching in 'type' filter.

	// Strategy: Get all objects owned by any of the addresses in the registry?
	// Simplified for now: just return empty or implement a wider search.
	return nil
}

func parseObjectNodes(data map[string]interface{}, filterType string) []DiscoveredObject {
	var results []DiscoveredObject

	objs, ok := data["objects"].(map[string]interface{})
	if !ok {
		return nil
	}
	nodes, ok := objs["nodes"].([]interface{})
	if !ok {
		return nil
	}

	for _, n := range nodes {
		node, ok := n.(map[string]interface{})
		if !ok {
			continue
		}

		id, _ := node["address"].(string)

		moveObj, _ := node["asMoveObject"].(map[string]interface{})
		if moveObj == nil {
			continue
		}
		contents, _ := moveObj["contents"].(map[string]interface{})
		if contents == nil {
			continue
		}
		typeObj, _ := contents["type"].(map[string]interface{})
		fullType := ""
		if typeObj != nil {
			fullType, _ = typeObj["repr"].(string)
		}

		name := ""
		if idx := strings.LastIndex(filterType, "::"); idx >= 0 {
			name = filterType[idx+2:]
		}

		results = append(results, DiscoveredObject{
			ID:   id,
			Type: shortenType(fullType),
			Name: name,
		})
	}

	return results
}

func shortenType(t string) string {
	if t == "" {
		return ""
	}
	parts := strings.Split(t, "::")
	if len(parts) < 3 {
		return t // Not a standard Move type
	}

	pkg := parts[0]
	if len(pkg) > 10 && strings.HasPrefix(pkg, "0x") {
		pkg = pkg[:6] + "..." + pkg[len(pkg)-4:]
	}

	return fmt.Sprintf("%s::%s::%s", pkg, parts[1], parts[2])
}
