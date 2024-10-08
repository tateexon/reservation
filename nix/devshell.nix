{ pkgs, scriptDir, go }:
with pkgs;
mkShell {
  nativeBuildInputs = [
    # basics
    bash
    # git
    curl
    gnumake
    jq
    dasel
    github-cli
    postgresql_16

    # go
    go
    go-mockery
    gotools
    gopls
    delve
    golangci-lint
    go-migrate

    #open api
    oapi-codegen

    # linting tools
    typos
    pre-commit
    python3
    shfmt
    shellcheck
  ];

  CGO_ENABLED = "0";
  GOROOT = "${go}/share/go";

  shellHook = ''
    # Uninstall pre-commit hooks in case they get messed up
    pre-commit uninstall > /dev/null || true
    pre-commit uninstall --hook-type pre-push > /dev/null || true

    # enable pre-commit hooks
    pre-commit install > /dev/null
    pre-commit install -f --hook-type pre-push > /dev/null

    # setup go bin
    export GOBIN=$HOME/.nix-go/bin
    mkdir -p $GOBIN
    export PATH=$GOBIN:$PATH
  '';
}
