// Desk Buddy, local mode: the same backend as on Cloudflare (src/index.js), run on the
// user's own computer. The database is a SQLite file instead of D1; the Kindle builds are
// served from a folder; nothing goes to Cloudflare.
//
// Built into one standalone program (no Node or Bun needed on the computer):
//   cd worker && npm run build:local      → local/dist/desk-buddy-server
// Run:
//   desk-buddy-server --data <folder> --key <key> [--port 8787] [--assets <folder>] [--tz Europe/London]
//
// It also answers on UDP port 8788: a Kindle on the same Wi-Fi sends "desk-buddy?" and
// gets back {"name", "port"}, so it can find this computer without typing an address.

import worker from "../src/index.js";
import { Database } from "bun:sqlite";
import { hostname } from "node:os";
import { join } from "node:path";
import { mkdirSync } from "node:fs";

const arg = (name, def) => {
  const i = process.argv.indexOf("--" + name);
  return i > 0 ? process.argv[i + 1] : def;
};
const dataDir = arg("data", ".");
const key = arg("key", process.env.KEY);
const keyOld = arg("key-old", process.env.KEY_OLD); // after a key change: the old key still works and gets the new one
const cloud = arg("cloud", process.env.CLOUD || ""); // the cloud's address, passed on to the Kindle
const port = +arg("port", 8787);
const assets = arg("assets", join(dataDir, "public"));
const tz = arg("tz", Intl.DateTimeFormat().resolvedOptions().timeZone || "Europe/London");
if (!key) {
  console.error("desk-buddy-server: --key is required");
  process.exit(2);
}
mkdirSync(dataDir, { recursive: true });

// ----- D1, on SQLite -----
const sqlite = new Database(join(dataDir, "desk-buddy.sqlite"), { create: true });
sqlite.exec("pragma journal_mode = wal");
sqlite.exec(`create table if not exists tasks(
  id integer primary key, profile text not null, text text not null,
  done integer not null default 0, pinned integer not null default 0,
  tag text, pos real not null, created real not null,
  parent_id integer, notes text, done_at real);
create table if not exists kv(k text primary key, v text not null);`);
try { sqlite.exec("alter table tasks add column done_at real"); } catch {} // older databases

const clean = (args) => args.map((a) => (a === undefined ? null : typeof a === "boolean" ? (a ? 1 : 0) : a));

class Statement {
  constructor(sql, args = []) {
    this.sql = sql;
    this.args = args;
  }
  bind(...args) {
    return new Statement(this.sql, clean(args));
  }
  runSync() {
    const r = sqlite.query(this.sql).run(...this.args);
    return { success: true, meta: { changes: r.changes, last_row_id: Number(r.lastInsertRowid) } };
  }
  async run() {
    return this.runSync();
  }
  async first(col) {
    const row = sqlite.query(this.sql).get(...this.args);
    return row == null ? null : col ? row[col] : row;
  }
  async all() {
    return { success: true, results: sqlite.query(this.sql).all(...this.args) };
  }
}

const DB = {
  prepare: (sql) => new Statement(sql),
  async batch(statements) {
    return sqlite.transaction(() => statements.map((s) => s.runSync()))();
  },
};

const env = { DB, KEY: key, TZ: tz, LOCAL: "1", CLOUD: cloud, ...(keyOld ? { KEY_OLD: keyOld } : {}) };
const ctx = { waitUntil: (p) => Promise.resolve(p).catch((e) => console.error("background:", e)), passThroughOnException() {} };

// ----- HTTP -----
Bun.serve({
  port,
  hostname: "0.0.0.0",
  async fetch(request) {
    const path = new URL(request.url).pathname;
    // the Kindle's updates and the phone page's fonts: plain files, no key (as on Cloudflare)
    if ((path.startsWith("/kindle/") || path.startsWith("/fonts/") || path.startsWith("/windows/")) && !path.includes("..")) {
      const f = Bun.file(join(assets, path));
      if (await f.exists()) return new Response(f);
    }
    return worker.fetch(request, env, ctx);
  },
});

// ----- finding this computer from the Kindle -----
const name = hostname().replace(/\.local$/, "");
await Bun.udpSocket({
  port: port + 1,
  socket: {
    data(sock, buf, rport, raddr) {
      if (buf.toString().trim() === "desk-buddy?") sock.send(JSON.stringify({ name, port }), rport, raddr);
    },
  },
});

console.log(`Desk Buddy local server: http://${name}.local:${port} (data in ${dataDir}, time zone ${tz})`);
