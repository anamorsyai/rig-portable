#!/usr/bin/env node
// zen-bridge.mjs — OpenAI /v1/chat/completions frontend -> opencode server session API backend.
// Why: opencode zen's free tier is validated SERVER-SIDE — only requests that carry a real
// opencode session pass. Direct calls (any headers, any key) get FreeTierError. The ONLY
// proven working path is through opencode itself (CLI `run` or the server's session API:
// create session -> switch model -> prompt -> poll for the assistant reply).
// This bridge exposes the OpenAI shape the anthropic-shim already speaks, and backs it with
// one reusable opencode session per model, prompting it and polling the reply.
// Zero deps. Node >= 18. Env: BRIDGE_PORT (default 9097), OC_SERVER, OC_PASSWORD, OC_LOCATION.
import http from "node:http";
import crypto from "node:crypto";

const PORT = +(process.env.BRIDGE_PORT || 9097);
const OC = process.env.OC_SERVER || "http://127.0.0.1:4099";
const PW = process.env.OC_PASSWORD || "";
const LOC = process.env.OC_LOCATION || "";
const AUTH = "Basic " + Buffer.from("opencode:" + PW).toString("base64");
const log = (...a) => console.log(new Date().toISOString(), ...a);

// one session per model id; created lazily, model-switched per request
const sessions = new Map();
async function api(method, path, body) {
  const r = await fetch(OC + path, {
    method,
    headers: { authorization: AUTH, "content-type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
  // 204 No Content is a valid success (model switch returns empty) — treat as {}
  if (r.status === 204) return {};
  const t = await r.text();
  let j; try { j = JSON.parse(t); } catch { throw new Error(`${path}: ${r.status} ${t.slice(0,150)}`); }
  if (r.status >= 400) throw new Error(`${path}: ${r.status} ${j.message || t.slice(0,150)}`);
  return j;
}
async function sessionFor(modelId) {
  let s = sessions.get(modelId);
  if (!s) {
    const r = await api("POST", "/api/session", { title: "zen-bridge", directory: LOC || undefined });
    s = r.data.id;
    sessions.set(modelId, s);
    log("session created for", modelId, "->", s);
  }
  await api("POST", `/api/session/${s}/model`, { model: { id: modelId, providerID: "opencode" } });
  return s;
}

function extractText(body) {
  // last user message text (string or content-parts array)
  const msgs = body.messages || [];
  const lastUser = [...msgs].reverse().find(m => m.role === "user");
  if (!lastUser) return "";
  const c = lastUser.content;
  if (typeof c === "string") return c;
  if (Array.isArray(c)) return c.map(p => p.text || "").join("\n");
  return String(c ?? "");
}

const server = http.createServer(async (req, res) => {
  if (req.method === "GET" && req.url === "/v1/models") {
    res.writeHead(200, { "content-type": "application/json" });
    return res.end(JSON.stringify({ object: "list", data: [
      { id: "opencode/big-pickle", object: "model" },
      { id: "opencode/mimo-v2.6-flash-free", object: "model" },
      { id: "opencode/nemotron-3-ultra-free", object: "model" },
      { id: "opencode/ling-3.0-flash-fin-free", object: "model" },
      { id: "opencode/longcat-2.5-preview-free", object: "model" },
      { id: "opencode/muse-spark-1.3-contributor-free", object: "model" },
      { id: "opencode/fledge-alpha-free", object: "model" },
    ]}));
  }
  if (req.method !== "POST" || !req.url.startsWith("/v1/chat/completions")) {
    res.writeHead(404, { "content-type": "application/json" });
    return res.end(JSON.stringify({ error: { message: "not found" } }));
  }
  let b = ""; req.on("data", c => b += c);
  req.on("end", async () => {
    const t0 = Date.now();
    try {
      const body = JSON.parse(b || "{}");
      let model = String(body.model || "big-pickle").replace(/^opencode\//, "");
      const text = extractText(body);
      if (!text.trim()) throw new Error("empty prompt");
      const sid = await sessionFor(model);
      const mid = "msg_" + crypto.randomBytes(8).toString("hex");
      await api("POST", `/api/session/${sid}/prompt`, { id: mid, text });
      // poll for the assistant reply (model-scoped, newest after our prompt)
      let reply = "";
      for (let i = 0; i < 45; i++) {
        await new Promise(r => setTimeout(r, 4000));
        const d = await api("GET", `/api/session/${sid}/message`);
        const items = d.data || [];
        // the assistant message that came after our user message
        const uIdx = items.findIndex(m => m.id === mid);
        const after = uIdx >= 0 ? items.slice(0, uIdx) : items; // newest-first at server
        const a = (uIdx >= 0 ? after : items).find(m => m.type === "assistant" && m.model?.id === model && m.finish);
        if (a) { const c = a.content || []; reply = c.map(p => p.text || "").join(""); break; }
      }
      if (!reply) throw new Error("timeout waiting for reply");
      log(`ok ${model} in ${Date.now() - t0}ms`);
      res.writeHead(200, { "content-type": "application/json" });
      res.end(JSON.stringify({
        id: "chatcmpl-bridge-" + mid, object: "chat.completion", created: Math.floor(Date.now()/1000),
        model, choices: [{ index: 0, message: { role: "assistant", content: reply }, finish_reason: "stop" }],
        usage: { prompt_tokens: 0, completion_tokens: 0, total_tokens: 0 },
      }));
    } catch (e) {
      log("error:", e.message);
      res.writeHead(502, { "content-type": "application/json" });
      res.end(JSON.stringify({ error: { message: e.message, type: "bridge_error" } }));
    }
  });
});
server.listen(PORT, "127.0.0.1", () => log(`zen-bridge on :${PORT} -> ${OC}${LOC ? " @" + LOC : ""}`));
