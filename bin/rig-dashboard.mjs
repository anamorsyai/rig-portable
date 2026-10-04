#!/usr/bin/env node
// rig-dashboard.mjs — control server + dashboard SPA host for the shim rig.
// Zero deps. Node >= 18. Port :9087, binds 0.0.0.0 (LAN).
// Controls: shim conf (/etc/conf.d/anthropic-shim), claude settings
// (~/.claude/settings.json env), snapshots/revert, proxy restart/log/test.
// Masters are kept in sync (secrets master + template settings) after writes.
import http from "node:http";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import net from "node:net";
import { randomBytes } from "node:crypto";
import { execFile } from "node:child_process";

const PORT = +(process.env.DASH_PORT || 9087);
const HOST = process.env.DASH_HOST || "127.0.0.1";
const CONF = process.env.SHIM_CONF || "/etc/conf.d/anthropic-shim";
const HOME = os.homedir();
const SETTINGS = process.env.SETTINGS_JSON || path.join(HOME, ".claude", "settings.json");
const REPO = process.env.RIG_REPO || path.join(HOME, "workspaces", "rig-portable");
const SECRETS_MASTER = path.join(REPO, "secrets", "anthropic-shim.conf");
const TEMPLATE_SETTINGS = path.join(REPO, "claude-official", "settings.json");
const SNAPSHOT_DIR = process.env.SNAPSHOT_DIR || path.join(HOME, ".claude", "dashboard-snapshots");
const LOG_FILE = "/var/log/shim.log";
const PID_FILE = "/run/anthropic-shim.pid";
const SHIM_TEST_URL = "http://127.0.0.1:9086/v1/messages";
const KEEP_SNAPSHOTS = 50;
const RIG_STATE = process.env.RIG_STATE || path.join(SNAPSHOT_DIR, "rig-state.json");
const DASH_PIDFILE = "/run/rig-dashboard.pid";
const DEFAULT_BIFROST_CMD = "rc-service bifrost restart";

// Rig-wide service registry: every daemon the panel should see + control.
// type: "openrc" (rc-service managed), "tcp" (port probe), "self" (this dashboard —
//       restarting it is deferred so we don't kill our own reply socket mid-response)
// probe: {port} TCP-bind check; {pidfile} liveness via PID.
// restart: cli string OR {argv:[cmd,...]}. Only type "openrc" gets rc-service restart.
const SERVICES = [
  { name: "anthropic-shim", label: "Shim proxy",   type: "openrc", probe: { port: 9086 }, pidfile: "/run/anthropic-shim.pid", service: "anthropic-shim", role: "Anthropic→OpenAI failover proxy" },
  { name: "tls-relay",      label: "TLS relay",    type: "tcp",    probe: { port: 9097 }, role: "zen TLS-fingerprint relay (node TLS — go's crypto/tls gets zen 403)" },
  { name: "rig-dashboard",  label: "Dashboard",    type: "self",   probe: { port: 9087 }, pidfile: DASH_PIDFILE, service: "rig-dashboard", role: "This control panel" },
  { name: "bifrost",        label: "Bifrost",      type: "openrc", probe: null,                                           service: "bifrost", role: "Second proxy unit (unregistered)" },
  { name: "burp",           label: "Burp MCP",     type: "tcp",    probe: { port: 9876 }, role: "Raw HTTP (burp MCP)" },
  { name: "caido",          label: "Caido MCP",    type: "tcp",    probe: { port: 3333 }, role: "Traffic analysis MCP" },
  { name: "browser",        label: "Browser ctrl", type: "tcp",    probe: { port: 8089 }, role: "Real-browser automation" },
];

function tcpAlive(port, host = "127.0.0.1", timeout = 900) {
  return new Promise((res) => {
    if (!port) return res(false);
    const s = net.connect(port, host);
    let done = false;
    const close = (v) => { if (!done) { done = true; try { s.destroy(); } catch {} res(v); } };
    s.setTimeout(timeout, () => close(false));
    s.on("connect", () => close(true));
    s.on("error", () => close(false));
  });
}
async function probeAlive(svc) {
  if (svc.probe && svc.probe.port) return tcpAlive(svc.probe.port);
  if (svc.pidfile) {
    try { const pid = +fs.readFileSync(svc.pidfile, "utf8").trim(); return process.kill(pid, 0); }
    catch { return false; }
  }
  return false;
}
async function serviceStatus() {
  const out = [];
  for (const svc of SERVICES) {
    const pid = svc.pidfile ? (() => { try { return +fs.readFileSync(svc.pidfile, "utf8").trim(); } catch { return null; } })() : null;
    out.push({
      name: svc.name, label: svc.label, type: svc.type, role: svc.role,
      running: await probeAlive(svc),
      pid, port: svc.probe && svc.probe.port || null,
      restartable: svc.type === "openrc" || svc.type === "self",
    });
  }
  return out;
}
async function restartService(name) {
  const svc = SERVICES.find(s => s.name === name);
  if (!svc) return { ok: false, error: "unknown service: " + name };
  if (!svc.restartable) return { ok: false, error: svc.label + " is not restart-managed (probe-only)" };
  if (svc.type === "openrc") {
    const r = await run("rc-service", [svc.service, "restart"], 20000);
    return { ok: r.ok, cmdOut: (r.stdout + r.stderr).trim() };
  }
  if (svc.type === "self") {
    // Defer: spawn rc-service restart of OUR OWN unit detached; reply first, then exit is fine.
    const r = await run("rc-service", ["rig-dashboard", "restart"], 0).catch(e => ({ ok: false }));
    return { ok: true, note: r && r.ok === false ? "restart command scheduled — panel will reconnect" : "restarting… panel will reconnect" };
  }
  return { ok: false, error: "unsupported type" };
}

/* ---------- persistent rig state (budget + alert rules) ---------- */
function loadRigState() {
  try { return JSON.parse(fs.readFileSync(RIG_STATE, "utf8")); } catch { return { budget: {}, alerts: [] }; }
}
function saveRigState(st) {
  fs.mkdirSync(path.dirname(RIG_STATE), { recursive: true });
  const tmp = RIG_STATE + ".tmp-" + process.pid;
  fs.writeFileSync(tmp, JSON.stringify(st, null, 2));
  fs.renameSync(tmp, RIG_STATE);
}

fs.mkdirSync(SNAPSHOT_DIR, { recursive: true });

/* ---------- logging ---------- */
const log = (...a) => console.log(new Date().toISOString(), ...a);

/* ---------- config file IO ---------- */
function readFileSafe(p) { try { return fs.readFileSync(p, "utf8"); } catch { return null; } }

