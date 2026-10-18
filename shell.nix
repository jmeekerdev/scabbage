# Dev tools only; deployment uses Docker (see compose.yaml).
{ pkgs ? import <nixpkgs> { } }:
pkgs.mkShell {
  packages = with pkgs; [
    go
    gcc # for `go test -race`
    nodejs_22
    buf
    protoc-gen-go
    gnumake
    golangci-lint
  ];
}
