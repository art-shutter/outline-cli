{
  description = "outline-cli — command-line interface for managing Outline VPN servers";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [
        "aarch64-darwin"
        "x86_64-darwin"
        "x86_64-linux"
        "aarch64-linux"
      ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      version = self.shortRev or self.dirtyShortRev or "dev";
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          outline-cli = pkgs.buildGoModule {
            pname = "outline-cli";
            inherit version;
            src = ./.;

            vendorHash = "sha256-za4UlLocx8LqcOaD4Ag5Mnt3ONG7No+XsnxTaiFDRT0=";

            env.CGO_ENABLED = 0;

            ldflags = [
              "-s"
              "-w"
              "-X main.Version=${version}"
            ];

            meta = {
              description = "A command-line interface for managing Outline VPN servers";
              homepage = "https://github.com/art-shutter/outline-cli";
              license = pkgs.lib.licenses.mit;
              mainProgram = "outline-cli";
              platforms = pkgs.lib.platforms.unix;
            };
          };

          default = self.packages.${system}.outline-cli;
        }
      );

      apps = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
          mkTask =
            name: runtimeInputs: text:
            let
              drv = pkgs.writeShellApplication {
                inherit name runtimeInputs text;
              };
            in
            {
              type = "app";
              program = "${drv}/bin/${name}";
            };
        in
        {
          fmt = mkTask "oc-fmt" [ pkgs.go pkgs.gofumpt pkgs.golangci-lint ] ''
            gofumpt -l -w .
            golangci-lint fmt
          '';

          lint = mkTask "oc-lint" [ pkgs.go pkgs.golangci-lint ] ''
            golangci-lint run ./...
          '';

          build = mkTask "oc-build" [ pkgs.go ] ''
            mkdir -p bin
            CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.Version=${version}" -o bin/outline-cli ./cmd/outline-cli
          '';

          test = mkTask "oc-test" [ pkgs.go ] ''
            go test -v -race ./...
          '';

          release = mkTask "oc-release" [ pkgs.git pkgs.go pkgs.goreleaser ] ''
            goreleaser release --clean "$@"
          '';

          release-snapshot = mkTask "oc-release-snapshot" [ pkgs.git pkgs.go pkgs.goreleaser ] ''
            goreleaser release --snapshot --clean "$@"
          '';
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
          appBins = map (a: pkgs.writeShellScriptBin (baseNameOf a.program) ''exec ${a.program} "$@"'') (
            builtins.attrValues self.apps.${system}
          );
        in
        {
          default = pkgs.mkShell {
            packages = [
              pkgs.go
              pkgs.gofumpt
              pkgs.golangci-lint
              pkgs.gopls
              pkgs.goreleaser
            ]
            ++ appBins;

            shellHook = ''
              export GODEBUG=netdns=cgo
              echo "outline-cli devShell active (go: $(go version | awk '{print $3}'), gofumpt, golangci-lint, goreleaser)"
              echo "Tasks: nix run .#{fmt,lint,build,test,release-snapshot}"
              echo "   or: oc-{fmt,lint,build,test,release-snapshot}"
            '';
          };
        }
      );
    };
}
