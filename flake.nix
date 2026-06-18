{
  description = "Browser-based Markdown/Obsidian notes viewer";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forAllSystems = f:
        nixpkgs.lib.genAttrs systems (system:
          f nixpkgs.legacyPackages.${system}
        );
    in
    {
      packages = forAllSystems (pkgs: {
        default = pkgs.buildGoModule {
          pname = "notes-web";
          version = "0.1.0";

          src = ./.;
          subPackages = [ "cmd/notes-web" ];

          # Replace with the sha256 reported by `nix build` after first run.
          vendorHash = null;

          ldflags = [
            "-s"
            "-w"
          ];

          meta = {
            description = "Browser-based Markdown/Obsidian notes viewer";
            homepage = "https://github.com/arkan/notes-web";
            mainProgram = "notes-web";
          };
        };
      });

      apps = forAllSystems (pkgs: {
        default = {
          type = "app";
          program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.default}/bin/notes-web";
        };
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = [
            pkgs.go
            pkgs.gopls
            pkgs.gotools
            pkgs.nodejs
          ];
        };
      });

      nixosModules.default = import ./nix/nixos-module.nix { inherit self; };
      nixosModules.notes-web = self.nixosModules.default;
    };
}
