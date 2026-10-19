package doctor

import (
	"reflect"
	"testing"
)

// Automated tests for CPU features
func TestMissingCPUFeatures(t *testing.T) {
	tests := []struct {
		name        string
		flags       Evidence
		wantMissing []string
		wantKnown   bool
	}{
		{
			name:        "both supported",
			flags:       Evidence{Source: "fixture", Value: "adx bmi2 avx sse"},
			wantMissing: nil,
			wantKnown:   true,
		},
		{
			name:        "ADX missing",
			flags:       Evidence{Source: "fixture", Value: "bmi2 avx sse"},
			wantMissing: []string{"ADX"},
			wantKnown:   true,
		},
		{
			name:        "BMI2 missing",
			flags:       Evidence{Source: "fixture", Value: "adx avx sse"},
			wantMissing: []string{"BMI2"},
			wantKnown:   true,
		},
		{
			name:        "both missing",
			flags:       Evidence{Source: "fixture", Value: "avx sse"},
			wantMissing: []string{"ADX", "BMI2"},
			wantKnown:   true,
		},
		{
			name:        "empty flags",
			flags:       Evidence{Source: "fixture", Value: ""},
			wantMissing: []string{"ADX", "BMI2"},
			wantKnown:   true,
		},
		{
			name:        "case insensitive",
			flags:       Evidence{Source: "fixture", Value: "ADX BMI2 AVX"},
			wantMissing: nil,
			wantKnown:   true,
		},
		{
			name:      "unavailable flags",
			flags:     Evidence{Source: "fixture", Reason: "instruction flags unavailable"},
			wantKnown: false,
		},
		{
			name:      "no source",
			flags:     Evidence{Value: "adx bmi2"},
			wantKnown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMissing, gotKnown := MissingCPUFeatures(tt.flags)

			if gotKnown != tt.wantKnown {
				t.Fatalf("known = %v, want %v", gotKnown, tt.wantKnown)
			}

			if !reflect.DeepEqual(gotMissing, tt.wantMissing) {
				t.Errorf("missing = %v, want %v", gotMissing, tt.wantMissing)
			}
		})
	}
}

func TestMissingSuiCPUFeaturesArchitecture(t *testing.T) {
	for _, arch := range []string{"amd64", "x86_64", "arm64", "aarch64", "", "386"} {
		t.Run(arch, func(t *testing.T) {
			cpu := CPUInfo{Architecture: Evidence{Source: "fixture", Value: arch}, Flags: Evidence{Source: "fixture", Value: "sse"}}
			missing, known := MissingSuiCPUFeatures(cpu)
			wantKnown := arch == "amd64" || arch == "x86_64"
			if known != wantKnown || (known && len(missing) != 2) {
				t.Fatalf("missing=%v known=%v for %s", missing, known, arch)
			}
			cpu.Architecture.Reason = "unavailable"
			if _, known := MissingSuiCPUFeatures(cpu); known {
				t.Fatal("unavailable architecture treated as confirmed x86")
			}
		})
	}
}
