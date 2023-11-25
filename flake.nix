{
  description = "Projet IFT605";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-23.05";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        buildDeps = with pkgs; [ git go gnumake ];
        devDeps = with pkgs;
          buildDeps ++ [
            delve
            golangci-lint
            gopls
            gotestsum
            gotools
            grpc-gateway
            httpie
            mage
            protobuf
            protoc-gen-go
            protoc-gen-go-grpc
          ];
      in { devShell = pkgs.mkShell { buildInputs = devDeps; }; });
}
