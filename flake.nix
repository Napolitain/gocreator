{
  description = "A CLI tool for creating videos with translations and audio narration";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = "0.0.0";
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "gocreator";
          inherit version;

          src = ./.;

          vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";

          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];

          subPackages = [ "cmd/gocreator" ];

          meta = with pkgs.lib; {
            description = "A CLI tool for creating videos with translations and audio narration";
            homepage = "https://github.com/Napolitain/gocreator";
            license = licenses.gpl3;
            maintainers = [ ];
            mainProgram = "gocreator";
          };
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/gocreator";
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
          ];
        };
      }
    );
}