// parse shell-sourced conf into { exports: {KEY: value}, quotes: {KEY: "'"|'"'} }
// Handles: KEY="value", KEY='value', KEY=unquoted, and inline comments after the
// closing quote: KEY="value"   # comment  -> value = "value" (comment dropped).
function parseConf(text) {
  const exports = {}, quotes = {};
  for (const l of text.split("\n")) {
    if (!l.startsWith("export ")) continue;
    const eq = l.indexOf("=", 7);
    if (eq < 0) continue;
    const key = l.slice(7, eq).trim();
    let raw = l.slice(eq + 1).trim();
    // A trailing # starts a shell comment. Only split on a # that is preceded by
    // whitespace (e.g. `"15000"   # floor...`), never one inside the value, so a
    // key/value with no comment is untouched.
    const hash = raw.search(/\s+#/);
    if (hash >= 0) raw = raw.slice(0, hash).trim();
    let q = null;
    if ((raw.startsWith('"') && raw.endsWith('"')) || (raw.startsWith("'") && raw.endsWith("'"))) {
      q = raw[0]; raw = raw.slice(1, -1);
    }
    exports[key] = raw; quotes[key] = q || '"';
  }
  return { exports, quotes };
}

function renderConf(exports, quotes) {
  const text = readFileSafe(CONF) || "";
  const out = [];
  const lines = text.split("\n");
  const done = new Set();
  for (const l of lines) {
    if (l.startsWith("export ")) {
      const eq = l.indexOf("=", 7);
      if (eq >= 0) {
        const key = l.slice(7, eq).trim();
        if (key in exports && !done.has(key)) {
          const q = quotes[key] || '"';
          const v = exports[key];
          // never allow a value containing the quote char to break the shell line
          const safe = String(v).replaceAll(q, "\\" + q);
          out.push(`export ${key}=${q}${safe}${q}`);
          done.add(key);
          continue;
        }
      }
    }
    out.push(l);
  }
  // append newly added keys (not already present)
  for (const [k, v] of Object.entries(exports)) {
    if (!done.has(k)) {
      const q = quotes[k] || '"';
      out.push(`export ${k}=${q}${String(v).replaceAll(q, "\\" + q)}${q}`);
    }
  }
  return out.join("\n");
}

function writeConf(exports, quotes) {
  const before = readFileSafe(CONF) || "";
  const next = renderConf(exports, quotes);
  if (next === before) return false;
  // atomic write
  const tmp = CONF + ".tmp-" + process.pid;
  fs.writeFileSync(tmp, next);
  fs.renameSync(tmp, CONF);
  // keep secrets master synced
  try {
    fs.mkdirSync(path.dirname(SECRETS_MASTER), { recursive: true });
    fs.copyFileSync(CONF, SECRETS_MASTER);
  } catch (e) { log("secrets master sync failed:", e.message); }
  return true;
}

// update one conf var (value may be string). returns true if changed
function setConfVar(key, value, { singleQuote = false } = {}) {
  const text = readFileSafe(CONF);
  if (!text) return false;
  const { exports, quotes } = parseConf(text);
  if (singleQuote) quotes[key] = "'";
  const before = exports[key];
  exports[key] = String(value);
  const changed = before !== exports[key];
  if (changed) writeConf(exports, quotes);
  return changed;
}

function getConf() {
  const text = readFileSafe(CONF);
  if (!text) return { exports: {}, quotes: {}, upProviders: {}, chain: [], knownModels: [] };
  const { exports, quotes } = parseConf(text);
  let upProviders = {};
  try { upProviders = JSON.parse(exports.UPSTREAMS || "{}"); } catch {}
  const chain = (exports.FALLBACK_CHAIN || "").split(",").map(s => s.trim()).filter(Boolean);
  const knownModels = (exports.KNOWN_MODELS || "").split(",").map(s => s.trim()).filter(Boolean);
  return { exports, quotes, upProviders, chain, knownModels };
}

/* ---------- masking ---------- */
// Provider secrets are matched by naming CONVENTION (every one of them ends in
// "_KEY", e.g. DAHL_KEY, BAI_KEY, BRAVE_KEY) rather than a hardcoded list —
// a hardcoded set silently stops covering a var the moment a new provider is
// added to the conf (observed live: BAI_KEY and BRAVE_KEY are real secrets in
// the current conf that a stale hardcoded list left completely unmasked in
// both /api/state and the default /api/config/raw view).
function isKeyVar(k) { return /_KEY$/.test(k); }
function mask(v) {
  if (v == null || v === "") return "";
  if (v.length <= 10) return "***";
  return v.slice(0, 5) + "·····" + v.slice(-4);
}
function maskedExports(exports) {
  const out = { ...exports };
  for (const k of Object.keys(out)) if (isKeyVar(k)) out[k] = mask(out[k]);
  return out;
}

/* ---------- auth ---------- */
// Auth model: if DASH_TOKEN is not set, auto-generate one and persist it to a
// root-only file so the key-exposing API is NEVER open by default AND the
// operator can still retrieve the token. Operators may set DASH_TOKEN in
// /etc/init.d/rig-dashboard to use a stable token instead.
const DASH_TOKEN_FILE = "/run/rig-dashboard.token";
function loadOrMakeToken() {
  const envTok = process.env.DASH_TOKEN;
  if (envTok) return envTok;
  try {
    const seen = fs.readFileSync(DASH_TOKEN_FILE, "utf8").trim();
    if (seen) return seen;
  } catch {}
  const tok = randomBytes(24).toString("hex");
  try {
    fs.writeFileSync(DASH_TOKEN_FILE, tok + "\n", { mode: 0o600 });
    fs.chmodSync(DASH_TOKEN_FILE, 0o600);
  } catch (e) {}
  return tok;
}
const DASH_TOKEN = loadOrMakeToken();
function randomToken(len = 24) {
  const b = randomBytes(len).toString("hex");
  return b;
}
function authRequired() { return !!DASH_TOKEN; }
function authorized(req) {
  if (!authRequired()) return true;
  const h = req.headers["authorization"] || "";
  const m = h.match(/^Bearer\s+(.+)$/i);
  const tok = m ? m[1].trim() : "";
  return !!(tok && tok === DASH_TOKEN);
}

/* ---------- settings.json ---------- */
function getSettings() {
  const t = readFileSafe(SETTINGS) || "{}";
  try { return JSON.parse(t); } catch { return { env: {} }; }
}
function writeSettings(settings) {
  const tmp = SETTINGS + ".tmp-" + process.pid;
  fs.writeFileSync(tmp, JSON.stringify(settings, null, 2));
  fs.renameSync(tmp, SETTINGS);
  // sync template env block (keep template's own extra keys)
  try {
    const t = readFileSafe(TEMPLATE_SETTINGS);
    if (t) {
      const tmp2 = JSON.parse(t);
      tmp2.env = { ...tmp2.env, ...(settings.env || {}) };
      fs.writeFileSync(TEMPLATE_SETTINGS, JSON.stringify(tmp2, null, 2));
    }
  } catch (e) { log("template settings sync failed:", e.message); }
}

/* ---------- snapshots ---------- */
function snapId() {
  const d = new Date();
  const p = (n) => String(n).padStart(2, "0");
  return `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}${p(d.getSeconds())}`;
}
function listSnapshots() {
  try {
    return fs.readdirSync(SNAPSHOT_DIR).filter(f => f.endsWith(".json")).sort().map(f => {
      const p = path.join(SNAPSHOT_DIR, f);
      const st = fs.statSync(p);
      let label = f.replace(".json", "");
      try { const j = JSON.parse(fs.readFileSync(p, "utf8")); if (j.label) label = j.label; } catch {}
      return { id: f.replace(".json", ""), file: f, mtime: st.mtimeMs, label };
    }).reverse();
  } catch { return []; }
}
function makeSnapshot(label = "") {
  const id = snapId() + "-" + Math.random().toString(36).slice(2, 6);
  const conf = readFileSafe(CONF) || "";
  const settings = readFileSafe(SETTINGS) || "";
  const secretMaster = readFileSafe(SECRETS_MASTER) || "";
  const obj = { id, label, at: Date.now(), conf, settings, secretMaster };
  const file = path.join(SNAPSHOT_DIR, id + ".json");
  fs.writeFileSync(file, JSON.stringify(obj, null, 2));
  // prune old
  const all = fs.readdirSync(SNAPSHOT_DIR).filter(f => f.endsWith(".json")).sort();
  for (const f of all.slice(0, Math.max(0, all.length - KEEP_SNAPSHOTS))) {
    try { fs.unlinkSync(path.join(SNAPSHOT_DIR, f)); } catch {}
  }
  return id;
}
// Snapshot ids are always machine-generated (snapId()+"-"+random, see
// makeSnapshot) and never contain "/" or "..", but this id also arrives
// straight from client input (GET /api/snapshot/:id, POST /api/revert
// {id}) — reject anything outside that safe shape before it reaches a path
// join, so a crafted id can't escape SNAPSHOT_DIR and read (or, via revert,
// feed attacker-influenced content back into) an arbitrary .json file.
function isSafeSnapshotId(id) { return typeof id === "string" && /^[A-Za-z0-9_-]+$/.test(id); }
function loadSnapshot(id) {
  if (!isSafeSnapshotId(id)) return null;
  const file = path.join(SNAPSHOT_DIR, id + ".json");
  try { return JSON.parse(fs.readFileSync(file, "utf8")); } catch { return null; }
}
function revertTo(id) {
  const snap = loadSnapshot(id);
  if (!snap) return { ok: false, error: "snapshot not found" };
  makeSnapshot("auto before revert:" + id);
  if (snap.conf) { const tmp = CONF + ".tmp-" + process.pid; fs.writeFileSync(tmp, snap.conf); fs.renameSync(tmp, CONF); fs.copyFileSync(CONF, SECRETS_MASTER); }
  if (snap.settings) { const tmp = SETTINGS + ".tmp-" + process.pid; fs.writeFileSync(tmp, snap.settings); fs.renameSync(tmp, SETTINGS); }
  const alive = isProxyAlive();
  return { ok: true, proxyWasAlive: alive };
}

/* ---------- proxy control ---------- */
function isProxyAlive() {
  try {
    return fs.existsSync(PID_FILE) && process.kill(+fs.readFileSync(PID_FILE, "utf8").trim(), 0);
  } catch { return false; }
}
function proxyPid() {
  try { return +fs.readFileSync(PID_FILE, "utf8").trim(); } catch { return null; }
}
function run(cmd, args, timeout = 15000) {
  return new Promise((res) => {
    execFile(cmd, args, { timeout }, (err, stdout, stderr) => {
      res({ ok: !err, code: err ? err.code : 0, stdout: (stdout || ""), stderr: (stderr || "") });
    });
  });
}
async function restartProxy() {
  // OpenRC only exists on real Alpine hosts — AlpineTerm/proot has none, so fall
  // back to pkill + nohup restart (same fallback the oc panel uses).
  let r = await run("rc-service", ["anthropic-shim", "restart"], 20000).catch(() => ({ ok: false }));
  if (!r.ok) {
    const haveRC = await run("sh", ["-c", "command -v rc-service >/dev/null && [ -d /etc/init.d ] && echo yes"], 2000).catch(() => ({ stdout: "" }));
    if (!(haveRC.stdout || "").includes("yes")) {
      // the shared restart script: pkill pattern matches only real binary paths
      // (a broad pattern matches this dashboard's own argv and kills the restart)
      const script = REPO + "/bin/restart-shim.sh";
      r = await run("sh", [script], 20000);
      if (!r.ok) r = await run("sh", ["/root/claude-code-hunting-rig-portable/bin/restart-shim.sh"], 20000);
      r.stdout = r.stdout || "restarted via restart-shim.sh (no OpenRC)";
    }
  }
  // health: wait for :9086 to answer
  let alive = false, health = null;
  for (let i = 0; i < 20; i++) {
    await new Promise(r2 => setTimeout(r2, 250));
    alive = isProxyAlive();
    if (alive) {
      try {
        const ctl = new AbortController();
        const t = setTimeout(() => ctl.abort(), 1500);
        const resp = await fetch(SHIM_TEST_URL.replace("/v1/messages", "/"), { signal: ctl.signal });
        clearTimeout(t);
        if (resp.ok) { health = "ok"; break; }
      } catch {}
    }
  }
  return { ok: r.ok, cmdOut: (r.stdout + r.stderr).trim(), alive, health };
}
// Bounded tail read: only pull the trailing bytes likely to cover `lines`
// lines, instead of loading the whole file (up to the 20MB shim-logrotate
// cap) just to keep the last handful. This is called on nearly every
// dashboard API (stats/usage/log/alerts) plus, now, every periodic alert
// evaluation — reading 20MB per call adds up fast at that frequency.
function tailLog(lines = 60) {
  try {
    const fd = fs.openSync(LOG_FILE, "r");
    try {
      const { size } = fs.fstatSync(fd);
      // Real log lines here average well under 300 bytes; pad generously.
      const wanted = Math.min(size, Math.max(65536, lines * 300));
      const buf = Buffer.alloc(wanted);
      fs.readSync(fd, buf, 0, wanted, size - wanted);
      let text = buf.toString("utf8");
      // A partial read starts mid-line — drop the (possibly truncated) first
      // line, unless the read actually covered the whole file from byte 0.
      if (wanted < size) text = text.slice(text.indexOf("\n") + 1);
      return text.split("\n").filter(Boolean).slice(-lines);
    } finally {
      fs.closeSync(fd);
    }
  } catch { return []; }
}

/* ---------- stats (derived from shim log) ---------- */
// Strip the trailing request-id prefix the Go shim writes between the timestamp
// and the message (e.g. "2026-... [reqdl3c3gzvl86p9] -> ..."), so the same regexes
// parse both the old Node log format ("attempt 1 model=...") and the new Go format
// ("[req...] attempt model=..."). Returns {log, ts}.
function splitLogLine(l) {
  const m = l.match(/^(\S+)/);
  const ts = m ? Date.parse(m[1]) : NaN;
  let rest = m ? l.slice(m[1].length).trimStart() : l;
  rest = rest.replace(/^\[req[^\]]*\]\s*/, "").replace(/^\[-\]\s*/, "");
  return { rest, ts };
}

