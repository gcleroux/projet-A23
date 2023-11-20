{
  description = "Projet IFT605";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        buildDeps = with pkgs; [ git go gnumake ];
        devDeps = with pkgs;
          buildDeps ++ [
            golangci-lint
            gopls
            gotestsum
            gotools
            httpie
            mage
            protobuf
            protoc-gen-go
            protoc-gen-go-grpc
          ];
      in { devShell = pkgs.mkShell { buildInputs = devDeps; }; });
}
