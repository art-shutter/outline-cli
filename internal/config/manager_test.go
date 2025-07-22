package config

import (
	"testing"
)

func TestParseDataSize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
		hasError bool
	}{
		{"empty string", "", 0, false},
		{"1 byte", "1B", 1, false},
		{"1 kilobyte", "1KB", 1000, false},
		{"1 megabyte", "1MB", 1000000, false},
		{"1 gigabyte", "1GB", 1000000000, false},
		{"1 terabyte", "1TB", 1000000000000, false},
		{"decimal values", "1.5GB", 1500000000, false},
		{"large decimal", "2.5TB", 2500000000000, false},
		{"with spaces", " 1GB ", 1000000000, false},
		{"lowercase", "1gb", 1000000000, false},
		{"mixed case", "1Gb", 1000000000, false},
		{"kibibyte", "1KiB", 1024, false},
		{"mebibyte", "1MiB", 1048576, false},
		{"gibibyte", "1GiB", 1073741824, false},
		{"tebibyte", "1TiB", 1099511627776, false},
		{"large number", "1000MB", 1000000000, false},
		{"zero", "0GB", 0, false},
		{"number without unit", "1000", 1000, false},
		{"with multiple spaces", "1  GB", 1000000000, false},
		{"decimal without unit", "1.5", 1, false},

		// Invalid inputs
		{"invalid format", "invalid", 0, true},
		{"unknown unit", "1ZB", 0, true},
		{"negative number", "-1GB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDataSize(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("ParseDataSize(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("ParseDataSize(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("ParseDataSize(%q) = %d, want %d", tt.input, result, tt.expected)
				}
			}
		})
	}
}
