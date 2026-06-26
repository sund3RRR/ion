package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	embeddedconfig "github.com/sund3RRR/ion/config"
)

func TestLoadDefaultConfig(t *testing.T) {
	resetConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	assertEqual(t, cfg.Profiles.System.Dir, "/var/lib/ion/profiles/system")
	assertEqual(t, cfg.Profiles.User.Dir, "$XDG_STATE_HOME/.ion/profiles/$USER")
	assertEqual(t, cfg.Daemon.Socket.Path, "/run/ion/iond.sock")
}

func TestLoadEnvOverrides(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("ION_PROFILES_SYSTEM_DIR", "/env/system")
	t.Setenv("ION_DAEMON_SOCKET_PATH", "/env/iond.sock")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	assertEqual(t, cfg.Profiles.System.Dir, "/env/system")
	assertEqual(t, cfg.Daemon.Socket.Path, "/env/iond.sock")
}

func TestLoadConfigPathOverride(t *testing.T) {
	resetConfigEnv(t)

	path := filepath.Join(t.TempDir(), "override.yaml")
	writeConfig(t, path, `
profiles:
  system:
    dir: "/override/system"
  user:
    dir: "/override/user"
daemon:
  socket:
    path: "/override/socket"
user:
  name: "override-user"
  group: "override-group"
`)

	t.Setenv("ION_CONFIG_PATH", path)
	t.Setenv("ION_PROFILES_SYSTEM_DIR", "/env/system")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	assertEqual(t, cfg.Profiles.System.Dir, "/override/system")
	assertEqual(t, cfg.Daemon.Socket.Path, "/override/socket")
	assertEqual(t, cfg.User.Name, "override-user")
}

func TestLoadConfigPathOverrideMissingFile(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("ION_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.yaml"))

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want missing override config error")
	}
}

func TestWriteDefault(t *testing.T) {
	resetConfigEnv(t)

	path := filepath.Join(t.TempDir(), "ion", "config.yaml")
	if err := WriteDefault(path); err != nil {
		t.Fatalf("WriteDefault() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written default config: %v", err)
	}
	if string(data) != string(embeddedconfig.DefaultYAML) {
		t.Fatal("written default config does not match embedded default config")
	}

	t.Setenv("ION_CONFIG_PATH", path)
	if _, err := Load(); err != nil {
		t.Fatalf("Load() written default config error = %v", err)
	}
}

func resetConfigEnv(t *testing.T) {
	t.Helper()

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg-config"))

	for _, item := range os.Environ() {
		key, _, ok := strings.Cut(item, "=")
		if ok && strings.HasPrefix(key, envPrefix) {
			t.Setenv(key, "")
		}
	}
}

func writeConfig(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create config dir %q: %v", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config %q: %v", path, err)
	}
}

func assertEqual(t *testing.T, got, want string) {
	t.Helper()

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
