#!/bin/sh
# Desk Buddy: put your own server online, on Cloudflare's free plan. Run it once.
#
#   sh setup.sh [key]
#
# key: Desk Buddy's key, so your Kindle and computer keep working with the new server. The
# Mac app's Server › "Your own cloud server" puts it in a private file named `key` next to this
# script instead, so it's never typed or pasted anywhere. Without either, a new key is made.
# Needs Node.js (https://nodejs.org). Your browser opens once to log in to Cloudflare.
# It makes its own settings file (wrangler.desk-buddy.toml) and never changes wrangler.toml.
# Set DRY_RUN=1 to only check and print what it would do.
set -e
cd "$(dirname "$0")"
KEY="${1:-$(cat key 2>/dev/null)}"
[ -n "$KEY" ] || KEY=$(openssl rand -hex 12)
NAME="${DESK_BUDDY_NAME:-desk-buddy}"
CONF=wrangler.desk-buddy.toml
TZ_NAME=$(readlink /etc/localtime 2>/dev/null | sed 's#.*/zoneinfo/##')
[ -n "$TZ_NAME" ] || TZ_NAME=Europe/London
run() { if [ -n "$DRY_RUN" ]; then echo "   (would run) $*"; else "$@"; fi; }

echo ""
echo "  Desk Buddy · putting your server online"
echo "  ---------------------------------------"
if ! command -v npx >/dev/null 2>&1; then
  echo "  Please install Node.js first (the LTS version): https://nodejs.org"
  echo "  Then run this again."
  exit 1
fi

echo "  1/5  Getting the tools ready…"
run npm install --silent --no-audit --no-fund >/dev/null 2>&1

echo "  2/5  Logging in to Cloudflare (your browser may open)…"
if [ -z "$DRY_RUN" ] && ! npx wrangler whoami 2>/dev/null | grep -q "logged in"; then
  npx wrangler login
fi

echo "  3/5  Making your database…"
if [ -n "$DRY_RUN" ]; then
  ID=00000000-0000-0000-0000-000000000000
else
  npx wrangler d1 create "$NAME" >/dev/null 2>&1 || true # already there from an earlier run: fine
  ID=$(npx wrangler d1 info "$NAME" --json 2>/dev/null | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{try{const j=JSON.parse(s);console.log(j.uuid||j.database_id||"")}catch{console.log("")}})')
fi
if [ -z "$ID" ]; then
  echo "  Couldn't make or find the database. Check you're logged in (npx wrangler whoami) and try again."
  exit 1
fi
cat > "$CONF" <<EOF
name = "$NAME"
main = "src/index.js"
compatibility_date = "2026-09-01"

[assets]
directory = "./public"

[vars]
TZ = "$TZ_NAME"

[[d1_databases]]
binding = "DB"
database_name = "$NAME"
database_id = "$ID"
EOF
run npx wrangler d1 execute "$NAME" --remote --file schema.sql -c "$CONF" -y >/dev/null

echo "  4/5  Putting the server online…"
if [ -n "$DRY_RUN" ]; then
  URL="https://$NAME.example.workers.dev"
  echo "   (would run) npx wrangler deploy -c $CONF"
else
  URL=$(npx wrangler deploy -c "$CONF" 2>&1 | grep -oE 'https://[a-zA-Z0-9.-]+\.workers\.dev' | head -1)
fi
if [ -z "$URL" ]; then
  echo "  Couldn't put it online. Run: npx wrangler deploy -c $CONF   to see why."
  exit 1
fi

echo "  5/5  Locking it with your key…"
if [ -n "$DRY_RUN" ]; then
  echo "   (would run) npx wrangler secret put KEY -c $CONF"
else
  printf '%s' "$KEY" | npx wrangler secret put KEY -c "$CONF" >/dev/null
fi

printf '%s' "$URL" | pbcopy 2>/dev/null || true
echo ""
echo "  Done! Your server is online:"
echo ""
echo "     $URL"
echo ""
echo "  It's copied. Paste it into Desk Buddy › Server › Your own cloud server, and click Connect."
echo ""
