{ pkgs, lib, config, inputs, ... }:

{
  # https://devenv.sh/packages/
  packages = [
    pkgs.git
    pkgs.goreleaser
  ];

  # https://devenv.sh/languages/
  languages.go.enable = true;
}
