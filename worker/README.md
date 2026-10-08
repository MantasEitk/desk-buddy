# worker: the Desk Buddy server

One small server holds everything: tasks, calendars, weather, the Deck, music, focus timer, settings and the Kindle's updates. The same code runs in two places:

- **On your computer** (the default): the Desk Buddy app runs it for you. Nothing to set up. `local/server.js` is the wrapper (SQLite instead of D1), built into one program with Bun.
- **On Cloudflare** (optional): your own free Cloudflare Worker with a D1 database, so the Kindle and your phone work while the computer sleeps. Set it up from the app (Server › Your own cloud server), or run `sh setup.sh` here.

Every request needs the key: an `X-Key` header, or the cookie a phone gets after opening the page's QR link once.

## Files

- `src/index.js`: routes, rules and the phone page (`/app`)
- `src/calendar.js` (calendar links, ICS), `src/weather.js` (Open-Meteo), `src/demo.js` (made-up data for photos)
- `src/release.json`: the current Kindle build (written by `kindle/desk/release.sh`)
- `schema.sql`: the database tables
- `setup.sh`: puts your own server online on Cloudflare (needs Node.js; `DRY_RUN=1` only prints what it would do)
- `local/server.js`: the same server for your computer

## Develop

```sh
npm install
npx wrangler d1 execute desk-buddy --local --file schema.sql
echo 'KEY=dev' > .dev.vars
npx wrangler dev --port 8790
```

Or the computer version: `bun local/server.js --data /tmp/db --key dev --port 8787`.
