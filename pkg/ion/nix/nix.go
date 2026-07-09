// Package nix adapts gonix operations for ION package resolution and
// realization.
package nix

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sund3RRR/gonix"
	gonixflake "github.com/sund3RRR/gonix/flake"
)

// ErrPkgNotFound is returned when a package attribute cannot be found in a
// flake's package outputs for the active system.
var ErrPkgNotFound = errors.New("package not found")

// Config contains gonix client settings used by Nix.
type Config struct {
	// Cores is the number of build cores to make available to Nix.
	Cores int
	// MaxJobs is the maximum number of concurrent Nix build jobs.
	MaxJobs int
	// Verbosity controls gonix logging detail.
	Verbosity gonix.Verbosity
	// LogFormat controls the format used for gonix logs.
	LogFormat gonix.LogFormat
	// LogSinkPath is the path gonix should write logs to, when configured.
	LogSinkPath string
	// StoreURI is the Nix store URI used by the gonix client.
	StoreURI string
}

// Nix adapts gonix source and package operations for ION.
type Nix struct {
	client *gonix.Client
	system string
}

// New creates a Nix adapter with an owned gonix client.
func New(config Config) (*Nix, error) {
	client, err := gonix.NewClient(gonix.ClientConfig{
		Cores:       config.Cores,
		MaxJobs:     config.MaxJobs,
		Verbosity:   config.Verbosity,
		LogFormat:   config.LogFormat,
		LogSinkPath: config.LogSinkPath,
		Store: gonix.StoreConfig{
			URI: config.StoreURI,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("nix: create gonix client: %w", err)
	}

	return &Nix{
		client: client,
		system: gonix.DefaultSystem(),
	}, nil
}

// System returns the Nix system used for package resolution and realization.
func (n *Nix) System() string {
	return n.system
}

// Close releases resources owned by n.
func (n *Nix) Close() error {
	return n.client.Close()
}

// OpenFlake resolves a source flake lock in memory without writing flake.lock.
func (n *Nix) OpenFlake(ctx context.Context, ref string) (*gonixflake.Flake, error) {
	flake, err := n.client.OpenFlake(ref, gonixflake.WithLockMode(gonixflake.LockModeVirtual))
	if err != nil {
		return nil, fmt.Errorf("nix: open flake: %w", err)
	}

	return flake, nil
}

// OpenLockedFlake opens ref using the supplied flake lock information.
func (n *Nix) OpenLockedFlake(ctx context.Context, ref string, lockInfo *gonixflake.LockInfo) (*gonixflake.Flake, error) {
	flake, err := n.client.OpenFlake(ref,
		gonixflake.WithLockMode(gonixflake.LockModeCheck),
		gonixflake.WithReferenceLockInfo(*lockInfo),
	)
	if err != nil {
		return nil, fmt.Errorf("nix: open flake: %w", err)
	}

	return flake, nil
}

// PullPackageRequest describes a package derivation output to realize.
type PullPackageRequest struct {
	// Attr is the package attribute to resolve.
	Attr string
	// System is the Nix system to resolve for.
	System string
	// OutputName is the derivation output name to realize.
	OutputName string
}

// RealizeOutput realizes outputName from drvPath.
func (n *Nix) RealizeOutput(ctx context.Context, drvPath string, outputName string) (*RealizedOutput, error) {
	realized, err := n.client.RealizeOutput(ctx, drvPath, outputName)
	if err != nil {
		return nil, fmt.Errorf("nix: realize derivation: %w", err)
	}

	return &RealizedOutput{
		OutputName: realized.OutputName,
		StorePath:  realized.StorePath,
		RealPath:   realized.RealPath,
		DrvPath:    drvPath,
		Name:       realized.Name,
		Hash:       realized.Hash,
	}, nil
}

type Derivation struct {
	Path    string   `nix:"drvPath" validate:"required"`
	Outputs []string `nix:"outputs"`
}

// CallDerivation evaluates expr as a Nix function and applies it to args,
// passing each value as a Nix string, then returns the resulting
// derivation's store path and declared output names.
//
// Arguments are passed as real Nix values through the gonix evaluator
// (eval.String, eval.Attrs) rather than substituted into expr as text, so
// arbitrary Go string values need no Nix-syntax escaping.
func (n *Nix) CallDerivation(
	ctx context.Context,
	expr string,
	args any,
) (Derivation, error) {
	var value Derivation
	if err := n.client.EvalWithArgs(ctx, expr, args, &value); err != nil {
		return Derivation{}, fmt.Errorf("nix: evaluate derivation function: %w", err)
	}
	if value.Path == "" {
		return Derivation{}, errors.New("nix: evaluated derivation has empty drv path")
	}

	return value, nil
}

func (n *Nix) ResolvePackage(ctx context.Context, flake *gonixflake.Flake, attr string, system string) (*ResolvedPackage, error) {
	pkg, err := n.resolvePackagePath(ctx, flake, attr, legacyAttrPath(attr, system), system)
	if err == nil {
		return pkg, nil
	}

	if !isMissingAttributeError(err) {
		return nil, fmt.Errorf("nix: resolve package path: %w", err)
	}

	pkg, err = n.resolvePackagePath(ctx, flake, attr, packageAttrPath(attr, system), system)
	if err == nil {
		return pkg, nil
	}

	if !isMissingAttributeError(err) {
		return nil, fmt.Errorf("nix: resolve package path: %w", err)
	}

	return nil, ErrPkgNotFound
}

func (n *Nix) resolvePackagePath(
	ctx context.Context,
	flake *gonixflake.Flake,
	attr string,
	path []string,
	system string,
) (*ResolvedPackage, error) {
	type resolvedValues struct {
		Name    string   `nix:"name"`
		Version string   `nix:"version"`
		DrvPath string   `nix:"drvPath" validate:"required"`
		Outputs []string `nix:"outputs"`
	}

	var value resolvedValues
	if err := n.client.EvalFlakeOutput(ctx, flake, path, &value); err != nil {
		return nil, fmt.Errorf("nix: evaluate package path %q: %w", strings.Join(path, "."), err)
	}
	if value.DrvPath == "" {
		return nil, errors.New("nix: resolved package has empty drv path")
	}

	return &ResolvedPackage{
		Attr:        attr,
		AttrPath:    append([]string(nil), path...),
		System:      system,
		Name:        value.Name,
		Version:     value.Version,
		DrvPath:     value.DrvPath,
		OutputNames: value.Outputs,
	}, nil
}

func legacyAttrPath(attr string, system string) []string {
	segments := strings.Split(attr, ".")
	return append([]string{"legacyPackages", system}, segments...)
}

func packageAttrPath(attr string, system string) []string {
	segments := strings.Split(attr, ".")
	return append([]string{"packages", system}, segments...)
}

func isMissingAttributeError(err error) bool {
	var nixErr *gonix.Error
	if errors.As(err, &nixErr) && nixErr.Code == gonix.ErrorCodeKey && nixErr.Message == "missing attribute" {
		return true
	}
	return false
}
