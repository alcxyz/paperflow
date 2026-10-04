{ lib, buildGoModule
, version ? import ./build-version.nix {
    version = builtins.replaceStrings ["\n"] [""] (builtins.readFile ./VERSION);
  }
}:

buildGoModule {
  pname = "paperflow";
  inherit version;

  src = ./.;

  vendorHash = "sha256-UBy4cGYMBdkiQPC1NKG6mBVV2szFxN35Hn/7zu3emgc=";

  subPackages = [ "cmd/paperflow" ];

  ldflags = [ "-s" "-w" "-X main.version=${version}" ];

  meta = with lib; {
    description = "File organizer and Paperless-ngx ingestion tool";
    homepage = "https://github.com/alcxyz/paperflow";
    license = licenses.mit;
    mainProgram = "paperflow";
  };
}
