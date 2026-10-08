#!/bin/sh
# Builds Desk Buddy for Windows (x64) on the Mac: dist/Desk Buddy.exe. Needs Go.
# The icon and version info come from rsrc_windows_amd64.syso (made with go-winres, see HANDOVER).
# Not stripped (no -s -w) on purpose: antivirus guesses flag stripped, unsigned Go programs more often.
set -e
cd "$(dirname "$0")"
go vet . && GOOS=windows GOARCH=amd64 go vet .
mkdir -p dist
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H=windowsgui" -o "dist/Desk Buddy.exe" .
ls -la dist
