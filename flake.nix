{
  description = "A command-line interface for managing Outline VPN servers with full API integration";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
        version = self.rev or "dev";
      in {
        formatter = pkgs.alejandra;

        packages.outline-cli = pkgs.buildGoModule {
          pname = "outline-cli";
          inherit version;
          src = builtins.path {
            path = ./src;
            name = "source";
          };

          vendorHash = "sha256-0sCb33khZSpMxmUnvO7ASagONP3ytjAGYMH/2aAJDxA=";

          env.CGO_ENABLED = 0;

          ldflags = [
            "-s -w -X main.Version=${version}"
          ];

          meta = with pkgs.lib; {
            description = "A command-line interface for managing Outline VPN servers with full API integration";
            homepage = "https://github.com/art-shutter/outline-cli";
            maintainers = ["art-shutter"];
            mainProgram = "outline-cli";
            platforms = platforms.unix ++ platforms.darwin;
          };
        };

        packages.default = self.packages.${system}.outline-cli;

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            golangci-lint
            gopls
            just
            pre-commit
          ];
          shellHook = ''
            export PATH="$(pwd)/result/bin:''${PATH}"
            pre-commit install --install-hooks --overwrite
          '';
        };
      }
    );
}
