#!/bin/sh
# Build main.k65proj and start Atari800MacX with the fresh binary.
# K65 defaults to the compiler bundled by the IntelliJ K65 plugin.
set -e
cd "$(dirname "$0")/.."
K65=${K65:-$(ls -t "$HOME"/Library/Caches/JetBrains/*/k65-intellij/*/out/k65.exe 2>/dev/null | head -1)}
[ -x "$K65" ] || { echo "k65 compiler not found; set K65=/path/to/k65.exe" >&2; exit 1; }
"$K65" @main.k65proj
git checkout -q -- test_raw.sym 2>/dev/null || true
python3 tools/make_atr.py
osascript -e 'quit app "Atari800MacX"' 2>/dev/null || true
sleep 1
open -a Atari800MacX out/cats-ai-busters.atr
