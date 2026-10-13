#!/bin/sh
# Cross-compile the Windows GUI from Linux. Output: dist/PolycompManager.exe (64-bit) and dist/PolycompManager-32bit.exe
set -e
cd "$(dirname "$0")"
export PATH="$HOME/.local/go/bin:$PATH"
go test ./...
# embed manifest + icon as a Windows resource object (walk needs the Common-Controls v6 manifest)
go run github.com/akavel/rsrc@v0.10.2 -manifest cmd/polycomp-gui/app.manifest -ico cmd/polycomp-gui/app.ico -arch amd64 -o cmd/polycomp-gui/rsrc_windows_amd64.syso
go run github.com/akavel/rsrc@v0.10.2 -manifest cmd/polycomp-gui/app.manifest -ico cmd/polycomp-gui/app.ico -arch 386   -o cmd/polycomp-gui/rsrc_windows_386.syso
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -H windowsgui" -o dist/PolycompManager.exe ./cmd/polycomp-gui
GOOS=windows GOARCH=386   CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -H windowsgui" -o dist/PolycompManager-32bit.exe ./cmd/polycomp-gui
ls -l dist/
