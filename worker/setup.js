#!/usr/bin/env node
// Desk Buddy: put your own server online, on Cloudflare's free plan (any computer with Node.js).
// The same steps as setup.sh: node setup.js [key]
// The key comes from the argument, or a private file named `key` next to this script (Desk Buddy
// writes it there), or a new one is made. Set DRY_RUN=1 to only print what it would do.
const { execSync, spawnSync } = require("child_process");
const fs = require("fs");
const path = require("path");
process.chdir(__dirname);
const dry = !!process.env.DRY_RUN;
const name = process.env.DESK_BUDDY_NAME || "desk-buddy";
const conf = "wrangler.desk-buddy.toml";
let key = process.argv[2] || (fs.existsSync("key") ? fs.readFileSync("key", "utf8").trim() : "");
if (!key) key = require("crypto").randomBytes(12).toString("hex");
const tz = Intl.DateTimeFormat().resolvedOptions().timeZone || "Europe/London";
const npx = process.platform === "win32" ? "npx.cmd" : "npx";
const run = (args, opts = {}) => {
  if (dry) return console.log("   (would run) npx " + args.join(" ")), { status: 0, stdout: "" };
  return spawnSync(npx, args, { stdio: opts.capture ? ["inherit", "pipe", "inherit"] : "inherit", encoding: "utf8", shell: process.platform === "win32", input: opts.input });
};
const fail = (msg) => { console.log("\n  " + msg + "\n"); process.exit(1); };

console.log("\n  Desk Buddy · putting your server online\n  ---------------------------------------");
console.log("  1/5  Getting the tools ready…");
if (!dry) execSync(process.platform === "win32" ? "npm.cmd install --silent --no-audit --no-fund" : "npm install --silent --no-audit --no-fund", { stdio: "ignore" });
console.log("  2/5  Logging in to Cloudflare (your browser may open)…");
if (!dry && !/logged in/i.test(run(["wrangler", "whoami"], { capture: true }).stdout || "")) run(["wrangler", "login"]);
console.log("  3/5  Making your database…");
let id = "00000000-0000-0000-0000-000000000000";
if (!dry) {
  run(["wrangler", "d1", "create", name], { capture: true }); // already there from an earlier run: fine
  try { const j = JSON.parse(run(["wrangler", "d1", "info", name, "--json"], { capture: true }).stdout); id = j.uuid || j.database_id || ""; } catch { id = ""; }
}
if (!id) fail("Couldn't make or find the database. Check you're logged in (npx wrangler whoami) and try again.");
(dry ? (t) => console.log("   (would write) " + conf) : (t) => fs.writeFileSync(conf, t))(`name = "${name}"\nmain = "src/index.js"\ncompatibility_date = "2026-09-01"\n\n[assets]\ndirectory = "./public"\n\n[vars]\nTZ = "${tz}"\n\n[[d1_databases]]\nbinding = "DB"\ndatabase_name = "${name}"\ndatabase_id = "${id}"\n`);
run(["wrangler", "d1", "execute", name, "--remote", "--file", "schema.sql", "-c", conf, "-y"], { capture: true });
console.log("  4/5  Putting the server online…");
let url = `https://${name}.example.workers.dev`;
if (!dry) {
  const out = run(["wrangler", "deploy", "-c", conf], { capture: true }).stdout || "";
  url = (out.match(/https:\/\/[a-zA-Z0-9.-]+\.workers\.dev/) || [""])[0];
}
if (!url) fail(`Couldn't put it online. Run: npx wrangler deploy -c ${conf}   to see why.`);
console.log("  5/5  Locking it with your key…");
run(["wrangler", "secret", "put", "KEY", "-c", conf], { capture: true, input: key });
console.log(`\n  Done! Your server is online:\n\n     ${url}\n\n  Paste it into Desk Buddy › Server › Your own cloud server, and click Connect.\n`);
