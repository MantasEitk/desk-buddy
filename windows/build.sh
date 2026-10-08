#!/bin/sh
# Builds Desk Buddy for Windows (x64) on the Mac, as a folder and a zip:
#   dist/Desk Buddy for Windows/  Desk Buddy.exe, desk-buddy-server.exe (the server, for
#                                 "On this PC"), public/ (the Kindle's update files, fonts)
#   dist/Desk-Buddy-for-Windows.zip
# Needs Go, and `npm install` in ../worker (for Bun, which builds the server).
# The icon and version info come from rsrc_windows_amd64.syso (made with go-winres, see HANDOVER).
# Not stripped (no -s -w) on purpose: antivirus guesses flag stripped, unsigned Go programs more often.
set -e
cd "$(dirname "$0")"
go vet . && GOOS=windows GOARCH=amd64 go vet .
OUT="dist/Desk Buddy for Windows"
rm -rf dist && mkdir -p "$OUT"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H=windowsgui" -o "$OUT/Desk Buddy.exe" .
(cd ../worker && node_modules/.bin/bun build local/server.js --compile --minify --target=bun-windows-x64 --outfile "../windows/$OUT/desk-buddy-server.exe" >/dev/null)
cp -R ../worker/public "$OUT/public"
# the cloud setup kit (Server › Your own cloud server): the server's code + setup.js
mkdir -p "$OUT/cloud-kit"
cp -R ../worker/src ../worker/schema.sql ../worker/setup.js "$OUT/cloud-kit/"
node -e 'const p=require("../worker/package.json");delete p.devDependencies.bun;delete p.scripts;console.log(JSON.stringify(p,null,2))' > "$OUT/cloud-kit/package.json"
(cd dist && zip -q -r -X Desk-Buddy-for-Windows.zip "Desk Buddy for Windows")
ls -la dist "$OUT"
