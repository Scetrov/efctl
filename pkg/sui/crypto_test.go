package sui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveAddressFromPrivateKey_AdminKey(t *testing.T) {
	// Known test vector from a real deployment:
	// Admin private key → admin address
	addr, err := DeriveAddressFromPrivateKey("suiprivkey1qzgv6g33hpr66xkvu94lff8l3smw9ggq8w54rvkse7cdxy0yjjsh7dxgser") // gitleaks:allow
	require.NoError(t, err)
	assert.Equal(t, "0x1cde4f2de0639971fbb9261591f4bbe8d100b695dddae5408e79df84ad2ba05a", addr)
}

func TestDeriveAddressFromPrivateKey_InvalidHRP(t *testing.T) {
	_, err := DeriveAddressFromPrivateKey("bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4")
	assert.Error(t, err)
}

func TestDeriveAddressFromPrivateKey_InvalidBech32(t *testing.T) {
	_, err := DeriveAddressFromPrivateKey("notabech32string")
	assert.Error(t, err)
}

func TestDeriveAddressFromPrivateKey_EmptyString(t *testing.T) {
	_, err := DeriveAddressFromPrivateKey("")
	assert.Error(t, err)
}

func TestDeriveAddressFromPrivateKey_Deterministic(t *testing.T) {
	key := "suiprivkey1qzgv6g33hpr66xkvu94lff8l3smw9ggq8w54rvkse7cdxy0yjjsh7dxgser" // gitleaks:allow
	addr1, err1 := DeriveAddressFromPrivateKey(key)
	addr2, err2 := DeriveAddressFromPrivateKey(key)
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, addr1, addr2, "derivation must be deterministic")
}
