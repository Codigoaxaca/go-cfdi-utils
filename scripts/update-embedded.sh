#!/usr/bin/env bash
# Regenera el respaldo incluido (satresources/embedded/data) descargando la
# versión actual de los recursos del SAT. Solo para desarrollo.
set -euo pipefail
cd "$(dirname "$0")/.."
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
go run ./cmd/cfdi-sat-resources -dir "$tmp/store" update
go run ./cmd/cfdi-sat-resources -dir "$tmp/store" export "$tmp/data"
rm -rf satresources/embedded/data
mv "$tmp/data" satresources/embedded/data
echo "respaldo actualizado en satresources/embedded/data"
