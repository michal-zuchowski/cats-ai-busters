#!/usr/bin/env bash
set -eu

project_dir="$(cd "$(dirname "$0")/.." && pwd)"
compiler_dir="$(mktemp -d)"
git clone --quiet --depth 1 --branch v0.3.11 https://github.com/michal-zuchowski/k65.git "$compiler_dir/k65"
make -C "$compiler_dir/k65" init out/k65.exe
cd "$project_dir"
"$compiler_dir/k65/out/k65.exe" @main.k65proj
