{ pkgs, label }:
''
  if [ -d "$out/share/applications" ]; then
    for f in "$out"/share/applications/*.desktop; do
      [ -e "$f" ] || continue
      ${pkgs.gnused}/bin/sed -i -E 's/^(Name=.*)$/\1${label}/' "$f"
    done
  fi
''
