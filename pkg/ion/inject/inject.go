package inject

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/sund3RRR/ion/pkg/ion/nix"
)

// wrapperExpr is the Nix function that adapts a package output, embedded
// from inject.nix.
//
//go:embed inject.nix
var wrapperExpr string

//go:embed placeholder.nix
var placeholderExpr string

//go:embed nixgl.nix
var nixGLExpr string

// Injector adapts a realized package into a new wrapped derivation by
// evaluating and realizing generated Nix code.
type Injector struct {
	nix *nix.Nix
}

// New creates an Injector that evaluates and realizes wrapper derivations
// through n.
func New(n *nix.Nix) *Injector {
	return &Injector{nix: n}
}

// Request describes the package output to adapt.
type Request struct {
	// Attr is the user-facing package attribute being adapted.
	Attr string
	// FlakeRef is the source flake reference used to resolve nixpkgs helpers.
	FlakeRef string
	// System is the Nix system used for the wrapper derivation.
	System string
	// StorePath is the realized package output to wrap.
	StorePath string
	// OutputName is the wrapper derivation output to realize.
	OutputName string
	// Tweaks configures the package-output transformations applied by Nix.
	Tweaks Tweaks
}

// Tweaks groups all injector tweaks passed from Go into the Nix dispatcher.
type Tweaks struct {
	// Placeholder configures desktop-entry placeholder labeling.
	Placeholder PlaceholderTweakConfig
	// NixGL configures nixGL wrapping. It is reserved for future use.
	NixGL NixGLTweakConfig
}

// PlaceholderTweakConfig configures the desktop placeholder tweak.
type PlaceholderTweakConfig struct {
	// Enabled controls whether the tweak is applied.
	Enabled bool
	// Label is appended to desktop-entry Name fields when Enabled is true.
	Label string
}

// NixGLTweakConfig configures the nixGL tweak.
type NixGLTweakConfig struct {
	// Enabled controls whether the tweak is applied.
	Enabled bool
}

const defaultPlaceholderLabel = " (ion)"

// AdaptPackage creates and realizes an adapted derivation for req.
func (inj *Injector) AdaptPackage(ctx context.Context, req Request) (*nix.RealizedOutput, error) {
	tweaks := normalizeTweaks(req)
	drv, err := inj.nix.CallDerivation(ctx, wrapperExpr, map[string]any{
		"flakeRef": req.FlakeRef,
		"system":   req.System,
		"basePath": req.StorePath,
		"name":     req.Attr + "-ion",
		"tweaks": map[string]any{
			"placeholder": map[string]any{
				"enabled": tweaks.Placeholder.Enabled,
				"label":   tweaks.Placeholder.Label,
			},
			"nixgl": map[string]any{
				"enabled": tweaks.NixGL.Enabled,
			},
		},
		"tweakSources": map[string]any{
			"placeholder": placeholderExpr,
			"nixgl":       nixGLExpr,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("inject: evaluate wrapper derivation: %w", err)
	}

	realized, err := inj.nix.RealizeOutput(ctx, drv.Path, req.OutputName)
	if err != nil {
		return nil, fmt.Errorf("inject: realize wrapper derivation: %w", err)
	}

	return realized, nil
}

func normalizeTweaks(req Request) Tweaks {
	tweaks := req.Tweaks
	if tweaks.Placeholder.Enabled && tweaks.Placeholder.Label == "" {
		tweaks.Placeholder.Label = defaultPlaceholderLabel
	}
	return tweaks
}
