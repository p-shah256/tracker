
{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = [
    pkgs.python3
    pkgs.python3Packages.pip
    pkgs.python311Packages.virtualenv
  ];

  shellHook = ''
    python -m venv .venv
    source .venv/bin/activate
    pip install --upgrade pip
  '';
}

