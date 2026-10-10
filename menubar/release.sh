#!/bin/sh
# Offers the current Mac version (patchnotes.json's newest) at /mac/release.json on Desk Buddy
# servers: About › Updates compares it with its own version and shows the notes. Then deploy the
# cloud (cd ../worker && npx wrangler deploy) and/or ./install.sh (this Mac's server serves it).
# With a download: ./release.sh <zip> <url>  (url: where the zip is, e.g. a GitHub Release
# asset; the zip holds "Desk Buddy.app" and must NOT hold backend/config.json or any key).
# Without one, the apps only say a new version is out.
set -e
cd "$(dirname "$0")"
PUB=../worker/public/mac
mkdir -p "$PUB"
python3 - "$@" > "$PUB/release.json" <<'PY'
import hashlib, json, os, sys
pn = json.load(open("patchnotes.json"))
r = {"version": pn[0]["version"], "notes": pn[0]["items"], "patchnotes": pn[:5]}
if len(sys.argv) > 2:
    z, url = sys.argv[1], sys.argv[2]
    r.update(sha256=hashlib.sha256(open(z, "rb").read()).hexdigest(), size=os.path.getsize(z), url=url)
print(json.dumps(r))
PY
echo "Offered Mac $(python3 -c 'import json;print(json.load(open("../worker/public/mac/release.json"))["version"])')"
