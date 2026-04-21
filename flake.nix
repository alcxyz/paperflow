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
        version = builtins.replaceStrings ["\n"] [""] (builtins.readFile ./VERSION);
      in {
        packages = rec {
          paperflow = pkgs.callPackage ./default.nix { inherit version; };
          default = paperflow;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [ go gopls gotools goreleaser ];
        };
      }
    );
}
