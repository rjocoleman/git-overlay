package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSymlinkSpecUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected SymlinkSpec
		wantErr  bool
	}{
		{
			name:  "string form",
			input: `"app"`,
			expected: SymlinkSpec{
				From:   "app",
				To:     "app",
				String: "app",
			},
		},
		{
			name: "struct form",
			input: `
from: src/lib
to: library`,
			expected: SymlinkSpec{
				From: "src/lib",
				To:   "library",
			},
		},
		{
			name:    "invalid yaml",
			input:   `{`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got SymlinkSpec
			err := yaml.Unmarshal([]byte(tt.input), &got)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if got.From != tt.expected.From {
				t.Errorf("From = %v, want %v", got.From, tt.expected.From)
			}
			if got.To != tt.expected.To {
				t.Errorf("To = %v, want %v", got.To, tt.expected.To)
			}
			if got.String != tt.expected.String {
				t.Errorf("String = %v, want %v", got.String, tt.expected.String)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		errField string
	}{
		{
			name: "valid config",
			input: `
upstream:
  url: "https://github.com/example/repo.git"
  ref: "main"
symlinks:
  - app
  - config`,
			wantErr: false,
		},
		{
			name: "missing url",
			input: `
upstream:
  ref: "main"
symlinks:
  - app`,
			wantErr:  true,
			errField: "upstream.url",
		},
		{
			name: "missing ref",
			input: `
upstream:
  url: "https://github.com/example/repo.git"
symlinks:
  - app`,
			wantErr:  true,
			errField: "upstream.ref",
		},
		{
			name: "valid complex symlinks",
			input: `
upstream:
  url: "https://github.com/example/repo.git"
  ref: "main"
symlinks:
  - app
  - from: src/lib
    to: library`,
			wantErr: false,
		},
		{
			name: "valid with link mode",
			input: `
upstream:
  url: "https://github.com/example/repo.git"
  ref: "main"
symlinks:
  - app
link_mode: "hardlink"`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			err := yaml.Unmarshal([]byte(tt.input), &cfg)
			if err != nil {
				t.Fatalf("Failed to unmarshal config: %v", err)
			}

			// Validate required fields
			var validationErr error
			if cfg.Upstream.URL == "" {
				validationErr = ErrMissingURL
			} else if cfg.Upstream.Ref == "" {
				validationErr = ErrMissingRef
			}

			if (validationErr != nil) != tt.wantErr {
				t.Errorf("Validation error = %v, wantErr %v", validationErr, tt.wantErr)
			}
		})
	}
}
