# Adapts a realized package output: copies it and appends a label to the
# Name of every .desktop entry under share/applications.
{ flakeRef, system, basePath, name, label }:
let
  flake = builtins.getFlake flakeRef;
  pkgs = flake.legacyPackages.${system};
  base = builtins.storePath basePath;
in
pkgs.runCommand name { } ''
  mkdir -p $out
  cp -a ${base}/. "$out"/
  chmod -R u+w "$out"

  if [ -d "$out/share/applications" ]; then
    for f in "$out"/share/applications/*.desktop; do
      [ -e "$f" ] || continue
      ${pkgs.gnused}/bin/sed -i -E 's/^(Name=.*)$/\1${label}/' "$f"
    done
  fi
''
