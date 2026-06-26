# AGENTS.md

Guidance for agents working on the `ion` repository.

## Project Goal

ION is a Go package manager layer built on top
of Nix and `gonix`.

ION is responsible for:

- resolving packages from flakes;
- indexing and searching packages;
- managing user, system, and custom profiles;
- activating profile generations;
- exposing all core operations through a Unix socket daemon;
- running package injectors for anchors such as binaries, libraries, desktop
  files, services, and other package paths.

`gonix` is the intended gateway to Nix behavior. Do not shell out to `nix` for
core package-manager behavior unless it is explicitly a temporary diagnostic,
compatibility check, or documented fallback.

## Project Layout

- `cmd/ion`: executable entrypoint. Keep it thin: context setup, signals, and
  delegation to `internal/app/cli`.
- `internal/app/cli`: Kong CLI app, commands, runtime wiring, config loading,
  and CLI-specific presentation.
- `internal/app/daemon`: Unix socket daemon, HTTP server lifecycle, and
  ConnectRPC service implementations.
- `internal/config`: YAML/env config loading and default config writing.
- `internal/version`: build-time version values set by `-ldflags -X`.
- `api/proto`: protobuf service contracts. This is the source of truth for IPC.
- `pkg/ion`: public ION API and business-logic interfaces intended for other
  Go projects to import.
- `pkg/pb`: generated protobuf and ConnectRPC Go code. Do not edit by hand.
- `pkg/types`: public domain types and models.
- `pkg`: container for public importable ION packages.
- `config`: embedded default YAML config.
- `flake.nix`: Nix packages and dev shell.
- `Makefile`: project commands, usually executed through `nix develop`.

## Architecture

Keep dependency direction strict:

- `cmd/ion` may import `internal/app/cli`.
- `internal/app/*` may import `pkg/ion`, `pkg/types`, generated `pkg/pb`,
  config, and version packages.
- packages under `pkg/` must not import `internal`.
- `api/proto` drives generated `pkg/pb`; generated code should not be edited.

The CLI and daemon are adapters. Core behavior should live in `pkg/ion`, with
domain types under `pkg/types`.

The daemon uses ConnectRPC over `net/http` on a Unix socket. Keep service
handlers small: decode request, call core, encode response. Avoid putting
business rules directly in RPC handlers.

Kong commands should stay thin. Parse command input, resolve runtime/config,
then call core or daemon clients. Do not let CLI structs become the domain
model.

## Stack

- Language: Go 1.25.
- CLI: `github.com/alecthomas/kong`.
- Config: `koanf` with YAML files and `ION_*` environment overrides.
- IPC: ConnectRPC over Unix sockets.
- Serialization: protobuf.
- Codegen: `buf generate`.
- Nix integration: `gonix` and Nix C/C++ libraries from the development shell.
- Build/dev environment: flakes, `gonix.lib.${system}.mkDevShell`, and
  `buildGoModule`.
- Static Linux build: musl via Nix `pkgsStatic`.

## Protobuf Workflow

Edit `.proto` files under `api/proto` only.

Regenerate generated Go code with:

```sh
make generate
```

or, if already inside the dev shell:

```sh
buf generate
```

Generated code is written to `pkg/pb`. Do not manually edit files under
`pkg/pb`; change the proto contract and regenerate instead.

After proto changes, run:

```sh
buf generate
go test ./...
```

If generated output changes unexpectedly, inspect the diff before continuing.

## Config

Runtime config is loaded by `internal/config`.

Normal precedence:

1. embedded default config from `config/config.yaml`;
2. `/etc/ion/config.yaml`;
3. `$XDG_CONFIG_HOME/ion/config.yaml`, or `$HOME/.config/ion/config.yaml`;
4. `ION_*` environment variables.

`ION_CONFIG_PATH` is a strict override: when set, only that file is loaded. It
does not merge with defaults, system config, user config, or other `ION_*`
variables.

Use `config.WriteDefault(path)` when code needs to materialize the built-in
default YAML.

## Versioning

The source default version is `dev` in `internal/version`.

Release builds should set:

```sh
-X github.com/sund3RRR/ion/internal/version.Version=<version>
-X github.com/sund3RRR/ion/internal/version.Commit=<commit>
-X github.com/sund3RRR/ion/internal/version.Date=<utc-date>
```

The CLI exposes both:

```sh
ion --version
ion version
```

The daemon's `Version` RPC should return the same version string.

## Development Commands

Preferred commands from the repository root:

```sh
make generate
make test
make lint
make check
```

After changing code, always run:

```sh
make check
```

If `make check` cannot be run, state that explicitly in the final report and
explain why.

Build commands:

```sh
make build
make build-dynamic
make build-static
make build-dev
```

Clean local build artifacts:

```sh
make clean
```

Direct commands that are useful while iterating:

```sh
go test ./...
go run ./cmd/ion --help
go run ./cmd/ion --version
buf generate
```

Nix commands:

```sh
nix develop path:.
nix build path:.
nix build path:.#dynamic
nix build path:.#static
```

On Darwin, `packages.default` is the native dynamic package. On Linux,
`packages.default` is the static musl package.

## Testing

Add focused tests near the package being changed.

Use daemon tests for Unix socket lifecycle and RPC behavior. Prefer real
ConnectRPC clients over ad hoc handler calls when testing daemon IPC.

For CLI changes, check at least:

```sh
go run ./cmd/ion --help
go run ./cmd/ion <command> --help
```

For versioning changes, check both CLI entrypoints:

```sh
go run ./cmd/ion --version
go run ./cmd/ion version
```

If a Nix command cannot be run because local Nix is fetching, locked, or
blocked by the daemon, state that explicitly in the final report and still run
the closest direct Go or buf command.

## Style And Boundaries

Keep code small and explicit. Prefer typed structs and interfaces over global
state. Keep comments concise and useful.

## Go Documentation

All public Go entities must have concise godoc comments. After adding or
changing public API documentation, verify the rendered output with `go doc` for
the affected package or symbols.

Do not:

- edit generated protobuf files by hand;
- put business logic in `cmd/ion`;
- import `internal` packages from `pkg`;
- make CLI or RPC request structs the canonical domain model;
- shell out to Nix for core ION behavior when a `gonix` path exists;
- silently ignore daemon shutdown, socket, or config errors.

When adding public API under `pkg/ion`, use clear Go names and keep it
importable by external projects without pulling in CLI or daemon internals.
