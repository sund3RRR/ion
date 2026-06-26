{
  description = "ION";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    gonix = {
      url = "github:sund3RRR/gonix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      gonix,
    }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
          lib = pkgs.lib;

          appVersion = self.shortRev or self.dirtyShortRev or "dev";
          commit = self.shortRev or self.dirtyShortRev or "unknown";
          date = self.lastModifiedDate or "unknown";

          versionPackage = "github.com/sund3RRR/ion/internal/version";
          commonLdflags = [
            "-s"
            "-w"
            "-X=${versionPackage}.Version=${appVersion}"
            "-X=${versionPackage}.Commit=${commit}"
            "-X=${versionPackage}.Date=${date}"
          ];

          mkIon =
            goPkgs:
            {
              static ? false,
            }:
            goPkgs.buildGoModule {
              pname = "ion";
              version = appVersion;
              src = lib.cleanSource ./.;
              vendorHash = "sha256-cVn74Wljuu9tuK6Z8MCpjtdEpU6843MNfz0GSVZ3ZPk=";

              subPackages = [ "cmd/ion" ];
              preBuild = ''
                export CGO_ENABLED=1
              '';

              tags = lib.optionals static [
                "netgo"
                "osusergo"
              ];

              ldflags =
                commonLdflags
                ++ lib.optionals static [
                  "-linkmode=external"
                  "-extldflags=-static"
                ];

              meta = {
                description = "ION";
                homepage = "https://github.com/sund3RRR/ion";
                license = lib.licenses.mit;
                mainProgram = "ion";
              };
            };

          dynamicPackage = mkIon pkgs { };
          staticPackage = mkIon pkgs.pkgsStatic { static = true; };
        in
        {
          default = if pkgs.stdenv.isLinux then staticPackage else dynamicPackage;
          dynamic = dynamicPackage;
        }
        // lib.optionalAttrs pkgs.stdenv.isLinux {
          static = staticPackage;
        }
      );

      apps = forAllSystems (
        system:
        {
          default = {
            type = "app";
            program = "${self.packages.${system}.default}/bin/ion";
          };
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          default = gonix.lib.${system}.mkDevShell {
            packages = [
              pkgs.buf
              pkgs.protobuf
              pkgs.protoc-gen-go
              pkgs.protoc-gen-connect-go
              pkgs.gnumake
            ];
          };
        }
      );
    };
}
