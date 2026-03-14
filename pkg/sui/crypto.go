package sui

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/blake2b"
)

// DeriveAddressFromPrivateKey derives a Sui address from a bech32-encoded
// private key (suiprivkey1...) without shelling out to the sui CLI.
//
// Algorithm:
//  1. Bech32-decode the key → 33 bytes (1 flag byte + 32-byte Ed25519 seed)
//  2. Derive the Ed25519 public key from the seed
//  3. Hash (flag_byte || public_key) with BLAKE2b-256
//  4. Return the result as "0x" + hex
func DeriveAddressFromPrivateKey(privkey string) (string, error) {
	hrp, data, err := bech32Decode(privkey)
	if err != nil {
		return "", fmt.Errorf("bech32 decode: %w", err)
	}
	if hrp != "suiprivkey" {
		return "", fmt.Errorf("unexpected HRP %q, expected \"suiprivkey\"", hrp)
	}
	if len(data) != 33 {
		return "", fmt.Errorf("unexpected payload length %d, expected 33", len(data))
	}

	flag := data[0] // 0x00 = Ed25519
	seed := data[1:]

	pub := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)

	// Sui address = BLAKE2b-256( flag || public_key )
	msg := make([]byte, 0, 33)
	msg = append(msg, flag)
	msg = append(msg, pub...)
	hash := blake2b.Sum256(msg)

	return "0x" + hex.EncodeToString(hash[:]), nil
}

// ────────────────────────────────────────────────────────────────────
// Minimal Bech32 implementation (BIP-173).
// We inline this to avoid pulling in a full Bitcoin dependency.
// ────────────────────────────────────────────────────────────────────

const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

func bech32Decode(bech string) (string, []byte, error) {
	bech = strings.ToLower(bech)
	pos := strings.LastIndexByte(bech, '1')
	if pos < 1 || pos+7 > len(bech) {
		return "", nil, fmt.Errorf("invalid bech32 separator position")
	}
	hrp := bech[:pos]
	dataPart := bech[pos+1:]

	values := make([]int, len(dataPart))
	for i, c := range dataPart {
		idx := strings.IndexByte(bech32Charset, byte(c)) // #nosec G115
		if idx < 0 {
			return "", nil, fmt.Errorf("invalid bech32 character %q", c)
		}
		values[i] = idx
	}

	if !bech32VerifyChecksum(hrp, values) {
		return "", nil, fmt.Errorf("bech32 checksum mismatch")
	}

	// Strip the 6-character checksum
	conv, err := bech32ConvertBits(values[:len(values)-6], 5, 8, false)
	if err != nil {
		return "", nil, fmt.Errorf("bit conversion: %w", err)
	}

	return hrp, conv, nil
}

func bech32Polymod(values []int) int {
	gen := [5]int{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}
	chk := 1
	for _, v := range values {
		b := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ v
		for i := 0; i < 5; i++ {
			if (b>>uint(i))&1 == 1 {
				chk ^= gen[i]
			}
		}
	}
	return chk
}

func bech32HRPExpand(hrp string) []int {
	ret := make([]int, 0, len(hrp)*2+1)
	for _, c := range hrp {
		ret = append(ret, int(c>>5))
	}
	ret = append(ret, 0)
	for _, c := range hrp {
		ret = append(ret, int(c&31))
	}
	return ret
}

func bech32VerifyChecksum(hrp string, data []int) bool {
	vals := append(bech32HRPExpand(hrp), data...) //nolint:gocritic
	return bech32Polymod(vals) == 1
}

func bech32ConvertBits(data []int, fromBits, toBits uint, pad bool) ([]byte, error) {
	acc := 0
	bits := uint(0)
	maxV := (1 << toBits) - 1
	var ret []byte
	for _, v := range data {
		if v < 0 || v>>fromBits != 0 {
			return nil, fmt.Errorf("invalid data value %d", v)
		}
		acc = (acc << fromBits) | v
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			ret = append(ret, byte((acc>>bits)&maxV)) // #nosec G115
		}
	}
	if pad {
		if bits > 0 {
			ret = append(ret, byte((acc<<(toBits-bits))&maxV)) // #nosec G115
		}
	} else if bits >= fromBits {
		return nil, fmt.Errorf("excess padding")
	} else if (acc<<(toBits-bits))&maxV != 0 {
		return nil, fmt.Errorf("non-zero padding")
	}
	return ret, nil
}
