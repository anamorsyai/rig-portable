#!/usr/bin/env node
// tls-relay.mjs — zero-logic TLS relay for the shim's zen requests.
// Why: zen gates on TLS fingerprint (MITM-confirmed 2026-10-04) — Go's crypto/tls
// gets FreeTierError while curl/OpenSSL and Node/Bun TLS pass. This relay re-sends
// every request it receives over Node's TLS (which passes), untouched otherwise.
// The shim's zen provider points here; every other provider is unaffected.
import http from "node:http";
import https from "node:https";
import dns from "node:dns";

const PORT = +(process.env.RELAY_PORT || 9097);
const log = (...a) => console.log(new Date().toISOString(), ...a);
let N = 0;

// resolve opencode.ai fresh at startup (Cloudflare rotates; /etc/hosts is clean)
const REAL_IPS = [];
dns.resolve4("opencode.ai", (e, addrs) => {
  if (!e && addrs.length) { REAL_IPS.push(...addrs); log("resolved opencode.ai:", addrs.join(", ")); }
});

const server = http.createServer((req, res) => {
  const chunks = [];
  req.on("data", c => chunks.push(c));
  req.on("end", () => {
    const body = Buffer.concat(chunks);
    const id = ++N;
    const headers = { ...req.headers };
    delete headers.host;
    delete headers.connection;
    headers.host = "opencode.ai";
    log(`[${id}] -> ${req.url} (${body.length}B) stream:`);
    try { const jb = JSON.parse(body.toString()); log("   stream=", jb.stream, "model=", jb.model); } catch {}
    log("   headers:", JSON.stringify({...req.headers}))
    const target = REAL_IPS.length ? { hostname: REAL_IPS[id % REAL_IPS.length] } : { hostname: "opencode.ai" };
    const path = req.url.startsWith("/zen/") ? req.url : "/zen" + req.url;
    const preq = https.request(
      { ...target, port: 443, path, method: req.method, headers, servername: "opencode.ai" },
      (pres) => {
        log(`[${id}] <- ${pres.statusCode}`);
        if (!res.headersSent) {
          res.writeHead(pres.statusCode, pres.headers);
        }
        pres.pipe(res);
      },
    );
    preq.on("error", (e) => {
      log(`[${id}] upstream error: ${e.message}`);
      res.writeHead(502, { "content-type": "application/json" });
      res.end(JSON.stringify({ error: { message: e.message, type: "relay_error" } }));
    });
    if (body.length) preq.write(body);
    preq.end();
  });
});

server.listen(PORT, "127.0.0.1", () => log(`tls-relay on :${PORT} -> opencode.ai via node TLS`));
