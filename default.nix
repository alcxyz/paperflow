{ lib, buildGoModule }:

let
  version = "0.2.0";
in
buildGoModule {
  pname = "paperflow";
  inherit version;

  src = ./.;

  vendorHash = "sha256-1t9mjvosmRJiyRugnd8qzwx+9ZAg2ayrCNySp5dIRaM=";

  subPackages = [ "cmd/paperflow" ];

  ldflags = [ "-s" "-w" "-X main.version=v${version}" ];

  meta = with lib; {
    description = "File organizer and Paperless-ngx ingestion tool";
    homepage = "https://github.com/alcxyz/paperflow";
    license = licenses.mit;
    mainProgram = "paperflow";
  };
}