function computeStats() {
  const lines = tailLog(5000);
  let requests = 0, attempts = 0, e429 = 0, e400 = 0, e5xx = 0, connectFails = 0, cooldowns = 0;
  const perProvider = new Map();
  const buckets = new Array(24).fill(0);
  const now = Date.now();
  for (const l of lines) {
    const { rest, ts } = splitLogLine(l);
    // request line: "-> <requested> => <chain|...> tools: N msgs: N"
    if (rest.startsWith("-> ") && rest.includes(" => ")) {
      requests++;
      if (!Number.isNaN(ts)) {
        const hr = Math.floor((now - ts) / 3600e3);
        if (hr >= 0 && hr < 24) buckets[23 - hr]++;
      }
      continue;
    }
    // attempt line: "attempt N model=..." (old Node) OR "attempt model=..." (Go).
    const am = rest.match(/^attempt(?:\s+\d+)?\s+model=([^@\s]+)@([^#\s]+)/);
    if (am) { attempts++; perProvider.set(am[2], (perProvider.get(am[2]) || 0) + 1); continue; }
    // upstream error: "upstream <code>:" or "nonStream upstream <code>:"
    const em = rest.match(/(?:nonStream\s+)?upstream\s+(\d{3})/);
    if (em) { const c = +em[1]; if (c === 429) e429++; else if (c === 400 || c === 404) e400++; else if (c >= 500) e5xx++; continue; }
    if (/^connect fail/i.test(rest)) { connectFails++; continue; }
    if (/cooldown/i.test(rest)) { cooldowns++; }
  }
  return {
    requests, attempts, failovers: Math.max(0, attempts - requests),
    e429, e400, e5xx, connectFails, cooldowns,
    perProvider: [...perProvider.entries()].map(([provider, n]) => ({ provider, attempts: n })).sort((a, b) => b.attempts - a.attempts),
    hourly: buckets,
    linesAnalyzed: lines.length,
  };
}

/* ---------- token usage meter (derived from shim "usage ..." lines) ---------- */
function computeUsage() {
  const lines = tailLog(5000);
  let requests = 0;
  const perProvider = new Map(), perModel = new Map();
  const buckets = new Array(24).fill(0);
  const now = Date.now();
  for (const l of lines) {
    // "usage at=<provider|- > model=<model|- > uIn=<in> uOut=<out> ms=<ms>" ; optional
    // [req...] prefix from the Go shim is tolerated by matching anywhere in the line.
    const m = l.match(/usage at=(\S+) model=(\S+) uIn=(\d+) uOut=(\d+) ms=(\d+)/);
    if (!m) continue;
    requests++;
    const at = m[1], model = m[2];
    const inp = +m[3], out = +m[4];
    // resolve hour bucket from the log line timestamp
    const tm = l.match(/^(\S+)/);
    const ts = tm ? Date.parse(tm[1]) : NaN;
    if (!Number.isNaN(ts)) {
      const hr = Math.floor((now - ts) / 3600e3);
      if (hr >= 0 && hr < 24) buckets[23 - hr] += inp + out;
    }
    if (at !== "-") {
      let p = perProvider.get(at); if (!p) { p = { provider: at, in: 0, out: 0, total: 0 }; perProvider.set(at, p); }
      p.in += inp; p.out += out; p.total += inp + out;
    }
    if (model !== "-") {
      let q = perModel.get(model); if (!q) { q = { model, in: 0, out: 0, total: 0 }; perModel.set(model, q); }
      q.in += inp; q.out += out; q.total += inp + out;
    }
  }
  const pp = [...perProvider.values()].sort((a, b) => b.total - a.total);
  const pm = [...perModel.values()].sort((a, b) => b.total - a.total);
  const r = pp.reduce((a, p) => ({ in: a.in + p.in, out: a.out + p.out }), { in: 0, out: 0 });
  return {
    requests,
    totalIn: r.in,
    totalOut: r.out,
    total: r.in + r.out,
    perProvider: pp,
    perModel: pm,
    hourlyTokens: buckets,
  };
}

/* ---------- budget (spend caps) ---------- */
const TOK_COST_KEYS = ["BUDGET_DAILY_TOKENS", "BUDGET_PROVIDER_TOKENS", "BUDGET_REQ_PER_MIN"];
function computeBudget() {
  const st = loadRigState();
  const { exports } = getConf();
  const usage = computeUsage();
  const dailyCap = +(exports.BUDGET_DAILY_TOKENS || st.budget.dailyTokens) || 0;
  const provCaps = (() => { try { return JSON.parse(exports.BUDGET_PROVIDER_TOKENS || "{}"); } catch { return st.budget.providers || {}; } })();
  const reqMin = +(exports.BUDGET_REQ_PER_MIN || st.budget.reqPerMin) || 0;
  const providers = (usage.perProvider || []).map(p => ({
    provider: p.provider, in: p.in, out: p.out, total: p.total,
    cap: provCaps[p.provider] || 0,
    over: !!(provCaps[p.provider] && p.total >= provCaps[p.provider]),
  }));
  // aggregate a simple per-hour req rate from the last few minutes of log
  let latestReq = 0;
  const now = Date.now();
  const lines = tailLog(500);
  for (const l of lines) {
    const m = l.match(/^(\S+)/);
    const ts = m ? Date.parse(m[1]) : NaN;
    if (!Number.isNaN(ts) && (now - ts) < 5 * 60e3 && l.includes(" => ")) latestReq++;
  }
  return {
    dailyCap, dailyTokens: usage.total, dailyOver: !!(dailyCap && usage.total >= dailyCap),
    reqPerMin: reqMin, reqRecent: Math.round(latestReq / 5), // per-min over last 5m
    providers,
  };
}

/* ---------- alert rules + auto-recovery ---------- */
const DEFAULT_ALERT_RULES = [
  { id: "svc-anthropic-shim", name: "Shim down", kind: "service", target: "anthropic-shim", action: "restart", enabled: true, msg: "shim proxy unreachable" },
  { id: "svc-bifrost",       name: "Bifrost down", kind: "service", target: "bifrost", action: "restart", enabled: false, msg: "bifrost unreachable" },
  { id: "svc-dashboard",     name: "Dashboard down", kind: "service", target: "rig-dashboard", action: "restart", enabled: false, msg: "dashboard unreachable" },
  { id: "storm-failover",    name: "Failover storm", kind: "rate", field: "failovers", over: 30, windowS: 300, metric: "failovers/5m", action: "none", enabled: true, msg: "failover storm" },
  { id: "drift-5xx",         name: "5xx spike", kind: "rate", field: "e5xx", over: 20, windowS: 300, metric: "5xx/5m", action: "none", enabled: true, msg: "5xx upstream spike" },
  { id: "storm-429",         name: "429 storm", kind: "rate", field: "e429", over: 20, windowS: 300, metric: "429/5m", action: "none", enabled: true, msg: "429 upstream storm" },
  { id: "drift-connect",     name: "Connect fails", kind: "rate", field: "connectFails", over: 30, windowS: 300, metric: "connect-fail/5m", action: "none", enabled: true, msg: "connect failures" },
];
function alertsState() {
  const st = loadRigState();
  const rules = (st.alerts && st.alerts.length ? st.alerts : DEFAULT_ALERT_RULES);
  const s = computeStats();
  const svcs = { }; // name -> running
  // evaluate service rules sync via pidfile where possible; TCP probes are async — handled below
  return { rules, budget: computeBudget(), stats: { failovers: s.failovers, e5xx: s.e5xx, e429: s.e429, connectFails: s.connectFails } };
}
// Evaluate alert rules against live state; persist fired events; apply auto-actions.
async function evaluateAlerts(force = false) {
  const st = loadRigState();
  const rules = (st.alerts && st.alerts.length ? st.alerts : DEFAULT_ALERT_RULES);
  const run = st.alerts && st.alerts.length ? st.alerts : DEFAULT_ALERT_RULES;
  const s = computeStats();
  const svcs = await serviceStatus();
  const now = Date.now();
  const events = [];
  for (const r of run) {
    if (!r.enabled) continue;
    if (r.kind === "service") {
      const svc = svcs.find(x => x.name === r.target);
      const running = svc ? svc.running : true;
      if (!running && r.action === "restart" && svc && svc.restartable) {
        const rr = await restartService(r.target);
        const okNow = !!(await serviceStatus()).find(x => x.name === r.target).running;
        events.push({ at: now, rule: r.name, kind: "service", target: r.target, msg: r.msg + " → auto-restarted" + (okNow ? " (ok)" : " (still down)"), action: "restart", ok: okNow });
      } else if (!running) {
        events.push({ at: now, rule: r.name, kind: "service", target: r.target, msg: r.msg, action: "none" });
      }
      continue;
    }
    // rate rules: estimate rate from the recent log tail (approx — recompute window)
    const field = r.field;
    const recent = (() => { // count events in windowS from log tail
      const windowS = r.windowS || 300;
      const now2 = Date.now(); let c = 0;
      for (const l of tailLog(4000)) {
        const m = l.match(/^(\S+)/);
        const ts = m ? Date.parse(m[1]) : NaN;
        if (Number.isNaN(ts) || (now2 - ts) > windowS * 1e3) continue;
        if (field === "failovers" && /^attempt\s+\d+\s+model=/i.test(l)) c++;
        else if (field === "e5xx" && /^upstream\s+5\d\d/i.test(l)) c++;
        else if (field === "e429" && /^upstream\s+429/i.test(l)) c++;
        else if (field === "connectFails" && /^connect fail/i.test(l)) c++;
      }
      return c;
    })();
    if (recent > (r.over || 0)) {
      events.push({ at: now, rule: r.name, kind: "rate", metric: r.metric, value: recent, msg: r.msg + ` (${recent} in window)`, action: r.action || "none" });
    }
  }
  const prev = st.events || [];
  st.events = [...events, ...prev].slice(0, 200);
  const live = (await serviceStatus());
  st.lastEval = now;
  // re-run budget over a fresh stats read
  st.lastBudget = computeBudget();
  saveRigState(st);
  return { fired: events, services: live, budget: st.lastBudget };
}

async function loadAlertsFull() {
  const st = loadRigState();
  const rules = (st.alerts && st.alerts.length ? st.alerts : DEFAULT_ALERT_RULES);
  return {
    rules,
    events: st.events || [],
    budget: st.lastBudget || computeBudget(),
    stats: alertsState().stats,
    services: await serviceStatus(),
    lastEval: st.lastEval || 0,
  };
}

/* ---------- test request ----------
   Two-stage probe so the UI tells the truth about a model:
   - "pinned": force the model to its own provider (if it has one in UPSTREAMS)
     and verify an actual expected word appears in the output bytes. This is the
     only way to detect a burned-out model — the chain would otherwise mask it.
   - "chain": normal chain path (default model selection), measures first-byte.
*/
const STREAM_EVENT = "data:";
const okRe = /"text":"([^"]*)"|content":"OK"|"text"\s*:\s*"([^\"]*\bOK\b[^\"]*)"/i;

// Stateful SSE frame buffer: handles a data: JSON frame split across
// network chunks (TextDecoder streamed). Feed .push(decodedChunk) then
// drain complete JSON frames. Returns accumulated content text across frames.
function makeSseParser() {
  let buf = "";
  let text = "";   // content
  let reason = ""; // reasoning
  const t = (o) => (o == null ? "" : String(o));
  return {
    // returns the content text accumulated so far after consuming this chunk
    push(chunk) {
      buf += chunk;
      let idx;
      while ((idx = buf.indexOf("\n")) >= 0) {
        const line = buf.slice(0, idx);
        buf = buf.slice(idx + 1);
        let s = line.endsWith("\r") ? line.slice(0, -1) : line;
        if (s.startsWith(STREAM_EVENT)) s = s.slice(STREAM_EVENT.length).trim();
        if (!s || !s.startsWith("{")) continue;
        try {
          const j = JSON.parse(s);
          const d = (j.choices && j.choices[0] && j.choices[0].delta) || j.delta || j;
          const c = d.text || d.content || "";
          const r = d.reasoning_content || "";
          if (c) text += t(c);
          if (r) reason += t(r);
        } catch {}
      }
      return text || reason;
    },
    final() { return text || reason; },
  };
}

// legacy single-shot helper (also used by direct probe tail) — kept for content
function parseSseText(chunk) {
  const p = makeSseParser();
  return p.push(chunk) || p.final();
}
function expectedFor(model) {
  if (/^big-pickle(@|$)/.test(model) || /^hy3-free(@|$)/.test(model)) return "OK";
  return null;
}
// expand a possible "$ENV_VAR" reference using values already loaded in this
// process (the dashboard service sources the same conf as the shim, so key
// variables like UPSTREAM_KEY / DAHL_KEY are present here).
function deref(v) {
  if (v == null) return "";
  const s = String(v).trim();
  if (s.startsWith("$")) return process.env[s.slice(1)] || "";
  return s;
}
function safeParseJson(t) { try { return JSON.parse(t); } catch { return null; } }
function testModel(model, maxTokens = 256, opts = {}) {
  return new Promise(async (resolve) => {
    const body = { model, max_tokens: maxTokens, messages: [{ role: "user", content: "Reply with exactly OK" }] };
    const stream = opts.stream !== false;
    if (stream) body.stream = true;
    const start = Date.now();
    const respond = (o) => resolve({ ...o, ms: Date.now() - start });
    let ctl, t;
    try {
      ctl = new AbortController();
      t = setTimeout(() => ctl.abort(), opts.timeout || 45000);
      const resp = await fetch(SHIM_TEST_URL, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify(body),
        signal: ctl.signal,
      });
      let hadContent = false, firstByte = null, outText = "";
      if (stream && resp.ok && resp.body) {
        const rd = resp.body.getReader();
        const dec = new TextDecoder();
        const sse = makeSseParser();
        for (;;) {
          const { done, value } = await rd.read();
          if (done) break;
          const s = dec.decode(value, { stream: true });
          if (firstByte == null && s.trim()) firstByte = Date.now() - start;
          outText = sse.push(s);
          if (!hadContent && outText.trim()) hadContent = true;
        }
      } else {
        const text = await resp.text().catch(() => "");
        if (firstByte == null) firstByte = Date.now() - start;
        const j = safeParseJson(text);
        if (j && j.content) {
          outText = (Array.isArray(j.content) ? j.content.map(b => b.text || "").join("") : String(j.content));
          hadContent = outText.trim().length > 0;
        } else {
          outText = text;
          hadContent = /"content"\s*:/.test(text) || text.trim().length > 0;
        }
      }
      clearTimeout(t);
      const expected = expectedFor(model);
      const ok = resp.ok && hadContent && (!expected || new RegExp(expected, "i").test(outText));
      const snippet = String(outText || "no generated text").slice(0, 140);
      respond({ ok, status: resp.status, firstByteMs: firstByte, error: ok ? undefined : (hadContent ? `expected "${expected}" in output` : "no usable content"), snippet });
    } catch (e) {
      if (t) clearTimeout(t);
      respond({ ok: false, status: 0, error: "error: " + e.message });
    }
  });
}

