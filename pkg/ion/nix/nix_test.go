package nix

import (
	"errors"
	"reflect"
	"testing"

	"github.com/sund3RRR/gonix"
)

func TestLegacyAttrPath(t *testing.T) {
	got := legacyAttrPath("hello", "x86_64-linux")
	want := []string{"legacyPackages", "x86_64-linux", "hello"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("legacyAttrPath() = %#v, want %#v", got, want)
	}
}

func TestLegacyAttrPathSplitsNestedAttribute(t *testing.T) {
	got := legacyAttrPath("gnome.gedit", "aarch64-darwin")
	want := []string{"legacyPackages", "aarch64-darwin", "gnome", "gedit"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("legacyAttrPath() = %#v, want %#v", got, want)
	}
}

func TestPackageAttrPath(t *testing.T) {
	got := packageAttrPath("gnome.gedit", "x86_64-linux")
	want := []string{"packages", "x86_64-linux", "gnome", "gedit"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("packageAttrPath() = %#v, want %#v", got, want)
	}
}

func TestIsMissingAttributeError(t *testing.T) {
	err := &gonix.Error{
		Code:    gonix.ErrorCodeKey,
		Message: "missing attribute",
	}

	if !isMissingAttributeError(err) {
		t.Fatal("isMissingAttributeError() = false, want true")
	}
}

func TestIsMissingAttributeErrorUnwraps(t *testing.T) {
	err := &gonix.Error{
		Code:    gonix.ErrorCodeKey,
		Message: "missing attribute",
	}

	if !isMissingAttributeError(errors.Join(errors.New("context"), err)) {
		t.Fatal("isMissingAttributeError() = false, want true for wrapped error")
	}
}

func TestIsMissingAttributeErrorRejectsOtherErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "nil",
			err:  nil,
		},
		{
			name: "plain error",
			err:  errors.New("missing attribute"),
		},
		{
			name: "different gonix code",
			err: &gonix.Error{
				Code:    gonix.ErrorCodeUnknown,
				Message: "missing attribute",
			},
		},
		{
			name: "different gonix message",
			err: &gonix.Error{
				Code:    gonix.ErrorCodeKey,
				Message: "missing key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isMissingAttributeError(tt.err) {
				t.Fatal("isMissingAttributeError() = true, want false")
			}
		})
	}
}
