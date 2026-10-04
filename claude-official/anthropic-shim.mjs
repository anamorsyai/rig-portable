#!/usr/bin/env node
// anthropic-shim.mjs — Anthropic /v1/messages frontend -> OpenAI chat/completions upstream.
// Thinking blocks from reasoning_content, full streaming grammar, tool_use roundtrip,
// image passthrough, and model FAILOVER chain (on upstream error or empty stream) with
// self-healing per-candidate cooldowns, request body cap, and a /v1/models endpoint.
// Zero deps. Node >= 18. Env: PORT, UPSTREAM, UPSTREAM_KEY, UPSTREAM_UA, KNOWN_MODELS,
// FALLBACK_MODEL, MODEL_MAP, FALLBACK_CHAIN.
import http from "node:http";

const PORT = +(process.env.PORT || 9086);
const UPSTREAM = process.env.UPSTREAM || "https://opencode.ai/zen/v1/chat/completions";
const UPSTREAM_KEY = process.env.UPSTREAM_KEY || "public";
const UPSTREAM_UA = process.env.UPSTREAM_UA || "opencode/latest/2.0.21/cli";
const MODEL_MAP = JSON.parse(process.env.MODEL_MAP || "{}");
const KNOWN_MODELS = new Set((process.env.KNOWN_MODELS ||
  "big-pickle,moonshotai/Kimi-K2.6,mimo-v2.5-free,nemotron-3-ultra-free,nemotron-3.5-lightning-free,hy3-free").split(","));
const ATTEMPT_TIMEOUT_MS = parseInt(process.env.ATTEMPT_TIMEOUT_MS || "45000", 10);
const STREAM_IDLE_TIMEOUT_MS = parseInt(process.env.STREAM_IDLE_TIMEOUT_MS || "120000", 10);
const TOTAL_BUDGET_MS = parseInt(process.env.TOTAL_BUDGET_MS || "150000", 10);
// self-heal cooldowns + body cap — values are tuned live in /etc/conf.d/anthropic-shim
const COOLDOWN_BASE_MS = parseInt(process.env.COOLDOWN_BASE_MS || "2500", 10);
const COOLDOWN_MAX_MS = parseInt(process.env.COOLDOWN_MAX_MS || "30000", 10);
const COOLDOWN_AUTH_MS = parseInt(process.env.COOLDOWN_AUTH_MS || "20000", 10);
const MAX_BODY_BYTES = parseInt(process.env.MAX_BODY_BYTES || "10485760", 10);
// Slow-header threshold: usable responses that took longer than this to first-byte are
// treated as congestion risk — they complete, but the candidate is cooled so the *next*
// request prefers a faster secondary until the primary's latency recovers.
const SLOW_HEADER_MS = parseInt(process.env.SLOW_HEADER_MS || "9000", 10);
const FALLBACK_MODEL = process.env.FALLBACK_MODEL || "big-pickle";
const FALLBACK_CHAIN = (process.env.FALLBACK_CHAIN ||
  "big-pickle,moonshotai/Kimi-K2.6,mimo-v2.5-free").split(",");

// ---- multi-provider routing ----
// UPSTREAMS = {"name":{"url","key","ua","models":[...]}}; "$ENV" refs in key/ua resolved at load.
// Routing per model: MODEL_PROVIDER override > first provider listing it > default (first listed).
let PROVIDERS = {};
try {
  const raw = JSON.parse(process.env.UPSTREAMS || "null");
  if (raw && typeof raw === "object") {
    for (const [name, p] of Object.entries(raw)) {
      const deref = (v) => typeof v === "string" && v.startsWith("$") ? process.env[v.slice(1)] : v;
      // thinking_param is the canonical flag; also honor reasoning_param (zen in the live
      // conf is spelled reasoning_param) so chat_template_kwargs={thinking:true} is sent.
      const tp = !!(p.thinking_param || p.reasoning_param);
      PROVIDERS[name] = { url: deref(p.url), key: deref(p.key) || "", ua: deref(p.ua) || UPSTREAM_UA, models: p.models || [], thinking_param: tp };
    }
  }
} catch (e) { console.error("UPSTREAMS parse failed:", e.message); }
if (!Object.keys(PROVIDERS).length)
  PROVIDERS.default = { url: UPSTREAM, key: UPSTREAM_KEY, ua: UPSTREAM_UA, models: [...KNOWN_MODELS] };
// key rotation: provider.key may be comma-separated; normalized to prov.keys[]
let RR = 0; // global round-robin counter so concurrent requests spread across keys
for (const p of Object.values(PROVIDERS))
  p.keys = (typeof p.key === "string" && p.key.includes(",")) ? p.key.split(",").map(s => s.trim()).filter(Boolean) : (p.key ? [p.key] : [""]);