/* ---------- API ---------- */
function json(res, status, obj) {
  res.writeHead(status, { "Content-Type": "application/json" });
  res.end(JSON.stringify(obj));
}
const getBody = (req) => new Promise((resolve) => {
  let raw = ""; let size = 0;
  req.on("data", (c) => { size += c.length; if (size > 2_000_000) { req.destroy(); return; } raw += c; });
  req.on("end", () => { try { resolve(raw ? JSON.parse(raw) : {}); } catch { resolve(null); } });
  req.on("error", () => resolve(null));
});

function serveSpa(res) {
  const spaPath = path.join(path.dirname(new URL(import.meta.url).pathname), "..", "claude-official", "dashboard.html");
  let html = readFileSafe(spaPath);
  if (!html) {
    html = `<!doctype html><html><head><title>Rig Dashboard</title></head><body><h1>dashboard.html missing</h1>
<p>Expected at: <code>${spaPath}</code></p></body></html>`;
  }
  res.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
  res.end(html);
}

const state = () => {
  const { exports, quotes, upProviders, chain, knownModels } = getConf();
  const settings = getSettings();
  return {
    conf: maskedExports(exports),
    providers: upProviders,
    chain,
    knownModels,
    settingsEnv: settings.env || {},
    status: { proxyAlive: isProxyAlive(), pid: proxyPid(), logModified: (() => { try { return fs.statSync(LOG_FILE).mtimeMs; } catch { return 0; } })() },
  };
};

