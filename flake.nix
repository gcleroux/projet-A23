{
  description = "Projet IFT605";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-23.05";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = inputs@{ self, ... }:
    inputs.flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import inputs.nixpkgs { inherit system; };
        buildDeps = with pkgs; [
          git
          go
          grpc-gateway
          protobuf
          protoc-gen-go
          protoc-gen-go-grpc
          mage
        ];
        devDeps = with pkgs;
          buildDeps ++ [ delve golangci-lint gopls gotestsum gotools httpie ];
      in { devShell = pkgs.mkShell { buildInputs = devDeps; }; });
}
