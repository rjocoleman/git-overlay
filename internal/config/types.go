package config

import "errors"

var (
	// ErrMissingURL is returned when the upstream URL is not provided
	ErrMissingURL = errors.New("upstream.url is required")
	// ErrMissingRef is returned when the upstream ref is not provided
	ErrMissingRef = errors.New("upstream.ref is required")
)

// Config represents the root configuration structure
type Config struct {
	Upstream  UpstreamConfig `yaml:"upstream"`
	Symlinks  []SymlinkSpec  `yaml:"symlinks"`
	LinkMode  string         `yaml:"link_mode,omitempty"`
	DebugMode bool          `yaml:"debug,omitempty"`
}

// UpstreamConfig holds upstream repository configuration
type UpstreamConfig struct {
	URL string `yaml:"url"`
	Ref string `yaml:"ref"`
}

// SymlinkSpec defines a symlink mapping
type SymlinkSpec struct {
	From string `yaml:"from,omitempty"`
	To   string `yaml:"to,omitempty"`
	// If string form is used, both From and To will be the same
	String string `yaml:"-"`
}

// UnmarshalYAML implements custom YAML unmarshaling
func (s *SymlinkSpec) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try string form first
	var str string
	if err := unmarshal(&str); err == nil {
		s.From = str
		s.To = str
		s.String = str
		return nil
	}

	// Fall back to struct form
	type alias SymlinkSpec
	var v alias
	if err := unmarshal(&v); err != nil {
		return err
	}
	*s = SymlinkSpec(v)
	return nil
}
