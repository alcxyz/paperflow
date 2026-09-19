{
  description = "File organizer and Paperless-ngx ingestion tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        sourceVersion = builtins.replaceStrings ["\n"] [""] (builtins.readFile ./VERSION);
        revision = self.rev or self.dirtyRev or "unknown";
        developmentVersion = import ./build-version.nix {
          version = sourceVersion;
          inherit revision;
        };
        releaseVersion = import ./build-version.nix {
          version = sourceVersion;
          inherit revision;
          release = true;
        };
      in {
        packages = (rec {
          paperflow = pkgs.callPackage ./default.nix { version = developmentVersion; };
          default = paperflow;
        }) // pkgs.lib.optionalAttrs
          (builtins.match "[0-9a-f]{7,64}" revision != null)
          { release = pkgs.callPackage ./default.nix { version = releaseVersion; }; };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [ go gopls gotools goreleaser ];
        };
      }
    );
}
