package inject

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/sund3RRR/gonix"
	"github.com/sund3RRR/ion/pkg/ion/nix"
)

// wrapperExpr is the Nix function that adapts a package output, embedded
// from inject.nix. It takes { flakeRef, system, basePath, name, label }.
//
//go:embed inject.nix
var wrapperExpr string

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
	Attr             string
	FlakeRef         string
	System           string
	StorePath        string
	OutputName       string
	NixGLTweak       bool
	PlaceholderTweak bool
}

func (inj *Injector) AdaptPackage(ctx context.Context, req Request) (gonix.RealizedOutput, string, error) {
	drvPath, _, err := inj.nix.CallDerivation(ctx, wrapperExpr, map[string]string{
		"flakeRef": req.FlakeRef,
		"system":   req.System,
		"basePath": req.StorePath,
		"name":     req.Attr + "-ion",
		"label":    " (ion)",
	})
	if err != nil {
		return gonix.RealizedOutput{}, "", fmt.Errorf("inject: evaluate wrapper derivation: %w", err)
	}

	realized, err := inj.nix.RealizeOutput(ctx, drvPath, req.OutputName)
	if err != nil {
		return gonix.RealizedOutput{}, "", fmt.Errorf("inject: realize wrapper derivation: %w", err)
	}

	return realized, drvPath, nil
}
