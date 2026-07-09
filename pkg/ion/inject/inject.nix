{
  flakeRef,
  system,
  basePath,
  name,
  tweaks,
  tweakSources,
}:
let
  flake = builtins.getFlake flakeRef;
  pkgs = flake.legacyPackages.${system};
  base = builtins.storePath basePath;

  placeholder = import (builtins.toFile "ion-placeholder-tweak.nix" tweakSources.placeholder);
  nixgl = import (builtins.toFile "ion-nixgl-tweak.nix" tweakSources.nixgl);

  enabledTweaks = builtins.filter (tweak: tweak.enabled) [
    {
      enabled = tweaks.placeholder.enabled;
      apply = placeholder {
        inherit pkgs;
        label = tweaks.placeholder.label;
      };
    }
    {
      enabled = tweaks.nixgl.enabled;
      apply = nixgl {
        inherit pkgs;
      };
    }
  ];

  script = builtins.concatStringsSep "\n" (map (tweak: tweak.apply) enabledTweaks);
in
pkgs.runCommand name { } ''
  mkdir -p "$out"
  cp -a ${base}/. "$out"/
  chmod -R u+w "$out"

  ${script}
''
