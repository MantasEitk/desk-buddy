#!/bin/sh
# Offers the current Windows app as an update: builds it, puts Desk Buddy.exe (under its
# SHA-256) and release.json in ../worker/public/windows/, where Desk Buddy servers serve them.
# Then deploy the cloud (cd ../worker && npx wrangler deploy) and/or reinstall the Mac app
# (../menubar/install.sh) so its server has them. Patch notes: patchnotes.json (newest first,
# its first version must match main.go's version).
set -e
cd "$(dirname "$0")"
./build.sh > /dev/null
VERSION=$(sed -n 's/^const version = "\(.*\)"/\1/p' main.go)
EXE="dist/Desk Buddy for Windows/Desk Buddy.exe"
SHA=$(shasum -a 256 "$EXE" | cut -d' ' -f1)
SIZE=$(wc -c < "$EXE" | tr -d ' ')
PUB=../worker/public/windows
mkdir -p "$PUB" && rm -f "$PUB"/desk-buddy-*.exe
cp "$EXE" "$PUB/desk-buddy-$SHA.exe"
python3 - "$VERSION" "$SHA" "$SIZE" > "$PUB/release.json" <<'PY'
import json, sys
v, sha, size = sys.argv[1], sys.argv[2], int(sys.argv[3])
pn = json.load(open("patchnotes.json"))
assert pn[0]["version"] == v, "patchnotes.json should start with version " + v
print(json.dumps({"version": v, "sha256": sha, "size": size, "file": f"/windows/desk-buddy-{sha}.exe",
                  "notes": pn[0]["items"], "patchnotes": pn[:5]}))
PY
echo "Offered Windows $VERSION ($SHA, $SIZE bytes)"
