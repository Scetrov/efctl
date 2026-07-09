package blake2b256

import (
	"encoding/hex"
	"testing"
)

type testVector struct {
	name     string
	inputHex string
	want     string
}

func TestSum256KnownVectors(t *testing.T) {
	// Expected digests were generated independently with Python's hashlib.blake2b
	// using digest_size=32, matching unkeyed BLAKE2b-256.
	tests := []testVector{
		{name: "empty", inputHex: "", want: "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8"},
		{name: "abc", inputHex: "616263", want: "bddd813c634239723171ef3fee98579b94964e3bb1cb3e427262c8c068d52319"},
		{name: "quick-brown-fox", inputHex: "54686520717569636b2062726f776e20666f78206a756d7073206f76657220746865206c617a7920646f67", want: "01718cec35cd3d796dd00020e0bfecb473ad23457d063b75eff29c0ffa2e58a9"},
		{name: "sui-address-message", inputHex: "00000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", want: "0ddaaec3ffac93977c83c3d7440e9e65663850d4861be2f48532548d0a463336"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := hex.DecodeString(tt.inputHex)
			if err != nil {
				t.Fatalf("decode input hex: %v", err)
			}

			got := Sum256(input)
			if hex.EncodeToString(got[:]) != tt.want {
				t.Fatalf("Sum256(%q) = %x, want %s", input, got, tt.want)
			}
		})
	}
}

func TestSum256BlockBoundaryVectors(t *testing.T) {
	// These cases exercise sizes around the 128-byte BLAKE2b block size. Inputs
	// are deterministic byte sequences where input[i] = byte(i).
	tests := []struct {
		length int
		want   string
	}{
		{length: 1, want: "03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314"},
		{length: 2, want: "01cf79da4945c370c68b265ef70641aaa65eaa8f5953e3900d97724c2c5aa095"},
		{length: 3, want: "3d8c3d594928271f44aad7a04b177154806867bcf918e1549c0bc16f9da2b09b"},
		{length: 31, want: "9adf65b53153b1caec84cd717e00e01c2000d0569704ce38d065180adee5d964"},
		{length: 32, want: "cb2f5160fc1f7e05a55ef49d340b48da2e5a78099d53393351cd579dd42503d6"},
		{length: 33, want: "b7634fe13c7aca3914ee896e22cfabc9da5b4f13e72a2ccbecb6d44bbda95bcc"},
		{length: 63, want: "29e41a64fbdd2fd27612228623c0702222bf367451e7324287f181cb3dcf7237"},
		{length: 64, want: "10d8e6d534b00939843fe9dcc4dae48cdf008f6b8b2b82b156f5404d874887f5"},
		{length: 65, want: "84c04ab082c8ae24206561f77397704b627892089a05887a2a1996472bcfe15d"},
		{length: 127, want: "f2fe67ff342e21b8f45e8f2e0bcd1d9243245d50ee6c78042e9c491388791c72"},
		{length: 128, want: "c3582f71ebb2be66fa5dd750f80baae97554f3b015663c8be377cfcb2488c1d1"},
		{length: 129, want: "f7f3c46ba2564ff4c4c162da1f5b605f9f1c4aa6a20652a9f9a337c1a2f5b9c9"},
		{length: 255, want: "1d0850ee9bca0abc9601e9deabe1418fedec2fb6ac4150bd5302d2430f9be943"},
		{length: 256, want: "39a7eb9fedc19aabc83425c6755dd90e6f9d0c804964a1f4aaeea3b9fb599835"},
		{length: 257, want: "45f7f084c30bac7cbae2e1963bc6e6b0d8cb227a12927e97fb941d288fb1f9a3"},
		{length: 1024, want: "f1551feeb252c7e60bb362205bd1ac2f70b145260a91d41e8c5d0a187549a5f2"},
	}

	for _, tt := range tests {
		t.Run(tt.want[:8], func(t *testing.T) {
			input := makeSequentialBytes(tt.length)
			got := Sum256(input)
			if hex.EncodeToString(got[:]) != tt.want {
				t.Fatalf("Sum256(%d sequential bytes) = %x, want %s", tt.length, got, tt.want)
			}
		})
	}
}

func makeSequentialBytes(length int) []byte {
	input := make([]byte, length)
	for i := range input {
		input[i] = byte(i)
	}
	return input
}
