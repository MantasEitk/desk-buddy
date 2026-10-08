#!/bin/sh
# Publishes the current build for the Kindle to pick up over Wi-Fi:
# builds, copies it to the backend's public files under its SHA-256, records it in
# worker/src/release.json, and deploys. The Kindle checks on opening and every 30 minutes.
set -e
cd "$(dirname "$0")"
./build.sh > /dev/null
SHA=$(shasum -a 256 dist/desk/desk | cut -d' ' -f1)
SIZE=$(wc -c < dist/desk/desk | tr -d ' ')
PUB=../../worker/public/kindle
rm -f "$PUB"/desk-*.bin
mkdir -p "$PUB"
cp dist/desk/desk "$PUB/desk-$SHA.bin"
RSHA=$(shasum -a 256 dist/desk/run.sh | cut -d' ' -f1)
rm -f "$PUB"/run-*.sh
cp dist/desk/run.sh "$PUB/run-$RSHA.sh"
# with the newest patch notes (patchnotes.json), so a waiting update can say what it brings
python3 - "$SHA" "$SIZE" "$RSHA" > ../../worker/src/release.json <<'PY'
import json, sys
sha, size, rsha = sys.argv[1], int(sys.argv[2]), sys.argv[3]
notes = json.load(open("patchnotes.json"))[:5]
print(json.dumps({"sha256": sha, "size": size, "file": f"/kindle/desk-{sha}.bin",
                  "run_sha256": rsha, "run_file": f"/kindle/run-{rsha}.sh", "notes": notes}))
PY
cd ../../worker && npx wrangler deploy 2>&1 | grep -E 'Deployed|rror|Uploaded' | head -3
echo "Released $SHA ($SIZE bytes)"
