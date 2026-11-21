#!/bin/sh
# Сборка установщика (Linux/macOS/WSL): нужен только Go 1.22+ и Python 3 для ресурсов.
set -e
cd "$(dirname "$0")"
python3 mksyso.py icon.ico setup.manifest rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H windowsgui -s -w" -o CapCut-Wiggle-Setup.exe .
echo "Готово: CapCut-Wiggle-Setup.exe"
