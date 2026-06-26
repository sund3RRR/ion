package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	embeddedconfig "github.com/sund3RRR/ion/config"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

const (
	delim             = "."
	defaultSystemPath = "/etc/ion/config.yaml"
	envPrefix         = "ION_"
	configPathEnv     = "ION_CONFIG_PATH"
)

// Config is the typed ION configuration loaded from YAML files and ION_* env vars.
type Config struct {
	Profiles ProfilesConfig `koanf:"profiles" yaml:"profiles"`
	Daemon   DaemonConfig   `koanf:"daemon" yaml:"daemon"`
	User     UserConfig     `koanf:"user" yaml:"user"`
}

type ProfilesConfig struct {
	System ProfileConfig `koanf:"system" yaml:"system"`
	User   ProfileConfig `koanf:"user" yaml:"user"`
}

type ProfileConfig struct {
	Dir string `koanf:"dir" yaml:"dir"`
}

type DaemonConfig struct {
	Socket SocketConfig `koanf:"socket" yaml:"socket"`
}

type SocketConfig struct {
	Path string `koanf:"path" yaml:"path"`
}

type UserConfig struct {
	Name  string `koanf:"name" yaml:"name"`
	Group string `koanf:"group" yaml:"group"`
}

// Load reads the ION config from defaults, system config, user config, and ION_* env.
func Load() (Config, error) {
	if configPath := os.Getenv(configPathEnv); configPath != "" {
		return loadSingleConfig(configPath)
	}

	userPath, err := defaultUserConfigPath()
	if err != nil {
		return Config{}, err
	}

	k := koanf.New(delim)
	parser := yaml.Parser()

	if err := k.Load(rawbytes.Provider(embeddedconfig.DefaultYAML), parser); err != nil {
		return Config{}, fmt.Errorf("load default config: %w", err)
	}

	if err := loadOptionalFile(k, defaultSystemPath, parser); err != nil {
		return Config{}, err
	}

	if err := loadOptionalFile(k, userPath, parser); err != nil {
		return Config{}, err
	}

	if err := k.Load(env.Provider(delim, env.Opt{
		Prefix:        envPrefix,
		TransformFunc: transformEnvKey,
	}), nil); err != nil {
		return Config{}, fmt.Errorf("load environment config: %w", err)
	}

	var cfg Config
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

// WriteDefault writes the built-in default configuration to path.
func WriteDefault(path string) error {
	if path == "" {
		return fmt.Errorf("write default config: empty path")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory %q: %w", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, embeddedconfig.DefaultYAML, 0o644); err != nil {
		return fmt.Errorf("write default config %q: %w", path, err)
	}

	return nil
}

func loadSingleConfig(path string) (Config, error) {
	k := koanf.New(delim)
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return Config{}, fmt.Errorf("load config file %q: %w", path, err)
	}

	var cfg Config
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

func loadOptionalFile(k *koanf.Koanf, path string, parser koanf.Parser) error {
	if path == "" {
		return nil
	}

	if err := k.Load(file.Provider(path), parser); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("load config file %q: %w", path, err)
	}

	return nil
}

func defaultUserConfigPath() (string, error) {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "ion", "config.yaml"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config path: %w", err)
	}

	return filepath.Join(home, ".config", "ion", "config.yaml"), nil
}

func transformEnvKey(key, value string) (string, any) {
	if key == configPathEnv {
		return "", nil
	}

	key = strings.TrimPrefix(key, envPrefix)
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "_", delim)

	return key, value
}