const DEFAULT_PROVIDER = Object.keys(PROVIDERS)[0];
// secret scrubbing: never let provider API keys reach the log file via upstream error bodies
const SECRET_RE = new RegExp(
  [...new Set(Object.values(PROVIDERS).flatMap(p => p.keys || []).filter(k => k && k.length >= 8))
        , ...(process.env.UPSTREAM_KEY && process.env.UPSTREAM_KEY !== "public" ? [process.env.UPSTREAM_KEY] : [])]
    .map(k => k.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")).join("|") || "(?!)x", "g");
const redact = (s) => String(s).replace(SECRET_RE, "[REDACTED]");
let MODEL_PROVIDER = {};
try { MODEL_PROVIDER = JSON.parse(process.env.MODEL_PROVIDER || "{}"); } catch {}
const providerFor = (m) =>
  MODEL_PROVIDER[m] || Object.keys(PROVIDERS).find(p => PROVIDERS[p].models.includes(m)) || DEFAULT_PROVIDER;

const resolveModel = (m) => MODEL_MAP[m] || (KNOWN_MODELS.has(m) ? m : FALLBACK_MODEL);
// candidates -> [{model,provider}]; chain entries: "model" or "model@provider" (forced provider)
const candidates = (rawModel) => {
  const first = resolveModel(rawModel);
  // if the chain pins this model to an explicit provider ("model@prov"), honor it as primary
  let pin = null;
  for (const e of FALLBACK_CHAIN) {
    if (e.includes("@")) {
      const [em, ep] = e.split("@", 2);
      if (resolveModel(em) === first && PROVIDERS[ep]) { pin = ep; break; }
    }
  }
  const out = [{ model: first, provider: pin || providerFor(first) }];
  for (const e of FALLBACK_CHAIN) {
    const [em, ep] = e.includes("@") ? e.split("@", 2) : [e, null];
    const m = resolveModel(em);
    const pr = ep && PROVIDERS[ep] ? ep : providerFor(m);
    if (!out.some(c => c.model === m && c.provider === pr)) out.push({ model: m, provider: pr });
  }
  // expand each candidate into per-key attempts; rotate starting key per request (429 relief)
  const rrOff = Math.abs(RR++);
  const expanded = [];
  for (const c of out) {
    const keys = PROVIDERS[c.provider]?.keys?.length ? PROVIDERS[c.provider].keys : [""];
    const off = keys.length > 1 ? rrOff % keys.length : 0;
    for (let k = 0; k < keys.length; k++) expanded.push({ ...c, keyIdx: keys.length > 1 ? (off + k) % keys.length : 0 });
  }
  return expanded;
};
const log = (...a) => console.log(new Date().toISOString(), ...a);

/* ---------- self-healing cooldowns (per model@provider) ---------- */
// Hanging/5xx/429/ghost candidates are skipped for an exponentially growing window
// (COOLDOWN_BASE_MS doubling up to COOLDOWN_MAX_MS; auth failures get COOLDOWN_AUTH_MS;
// 429 honors Retry-After when present). First usable payload resets the strike count so
// recovered providers rejoin the chain immediately. If every candidate is cooled we still
// try the full list (availability over purity).
const cooldowns = new Map(); // "model@provider" -> { until, fails, kind }
const candKey = (c) => c.model + "@" + c.provider;
const penalize = (cand, kind, retryAfterMs) => {
  const k = candKey(cand);
  const cur = cooldowns.get(k) || { fails: 0 };
  cur.fails++;
  const authMs = Math.max(COOLDOWN_AUTH_MS, COOLDOWN_BASE_MS);
  const base = kind === "auth" ? authMs : COOLDOWN_BASE_MS * Math.pow(2, cur.fails - 1);
  const cap = kind === "auth" ? authMs : COOLDOWN_MAX_MS;
  const ms = kind === "ratelimit" && retryAfterMs > 0 ? Math.min(retryAfterMs, cap) : Math.min(base, cap);
  cur.until = Date.now() + ms;
  cur.kind = kind || "fail";
  cooldowns.set(k, cur);
  log(`cooldown ${k} for ${ms}ms (${cur.kind}, strike ${cur.fails})`);
};
// reward only clears failure-class strikes; a latency cooldown from a slow-but-successful
// response stays until expiry so congestion doesn't re-hammer the hot primary next request.
const reward = (cand) => {
  const k = candKey(cand);
  const c = cooldowns.get(k);
  if (c && c.kind === "latency") return;
  if (cooldowns.delete(k)) log(`recovered ${k}`);
};
const penalizeSlow = (cand, headersMs) => {
  const k = candKey(cand);
  const ms = Math.min(COOLDOWN_MAX_MS, Math.max(COOLDOWN_BASE_MS, headersMs - SLOW_HEADER_MS + COOLDOWN_BASE_MS));
  const cur = cooldowns.get(k) || { fails: 0 };
  cur.fails++;
  cur.until = Date.now() + ms;
  cur.kind = "latency";
  cooldowns.set(k, cur);
  log(`cooldown ${k} for ${ms}ms (latency, slow-header ${headersMs}ms)`);
};
const coolFilter = (list) => {
  const now = Date.now();
  for (const [k, v] of cooldowns) if (v.until <= now) cooldowns.delete(k);
  const live = list.filter(c => !cooldowns.has(candKey(c)));
  return live.length ? live : list;
};

/* ---------- request translation: anthropic -> openai ---------- */
// zenSysprompt returns the opencode system-prompt fingerprint (multi-line, so it
// lives in a file — ZEN_SYSPROMPT_FILE; shell env values must stay single-line).
function zenSysprompt() {
  try {
    if (process.env.ZEN_SYSPROMPT_FILE) return fs.readFileSync(process.env.ZEN_SYSPROMPT_FILE, "utf8");
  } catch {}
  return process.env.ZEN_SYSPROMPT || "";
}

// zenHeaders builds the upstream request headers. opencode zen free-tier gate
// (MITM-confirmed 2026-10-04): zen validates (x-opencode-session id,
// x-opencode-project id) against its backend — both must be REAL ids created by
// opencode itself (persisted server-side, independent of any opencode process).
// The ids come from zen-ids.conf via env (ZEN_SESSION_ID/ZEN_PROJECT_ID).
function zenHeaders(cand, key, prov) {
  const h = { "Content-Type": "application/json", Authorization: "Bearer " + key, "User-Agent": prov.ua };
  const isZen = (cand && cand.provider === "zen") || prov === PROVIDERS.zen || prov.name === "zen";
  if (isZen) {
    h["X-Opencode-Client"] = "cli";
    // body fingerprint: prepend opencode's system prompt (zen validates it)
    if (cand && cand.__oaiBody) {
      const zsp = zenSysprompt();
      if (zsp && Array.isArray(cand.__oaiBody.messages)) {
        cand.__oaiBody.messages = [{ role: "system", content: zsp }, ...cand.__oaiBody.messages];
      }
    }
    if (process.env.ZEN_SESSION_ID) {
      const sid = process.env.ZEN_SESSION_ID;
      h["X-Opencode-Session"] = sid;
      h["X-Opencode-Session-Id"] = sid;
      h["X-Session-Id"] = sid;
      h["X-Session-Affinity"] = sid;
    }
    if (process.env.ZEN_PROJECT_ID) h["X-Opencode-Project"] = process.env.ZEN_PROJECT_ID;
  }
  return h;
}

function textOf(c) {
  if (typeof c === "string") return c;
  return (c || []).filter(b => b && b.type === "text").map(b => b.text).join("\n");
}
function convertBody(body) {
  const out = { model: resolveModel(body.model), stream: true };
  if (body.max_tokens) out.max_tokens = body.max_tokens;
  if (body.temperature != null) out.temperature = body.temperature;
  if (body.top_p != null) out.top_p = body.top_p;
  if (body.stop_sequences?.length) out.stop = body.stop_sequences;
  const msgs = [];
  if (body.system) {
    const s = typeof body.system === "string"
      ? body.system
      : Array.isArray(body.system) ? body.system.filter(b => b.type === "text").map(b => b.text).join("\n") : "";
    if (s) msgs.push({ role: "system", content: s });
  }
  for (const m of body.messages || []) {
    if (m.role === "user") {
      const parts = [];
      const blocks = Array.isArray(m.content) ? m.content : [{ type: "text", text: String(m.content ?? "") }];
      const flush = () => {
        if (!parts.length) return;
        const snap = [...parts];
        msgs.push({ role: "user", content: snap.length === 1 && snap[0].type === "text" ? snap[0].text : snap });
        parts.length = 0;
      };
      for (const b of blocks) {
        if (!b) continue;
        if (b.type === "tool_result") { flush(); msgs.push({ role: "tool", tool_call_id: b.tool_use_id, content: textOf(b.content) || "" }); }
        else if (b.type === "text") parts.push({ type: "text", text: b.text });
        else if (b.type === "image" && b.source?.type === "base64" && b.source.data)
          parts.push({ type: "image_url", image_url: { url: `data:${b.source.media_type || "image/png"};base64,${b.source.data}` } });
      }
      flush();
    } else if (m.role === "assistant") {
      let txt = ""; const toolCalls = [];
      for (const b of Array.isArray(m.content) ? m.content : []) {
        if (b.type === "text") txt += (txt ? "\n" : "") + b.text;
        else if (b.type === "tool_use")
          toolCalls.push({ id: b.id, type: "function", function: { name: b.name, arguments: JSON.stringify(b.input ?? {}) } });
      }
      const am = { role: "assistant", content: txt || null };
      if (toolCalls.length) am.tool_calls = toolCalls;
      if (am.content !== null || toolCalls.length) msgs.push(am);
    }
  }
  // opencode zen body fingerprint (MITM-confirmed 2026-10-04): zen validates the
  // SYSTEM PROMPT against opencode's own — prepend the captured opencode prompt
  // (from zen-sysprompt.txt via ZEN_SYSPROMPT_FILE) for zen-labeled requests.
  // convertBody has no provider context; used on the zen-bound path only when
  // the model resolves to the zen provider (checked at send time below).
  out.messages = msgs;
  if (Array.isArray(body.tools) && body.tools.length)
    out.tools = body.tools.map(t => ({ type: "function", function: { name: t.name, description: t.description || "", parameters: t.input_schema || { type: "object", properties: {} } } }));
  const tc = body.tool_choice;
  if (tc) {
    if (tc.type === "auto") out.tool_choice = "auto";
    else if (tc.type === "any") out.tool_choice = "required";
    else if (tc.type === "none") out.tool_choice = "none";
    else if (tc.type === "tool") out.tool_choice = { type: "function", function: { name: tc.name } };
  }
  return out;
}

/* ---------- response translation ---------- */
class Emitter {
  constructor(res) { this.res = res; this.closed = false; }
  send(event, data) { if (!this.closed) this.res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`); }
  end() { this.closed = true; this.res.end(); }
}
const safeJson = (s) => { try { return s ? JSON.parse(s) : {}; } catch { return { _raw: s }; } };

async function translateStream(clientReq, em, wantStream, model, oaiBody, candidatesList) {
  // candidate loop: fetch -> ok? -> stream; retry next model on connect error / bad status / empty stream
  let attempt = 0;
  const clientGoneRef = { gone: false }; // survives across candidate attempts
  if (wantStream && clientReq) clientReq.on("close", () => { clientGoneRef.gone = true; });
  const deadline = Date.now() + TOTAL_BUDGET_MS;
  const ordered = coolFilter(candidatesList); // skip self-healing-cooled candidates up front
  while (true) {
    attempt++;
    if (Date.now() > deadline) { log("total budget exceeded, giving up"); return null; }
    if (attempt > ordered.length) {
      log("all candidates exhausted");
      return null;
    }
    const cand = ordered[attempt - 1];
    if (clientGoneRef.gone) { log("client gone, abandoning chain"); return null; }
    const prov = PROVIDERS[cand.provider] || PROVIDERS[DEFAULT_PROVIDER];
    oaiBody.model = cand.model;
    if (prov.thinking_param) { if (!oaiBody.chat_template_kwargs) oaiBody.chat_template_kwargs = { thinking: true }; }
    else delete oaiBody.chat_template_kwargs;
    const keys = prov.keys?.length ? prov.keys : [""];
    const keyIdx = Number.isInteger(cand.keyIdx) && cand.keyIdx < keys.length ? cand.keyIdx : 0;
    log(`attempt ${attempt} model=${cand.model}@${cand.provider}#k${keyIdx}${keys.length > 1 ? "/" + keys.length : ""}`);
    const t0 = Date.now();
    let up;
    try {
      // hard per-attempt timeout so a hanging provider can't stall the chain
      const ac = new AbortController();
      const timer = setTimeout(() => ac.abort(), ATTEMPT_TIMEOUT_MS);
      try {
        if (cand.provider === "zen") {
          // body fingerprint prepend (grabbed before header-building reads it)
          cand.__oaiBody = oaiBody;
        }
        up = await fetch(prov.url, {
          method: "POST",
          headers: zenHeaders(cand, keys[keyIdx], prov),
          body: JSON.stringify(oaiBody),
          signal: ac.signal
        });
      } finally { clearTimeout(timer); }
    } catch (e) { log("connect fail:", e.message); penalize(cand, "connect"); continue; }
    if (!up.ok || !up.body) {
      const t = await up.text().catch(() => "");
      log("upstream", up.status, redact(t.slice(0, 160)));
      try { await up.body?.cancel(); } catch {} // free the pooled socket (await: undici can reject async on disturbed bodies)
      // Retry-After may be seconds ("12") or an HTTP-date; NaN from a date-form is no longer ignored
      const raHeader = up.headers?.get?.("retry-after");
      const raSec = parseInt(raHeader || "", 10);
      const raMs = (!isNaN(raSec)) ? raSec * 1000 : parseRetryAfterDate(raHeader);
      penalize(cand, up.status === 429 ? "ratelimit" : (up.status === 401 || up.status === 403) ? "auth" : "http", raMs);
      continue;
    }
    const headersMs = Date.now() - t0;
    log(`ok ${cand.model}@${cand.provider} headers in ${headersMs}ms`);

    // parse SSE; gate all client emission until first usable chunk (allows empty-stream retry)
    const st = { started: false, idx: 0, cur: null, tools: new Map(), openTools: new Set(), stopReason: "end_turn", uIn: 0, uOut: 0, agg: [], think: "", text: "", sawAny: false, sawPayload: false };
    const ensureStarted = () => {
      if (st.started || !wantStream) return;
      em.send("message_start", { type: "message_start", message: { id: "msg_" + Date.now().toString(36), type: "message", role: "assistant", model, content: [], stop_reason: null, stop_sequence: null, usage: { input_tokens: 0, output_tokens: 0, cache_creation_input_tokens: 0, cache_read_input_tokens: 0 } } });
      st.started = true;
    };
    const closeCur = () => {
      if (!st.cur) return;
      if (wantStream) {
        if (st.cur.kind === "thinking") em.send("signature_delta", { type: "content_block_delta", index: st.cur.index, delta: { type: "signature_delta", signature: "sig-shim-" + st.cur.index } });
        em.send("content_block_stop", { type: "content_block_stop", index: st.cur.index });
      }
      st.agg.push(st.cur.kind === "thinking" ? { type: "thinking", thinking: st.think } : { type: "text", text: st.text });
      st.think = ""; st.text = ""; st.cur = null;
    };

    const handleChunk = (j) => {
      const ch = j.choices?.[0];
      if (j.usage) { st.uIn = j.usage.prompt_tokens ?? st.uIn; st.uOut = j.usage.completion_tokens ?? st.uOut; }
      if (!ch) return;
      st.sawAny = true;
      ensureStarted();
      const d = ch.delta || {};
      const reason = d.reasoning_content ?? d.reasoning;
      if (typeof reason === "string" && reason) {
        st.sawPayload = true;
        if (st.cur?.kind !== "thinking") { closeCur(); st.cur = { kind: "thinking", index: st.idx++ }; if (wantStream) em.send("content_block_start", { type: "content_block_start", index: st.cur.index, content_block: { type: "thinking", thinking: "" } }); }
        if (wantStream) em.send("content_block_delta", { type: "content_block_delta", index: st.cur.index, delta: { type: "thinking_delta", thinking: reason } });
        st.think += reason;
      }
      if (typeof d.content === "string" && d.content) {
        st.sawPayload = true;
        if (st.cur?.kind !== "text") { closeCur(); st.cur = { kind: "text", index: st.idx++ }; if (wantStream) em.send("content_block_start", { type: "content_block_start", index: st.cur.index, content_block: { type: "text", text: "" } }); st.text = st.text || ""; }
        if (wantStream) em.send("content_block_delta", { type: "content_block_delta", index: st.cur.index, delta: { type: "text_delta", text: d.content } });
        st.text += d.content;
      }
      for (const tc of d.tool_calls || []) {
        st.sawPayload = true;
        closeCur();
        let ai = st.tools.get(tc.index);
        if (ai == null) {
          ai = st.idx++; st.tools.set(tc.index, ai); st.openTools.add(ai);
          const name = tc.function?.name || "";
          if (wantStream) em.send("content_block_start", { type: "content_block_start", index: ai, content_block: { type: "tool_use", id: tc.id || "toolu_" + Math.random().toString(36).slice(2), name, input: {} } });
          st.agg.push({ __tool: true, index: ai, id: tc.id || "", name, args: "" });
        }
        const arg = tc.function?.arguments || "";
        if (arg) {
          if (wantStream) em.send("content_block_delta", { type: "content_block_delta", index: ai, delta: { type: "input_json_delta", partial_json: arg } });
          const rec = st.agg.find(x => x.__tool && x.index === ai); if (rec) rec.args += arg;
        }
      }
      if (ch.finish_reason) st.stopReason = ch.finish_reason === "tool_calls" ? "tool_use" : ch.finish_reason === "length" ? "max_tokens" : "end_turn";
    };

    // read upstream SSE into handleChunk
    const reader = up.body.getReader();
    const dec = new TextDecoder(); let buf = "";
    let clientGone = false;
    if (wantStream && clientReq) clientReq.on("close", () => { clientGone = true; clientGoneRef.gone = true; try { reader.cancel(); } catch {} });
    let idleTimer = setInterval(() => {
      if (clientGone || Date.now() - lastRead > STREAM_IDLE_TIMEOUT_MS) {
        log(clientGone ? "client disconnected, aborting attempt" : "stream idle timeout", oaiBody.model);
        try { reader.cancel(); } catch {}
      }
    }, 1000);
    let lastRead = Date.now();
    try {
      while (!clientGone) {
        const { done, value } = await reader.read();
        if (done) break;
        lastRead = Date.now();
        buf += dec.decode(value, { stream: true });
        let nl;
        while ((nl = buf.indexOf("\n")) >= 0) {
          const line = buf.slice(0, nl).trim(); buf = buf.slice(nl + 1);
          if (!line.startsWith("data:")) continue;
          const p = line.slice(5).trim();
          if (p === "[DONE]") continue;
          try { handleChunk(JSON.parse(p)); } catch {}
        }
        // Backpressure: if the downstream client socket is congested (its write
        // buffer is full), pause upstream reads until it drains. Without this, a
        // fast model + slow client buffers unbounded bytes in memory.
        const sock = em.res;
        if (wantStream && sock && sock.writableNeedDrain) {
          await new Promise((resolve) => sock.once("drain", resolve));
        }
      }
    } catch (e) {
      log("stream read error:", e.message);
      if (clientGone) return null; // client gone: stop the whole chain
      if (st.started) { finalize(em, wantStream, st); em.end(); return; }
      penalize(cand, "stream");
      continue; // nothing emitted yet -> next candidate
    } finally {
      clearInterval(idleTimer);
    }

    if (clientGone) return null; // client gone mid-stream: nothing to deliver
    if (!st.sawAny || !st.sawPayload) { log("empty stream from", candKey(cand)); penalize(cand, "ghost"); continue; } // ghost stream / nothing usable -> retry
    if (headersMs > SLOW_HEADER_MS) penalizeSlow(cand, headersMs); else reward(cand); // fast = healthy, slow = congestion risk

    closeCur();
    for (const ai of [...st.openTools].sort((a, b) => a - b)) { if (wantStream) em.send("content_block_stop", { type: "content_block_stop", index: ai }); st.openTools.delete(ai); }
    const finalContent = [];
    let ti = 0;
    for (const c of st.agg) finalContent.push(c.__tool ? { type: "tool_use", id: c.id || "toolu_" + ti++, name: c.name, input: safeJson(c.args) } : c);
    if (!finalContent.length) finalContent.push({ type: "text", text: "" });

    if (wantStream) {
      finalize(em, true, st);
      em.end();
    } else {
      return { id: "msg_" + Date.now().toString(36), type: "message", role: "assistant", model, content: finalContent, stop_reason: st.stopReason, stop_sequence: null, usage: { input_tokens: st.uIn, output_tokens: st.uOut } };
    }
    return;
  }
}

function finalize(em, wantStream, st) {
  if (!wantStream) return;
  em.send("message_delta", { type: "message_delta", delta: { stop_reason: st.stopReason, stop_sequence: null }, usage: { input_tokens: st.uIn, output_tokens: st.uOut, cache_creation_input_tokens: 0, cache_read_input_tokens: 0 } });
  em.send("message_stop", { type: "message_stop" });
}

/* ---------- non-stream fast path ---------- */
// For non-streaming clients we can ask the upstream for one consolidated JSON body
// (stream:false) and return it directly, avoiding per-chunk SSE buffering. Falls back
// to the streaming translateStream path on any failure (availability over purity).
function parseReasonableArgs(s) {
  try { return JSON.parse(s || "{}"); } catch { return { _raw: s || "" }; }
}
function toAnthropicNonStream(model, j) {
  const ch = (j && j.choices && j.choices[0]) || {};
  const msg = ch.message || {};
  const content = [];
  // thinking/reasoning blocks
  const reason = msg.reasoning_content ?? msg.reasoning;
  if (typeof reason === "string" && reason) content.push({ type: "thinking", thinking: reason });
  // text content may be a string or an array of parts (OpenAI o1/gpt style)
  if (msg.content != null) {
    if (typeof msg.content === "string") { if (msg.content) content.push({ type: "text", text: msg.content }); }
    else if (Array.isArray(msg.content)) {
      for (const p of msg.content) {
        if (p && p.type === "text" && p.text) content.push({ type: "text", text: typeof p.text === "string" ? p.text : (p.text.value || "") });
      }
    }
  }
  // tool calls
  let ti = 0;
  for (const tc of (msg.tool_calls || [])) {
    const name = tc.function?.name || "";
    if (!name) continue; // tool_call without a name is unusable; skip
    content.push({ type: "tool_use", id: tc.id || "toolu_" + ti, name, input: parseReasonableArgs(tc.function?.arguments) });
    ti++;
  }
  if (!content.length) content.push({ type: "text", text: "" });
  const fr = ch.finish_reason;
  const stop = fr === "tool_calls" ? "tool_use" : fr === "length" ? "max_tokens" : "end_turn";
  const u = j.usage || {};
  return {
    id: "msg_" + Date.now().toString(36), type: "message", role: "assistant", model,
    content, stop_reason: stop, stop_sequence: null,
    usage: { input_tokens: u.prompt_tokens || 0, output_tokens: u.completion_tokens || 0, cache_creation_input_tokens: 0, cache_read_input_tokens: 0 }
  };
}
async function fetchNonStream(prov, oaiBody, keyIdx) {
  const keys = prov.keys?.length ? prov.keys : [""];
  const key = keys[keyIdx] || "";
  const ac = new AbortController();
  const timer = setTimeout(() => ac.abort(), ATTEMPT_TIMEOUT_MS);
  try {
    const up = await fetch(prov.url, {
      method: "POST",
      headers: zenHeaders({ provider: prov.name === "zen" || prov === PROVIDERS.zen ? "zen" : "", __oaiBody: oaiBody }, key, prov),
      // zen free tier requires stream:true (MITM-confirmed 2026-10-04); the SSE
      // reply is reassembled below. Other providers keep non-stream.
      body: JSON.stringify(prov === PROVIDERS.zen || prov.name === "zen" ? { ...oaiBody, stream: true } : { ...oaiBody, stream: false }),
      signal: ac.signal
    });
    if (!up.ok || !up.body) { const t = await up.text().catch(() => ""); return { ok: false, status: up.status, body: redact(t.slice(0, 160)), headers: up.headers }; }
    const j = await up.json().catch(() => null);
    return { ok: true, json: j, headers: up.headers };
  } catch (e) {
    return { ok: false, err: e.message };
  } finally { clearTimeout(timer); }
}
async function nonStreamFastPath(req, model, oaiBody, cands) {
  const deadline = Date.now() + TOTAL_BUDGET_MS;
  let attempt = 0;
  const ordered = coolFilter(cands);
  while (attempt < ordered.length) {
    if (Date.now() > deadline) { log("nonStream: total budget exceeded"); return null; }
    const cand = ordered[attempt++];
    if (attempt === 1 && req && req.aborted) return null;
    const prov = PROVIDERS[cand.provider] || PROVIDERS[DEFAULT_PROVIDER];
    const keys = prov.keys?.length ? prov.keys : [""];
    const keyIdx = Number.isInteger(cand.keyIdx) && cand.keyIdx < keys.length ? cand.keyIdx : 0;
    const oaiSent = { ...oaiBody, model: cand.model };
    if (prov.thinking_param) { if (!oaiSent.chat_template_kwargs) oaiSent.chat_template_kwargs = { thinking: true }; } else delete oaiSent.chat_template_kwargs;
    log(`nonStream attempt ${attempt} model=${cand.model}@${cand.provider}#k${keyIdx}`);
    const r = await fetchNonStream(prov, oaiSent, keyIdx);
    if (r.ok && r.json) {
      // empty payload (no content, no tool_calls) -> treat as ghost, let streaming path retry
      const ch = (r.json.choices && r.json.choices[0]) || {};
      const msg = ch.message || {};
      const hasContent = msg.content != null || (msg.reasoning_content || msg.reasoning) || (msg.tool_calls && msg.tool_calls.length);
      if (!hasContent) { log("nonStream: ghost payload from", candKey(cand)); penalize(cand, "ghost"); continue; }
      reward(cand);
      return toAnthropicNonStream(model, r.json);
    }
    // failure
    const code = r.status ? (r.status === 429 ? "ratelimit" : (r.status === 401 || r.status === 403) ? "auth" : "http") : "connect";
    log("nonStream upstream", r.status || "connect", r.err || r.body || "");
    const ra = parseInt(r.headers?.get?.("retry-after") || "0", 10);
    const raMs = !isNaN(ra) ? ra * 1000 : parseRetryAfterDate(r.headers?.get?.("retry-after"));
    penalize(cand, code, raMs > 0 ? raMs : 0);
    continue;
  }
  log("nonStream: all candidates exhausted");
  return null;
}
function parseRetryAfterDate(v) {
  if (!v || !isNaN(parseInt(v, 10))) return 0; // numeric form handled elsewhere; dates fall here
  try { const d = new Date(v); if (!isNaN(d.getTime())) return Math.max(0, d.getTime() - Date.now()); } catch {}
  return 0;
}

/* ---------- server ---------- */
const server = http.createServer(async (req, res) => {
  if (req.method === "GET") {
    if (req.url.includes("/v1/models")) {
      const ids = [...new Set([...Object.values(PROVIDERS).flatMap(p => p.models || []), ...KNOWN_MODELS])];
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ object: "list", data: ids.map(id => ({ id, object: "model", created: 0, owned_by: providerFor(id) })) }));
      return;
    }
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ status: "ok", upstream: UPSTREAM, uptime_s: Math.round(process.uptime()),
      chain: FALLBACK_CHAIN.join(","),
      cooling: [...cooldowns].filter(([, v]) => v.until > Date.now()).map(([k, v]) => ({ target: k, ms_left: v.until - Date.now(), kind: v.kind, strikes: v.fails })) }));
    return;
  }
  if (!req.url.includes("/messages")) { res.writeHead(404, { "Content-Type": "application/json" }); res.end(JSON.stringify({ type: "error", error: { type: "not_found_error", message: "not found" } })); return; }
  let raw = "", bodyBytes = 0, tooBig = false;
  try {
    for await (const c of req) { bodyBytes += c.length; if (bodyBytes > MAX_BODY_BYTES) { tooBig = true; break; } raw += c; }
  } catch { tooBig = true; } // aborted/premature-close mid-body: treat as rejected request
  if (tooBig) { res.writeHead(413, { "Content-Type": "application/json" }); res.end(JSON.stringify({ type: "error", error: { type: "request_too_large", message: "request body exceeds MAX_BODY_BYTES=" + MAX_BODY_BYTES } }), () => req.destroy()); return; }
  let body;
  try { body = JSON.parse(raw); } catch { res.writeHead(400, { "Content-Type": "application/json" }); res.end(JSON.stringify({ type: "error", error: { type: "invalid_request_error", message: "bad json" } })); return; }
  const wantStream = !!body.stream;
  const model = body.model || "unknown";
  const oaiBody = convertBody(body);
  const cands = candidates(body.model);
  log("->", model, "=>", cands.map(c => `${c.model}@${c.provider}`).join("|"), "tools:", oaiBody.tools?.length || 0, "msgs:", oaiBody.messages.length);

  const em = new Emitter(res);
  let result = null, failed = false;
  try {
    if (wantStream) {
      // Concurrency rate-limit guard: only ONE in-flight attempt per provider@key. When
      // the client fires a second streaming request at the same target (Cline parallel
      // tools), queue it instead of hitting the provider concurrently — free-tier
      // providers (zen/dahl) throttle *concurrency* per key and 429 on N simultaneous
      // calls. Queued requests emit periodic signature_delta keepalives (progress up
      // the wire) then run normally once the winner finishes. Different targets run
      // fully parallel. Non-streaming is intentionally NOT gated (no SSE to keep warm).
      const shortKey = (c) => c.model + "@" + c.provider + "#k" + (c.keyIdx ?? 0);
      const gate = (globalThis._shimInFlight ||= new Map());
      const target = shortKey(cands[0]);
      if (gate.has(target)) {
        log("queueing", target, "behind an in-flight attempt (concurrency guard)");
        const keepalive = setInterval(() => {
          try { em.send("signature_delta", { type: "content_block_delta", index: 0, delta: { type: "signature_delta", signature: "sig-queue-" + Math.floor(Date.now() / 300) } }); } catch {}
        }, 200);
        const qDeadline = Date.now() + TOTAL_BUDGET_MS;
        try {
          while (gate.has(target) && Date.now() < qDeadline) await new Promise(r => setTimeout(r, 100));
          if (Date.now() >= qDeadline) log("queue wait budget exceeded, firing anyway", target);
        } finally { clearInterval(keepalive); }
      }
      gate.set(target, true);
      try {
        // defer headers until first emission so hard-fail can still return HTTP error
        let headSent = false;
        const emProxy = new Emitter(res);
        emProxy.send = (event, data) => {
          if (emProxy.closed) return;
          if (!headSent) { res.writeHead(200, { "Content-Type": "text/event-stream", "Cache-Control": "no-cache", Connection: "keep-alive", "anthropic-version": "2023-06-01" }); headSent = true; }
          res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`);
        };
        const r = await translateStream(req, emProxy, true, model, oaiBody, cands);
        if (r === null) failed = true;
      } finally { gate.delete(target); }
    } else {
      // Fast path: ask upstream for one consolidated JSON (stream:false).
      // Falls back to the streamed path (which buffers then returns JSON) on failure.
      result = await nonStreamFastPath(req, model, oaiBody, cands);
      if (!result) result = await translateStream(null, em, false, model, oaiBody, cands);
      if (!result) failed = true;
      else { res.writeHead(200, { "Content-Type": "application/json" }); res.end(JSON.stringify(result)); }
    }
  } catch (e) {
    log("fatal:", e.message);
    failed = true;
  }
  if (failed) {
    if (res.headersSent) { try { em.end(); } catch {} }
    else {
      res.writeHead(502, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ type: "error", error: { type: "api_error", message: "All fallback models failed: " + cands.map(c => `${c.model}@${c.provider}`).join(", ") } }));
    }
  }
});

server.listen(PORT, "127.0.0.1", () => log(`anthropic-shim on :${PORT} -> ${UPSTREAM} | chain: ${FALLBACK_CHAIN.join(",")}`));

// resilience net: a proxy must never die on one bad request/rejection — log loudly, keep serving
const keepAlive = (sig) => (e) => log(`resilience: ${sig}:`, e?.stack || e);
process.on("uncaughtException", keepAlive("uncaughtException"));
process.on("unhandledRejection", keepAlive("unhandledRejection"));
