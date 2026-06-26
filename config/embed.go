package config

import _ "embed"

// DefaultYAML is the built-in fallback configuration.
//
//go:embed config.yaml
var DefaultYAML []byte
