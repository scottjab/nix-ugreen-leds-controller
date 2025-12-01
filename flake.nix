{
  description = "Nix package and NixOS module for UGREEN LEDs Controller";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
        package = pkgs.callPackage ./default.nix {
          kernel = pkgs.linuxPackages.kernel;
        };
      in
      {
        packages = {
          default = package;
          ugreen-leds-controller = package;
        };
      }
    )
    // {
      nixosModules = {
        default = import ./module.nix ./default.nix;
        ugreen-leds = import ./module.nix ./default.nix;
      };

      overlays.default = final: prev: {
        ugreen-leds-controller = final.callPackage ./default.nix {
          kernel = final.linuxPackages.kernel;
        };
      };
    };
}
