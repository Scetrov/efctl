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

func TestLocalMissingCPUFeaturesLive(t *testing.T) {
	missing, known := LocalMissingCPUFeatures()
	t.Logf("Local CPU feature data available: %v", known)
	t.Logf("Missing CPU features: %v", missing)
	if !known {
		t.Skip("local CPU feature data unavailable in this environment")
	}
}
