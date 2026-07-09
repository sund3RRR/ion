package inject

import "testing"

func TestNormalizeTweaksUsesExplicitConfig(t *testing.T) {
	got := normalizeTweaks(Request{
		Tweaks: Tweaks{
			Placeholder: PlaceholderTweakConfig{
				Enabled: true,
				Label:   " custom",
			},
			NixGL: NixGLTweakConfig{
				Enabled: true,
			},
		},
	})

	if !got.Placeholder.Enabled || got.Placeholder.Label != " custom" {
		t.Fatalf("Placeholder = %#v, want enabled custom label", got.Placeholder)
	}
	if !got.NixGL.Enabled {
		t.Fatal("NixGL.Enabled = false, want true")
	}
}

func TestNormalizeTweaksLeavesDisabledPlaceholderLabelEmpty(t *testing.T) {
	got := normalizeTweaks(Request{})
	if got.Placeholder.Enabled {
		t.Fatal("Placeholder.Enabled = true, want false")
	}
	if got.Placeholder.Label != "" {
		t.Fatalf("Placeholder.Label = %q, want empty", got.Placeholder.Label)
	}
}
