#!/bin/sh
# Builds everything a GitHub release needs, into dist/:
#   Desk-Buddy-for-Mac.zip, Desk-Buddy-for-Windows.zip, Desk-Buddy-for-Kindle.zip
#   desk-buddy-latest.json  what's in this release; the apps read it to update themselves
#                           (via /releases/latest/download/, so upload it with every release)
# Usage: ./release-files.sh v1.2   (the tag you'll give the release on GitHub)
# The Mac and Windows versions come from menubar/patchnotes.json and windows/patchnotes.json.
set -e
cd "$(dirname "$0")"
TAG=${1:?"give the release tag, e.g. v1.2"}
(cd menubar && ./package.sh > /dev/null)
(cd windows && ./build.sh > /dev/null)
git checkout -q worker/src/release.json 2>/dev/null || true # package.sh rewrites it
rm -rf dist && mkdir dist
cp menubar/dist/Desk-Buddy-for-Mac.zip menubar/dist/Desk-Buddy-for-Kindle.zip windows/dist/Desk-Buddy-for-Windows.zip dist/
python3 - "$TAG" > dist/desk-buddy-latest.json <<'PY'
import hashlib, json, os, sys
tag = sys.argv[1]
base = f"https://github.com/deskbuddy-app/desk-buddy/releases/download/{tag}/"
def part(notes, zip):
    data = open(f"dist/{zip}", "rb").read()
    n = json.load(open(notes))
    return {"version": n[0]["version"], "url": base + zip, "sha256": hashlib.sha256(data).hexdigest(),
            "size": len(data), "notes": n[0]["items"], "patchnotes": n[:5]}
print(json.dumps({"tag": tag, "mac": part("menubar/patchnotes.json", "Desk-Buddy-for-Mac.zip"),
                  "windows": part("windows/patchnotes.json", "Desk-Buddy-for-Windows.zip")}, indent=1, ensure_ascii=False))
PY
ls -la dist
