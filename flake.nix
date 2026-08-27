{
  description = "Bifrost development environment and Shen port environment composer";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "aarch64-darwin" "x86_64-darwin" "aarch64-linux" "x86_64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in {
      packages = forAllSystems (pkgs:
        let
          python = pkgs.python3.withPackages (packages: [ packages.pytest ]);
          base = [ pkgs.git pkgs.go python pkgs.uv pkgs.lefthook pkgs.direnv ];
        in {
          default = pkgs.buildGoModule {
            pname = "bifrost";
            version = "0.1.0-dev";
            src = self;
            vendorHash = null;
            doCheck = true;
          };
          toolchain = pkgs.buildEnv { name = "bifrost-toolchain"; paths = base; };
          env = pkgs.writeShellApplication {
            name = "bifrost-env";
            # Bifrost's own tools are always present; selected port flakes add
            # only their independently owned host toolchains. Reuse the Nix
            # installation which launched this app rather than nesting one.
            runtimeInputs = base;
            text = ''
              all_ports=(shen-cl shen-go shen-erl shen-rust shen-lua ShenScript shen-scheme shen-julia shen-swift shen-truffle)
              selected=()
              while [[ $# -gt 0 && "$1" != "--" ]]; do
                if [[ "$1" == "all" ]]; then selected=("''${all_ports[@]}"); else selected+=("$1"); fi
                shift
              done
              if [[ $# -gt 0 ]]; then shift; fi
              if [[ ''${#selected[@]} -eq 0 ]]; then
                echo "usage: nix run .#env -- PORT [PORT ...|all] -- COMMAND [ARG ...]" >&2
                exit 2
              fi
              workspace="''${BIFROST_PORTS_ROOT:-$(dirname "$PWD")}" 
              installables=()
              for port in "''${selected[@]}"; do
                if [[ ! -f "$workspace/$port/flake.nix" ]]; then
                  echo "bifrost-env: no port flake at $workspace/$port" >&2
                  exit 2
                fi
                installables+=("path:$workspace/$port#toolchain")
              done
              if [[ $# -eq 0 ]]; then set -- "''${SHELL:-sh}"; fi
              exec nix shell "''${installables[@]}" --command "$@"
            '';
          };
        });

      apps = forAllSystems (pkgs: {
        env = {
          type = "app";
          program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.env}/bin/bifrost-env";
        };
      });

      devShells = forAllSystems (pkgs:
        let python = pkgs.python3.withPackages (packages: [ packages.pytest ]);
        in {
          default = pkgs.mkShell {
            packages = [ pkgs.git pkgs.go python pkgs.uv pkgs.lefthook pkgs.direnv ];
            BIFROST_NIX_SHELL = "default";
          };
        });
    };
}
