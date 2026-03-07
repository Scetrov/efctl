//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"efctl/cmd"
	"github.com/pterm/pterm"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetFlags() {
	cmd.GraphqlEndpoint = "http://localhost:9125/graphql"
	cmd.Network = "localnet"

	root := cmd.GetRootCmd()
	// Reset all flags to their default values and mark them as not changed
	root.Flags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
		f.Changed = false
	})
	root.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
		f.Changed = false
	})
	
	if worldCmd, _, err := root.Find([]string{"world"}); err == nil {
		worldCmd.Flags().VisitAll(func(f *pflag.Flag) {
			f.Value.Set(f.DefValue)
			f.Changed = false
		})
	}
}

func TestWorldQuery_SSU(t *testing.T) {
	resetFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"address": "0x1111",
					"owner": map[string]interface{}{"__typename": "AddressOwner"},
					"asMoveObject": map[string]interface{}{
						"contents": map[string]interface{}{
							"type": map[string]interface{}{"repr": "0xpkg::smart_storage_unit::SmartStorageUnit"},
							"json": map[string]interface{}{
								"status":       "ONLINE",
								"network_node": "0xnode",
								"metadata":     "Some info",
								"owner":        "0xowner",
							},
						},
					},
					"dynamicFields": map[string]interface{}{
						"nodes": []interface{}{
							map[string]interface{}{
								"name": map[string]interface{}{"json": "SSU Inventory"},
								"value": map[string]interface{}{
									"json": []interface{}{
										map[string]interface{}{"name": "Carbonaceous Ore", "count": 100},
										map[string]interface{}{"name": "Silicate Minerals", "count": 10},
									},
								},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldStdout := os.Stdout
	r, w, errPipe := os.Pipe()
	require.NoError(t, errPipe)

	os.Stdout = w
	pterm.SetDefaultOutput(w)
	pterm.DisableColor()

	t.Cleanup(func() {
		pterm.EnableColor()
		pterm.SetDefaultOutput(oldStdout)
		os.Stdout = oldStdout
		if w != nil {
			_ = w.Close()
		}
		if r != nil {
			_ = r.Close()
		}
	})

	root := cmd.GetRootCmd()
	root.SetArgs([]string{"world", "query", "0x1111", "--endpoint", srv.URL})
	err := root.Execute()
	require.NoError(t, err)

	_ = w.Close()
	w = nil

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Smart Storage Unit (0x1111)")
	assert.Contains(t, output, "Type")
	assert.Contains(t, output, "0xpkg::smart_storage_unit::SmartStorageUnit")
	assert.Contains(t, output, "Status") // Check label exists (ignoring styling for now as contains still matches substring)
	assert.Contains(t, output, "ONLINE")
	assert.Contains(t, output, "Connected Node")
	assert.Contains(t, output, "0xnode")
	assert.Contains(t, output, "Inventories")
	assert.Contains(t, output, "SSU Inventory")
	assert.Contains(t, output, "Item: 100x Carbonaceous Ore")
	assert.Contains(t, output, "Item: 10x Silicate Minerals")
}

func TestWorldQuery_Gate(t *testing.T) {
	resetFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"address": "0x2222",
					"asMoveObject": map[string]interface{}{
						"contents": map[string]interface{}{
							"type": map[string]interface{}{"repr": "0xpkg::smart_gate::SmartGate"},
							"json": map[string]interface{}{
								"status":         "ONLINE",
								"network_node":   "0xnode123",
								"paired_gate_id": "0xpaired_addr",
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldStdout := os.Stdout
	r, w, errPipe := os.Pipe()
	require.NoError(t, errPipe)

	os.Stdout = w
	pterm.SetDefaultOutput(w)
	pterm.DisableColor()

	t.Cleanup(func() {
		pterm.EnableColor()
		pterm.SetDefaultOutput(oldStdout)
		os.Stdout = oldStdout
		if w != nil {
			_ = w.Close()
		}
		if r != nil {
			_ = r.Close()
		}
	})

	root := cmd.GetRootCmd()
	root.SetArgs([]string{"world", "query", "0x2222", "--endpoint", srv.URL})
	err := root.Execute()
	require.NoError(t, err)

	_ = w.Close()
	w = nil

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Smart Gate (0x2222)")
	assert.Contains(t, output, "Type")
	assert.Contains(t, output, "0xpkg::smart_gate::SmartGate")
	assert.Contains(t, output, "Status")
	assert.Contains(t, output, "ONLINE")
	assert.Contains(t, output, "Connected Node")
	assert.Contains(t, output, "0xnode123")
	assert.Contains(t, output, "Paired Gate ID")
	assert.Contains(t, output, "0xpaired_addr")
}

func TestWorldQuery_Turret(t *testing.T) {
	resetFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"address": "0x3333",
					"asMoveObject": map[string]interface{}{
						"contents": map[string]interface{}{
							"type": map[string]interface{}{"repr": "0xpkg::smart_turret::SmartTurret"},
							"json": map[string]interface{}{
								"status":       "ONLINE",
								"network_node": "0xtnode",
								"metadata":     "Turret Info",
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldStdout := os.Stdout
	r, w, errPipe := os.Pipe()
	require.NoError(t, errPipe)

	os.Stdout = w
	pterm.SetDefaultOutput(w)
	pterm.DisableColor()

	t.Cleanup(func() {
		pterm.EnableColor()
		pterm.SetDefaultOutput(oldStdout)
		os.Stdout = oldStdout
		if w != nil {
			_ = w.Close()
		}
		if r != nil {
			_ = r.Close()
		}
	})

	root := cmd.GetRootCmd()
	root.SetArgs([]string{"world", "query", "0x3333", "--endpoint", srv.URL})
	err := root.Execute()
	require.NoError(t, err)

	_ = w.Close()
	w = nil

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Smart Turret (0x3333)")
	assert.Contains(t, output, "Type")
	assert.Contains(t, output, "0xpkg::smart_turret::SmartTurret")
	assert.Contains(t, output, "Status")
	assert.Contains(t, output, "ONLINE")
	assert.Contains(t, output, "Connected Node")
	assert.Contains(t, output, "0xtnode")
	assert.Contains(t, output, "Metadata")
	assert.Contains(t, output, "Turret Info")
}

func TestWorldQuery_ComplexSSU(t *testing.T) {
	resetFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"address": "0x544a",
					"asMoveObject": map[string]interface{}{
						"contents": map[string]interface{}{
							"type": map[string]interface{}{
								"repr": "0xc88::storage_unit::StorageUnit",
							},
							"json": map[string]interface{}{
								"status": map[string]interface{}{
									"status": map[string]interface{}{
										"@variant": "ONLINE",
									},
								},
								"metadata": map[string]interface{}{
									"name":        "Heavy Storage",
									"description": "Premium SSU",
								},
								"owner_cap_id":     "0xowner_cap",
								"energy_source_id": "0xenergy_complex",
							},
						},
					},
					"dynamicFields": map[string]interface{}{
						"nodes": []interface{}{
							map[string]interface{}{
								"name": map[string]interface{}{"json": "Inventory A"},
								"value": map[string]interface{}{
									"json": map[string]interface{}{
										"items": map[string]interface{}{
											"contents": []interface{}{
												map[string]interface{}{
													"value": map[string]interface{}{
														"type_id":  "446",
														"quantity": 10,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldStdout := os.Stdout
	r, w, errPipe := os.Pipe()
	require.NoError(t, errPipe)

	os.Stdout = w
	pterm.SetDefaultOutput(w)
	pterm.DisableColor()

	t.Cleanup(func() {
		pterm.EnableColor()
		pterm.SetDefaultOutput(oldStdout)
		os.Stdout = oldStdout
		if w != nil {
			_ = w.Close()
		}
		if r != nil {
			_ = r.Close()
		}
	})

	root := cmd.GetRootCmd()
	root.SetArgs([]string{"world", "query", "0x544a", "--endpoint", srv.URL})
	err := root.Execute()
	require.NoError(t, err)

	_ = w.Close()
	w = nil

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Smart Storage Unit (0x544a)")
	assert.Contains(t, output, "Type")
	assert.Contains(t, output, "0xc88::storage_unit::StorageUnit")
	assert.Contains(t, output, "Status")
	assert.Contains(t, output, "ONLINE")
	assert.Contains(t, output, "Owner")
	assert.Contains(t, output, "0xowner_cap")
	assert.Contains(t, output, "Connected Node")
	assert.Contains(t, output, "0xenergy_complex")
	assert.Contains(t, output, "Metadata")
	assert.Contains(t, output, "name")
	assert.Contains(t, output, "Heavy Storage")
	assert.Contains(t, output, "description")
	assert.Contains(t, output, "Premium SSU")
	assert.Contains(t, output, "Item: 10x Type 446")
}

func TestWorldQuery_NetworkNode(t *testing.T) {
	resetFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"address": "0x4444",
					"asMoveObject": map[string]interface{}{
						"contents": map[string]interface{}{
							"type": map[string]interface{}{
								"repr": "0xworld::network_node::NetworkNode",
							},
							"json": map[string]interface{}{
								"status": map[string]interface{}{
									"status": map[string]interface{}{
										"@variant": "ONLINE",
									},
								},
								"owner_cap_id": "0xcap",
								"fuel": map[string]interface{}{
									"quantity":     "100",
									"max_capacity": "1000",
									"is_burning":   true,
								},
								"energy_source": map[string]interface{}{
									"current_energy_production": "500",
									"max_energy_production":     "1000",
									"total_reserved_energy":     "200",
								},
								"connected_assembly_ids": []interface{}{"0x1111", "0x2222"},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldStdout := os.Stdout
	r, w, errPipe := os.Pipe()
	require.NoError(t, errPipe)

	os.Stdout = w
	pterm.SetDefaultOutput(w)
	pterm.DisableColor()

	t.Cleanup(func() {
		pterm.EnableColor()
		pterm.SetDefaultOutput(oldStdout)
		os.Stdout = oldStdout
		if w != nil {
			_ = w.Close()
		}
		if r != nil {
			_ = r.Close()
		}
	})

	root := cmd.GetRootCmd()
	root.SetArgs([]string{"world", "query", "0x4444", "--endpoint", srv.URL})
	err := root.Execute()
	require.NoError(t, err)

	_ = w.Close()
	w = nil

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Network Node (0x4444)")
	assert.Contains(t, output, "Type")
	assert.Contains(t, output, "0xworld::network_node::NetworkNode")
	assert.Contains(t, output, "Status")
	assert.Contains(t, output, "ONLINE")
	assert.Contains(t, output, "Fuel")
	assert.Contains(t, output, "100 / 1000 (Burning: Yes)")
	assert.Contains(t, output, "Energy Production")
	assert.Contains(t, output, "500 / 1000 (Reserved: 200)")
	assert.Contains(t, output, "Connected Assemblies")
	assert.Contains(t, output, "2")
	assert.Contains(t, output, "0x1111")
	assert.Contains(t, output, "0x2222")
}

func TestWorldQuery_OwnerCap(t *testing.T) {
	resetFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"address": "0x000000000000000000000000000000000000000000000000000000000000000c",
					"asMoveObject": map[string]interface{}{
						"contents": map[string]interface{}{
							"type": map[string]interface{}{
								"repr": "0xworld::access::OwnerCap<0xworld::network_node::NetworkNode>",
							},
							"json": map[string]interface{}{
								"authorized_object_id": "0x4444",
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldStdout := os.Stdout
	r, w, errPipe := os.Pipe()
	require.NoError(t, errPipe)

	os.Stdout = w
	pterm.SetDefaultOutput(w)
	pterm.DisableColor()

	t.Cleanup(func() {
		pterm.EnableColor()
		pterm.SetDefaultOutput(oldStdout)
		os.Stdout = oldStdout
		if w != nil {
			_ = w.Close()
		}
		if r != nil {
			_ = r.Close()
		}
	})

	root := cmd.GetRootCmd()
	root.SetArgs([]string{"world", "query", "0x000000000000000000000000000000000000000000000000000000000000000c", "--endpoint", srv.URL})
	err := root.Execute()
	require.NoError(t, err)

	_ = w.Close()
	w = nil

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Owner Capability (0x000000000000000000000000000000000000000000000000000000000000000c)")
	assert.Contains(t, output, "Type")
	assert.Contains(t, output, "0xworld::access::OwnerCap<0xworld::network_node::NetworkNode>")
	assert.Contains(t, output, "Authorized Object")
	assert.Contains(t, output, "0x4444")
}

func TestWorldQuery_Character(t *testing.T) {
	resetFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"address": "0x000000000000000000000000000000000000000000000000000000000000000d",
					"asMoveObject": map[string]interface{}{
						"contents": map[string]interface{}{
							"type": map[string]interface{}{
								"repr": "0xworld::character::Character",
							},
							"json": map[string]interface{}{
								"tribe_id":          1,
								"character_address": "0xchar_addr",
								"owner_cap_id":      "0xowner_cap",
								"key": map[string]interface{}{
									"tenant": "dev",
								},
								"metadata": map[string]interface{}{
									"name": "Ace Pilot",
								},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldStdout := os.Stdout
	r, w, errPipe := os.Pipe()
	require.NoError(t, errPipe)

	os.Stdout = w
	pterm.SetDefaultOutput(w)
	pterm.DisableColor()

	t.Cleanup(func() {
		pterm.EnableColor()
		pterm.SetDefaultOutput(oldStdout)
		os.Stdout = oldStdout
		if w != nil {
			_ = w.Close()
		}
		if r != nil {
			_ = r.Close()
		}
	})

	root := cmd.GetRootCmd()
	root.SetArgs([]string{"world", "query", "0x000000000000000000000000000000000000000000000000000000000000000d", "--endpoint", srv.URL})
	err := root.Execute()
	require.NoError(t, err)

	_ = w.Close()
	w = nil

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Character (0x000000000000000000000000000000000000000000000000000000000000000d)")
	assert.Contains(t, output, "Type")
	assert.Contains(t, output, "0xworld::character::Character")
	assert.Contains(t, output, "Tribe ID")
	assert.Contains(t, output, "1")
	assert.Contains(t, output, "Character Address")
	assert.Contains(t, output, "0xchar_addr")
	assert.Contains(t, output, "Tenant")
	assert.Contains(t, output, "dev")
	assert.Contains(t, output, "name")
	assert.Contains(t, output, "Ace Pilot")
}

func TestWorldQuery_NetworkFlag(t *testing.T) {
	resetFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"address": "0x1111",
					"asMoveObject": map[string]interface{}{
						"contents": map[string]interface{}{
							"type": map[string]interface{}{"repr": "0xpkg::smart_storage_unit::SmartStorageUnit"},
							"json": map[string]interface{}{},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	// Override devnet for testing
	oldDevnet := cmd.NetworkEndpoints["devnet"]
	cmd.NetworkEndpoints["devnet"] = srv.URL
	defer func() { cmd.NetworkEndpoints["devnet"] = oldDevnet }()

	root := cmd.GetRootCmd()

	// Test devnet
	root.SetArgs([]string{"world", "query", "0x1111", "--network", "devnet"})

	// We capture output
	oldStdout := os.Stdout
	r, w, errPipe := os.Pipe()
	require.NoError(t, errPipe)

	os.Stdout = w
	pterm.SetDefaultOutput(w)
	pterm.DisableColor()

	t.Cleanup(func() {
		pterm.EnableColor()
		pterm.SetDefaultOutput(oldStdout)
		os.Stdout = oldStdout
		if w != nil {
			_ = w.Close()
		}
		if r != nil {
			_ = r.Close()
		}
	})

	err := root.Execute()
	require.NoError(t, err)

	_ = w.Close()
	w = nil

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, fmt.Sprintf("Querying world object 0x1111 (devnet) at %s...", srv.URL))
}

func TestWorldQuery_GovernorCap(t *testing.T) {
	resetFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"address": "0x5555",
					"asMoveObject": map[string]interface{}{
						"contents": map[string]interface{}{
							"type": map[string]interface{}{"repr": "0xc88::world::GovernorCap"},
							"json": map[string]interface{}{},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldStdout := os.Stdout
	r, w, errPipe := os.Pipe()
	require.NoError(t, errPipe)

	os.Stdout = w
	pterm.SetDefaultOutput(w)
	pterm.DisableColor()

	t.Cleanup(func() {
		pterm.EnableColor()
		pterm.SetDefaultOutput(oldStdout)
		os.Stdout = oldStdout
		if w != nil {
			_ = w.Close()
		}
		if r != nil {
			_ = r.Close()
		}
	})

	root := cmd.GetRootCmd()
	root.SetArgs([]string{"world", "query", "0x5555", "--endpoint", srv.URL})
	err := root.Execute()
	require.NoError(t, err)

	_ = w.Close()
	w = nil

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	output := buf.String()

	// "GovernorCap" should be derived as "Governor Cap"
	assert.Contains(t, output, "Governor Cap (0x5555)")
	assert.Contains(t, output, "Type")
	assert.Contains(t, output, "0xc88::world::GovernorCap")
}

func TestWorldQuery_AdminACL(t *testing.T) {
	resetFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"address": "0x7b6e",
					"asMoveObject": map[string]interface{}{
						"contents": map[string]interface{}{
							"type": map[string]interface{}{"repr": "0xc88::access::AdminACL"},
							"json": map[string]interface{}{},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldStdout := os.Stdout
	r, w, errPipe := os.Pipe()
	require.NoError(t, errPipe)

	os.Stdout = w
	pterm.SetDefaultOutput(w)
	pterm.DisableColor()

	t.Cleanup(func() {
		pterm.EnableColor()
		pterm.SetDefaultOutput(oldStdout)
		os.Stdout = oldStdout
		if w != nil {
			_ = w.Close()
		}
		if r != nil {
			_ = r.Close()
		}
	})

	root := cmd.GetRootCmd()
	root.SetArgs([]string{"world", "query", "0x7b6e", "--endpoint", srv.URL})
	err := root.Execute()
	require.NoError(t, err)

	_ = w.Close()
	w = nil

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	output := buf.String()

	// "AdminACL" should be overridden as "Admin Access Control List"
	assert.Contains(t, output, "Admin Access Control List (0x7b6e)")
}


