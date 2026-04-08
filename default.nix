{ lib, buildGoModule }:

buildGoModule {
  pname = "paperflow";
  version = "0.1.0";

  src = ./.;

  vendorHash = "sha256-1t9mjvosmRJiyRugnd8qzwx+9ZAg2ayrCNySp5dIRaM=";

  subPackages = [ "cmd/paperflow" ];

  meta = with lib; {
    description = "File organizer and Paperless-ngx ingestion tool";
    homepage = "https://github.com/alcxyz/paperflow";
    license = licenses.mit;
    mainProgram = "paperflow";
  };
}
