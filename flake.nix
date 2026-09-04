{
  description = "Bifrost development environment and Shen port environment composer";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "aarch64-darwin" "aarch64-linux" "x86_64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in {
      packages = forAllSystems (pkgs:
        let
          base = [ pkgs.git pkgs.go pkgs.lefthook pkgs.direnv ];
        in {
          default = pkgs.buildGoModule {
            pname = "bifrost";
            version = "0.1.0-dev";
            src = self;
            vendorHash = null;
            doCheck = true;
            meta.mainProgram = "bifrost";
          };
          toolchain = pkgs.buildEnv { name = "bifrost-toolchain"; paths = base; };
          env = pkgs.writeShellApplication {
            name = "bifrost-env";
            # Bifrost's own tools are always present; selected port flakes add
            # only their independently owned host toolchains. Reuse the Nix
            # installation which launched this app rather than nesting one.
            runtimeInputs = base;
            text = ''
              exec ${self.packages.${pkgs.stdenv.hostPlatform.system}.default}/bin/bifrost env "$@"
            '';
          };
        });

      apps = forAllSystems (pkgs: {
        default = {
          type = "app";
          program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.default}/bin/bifrost";
        };
        env = {
          type = "app";
          program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.env}/bin/bifrost-env";
        };
      });

      devShells = forAllSystems (pkgs: {
          default = pkgs.mkShell {
            packages = [ pkgs.git pkgs.go pkgs.lefthook pkgs.direnv ];
            BIFROST_NIX_SHELL = "default";
          };
        });
    };
}
