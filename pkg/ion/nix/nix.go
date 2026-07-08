// Package nix adapts gonix operations for ION package resolution and
// realization.
package nix

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sund3RRR/gonix"
	"github.com/sund3RRR/gonix/eval"
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

type PullPackageRequest struct {
	Attr       string
	System     string
	OutputName string
}

func (n *Nix) PullPackage(ctx context.Context, flake *gonixflake.Flake, req PullPackageRequest) (*RealizedPackage, error) {
	pkg, err := n.ResolvePackage(ctx, flake, req.Attr)
	if err != nil {
		return nil, fmt.Errorf("nix: resolve package: %w", err)
	}

	realized, err := n.client.RealizeOutput(ctx, pkg.DrvPath, req.OutputName)
	if err != nil {
		return nil, fmt.Errorf("nix: realize derivation: %w", err)
	}

	return &RealizedPackage{
		OutputName: realized.OutputName,
		StorePath:  realized.StorePath,
		RealPath:   realized.RealPath,
		Name:       realized.Name,
		Hash:       realized.Hash,
		Package:    *pkg,
	}, nil
}

func (n *Nix) RealizeOutput(ctx context.Context, drvPath string, outputName string) (gonix.RealizedOutput, error) {
	return n.client.RealizeOutput(ctx, drvPath, outputName)
}

// ResolvePackage resolves package metadata from a locked source.
func (n *Nix) ResolvePackage(ctx context.Context, flake *gonixflake.Flake, attr string) (*Package, error) {
	pkg, err := n.resolvePackagePath(ctx, flake, attr, legacyAttrPath(attr, n.system))
	if err == nil {
		return pkg, nil
	}

	if !isMissingAttributeError(err) {
		return nil, fmt.Errorf("nix: resolve package path: %w", err)
	}

	pkg, err = n.resolvePackagePath(ctx, flake, attr, packageAttrPath(attr, n.system))
	if err == nil {
		return pkg, nil
	}

	if !isMissingAttributeError(err) {
		return nil, fmt.Errorf("nix: resolve package path: %w", err)
	}

	return nil, ErrPkgNotFound
}

// System returns the Nix system used for package resolution and realization.
func (n *Nix) System() string {
	return n.system
}

// CallDerivation evaluates expr as a Nix function and applies it to args,
// passing each value as a Nix string, then returns the resulting
// derivation's store path and declared output names.
//
// Arguments are passed as real Nix values through the gonix evaluator
// (eval.String, eval.Attrs) rather than substituted into expr as text, so
// arbitrary Go string values need no Nix-syntax escaping.
func (n *Nix) CallDerivation(ctx context.Context, expr string, args map[string]string) (drvPath string, outputs []string, err error) {
	var value struct {
		DrvPath string   `nix:"drvPath" validate:"required"`
		Outputs []string `nix:"outputs"`
	}

	err = n.client.WithEvaluator(ctx, func(ev *eval.Evaluator) error {
		fn, err := ev.EvalString(expr, "<gonix>")
		if err != nil {
			return fmt.Errorf("nix: evaluate function: %w", err)
		}
		defer fn.Close() //nolint:errcheck

		attrs := make(map[string]eval.GoValue, len(args))
		for name, v := range args {
			attrs[name] = eval.String(v)
		}

		argsValue, err := ev.NewValue(eval.Attrs(attrs))
		if err != nil {
			return fmt.Errorf("nix: build function arguments: %w", err)
		}
		defer argsValue.Close() //nolint:errcheck

		result, err := ev.Call(fn, argsValue)
		if err != nil {
			return fmt.Errorf("nix: call function: %w", err)
		}
		defer result.Close() //nolint:errcheck

		return ev.Unmarshal(result, &value)
	})
	if err != nil {
		return "", nil, fmt.Errorf("nix: evaluate derivation function: %w", err)
	}
	if value.DrvPath == "" {
		return "", nil, errors.New("nix: evaluated derivation has empty drv path")
	}

	return value.DrvPath, value.Outputs, nil
}

func (n *Nix) resolvePackagePath(
	ctx context.Context,
	flake *gonixflake.Flake,
	attr string,
	path []string,
) (*Package, error) {
	type resolvedPackage struct {
		Name    string   `nix:"name"`
		Version string   `nix:"version"`
		DrvPath string   `nix:"drvPath" validate:"required"`
		Outputs []string `nix:"outputs"`
	}

	var value resolvedPackage
	if err := n.client.EvalFlakeOutput(ctx, flake, path, &value); err != nil {
		return nil, fmt.Errorf("nix: evaluate package path %q: %w", strings.Join(path, "."), err)
	}
	if value.DrvPath == "" {
		return nil, errors.New("nix: resolved package has empty drv path")
	}

	return &Package{
		Attr:        attr,
		AttrPath:    append([]string(nil), path...),
		System:      n.system,
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