async function handleApi(req, res, url, body) {
  const P = url.pathname;
  if (req.method === "GET" && P === "/api/state") return json(res, 200, state());
  if (req.method === "GET" && P === "/api/log") {
    const q = new URLSearchParams(url.search);
    return json(res, 200, { lines: tailLog(+(q.get("n") || 60)) });
  }
  if (req.method === "GET" && P === "/api/snapshots") return json(res, 200, { snapshots: listSnapshots() });
  if (req.method === "GET" && P === "/api/config/raw") {
    // Secrets are redacted by default. Opt in with ?keys=1 to fetch real key material.
    const q2 = new URLSearchParams(url.search);
    const withKeys = q2.get("keys") === "1";
    let conf = readFileSafe(CONF);
    let secretsMaster = null;
    if (!withKeys && conf) {
      // mask every *_KEY line so the raw dump never leaks provider keys
      const { exports, quotes } = parseConf(conf);
      for (const k of Object.keys(exports)) if (isKeyVar(k)) exports[k] = mask(exports[k]);
      conf = renderConf(exports, quotes);
      secretsMaster = null;
    } else {
      secretsMaster = readFileSafe(SECRETS_MASTER);
    }
    return json(res, 200, { conf, settings: readFileSafe(SETTINGS), secretsMaster });
  }
  if (req.method === "GET" && P === "/api/stats") return json(res, 200, { ok: true, ...computeStats() });
  if (req.method === "GET" && P === "/api/usage") return json(res, 200, { ok: true, ...computeUsage() });
  if (req.method === "GET" && P === "/api/services") return json(res, 200, { ok: true, services: await serviceStatus() });
  if (req.method === "GET" && P === "/api/shim-health") {
    // The shim's own health endpoint carries live per-provider cooldown state
    // and response-cache stats that the log-derived stats/usage views can't
    // see (those only see completed requests, never in-progress backoff).
    // Proxied server-side (not fetched by the browser directly) to keep every
    // dashboard data source flowing through one origin/error-handling path.
    try {
      const ctl = new AbortController();
      const t = setTimeout(() => ctl.abort(), 2000);
      const resp = await fetch("http://127.0.0.1:9086/", { signal: ctl.signal });
      clearTimeout(t);
      if (!resp.ok) return json(res, 200, { ok: false, error: "shim returned HTTP " + resp.status });
      const j = await resp.json();
      return json(res, 200, { ok: true, ...j });
    } catch (e) {
      return json(res, 200, { ok: false, error: e.message });
    }
  }
  if (req.method === "GET" && P === "/api/budget") return json(res, 200, { ok: true, ...computeBudget(), conf: { BUDGET_DAILY_TOKENS: getConf().exports.BUDGET_DAILY_TOKENS || "", BUDGET_PROVIDER_TOKENS: getConf().exports.BUDGET_PROVIDER_TOKENS || "", BUDGET_REQ_PER_MIN: getConf().exports.BUDGET_REQ_PER_MIN || "" } });
  if (req.method === "GET" && P === "/api/alerts") return json(res, 200, { ok: true, ...(await loadAlertsFull()) });
  if (req.method === "GET" && P === "/api/auth") return json(res, 200, { ok: true, required: authRequired(), hasToken: !!DASH_TOKEN });
  if (req.method === "GET" && P.startsWith("/api/snapshot/")) {
    const id = decodeURIComponent(P.slice("/api/snapshot/".length));
    const snap = loadSnapshot(id);
    if (!snap) return json(res, 404, { ok: false, error: "snapshot not found" });
    return json(res, 200, { ok: true, id: snap.id, label: snap.label, at: snap.at, conf: snap.conf, settings: snap.settings });
  }

  if (req.method === "POST") {
    if (P === "/api/snapshot") {
      const id = makeSnapshot((body && body.label) || "manual");
      return json(res, 200, { ok: true, id });
    }
    if (P === "/api/revert") {
      if (!body || !body.id) return json(res, 400, { ok: false, error: "missing id" });
      const r = revertTo(body.id);
      if (!r.ok) return json(res, 404, r);
      if (r.proxyWasAlive) { await restartProxy(); }
      return json(res, 200, { ok: true });
    }
    if (P === "/api/restart") {
      const r = await restartProxy();
      return json(res, 200, { ok: true, ...r });
    }
    if (P === "/api/test") {
      const model = (body && body.model) || "";
      if (!model) return json(res, 400, { ok: false, error: "missing model" });
      if (body && body.direct) {
        // true direct probe: bypass the shim, hit the provider endpoint itself.
        // The chain can never mask a burned model this way.
        const { upProviders } = getConf();
        let provName = null, prov = null;
        for (const [p, x] of Object.entries(upProviders)) {
          if ((x.models || []).includes(model)) { provName = p; prov = x; break; }
        }
        if (!prov) return json(res, 200, { ok: false, status: 0, error: `no provider claims "${model}"` });
        const url = prov.url;
        if (!url) return json(res, 200, { ok: false, status: 0, error: "provider has no url" });
        const ua = deref(prov.ua) || "curl/8.5.0";
        const key = deref(prov.key || "");
        if (!key) return json(res, 200, { ok: false, status: 0, error: "provider key missing/unresolvable" });
        const headers = { "content-type": "application/json", authorization: "Bearer " + key };
        if (ua) headers["user-agent"] = ua;
        const pe = url.replace(/\/chat\/completions.*$/, "/chat/completions");
        const pbody = { model, max_tokens: 256, stream: true, messages: [{ role: "user", content: "Reply with exactly OK" }] };
        const start = Date.now();
        const resp2 = await fetch(pe, { method: "POST", headers, body: JSON.stringify(pbody), signal: AbortSignal.timeout(30000) }).catch(e => ({ fail: e.message }));
        if (resp2.fail) return json(res, 200, { ok: false, status: 0, error: resp2.fail });
        if (!resp2.ok) {
          const t2 = await resp2.text().catch(() => "");
          return json(res, 200, { ok: false, status: resp2.status, error: (t2 || "HTTP " + resp2.status).slice(0, 160) });
        }
        let outText = "", firstByteMs = null, streamErr = null;
        const rd = resp2.body.getReader(), dec = new TextDecoder();
        const sse = makeSseParser();
        for (;;) {
          const { done, value } = await rd.read();
          if (done) break;
          const s = dec.decode(value, { stream: true });
          if (firstByteMs == null && s.trim()) firstByteMs = Date.now() - start;
          outText = sse.push(s);
          // detect upstream errors surfaced in-stream (bare JSON or wrapped in SSE data:)
          if (!streamErr) {
            for (const ln of s.split("\n")) {
              let jsonLine = ln;
              if (ln.startsWith(STREAM_EVENT)) jsonLine = ln.slice(STREAM_EVENT.length).trim();
              jsonLine = jsonLine.trim();
              if (!jsonLine.startsWith("{")) continue;
              const j = safeParseJson(jsonLine);
              if (j && (j.error || j.type === "error")) {
                const e = j.error || {};
                streamErr = (e.message || e.type || j.message || "upstream error").slice(0, 200);
                break;
              }
            }
          }
        }
        if (streamErr) return json(res, 200, { ok: false, status: resp2.status, pinned: provName, direct: true, firstByteMs, error: streamErr, snippet: outText || "no text", ms: Date.now() - start });
        const expected = expectedFor(model);
        const okF = outText.trim().length > 0 && (!expected || new RegExp(expected, "i").test(outText));
        return json(res, 200, { ok: okF, status: resp2.status, pinned: provName, direct: true, firstByteMs, error: okF ? undefined : (outText.trim() ? `output lacks "${expected}"` : "no generated text"), snippet: (outText || "no text").slice(0, 140), ms: Date.now() - start });
      }
      const r = await testModel(model);
      return json(res, 200, { ok: true, ...r });
    }
    if (P === "/api/apply") {
      if (!body) return json(res, 400, { ok: false, error: "bad json body" });
      const autoSnapId = makeSnapshot("auto before apply");
      const { confVars = {}, settingsEnv = null, restart = false } = body;
      let changed = false;
      for (const [k, v] of Object.entries(confVars)) {
        if (k === "UPSTREAMS" || k === "UPSTREAMS_JSON") continue; // handled via providers
        // empty value on a key var => keep existing (never blank out a live secret)
        if (isKeyVar(k) && (v == null || v === "")) continue;
        if (setConfVar(k, v)) changed = true;
      }
      if (settingsEnv) {
        const s = getSettings();
        for (const [k, v] of Object.entries(settingsEnv)) {
          if (v === "" && k.startsWith("ANTHROPIC_")) { delete s.env[k]; changed = true; continue; }
          if (s.env[k] !== v && v != null) { s.env[k] = v; changed = true; }
        }
        if (changed) writeSettings(s);
      }
      if (changed && restart) await restartProxy();
      return json(res, 200, { ok: true, changed, snapshot: autoSnapId, state: state() });
    }
    if (P === "/api/providers") {
      if (!body || !body.providers) return json(res, 400, { ok: false, error: "missing providers" });
      const { exports, quotes } = getConf();
      makeSnapshot("auto before providers");
      exports.UPSTREAMS = JSON.stringify(body.providers);
      quotes.UPSTREAMS = "'";
      const changed = writeConf(exports, quotes);
      if (body.restart) await restartProxy();
      return json(res, 200, { ok: true, changed, state: state() });
    }
    if (P === "/api/chain") {
      if (!body || !Array.isArray(body.chain)) return json(res, 400, { ok: false, error: "missing chain array" });
      makeSnapshot("auto before chain");
      setConfVar("FALLBACK_CHAIN", body.chain.join(","), { singleQuote: true });
      if (body.restart) await restartProxy();
      return json(res, 200, { ok: true, state: state() });
    }
    if (P === "/api/service/restart") {
      if (!body || !body.name) return json(res, 400, { ok: false, error: "missing service name" });
      const r = await restartService(body.name);
      return json(res, r.ok ? 200 : 400, { ok: r.ok, ...r });
    }
    if (P === "/api/budget") {
      if (!body || typeof body !== "object") return json(res, 400, { ok: false, error: "bad body" });
      makeSnapshot("auto before budget");
      const st = loadRigState();
      if (body.dailyTokens != null) st.budget.dailyTokens = +body.dailyTokens || 0;
      if (body.providers != null && typeof body.providers === "object") st.budget.providers = body.providers;
      if (body.reqPerMin != null) st.budget.reqPerMin = +body.reqPerMin || 0;
      saveRigState(st);
      const applied = [];
      const { exports, quotes } = getConf();
      const set = (k, v) => { if (exports[k] !== String(v ?? "")) { exports[k] = String(v ?? ""); applied.push(k); } };
      set("BUDGET_DAILY_TOKENS", body.dailyTokens ?? st.budget.dailyTokens);
      set("BUDGET_PROVIDER_TOKENS", JSON.stringify(st.budget.providers || {}));
      set("BUDGET_REQ_PER_MIN", body.reqPerMin ?? st.budget.reqPerMin);
      if (applied.length) writeConf(exports, quotes);
      if (body.restart) await restartProxy();
      return json(res, 200, { ok: true, applied, state: { ...computeBudget() } });
    }
    if (P === "/api/alerts") {
      if (!body || !Array.isArray(body.rules)) return json(res, 400, { ok: false, error: "missing rules array" });
      makeSnapshot("auto before alerts");
      const st = loadRigState();
      st.alerts = body.rules;
      st.modifiedBy = "dashboard";
      saveRigState(st);
      return json(res, 200, { ok: true, rules: st.alerts });
    }
    if (P === "/api/alert/eval") {
      const r = await evaluateAlerts(true);
      return json(res, 200, { ok: true, ...r });
    }
    if (P === "/api/auth/set") {
      if (!body || !body.token) return json(res, 400, { ok: false, error: "missing token" });
      const v = String(body.token);
      if (v.length < 8) return json(res, 400, { ok: false, error: "token < 8 chars" });
      return json(res, 200, { ok: true, note: "tokens are managed via DASH_TOKEN env on the rig-dashboard service — edit /etc/init.d/rig-dashboard or your launcher to persist one. This endpoint validates only." });
    }
  }
  return json(res, 404, { ok: false, error: "not found" });
}

