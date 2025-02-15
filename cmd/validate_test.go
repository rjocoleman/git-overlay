package cmd

import "testing"

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		path      string
		wantError bool
	}{
		{
			name:      "valid simple path",
			base:      "overlay",
			path:      "test.txt",
			wantError: false,
		},
		{
			name:      "valid nested path",
			base:      "overlay",
			path:      "subdir/test.txt",
			wantError: false,
		},
		{
			name:      "valid dotfile",
			base:      "overlay",
			path:      ".config",
			wantError: false,
		},
		{
			name:      "escape attempt with parent directory",
			base:      "overlay",
			path:      "../test.txt",
			wantError: true,
		},
		{
			name:      "escape attempt with absolute path",
			base:      "overlay",
			path:      "/etc/passwd",
			wantError: true,
		},
		{
			name:      "escape attempt with parent in middle",
			base:      "overlay",
			path:      "subdir/../../../test.txt",
			wantError: true,
		},
		{
			name:      "valid path with dot segments",
			base:      "overlay",
			path:      "subdir/./test.txt",
			wantError: false,
		},
		{
			name:      "valid path with multiple slashes",
			base:      "overlay",
			path:      "subdir//test.txt",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.base, tt.path)
			if (err != nil) != tt.wantError {
				t.Errorf("validatePath() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