/* ---------- server ---------- */
function denyAuth(res) {
  return json(res, 401, { ok: false, error: "unauthorized — set Authorization: Bearer <DASH_TOKEN>", auth: true });
}
const server = http.createServer(async (req, res) => {
  let url;
  try { url = new URL(req.url, `http://${req.headers.host || "localhost"}`); } catch { return json(res, 400, { ok: false, error: "bad url" }); }
  if (authRequired()) req._parsedUrl = url;
  const isApi = url.pathname.startsWith("/api/");
  // The SPA page is ALWAYS served so its login overlay can render; only the
  // API (except the /api/auth probe) is token-gated when DASH_TOKEN is set.
  if (url.pathname === "/" || url.pathname === "/index.html") {
    return serveSpa(res);
  }
  const isAuthProbe = isApi && url.pathname === "/api/auth";
  if (isApi && authRequired() && !isAuthProbe && !authorized(req)) return denyAuth(res);
  if (isApi) {
    const body = (req.method === "POST") ? await getBody(req) : null;
    try { await handleApi(req, res, url, body); }
    catch (e) { log("api error:", e.message); json(res, 500, { ok: false, error: e.message }); }
    return;
  }
  // favicon
  if (url.pathname === "/favicon.ico") { res.writeHead(204); res.end(); return; }
  json(res, 404, { ok: false, error: "not found" });
});

server.listen(PORT, HOST, () => log(`rig-dashboard on http://${HOST}:${PORT} | shim conf=${CONF} | settings=${SETTINGS}${!process.env.DASH_TOKEN ? ` | AUTO TOKEN (see ${DASH_TOKEN_FILE}): ${DASH_TOKEN}` : ""}`));

// ---------- autonomous alert loop ----------
// evaluateAlerts() already implements real auto-recovery (the default
// "Shim down" rule restarts anthropic-shim), but before this it only ever
// ran when a human happened to have the Alerts tab open and clicked
// "Evaluate now" — meaning a 3am crash sat down indefinitely with nobody
// watching, despite the recovery logic already existing and being enabled
// by default. Running it on a timer turns that into genuine unattended
// self-healing. 60s cadence: fast enough that a crash gets caught within a
// minute, infrequent enough that the periodic tailLog() reads it triggers
// are negligible overhead (see the bounded tailLog() above).
const ALERT_EVAL_INTERVAL_MS = 60_000;
setInterval(() => {
  evaluateAlerts(false).catch(e => log("periodic alert eval failed:", e.message));
}, ALERT_EVAL_INTERVAL_MS);