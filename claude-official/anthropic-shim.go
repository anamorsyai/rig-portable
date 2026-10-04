// anthropic-shim.go — Anthropic /v1/messages frontend -> OpenAI chat/completions upstream.
//
// Go rewrite of the Node shim. Provides:
//   - Anthropic streaming (SSE) and non-streaming /v1/messages frontend
//   - OpenAI-compatible chat/completions upstream with multi-provider failover chain
//   - Adaptive per-attempt timeout (tracks each provider's observed first-byte baseline)
//   - Per-provider@key degraded state with exponential backoff (smart failover)
//   - Thinking blocks (reasoning_content), tool_use roundtrip, image passthrough
//   - Full structured per-request logging with request IDs
//
// Configuration via environment variables (same names as the Node version):
//   PORT, and a single JSON blob UPSTREAMS. Optionally FALLBACK_CHAIN, KNOWN_MODELS,
//   FALLBACK_MODEL, and tuning vars (ATTEMPT_TIMEOUT_MS, TOTAL_BUDGET_MS, etc.).

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

type Provider struct {
	Name           string   `json:"-"`
	URL            string   `json:"url"`
	Key            string   `json:"key"`
	UA             string   `json:"ua"`
	Models         []string `json:"models"`
	ThinkingParam  bool     `json:"thinking_param"`
	ReasoningParam bool     `json:"reasoning_param"`
	MaxOutput      int      `json:"max_output"` // max_tokens cap for this provider (0 = unlimited)

	Keys []string `json:"-"`
}

var (
	cfgPort           = "9086"
	cfgFallbackModel  = "big-pickle"
	cfgFallbackChain  = []string{}
	cfgKnownModels    = map[string]bool{}
	cfgProviders      = map[string]*Provider{}
	cfgProviderOrder  = []string{}
	cfgModelProvider  = map[string]string{}
	cfgModelMap       = map[string]string{}
	cfgAttemptTimeout = 15 * time.Second
	cfgTotalBudget    = 180 * time.Second
	cfgCooldownBase   = 2500 * time.Millisecond
	cfgCooldownMax    = 30 * time.Second
	cfgCooldownAuth   = 20 * time.Second
	// Ratelimit-pressure ramp: after this many 429s within rlDecayWindow, the
	// whole provider is briefly cooled so requests skip it (saturated provider).
	rlProviderThreshold = 3
	rlDecayWindow       = 30 * time.Second
	cfgSlowHeaderMS     = 8000
	cfgMaxBodyBytes     = int64(10 << 20)

	// Response cache (exact-match on the forwarded request; safe replay of
	// identical re-sends / retries / background pings). See cache section below.
	cfgCacheEnabled      = true
	cfgCacheMaxEntries   = 512
	cfgCacheTTL          = 10 * time.Minute
	cfgCacheTempZeroOnly = true
	cfgCacheDir          = "/var/cache/anthropic-shim"
	cfgCachePersistMs    = 5 * time.Minute
	cfgBraveKey          = ""                              // optional Tavily API key (Bearer); empty = Tavily keyless free tier
	cfgBraveURL          = "https://api.tavily.com/search" // web-search provider endpoint; override in tests/local mock
)

// ---------------------------------------------------------------------------
// Per-provider@key health / degraded state
// ---------------------------------------------------------------------------

type healthKey struct {
	provider string
	keyIdx   int
}

type healthState struct {
	mu       sync.Mutex
	cooldown map[healthKey]*degraded
	pbox     map[string]*degraded        // provider-wide connect/timing-out cooldown (network-level, all keys)
	rl       map[string]rlState          // per-provider ratelimit pressure (decays)
	baseline map[healthKey]time.Duration // EMA of first-byte latency
}

type rlState struct {
	strikes int
	last    time.Time
}

type degraded struct {
	until   time.Time
	strikes int
	kind    string
}

var health = &healthState{
	cooldown: map[healthKey]*degraded{},
	pbox:     map[string]*degraded{},
	rl:       map[string]rlState{},
	baseline: map[healthKey]time.Duration{},
}

func (h *healthState) key(c *cand) healthKey { return healthKey{c.provider, c.keyIdx} }

func (h *healthState) isCooled(c *cand) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if pb, ok := h.pbox[c.provider]; ok && time.Now().Before(pb.until) {
		return true
	}
	d, ok := h.cooldown[h.key(c)]
	if !ok || time.Now().After(d.until) {
		if ok {
			delete(h.cooldown, h.key(c))
		}
		return false
	}
	return true
}

// cooldownRemaining reports whether c is currently cooled and, if so, how long
// until it becomes eligible again (the shorter of any provider-wide and
// per-key cooldown still in effect). Used by the failover loop to decide
// whether a short bounded wait-and-retry is worthwhile instead of declaring
// the whole chain exhausted just because every candidate happens to be in a
// brief cooldown window right now.
func (h *healthState) cooldownRemaining(c *cand) (time.Duration, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	wait := time.Duration(-1)
	if pb, ok := h.pbox[c.provider]; ok && now.Before(pb.until) {
		wait = pb.until.Sub(now)
	}
	if d, ok := h.cooldown[h.key(c)]; ok && now.Before(d.until) {
		if w := d.until.Sub(now); wait < 0 || w < wait {
			wait = w
		}
	}
	if wait < 0 {
		return 0, false
	}
	return wait, true
}

// jitter adds up to +20% randomized spread to a backoff duration so that many
// concurrent requests penalizing the same provider/key at the same instant
// don't all wake up and retry in the same lockstep instant (thundering herd).
func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}
	spread := int64(d) / 5
	if spread <= 0 {
		return d
	}
	return d + time.Duration(rand.Int63n(spread+1))
}

func (h *healthState) penalize(ctx context.Context, c *cand, kind string, retryAfter time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	k := h.key(c)
	cur := h.cooldown[k]
	if cur == nil {
		cur = &degraded{}
		h.cooldown[k] = cur
	}
	cur.strikes++
	var ms time.Duration
	base := cfgCooldownBase
	cap := cfgCooldownMax
	if kind == "auth" {
		base, cap = cfgCooldownAuth, cfgCooldownAuth
	} else if kind == "ratelimit" && retryAfter > 0 {
		ms = retryAfter
		if ms < base {
			ms = base
		}
		if ms > cap {
			ms = cap
		}
	}
	if ms == 0 {
		ms = base * time.Duration(1<<(cur.strikes-1))
		if ms > cap {
			ms = cap
		}
	}
	ms = jitter(ms)
	cur.until = time.Now().Add(ms)
	cur.kind = kind
	ReqLog(ctx, "cooldown %s@%s#k%d for %s (%s, strike %d)", c.model, c.provider, c.keyIdx, ms, kind, cur.strikes)
	// A connect/time-out is a provider-wide (network-level) failure: it affects
	// every key of this provider, so blacklist the whole provider briefly so the
	// chain jumps to the next provider instead of serially burning each key.
	if kind == "connect" {
		pb := h.pbox[c.provider]
		if pb == nil {
			pb = &degraded{}
			h.pbox[c.provider] = pb
		}
		pb.strikes++
		pbms := base * time.Duration(1<<(pb.strikes-1))
		if pbms > cap {
			pbms = cap
		}
		pb.until = time.Now().Add(jitter(pbms))
		pb.kind = "connect"
		ReqLog(ctx, "provider-wide cooldown %s for %s (connect strike %d)", c.provider, pbms, pb.strikes)
	}
	// Ratelimit pressure ramp: a provider where several keys are 429-ing within a
	// short window is saturated as a whole. Once pressure crosses a threshold, put
	// the whole provider on a brief cooldown so requests skip it (and land on the
	// next provider) instead of wasting a 2-key shot per request. A success on any
	// key (reward) clears the ramp, so a single hot key still lets rotation win.
	if kind == "ratelimit" {
		now := time.Now()
		rs := h.rl[c.provider]
		if now.Sub(rs.last) > rlDecayWindow {
			rs.strikes = 0
		}
		rs.strikes++
		rs.last = now
		h.rl[c.provider] = rs
		if rs.strikes >= rlProviderThreshold {
			pb := h.pbox[c.provider]
			if pb == nil || now.After(pb.until) {
				if pb == nil {
					pb = &degraded{}
					h.pbox[c.provider] = pb
				}
				pb.strikes++
				pbms := base * time.Duration(1<<(pb.strikes-1))
				if pbms > cap {
					pbms = cap
				}
				pb.until = now.Add(jitter(pbms))
				pb.kind = "ratelimit"
				ReqLog(ctx, "provider-wide cooldown %s for %s (ratelimit ramp strike %d)", c.provider, pbms, pb.strikes)
				// keep ramp modest so a short burst doesn't blacklist forever
				if rs.strikes > rlProviderThreshold+2 {
					rs.strikes = rlProviderThreshold // stop runaway escalation
				}
				h.rl[c.provider] = rs
			}
		}
	}
}

func (h *healthState) reward(ctx context.Context, c *cand) {
	h.mu.Lock()
	defer h.mu.Unlock()
	k := h.key(c)
	if d, ok := h.cooldown[k]; ok && d.kind != "ghost" {
		// keep latency/ghost cooldowns; clear failure strikes on success
		if d.kind != "latency" {
			delete(h.cooldown, k)
			ReqLog(ctx, "recovered %s@%s#k%d", c.model, c.provider, c.keyIdx)
		}
	}
	if pb, ok := h.pbox[c.provider]; ok && pb.kind != "ghost" {
		delete(h.pbox, c.provider)
		ReqLog(ctx, "provider-wide recovered %s", c.provider)
	}
	delete(h.rl, c.provider)
}

// adaptTimeout returns the per-attempt timeout for a candidate, based on its
// observed baseline: baseline*2 + margin, clamped to [baseline+2s, 60s].
// This replaces a fixed timeout that falsely aborted slow-but-valid providers.
func (h *healthState) adaptTimeout(c *cand, hardCap time.Duration) time.Duration {
	h.mu.Lock()
	base, ok := h.baseline[h.key(c)]
	h.mu.Unlock()
	if !ok {
		return minDuration(cfgAttemptTimeout, hardCap)
	}
	to := base*2 + 2*time.Second
	if to < base+2*time.Second {
		to = base + 2*time.Second
	}
	if to > hardCap {
		to = hardCap
	}
	return to
}

func (h *healthState) recordLatency(c *cand, d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	k := h.key(c)
	cur, ok := h.baseline[k]
	if !ok {
		h.baseline[k] = d
		return
	}
	// EMA with alpha 0.3
	h.baseline[k] = time.Duration(0.7*float64(cur) + 0.3*float64(d))
}

// ---------------------------------------------------------------------------
// Candidate chain
// ---------------------------------------------------------------------------

type cand struct {
	model    string
	provider string
	keyIdx   int
}

func deref(s string) string {
	if strings.HasPrefix(s, "$") {
		return os.Getenv(s[1:])
	}
	return s
}

func loadConfig() {
	if v := os.Getenv("PORT"); v != "" {
		cfgPort = v
	}
	if v := os.Getenv("FALLBACK_MODEL"); v != "" {
		cfgFallbackModel = v
	}
	if v := os.Getenv("FALLBACK_CHAIN"); v != "" {
		for _, e := range strings.Split(v, ",") {
			if e = strings.TrimSpace(e); e != "" {
				cfgFallbackChain = append(cfgFallbackChain, e)
			}
		}
	}
	if v := os.Getenv("KNOWN_MODELS"); v != "" {
		for _, m := range strings.Split(v, ",") {
			if m = strings.TrimSpace(m); m != "" {
				cfgKnownModels[m] = true
			}
		}
	}
	if v := os.Getenv("ATTEMPT_TIMEOUT_MS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfgAttemptTimeout = time.Duration(i) * time.Millisecond
		}
	}
	if v := os.Getenv("TOTAL_BUDGET_MS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfgTotalBudget = time.Duration(i) * time.Millisecond
		}
	}
	if v := os.Getenv("COOLDOWN_BASE_MS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfgCooldownBase = time.Duration(i) * time.Millisecond
		}
	}
	if v := os.Getenv("COOLDOWN_MAX_MS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfgCooldownMax = time.Duration(i) * time.Millisecond
		}
	}
	if v := os.Getenv("COOLDOWN_AUTH_MS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfgCooldownAuth = time.Duration(i) * time.Millisecond
		}
	}
	if v := os.Getenv("SLOW_HEADER_MS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfgSlowHeaderMS = i
		}
	}
	if v := os.Getenv("BRAVE_KEY"); v != "" {
		cfgBraveKey = strings.TrimSpace(v)
	}
	if v := os.Getenv("BRAVE_URL"); v != "" {
		cfgBraveURL = strings.TrimSpace(v)
	}
	if v := os.Getenv("MAX_BODY_BYTES"); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfgMaxBodyBytes = i
		}
	}
	if v := os.Getenv("CACHE_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfgCacheEnabled = b
		}
	}
	if v := os.Getenv("CACHE_MAX_ENTRIES"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfgCacheMaxEntries = i
		}
	}
	if v := os.Getenv("CACHE_TTL_MS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfgCacheTTL = time.Duration(i) * time.Millisecond
		}
	}
	if v := os.Getenv("CACHE_TEMP_ZERO_ONLY"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfgCacheTempZeroOnly = b
		}
	}
	if v := os.Getenv("CACHE_DIR"); v != "" {
		cfgCacheDir = v
	}
	if v := os.Getenv("CACHE_PERSIST_MS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfgCachePersistMs = time.Duration(i) * time.Millisecond
		}
	}
	if v := os.Getenv("MODEL_PROVIDER"); v != "" {
		if err := json.Unmarshal([]byte(v), &cfgModelProvider); err != nil {
			log.Print("MODEL_PROVIDER parse failed: ", err)
		}
	}
	if v := os.Getenv("MODEL_MAP"); v != "" {
		if err := json.Unmarshal([]byte(v), &cfgModelMap); err != nil {
			log.Print("MODEL_MAP parse failed: ", err)
		}
	}

	if v := os.Getenv("UPSTREAMS"); v != "" {
		raw := map[string]Provider{}
		if err := json.Unmarshal([]byte(v), &raw); err != nil {
			log.Print("UPSTREAMS parse failed: ", err)
		}
		for name, p := range raw {
			p.Name = name
			p.URL = deref(p.URL)
			p.Key = deref(p.Key)
			p.UA = deref(p.UA)
			// normalize key rotation pool
			if strings.Contains(p.Key, ",") {
				for _, k := range strings.Split(p.Key, ",") {
					if k = strings.TrimSpace(k); k != "" {
						p.Keys = append(p.Keys, k)
					}
				}
			} else if p.Key != "" {
				p.Keys = []string{p.Key}
			} else {
				p.Keys = []string{""}
			}
			cfgProviders[name] = &p
			cfgProviderOrder = append(cfgProviderOrder, name)
		}
	}
	if len(cfgProviders) == 0 {
		cfgProviderOrder = []string{"default"}
		cfgProviders["default"] = &Provider{
			Name: "default",
			URL:  os.Getenv("UPSTREAM"),
			UA:   os.Getenv("UPSTREAM_UA"),
			Keys: []string{os.Getenv("UPSTREAM_KEY")},
		}
	}
}

func providerFor(model string) string {
	if p, ok := cfgModelProvider[model]; ok {
		return p
	}
	for _, name := range cfgProviderOrder {
		for _, m := range cfgProviders[name].Models {
			if m == model {
				return name
			}
		}
	}
	if len(cfgProviderOrder) > 0 {
		return cfgProviderOrder[0]
	}
	return ""
}

func resolveModel(m string) string {
	if r, ok := cfgModelMap[m]; ok {
		return r
	}
	if cfgKnownModels[m] {
		return m
	}
	return cfgFallbackModel
}

// candidates builds the ordered attempt list, expanding providers with multiple
// keys into per-key attempts.
func candidates(rawModel string) []cand {
	first := resolveModel(rawModel)
	// pin if the chain specifies model@provider for the primary
	pin := ""
	for _, e := range cfgFallbackChain {
		if i := strings.Index(e, "@"); i > 0 {
			em, ep := e[:i], e[i+1:]
			if resolveModel(em) == first && cfgProviders[ep] != nil {
				pin = ep
				break
			}
		}
	}
	out := []cand{{model: first, provider: orDefault(pin, providerFor(first)), keyIdx: 0}}
	seen := map[string]bool{out[0].model + "@" + out[0].provider: true}
	for _, e := range cfgFallbackChain {
		ep := ""
		em := e
		if i := strings.Index(e, "@"); i > 0 {
			em, ep = e[:i], e[i+1:]
		}
		m := resolveModel(em)
		pr := ep
		if cfgProviders[pr] == nil {
			pr = providerFor(m)
		}
		k := m + "@" + pr
		if !seen[k] {
			seen[k] = true
			out = append(out, cand{model: m, provider: pr, keyIdx: 0})
		}
	}
	// expand into per-key, dedup. Multi-key providers (e.g. dahl with several
	// keys, each with its own concurrency limit) get round-robin as the starting
	// key so parallel requests spread across keys instead of always hammering
	// key index 0 (which saturates one key's concurrency first under bursts).
	var expanded []cand
	seen2 := map[string]bool{}
	for _, c := range out {
		prov := cfgProviders[c.provider]
		keys := prov.Keys
		if len(keys) < 2 {
			ki := 0
			if _, ok := seen2[c.model+"@"+c.provider+"#k0"]; !ok {
				seen2[c.model+"@"+c.provider+"#k0"] = true
				expanded = append(expanded, cand{c.model, c.provider, ki})
			}
			continue
		}
		start := int(atomic.AddInt64(&keyRotate, 1)-1) % len(keys)
		for n := 0; n < len(keys); n++ {
			ki := (start + n) % len(keys)
			kk := c.model + "@" + c.provider + "#k" + strconv.Itoa(ki)
			if !seen2[kk] {
				seen2[kk] = true
				expanded = append(expanded, cand{c.model, c.provider, ki})
			}
		}
	}
	return expanded
}

// minThinkingMaxTokens floors max_tokens whenever reasoning is switched on.
// Observed live: a small client max_tokens (e.g. Claude Code's small-fast-model
// calls, often 20-30) combined with a provider forced into reasoning mode can
// consume the ENTIRE budget on the <thinking> trace and return truly empty
// text (stop_reason:"max_tokens", no answer at all) — the model never got to
// the answer. That silently drops the turn's actual content and, cascading,
// starves Claude Code's own context/compaction accounting of a real reply.
// The floor is intentionally modest: enough room for a short thinking trace
// plus a real answer, well under every configured provider's MaxOutput cap
// (still clamped downstream by that cap regardless).
const minThinkingMaxTokens = 1024

// numTokens reads a max_tokens-shaped value that may arrive as float64 (json
// decode) or int64 (in-memory), returning ok=false for anything else/missing.
func numTokens(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	case int:
		return n, true
	}
	return 0, false
}

// applyThinking sets (or clears) the per-provider reasoning/thinking request
// shape on an outbound OpenAI-style body. The provider flags say what the
// backend is CAPABLE of being explicitly ASKED for; disabled reflects what
// THIS request actually wants (the client's own "thinking":{"type":"disabled"})
// and always wins, so an explicit opt-out isn't overridden by a provider that
// forces reasoning on.
//
// The max_tokens floor below is intentionally NOT gated on the provider's own
// reasoning_param/thinking_param flags: several providers (observed live,
// e.g. glm-5.3-flash@bai) emit a reasoning/thinking block unprompted even
// with neither flag set, so "did we explicitly ask for thinking" is not a
// reliable predicate for "might this response contain a thinking block that
// eats the budget." Only an explicit client opt-out should skip the floor.
func applyThinking(body map[string]interface{}, prov *Provider, disabled bool) {
	if prov.ReasoningParam && !disabled {
		body["reasoning"] = map[string]interface{}{"enabled": true}
	} else {
		delete(body, "reasoning")
	}
	if prov.ThinkingParam && !disabled {
		if body["chat_template_kwargs"] == nil {
			body["chat_template_kwargs"] = map[string]interface{}{"thinking": true}
		}
	} else {
		delete(body, "chat_template_kwargs")
	}
	if !disabled {
		if mt, ok := numTokens(body["max_tokens"]); !ok || mt < minThinkingMaxTokens {
			body["max_tokens"] = minThinkingMaxTokens
		}
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}


// zenSysprompt returns the opencode system-prompt fingerprint (multi-line, so it
// lives in a file referenced by ZEN_SYSPROMPT_FILE — shell-sourced conf.d values
// must stay single-line).
func zenSysprompt() string {
	if f := os.Getenv("ZEN_SYSPROMPT_FILE"); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			return string(b)
		}
	}
	return os.Getenv("ZEN_SYSPROMPT")
}

// genSessionID returns a fresh "ses_" + 26 random alphanumeric characters.
// Called per-request so every upstream call gets a unique X-Opencode-Session.
var sessionIDChars = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func genSessionID() string {
	b := make([]byte, 30)
	b[0], b[1], b[2], b[3] = 's', 'e', 's', '_'
	for i := 4; i < 30; i++ {
		b[i] = sessionIDChars[rand.Intn(len(sessionIDChars))]
	}
	return string(b)
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// Body translation: anthropic -> openai
// ---------------------------------------------------------------------------

func textOf(c interface{}) string {
	switch v := c.(type) {
	case string:
		return v
	}
	if arr, ok := c.([]interface{}); ok {
		var sb strings.Builder
		for _, b := range arr {
			if m, ok := b.(map[string]interface{}); ok && m["type"] == "text" {
				if t, ok := m["text"].(string); ok {
					sb.WriteString(t)
					sb.WriteString("\n")
				}
			}
		}
		return sb.String()
	}
	return ""
}

// imagePart converts a single Anthropic "image" content block (base64 source)
// into an OpenAI-style image_url part, or nil if the block isn't a usable
// inline image.
func imagePart(bm map[string]interface{}) map[string]interface{} {
	if bm["type"] != "image" {
		return nil
	}
	src, ok := bm["source"].(map[string]interface{})
	if !ok || src["type"] != "base64" {
		return nil
	}
	data, ok := src["data"].(string)
	if !ok || data == "" {
		return nil
	}
	mt := toString(src["media_type"])
	if mt == "" {
		mt = "image/png"
	}
	return map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "data:" + mt + ";base64," + data}}
}

// imagePartsOf scans a tool_result content value (string or block array) for
// any inline images it carries, returning them as OpenAI-style image_url
// parts. A plain-string tool_result content has none.
func imagePartsOf(c interface{}) []interface{} {
	arr, ok := c.([]interface{})
	if !ok {
		return nil
	}
	var out []interface{}
	for _, b := range arr {
		if bm, ok := b.(map[string]interface{}); ok {
			if p := imagePart(bm); p != nil {
				out = append(out, p)
			}
		}
	}
	return out
}

// convertBody translates an Anthropic /v1/messages request body into an
// OpenAI-style chat/completions body. The second return value reports
// whether the client explicitly disabled extended thinking
// (thinking:{type:"disabled"}) — kept out of the body map itself since the
// streaming failover path reuses the same map object across several
// sequential candidate attempts, and a value embedded in the map would need
// deleting/re-adding on every attempt instead of just being read.
func convertBody(body map[string]interface{}) (map[string]interface{}, bool) {
	out := map[string]interface{}{
		"model":  resolveModel(toString(body["model"])),
		"stream": true,
	}
	thinkingDisabled := false
	if v, ok := body["max_tokens"]; ok {
		out["max_tokens"] = v
	}
	if v, ok := body["temperature"]; ok && v != nil {
		out["temperature"] = v
	}
	if v, ok := body["top_p"]; ok && v != nil {
		out["top_p"] = v
	}
	if v, ok := body["stop_sequences"]; ok {
		if arr, ok := v.([]interface{}); ok && len(arr) > 0 {
			out["stop"] = arr
		}
	}

	var msgs []interface{}
	if sys, ok := body["system"]; ok && sys != nil {
		// Claude Code sends `system` either as a plain string OR (default in
		// modern versions) as an array of text blocks that may carry
		// cache_control breakpoints: [{"type":"text","text":"...","cache_control":{"type":"ephemeral"}}].
		// Join all text blocks so the full system prompt always reaches the
		// model (the previous string-only handling silently DROPPED block-array
		// system prompts).
		if s, ok := toStringOk(sys); ok && s != "" {
			msgs = append(msgs, map[string]interface{}{"role": "system", "content": s})
		} else if arr, ok := sys.([]interface{}); ok {
			var sb strings.Builder
			for _, b := range arr {
				if bm, ok := b.(map[string]interface{}); ok {
					if bm["type"] == "text" {
						if t, ok := bm["text"].(string); ok && t != "" {
							sb.WriteString(t)
							sb.WriteString("\n")
						}
					}
				}
			}
			if s := strings.TrimRight(sb.String(), "\n"); s != "" {
				msgs = append(msgs, map[string]interface{}{"role": "system", "content": s})
			}
		}
	}
	for _, m := range toSlice(body["messages"]) {
		mm, _ := m.(map[string]interface{})
		role := toString(mm["role"])
		if role == "user" {
			var parts []interface{}
			flush := func() {
				if len(parts) == 0 {
					return
				}
				if len(parts) == 1 {
					if t, ok := parts[0].(map[string]interface{}); ok && t["type"] == "text" {
						msgs = append(msgs, map[string]interface{}{"role": "user", "content": t["text"]})
						parts = parts[:0]
						return
					}
				}
				msgs = append(msgs, map[string]interface{}{"role": "user", "content": parts})
				parts = parts[:0]
			}
			// Accept a plain-string content (Claude Code's small-fast-model
			// requests, e.g. openai/gpt-oss-20b, send {"role":"user","content":
			// "..."} instead of the block-array form). Without this the message
			// was silently dropped, the body became empty and the whole fallback
			// chain walked and failed for nothing.
			if s, ok := mm["content"].(string); ok && s != "" {
				msgs = append(msgs, map[string]interface{}{"role": "user", "content": s})
				continue
			}
			for _, b := range toSlice(mm["content"]) {
				bm, _ := b.(map[string]interface{})
				switch bm["type"] {
				case "tool_result":
					flush()
					txt := textOf(bm["content"])
					if isErr, _ := bm["is_error"].(bool); isErr {
						// Most OpenAI-compatible backends have no separate is_error
						// channel on tool messages; without this a failed tool call
						// reads identically to a successful one and the model won't
						// self-correct.
						if txt == "" {
							txt = "(tool call failed with no output)"
						}
						txt = "Error: " + txt
					}
					msgs = append(msgs, map[string]interface{}{"role": "tool", "tool_call_id": toString(bm["tool_use_id"]), "content": txt})
					// tool_result content can itself carry image blocks (e.g. a
					// screenshot). A "tool" role message's content isn't reliably
					// accepted as a multimodal array across this provider set, so
					// forward any images as an immediate follow-up user message
					// instead of silently dropping them.
					if imgs := imagePartsOf(bm["content"]); len(imgs) > 0 {
						imgParts := append([]interface{}{map[string]interface{}{"type": "text", "text": "(image returned by the tool call above)"}}, imgs...)
						msgs = append(msgs, map[string]interface{}{"role": "user", "content": imgParts})
					}
				case "text":
					parts = append(parts, map[string]interface{}{"type": "text", "text": toString(bm["text"])})
				case "image":
					if p := imagePart(bm); p != nil {
						parts = append(parts, p)
					}
				default:
					// Unhandled block types (e.g. "document"/PDF attachments, or
					// future block kinds) must not silently vanish from the
					// conversation — that leaves the model reasoning over context
					// it was never actually given. Surface a visible placeholder
					// instead of dropping the block entirely.
					if t, ok := bm["type"].(string); ok && t != "" {
						parts = append(parts, map[string]interface{}{"type": "text", "text": "[unsupported attachment: " + t + "]"})
					}
				}
			}
			flush()
		} else if role == "assistant" {
			var txt strings.Builder
			var toolCalls []interface{}
			for _, b := range toSlice(mm["content"]) {
				bm, _ := b.(map[string]interface{})
				switch bm["type"] {
				case "text":
					if txt.Len() > 0 {
						txt.WriteString("\n")
					}
					txt.WriteString(toString(bm["text"]))
				case "tool_use":
					args, _ := json.Marshal(bm["input"])
					toolCalls = append(toolCalls, map[string]interface{}{
						"id":   toString(bm["id"]),
						"type": "function",
						"function": map[string]interface{}{
							"name":      toString(bm["name"]),
							"arguments": string(args),
						},
					})
				}
			}
			am := map[string]interface{}{"role": "assistant", "content": nil}
			if txt.Len() > 0 {
				am["content"] = txt.String()
			}
			if len(toolCalls) > 0 {
				am["tool_calls"] = toolCalls
			}
			if txt.Len() > 0 || len(toolCalls) > 0 {
				msgs = append(msgs, am)
			}
		}
	}
	out["messages"] = msgs

	if arr := toSlice(body["tools"]); len(arr) > 0 {
		var tools []interface{}
		for _, t := range arr {
			tm, _ := t.(map[string]interface{})
			params := tm["input_schema"]
			if params == nil {
				params = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
			}
			tools = append(tools, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        toString(tm["name"]),
					"description": toString(tm["description"]),
					"parameters":  params,
				},
			})
		}
		out["tools"] = tools
	}

	if tc, ok := body["tool_choice"].(map[string]interface{}); ok {
		switch tc["type"] {
		case "auto":
			out["tool_choice"] = "auto"
		case "any":
			out["tool_choice"] = "required"
		case "none":
			out["tool_choice"] = "none"
		case "tool":
			out["tool_choice"] = map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": toString(tc["name"])}}
		}
		if disable, ok := tc["disable_parallel_tool_use"].(bool); ok && disable {
			out["parallel_tool_calls"] = false
		}
	}

	// Respect an explicit client request to turn extended thinking off. The
	// per-provider reasoning/thinking flags below are an upstream CAPABILITY
	// (can this provider even do it), not client INTENT (does this particular
	// request want it) — without this, a request that explicitly disables
	// thinking still pays the latency/token cost of forced reasoning on a
	// provider configured with reasoning_param/thinking_param.
	if th, ok := body["thinking"].(map[string]interface{}); ok {
		if toString(th["type"]) == "disabled" {
			thinkingDisabled = true
		}
	}
	return out, thinkingDisabled
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := v.(bool); ok {
		return strconv.FormatBool(b)
	}
	if f, ok := v.(float64); ok {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return ""
}

func toStringOk(v interface{}) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func toSlice(v interface{}) []interface{} {
	if arr, ok := v.([]interface{}); ok {
		return arr
	}
	if s, ok := v.(string); ok {
		return []interface{}{s}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Logging
// ---------------------------------------------------------------------------

var mu io.Writer = os.Stdout

func logLine(rid, format string, a ...interface{}) {
	prefix := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	if rid != "" {
		fmt.Fprintf(mu, "%s [%s] %s\n", prefix, rid, fmt.Sprintf(format, a...))
	} else {
		fmt.Fprintf(mu, "%s %s\n", prefix, fmt.Sprintf(format, a...))
	}
}

var ridCounter int64

func newRID() string {
	n := "req" + strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.FormatInt(ridCounter, 36)
	ridCounter++
	return n
}

// Round-robin key rotation cursor so parallel requests on a multi-key provider
// (e.g. dahl) spread primary load across all keys instead of only key index 0.
var keyRotate int64

// Request-scoped logging ID: threaded through the request context instead of a
// global, so concurrent request-handler goroutines cannot clobber each other's
// log attribution (the old package-global currentRID made logs show [-]/wrong
// request IDs under concurrency).

type ridKey struct{}

func withRID(ctx context.Context, rid string) context.Context {
	return context.WithValue(ctx, ridKey{}, rid)
}

func ridFrom(ctx context.Context) string {
	if s, ok := ctx.Value(ridKey{}).(string); ok && s != "" {
		return s
	}
	return ""
}

func ReqLog(ctx context.Context, format string, a ...interface{}) {
	rid := ridFrom(ctx)
	if rid == "" {
		rid = "-"
	}
	logLine(rid, format, a...)
}

// ---------------------------------------------------------------------------
// Response translation (SSE upstream -> Anthropic SSE)
// ---------------------------------------------------------------------------

type emitter struct {
	w        io.Writer
	flusher  http.Flusher
	headSent bool
	header   http.Header
	closed   bool
}

func (e *emitter) send(event string, data interface{}) {
	if e.closed {
		return
	}
	b, _ := json.Marshal(data)
	fmt.Fprintf(e.w, "event: %s\ndata: %s\n\n", event, string(b))
	if e.flusher != nil {
		e.flusher.Flush()
	}
}

func (e *emitter) end() { e.closed = true }

// streamState tracks per-request translation state
type streamState struct {
	started    bool
	idx        int
	curKind    string
	curIndex   int
	tools      map[int]int // upstream tool index -> anthropic block index
	openTools  map[int]bool
	uIn        int
	uOut       int
	think      strings.Builder
	text       strings.Builder
	sawAny     bool
	sawPayload bool
	agg        []map[string]interface{}
	stop       string
}

func newStreamState() *streamState {
	return &streamState{
		tools:     map[int]int{},
		openTools: map[int]bool{},
		stop:      "end_turn",
	}
}

func (s *streamState) ensureStarted(em *emitter, model string) {
	if s.started {
		return
	}
	em.send("message_start", map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id": "msg_" + strconv.FormatInt(time.Now().UnixNano(), 36), "type": "message", "role": "assistant",
			"model": model, "content": []interface{}{}, "stop_reason": nil, "stop_sequence": nil,
			"usage": map[string]interface{}{"input_tokens": 0, "output_tokens": 0, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0},
		},
	})
	s.started = true
}

func (s *streamState) closeCur(em *emitter) {
	switch s.curKind {
	case "thinking":
		if em != nil {
			em.send("signature_delta", map[string]interface{}{"type": "content_block_delta", "index": s.curIndex, "delta": map[string]interface{}{"type": "signature_delta", "signature": "sig-shim-" + strconv.Itoa(s.curIndex)}})
			em.send("content_block_stop", map[string]interface{}{"type": "content_block_stop", "index": s.curIndex})
		}
		s.agg = append(s.agg, map[string]interface{}{"type": "thinking", "thinking": s.think.String()})
	case "text":
		if em != nil {
			em.send("content_block_stop", map[string]interface{}{"type": "content_block_stop", "index": s.curIndex})
		}
		s.agg = append(s.agg, map[string]interface{}{"type": "text", "text": s.text.String()})
	}
	s.think.Reset()
	s.text.Reset()
	s.curKind = ""
}

func (s *streamState) handleChunk(em *emitter, model string, j map[string]interface{}) {
	ch, _ := j["choices"].([]interface{})
	// Usage can arrive at the top level OR, for groq streaming, nested inside
	// x_groq.usage (groq never emits a top-level usage field in SSE chunks).
	u, _ := j["usage"].(map[string]interface{})
	if xg, ok := j["x_groq"].(map[string]interface{}); ok {
		if xu, ok := xg["usage"].(map[string]interface{}); ok && u == nil {
			u = xu
		}
	}
	if u != nil {
		s.uIn = asInt(u["prompt_tokens"])
		s.uOut = asInt(u["completion_tokens"])
	}
	if len(ch) == 0 {
		return
	}
	choice, _ := ch[0].(map[string]interface{})
	if choice == nil {
		return
	}
	s.sawAny = true
	s.ensureStarted(em, model)
	d, _ := choice["delta"].(map[string]interface{})
	if d == nil {
		d = map[string]interface{}{}
	}
	reason := ""
	if r, ok := d["reasoning_content"].(string); ok {
		reason = r
	} else if r, ok := d["reasoning"].(string); ok {
		reason = r
	}
	if reason != "" {
		s.sawPayload = true
		if s.curKind != "thinking" {
			s.closeCur(em)
			s.curKind = "thinking"
			s.curIndex = s.idx
			s.idx++
			if em != nil {
				em.send("content_block_start", map[string]interface{}{"type": "content_block_start", "index": s.curIndex, "content_block": map[string]interface{}{"type": "thinking", "thinking": ""}})
			}
		}
		if em != nil {
			em.send("content_block_delta", map[string]interface{}{"type": "content_block_delta", "index": s.curIndex, "delta": map[string]interface{}{"type": "thinking_delta", "thinking": reason}})
		}
		s.think.WriteString(reason)
	}
	if c, ok := d["content"].(string); ok && c != "" {
		s.sawPayload = true
		if s.curKind != "text" {
			s.closeCur(em)
			s.curKind = "text"
			s.curIndex = s.idx
			s.idx++
			if em != nil {
				em.send("content_block_start", map[string]interface{}{"type": "content_block_start", "index": s.curIndex, "content_block": map[string]interface{}{"type": "text", "text": ""}})
			}
		}
		if em != nil {
			em.send("content_block_delta", map[string]interface{}{"type": "content_block_delta", "index": s.curIndex, "delta": map[string]interface{}{"type": "text_delta", "text": c}})
		}
		s.text.WriteString(c)
	}
	for _, tc := range toSlice(d["tool_calls"]) {
		tcm, _ := tc.(map[string]interface{})
		s.sawPayload = true
		s.closeCur(em)
		ti := int(toFloat(tcm["index"]))
		ai, exists := s.tools[ti]
		if !exists {
			ai = s.idx
			s.idx++
			s.tools[ti] = ai
			s.openTools[ai] = true
			name, _ := tcm["function"].(map[string]interface{})
			if em != nil {
				id := s.tools[ti]
				_ = id
				em.send("content_block_start", map[string]interface{}{"type": "content_block_start", "index": ai, "content_block": map[string]interface{}{"type": "tool_use", "id": toString(tcm["id"]), "name": toString(name["name"]), "input": map[string]interface{}{}}})
			}
			s.agg = append(s.agg, map[string]interface{}{"__tool": true, "index": ai, "id": toString(tcm["id"]), "name": toString(name["name"]), "args": ""})
		}
		arg := ""
		if name, ok := tcm["function"].(map[string]interface{}); ok {
			arg = toString(name["arguments"])
		}
		if arg != "" {
			if em != nil {
				em.send("content_block_delta", map[string]interface{}{"type": "content_block_delta", "index": ai, "delta": map[string]interface{}{"type": "input_json_delta", "partial_json": arg}})
			}
			// append to agg rec
			for _, r := range s.agg {
				if r["__tool"] == true && r["index"] == float64(ai) {
					r["args"] = toString(r["args"]) + arg
					break
				}
			}
		}
	}
	if fr, ok := choice["finish_reason"].(string); ok {
		switch fr {
		case "tool_calls":
			s.stop = "tool_use"
		case "length":
			s.stop = "max_tokens"
		default:
			s.stop = "end_turn"
		}
	}
}

func toFloat(v interface{}) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

// asInt extracts an integer from float64/int/json.Number typed values (usage
// counts arrive as float64 after JSON decode but as int when an in-memory map
// was built directly).
func asInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case float32:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
	}
	return 0
}

// safeParseArgs parses tool arguments tolerant of partial JSON
func safeParseArgs(s string) interface{} {
	var v interface{}
	if s == "" {
		return map[string]interface{}{}
	}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return map[string]interface{}{"_raw": s}
	}
	return v
}

func finalize(em *emitter, s *streamState) {
	em.send("message_delta", map[string]interface{}{
		"type":  "message_delta",
		"delta": map[string]interface{}{"stop_reason": s.stop, "stop_sequence": nil},
		"usage": map[string]interface{}{"input_tokens": s.uIn, "output_tokens": s.uOut, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0},
	})
	em.send("message_stop", map[string]interface{}{"type": "message_stop"})
}

// ---------------------------------------------------------------------------
// failover chain streaming
// ---------------------------------------------------------------------------

// size-limited body reader
type limitedReader struct {
	r   io.Reader
	rem int64
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.rem <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.rem {
		p = p[:l.rem]
	}
	n, err := l.r.Read(p)
	l.rem -= int64(n)
	return n, err
}

func shouldAbort(httpStatus int) bool {
	return httpStatus == 401 || httpStatus == 403
}

var httpClient = &http.Client{
	Timeout: 0, // use per-candidate context timeouts
	Transport: &http.Transport{
		MaxIdleConnsPerHost:   8,
		MaxConnsPerHost:       8,
		IdleConnTimeout:       30 * time.Second,
		ResponseHeaderTimeout: 0, // managed via context
		DisableKeepAlives:     false,
		ForceAttemptHTTP2:     true,
	},
}

// newTTFBClient returns an http.Client whose ResponseHeaderTimeout bounds ONLY
// time-to-first-byte, leaving the streaming/response body unbounded (a long
// generation should not be killed by the short TTFB timeout).
func newTTFBClient(ttfb time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   8,
			MaxConnsPerHost:       8,
			IdleConnTimeout:       30 * time.Second,
			ResponseHeaderTimeout: ttfb,
			ForceAttemptHTTP2:     true,
		},
	}
}

// attemptOne performs a single candidate attempt (streaming). Returns:
//   - payloadOK: some usable content was emitted to the client
//   - terminal: true when no further candidates should be tried (client gone)
func attemptOne(ctx context.Context, w *emitter, c *cand, model string, oaiBody map[string]interface{}, isStream bool, thinkingDisabled bool, estIn int, cap *respCapture) (payloadOK bool, terminal bool) {
	prov := cfgProviders[c.provider]
	if prov == nil {
		ReqLog(ctx, "provider %s missing, skipping", c.provider)
		return false, false
	}
	to := health.adaptTimeout(c, cfgTotalBudget)
	// Per-attempt timeout covers ONLY connection + first-byte (headers), NOT the
	// whole stream — a long response (thinking + tool_calls) legitimately streams
	// far longer than TTFB. ResponseHeaderTimeout bounds TTFB, but the body can
	// stream as long as bytes keep arriving (idle timeout in the read loop below).
	client := newTTFBClient(to)

	key := ""
	if c.keyIdx < len(prov.Keys) {
		key = prov.Keys[c.keyIdx]
	}

	// Build request
	oaiBody["model"] = c.model
	// Per-provider thinking shape is deduced dynamically from each provider's own
	// flags (NOT hardcoded), matching the old Node shim:
	//   reasoning_param -> top-level "reasoning":{"enabled":true} (opencode-native; big-pickle@zen)
	//   thinking_param  -> "chat_template_kwargs":{"thinking":true} (dahl/Kimi, some zen models)
	// unless the client explicitly asked for thinking:disabled, which wins.
	applyThinking(oaiBody, prov, thinkingDisabled)
	// opencode zen free tier REQUIRES stream:true in the body (MITM-confirmed
	// 2026-10-04: a stream:false body gets FreeTierError even with valid ids).
	// The nonStream path therefore sends stream:true to zen and reassembles the
	// SSE reply; every other provider keeps its requested stream mode.
	if isStream || c.provider == "zen" {
		oaiBody["stream"] = true
	} else {
		oaiBody["stream"] = false
	}
	// opencode zen body fingerprint (MITM-confirmed 2026-10-04): zen also
	// validates the SYSTEM PROMPT against opencode's own. Prepend the captured
	// opencode prompt for zen requests — the model sees the real conversation
	// as the continuation that follows it. Other providers are untouched.
	if c.provider == "zen" {
		if zsp := zenSysprompt(); zsp != "" {
			existing := toSlice(oaiBody["messages"])
			combined := append([]interface{}{map[string]interface{}{"role": "system", "content": zsp}}, existing...)
			oaiBody["messages"] = combined
		}
	}
	// Clamp max_tokens to the provider's ceiling. Claude Code's fetch/web
	// summarization helper (compound-mini, e.g. fetch page content) sends a
	// large max_tokens that exceeds groq's hard cap (8192), which would 400 and
	// needlessly walk the fallback chain. Clamping keeps the request on-provider.
	if prov.MaxOutput > 0 {
		if mt, ok := oaiBody["max_tokens"].(float64); ok && int(mt) > prov.MaxOutput {
			oaiBody["max_tokens"] = prov.MaxOutput
		} else if mt, ok := oaiBody["max_tokens"].(int64); ok && int(mt) > prov.MaxOutput {
			oaiBody["max_tokens"] = prov.MaxOutput
		} else if _, ok := oaiBody["max_tokens"]; !ok {
			oaiBody["max_tokens"] = prov.MaxOutput
		}
	}
	bodyBytes, err := json.Marshal(oaiBody)
	if err != nil {
		ReqLog(ctx, "marshal body failed: %v", err)
		return false, false
	}

	// Request context carries no deadline; TTFB bounded by ResponseHeaderTimeout,
	// stream duration bounded per idle-gap below so long valid streams survive.
	req, err := http.NewRequestWithContext(ctx, "POST", prov.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		ReqLog(ctx, "new request failed: %v", err)
		health.penalize(ctx, c, "connect", 0)
		return false, false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	ua := prov.UA
	if ua == "" {
		ua = "opencode/latest/2.0.21/cli"
	}
	req.Header.Set("User-Agent", ua)
	// opencode v2 free-tier gate (MITM-confirmed 2026-10-04): zen validates the
	// (x-opencode-session, x-opencode-project) pair against its backend — both
	// REAL ids from ZEN_SESSION_ID/ZEN_PROJECT_ID for the zen provider alone.
	sid := genSessionID()
	if c.provider == "zen" {
		if zsid := os.Getenv("ZEN_SESSION_ID"); zsid != "" {
			sid = zsid
		}
		if zproj := os.Getenv("ZEN_PROJECT_ID"); zproj != "" {
			req.Header.Set("X-Opencode-Project", zproj)
		}
	}
	req.Header.Set("X-Opencode-Session", sid)
	req.Header.Set("X-Opencode-Session-Id", sid)
	req.Header.Set("X-Opencode-Client", "cli")
	req.Header.Set("X-Session-Id", sid)
	req.Header.Set("X-Session-Affinity", sid)

	t0 := time.Now()
	ReqLog(ctx, "attempt model=%s@%s#k%d (ttfb-timeout %s)", c.model, c.provider, c.keyIdx, to)
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			ReqLog(ctx, "client gone while waiting headers")
			return false, true
		}
		ReqLog(ctx, "connect/header fail: %v", err)
		health.penalize(ctx, c, "connect", 0)
		return false, false
	}
	defer resp.Body.Close()
	headersMs := time.Since(t0)
	health.recordLatency(c, headersMs)

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		ReqLog(ctx, "upstream %d: %s", resp.StatusCode, redact(string(b)))
		kind := "http"
		retryAfter := time.Duration(0)
		if resp.StatusCode == 429 {
			kind = "ratelimit"
			if ra, e := strconv.Atoi(resp.Header.Get("Retry-After")); e == nil && ra > 0 {
				retryAfter = time.Duration(ra) * time.Second
			}
		} else if shouldAbort(resp.StatusCode) {
			kind = "auth"
		}
		health.penalize(ctx, c, kind, retryAfter)
		return false, false
	}

	ReqLog(ctx, "ok %s@%s#k%d headers in %s", c.model, c.provider, c.keyIdx, headersMs)
	if !isStream {
		var j map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
			ReqLog(ctx, "non-stream decode failed: %v", err)
			health.penalize(ctx, c, "stream", 0)
			return false, false
		}
		var uIn, uOut int
		if u, ok := j["usage"].(map[string]interface{}); ok {
			if v, ok := u["prompt_tokens"].(float64); ok {
				uIn = int(v)
			}
			if v, ok := u["completion_tokens"].(float64); ok {
				uOut = int(v)
			}
		}
		ReqLog(ctx, "usage at=%s model=%s uIn=%d uOut=%d ms=%d attempt=1", c.provider, c.model, uIn, uOut, int(time.Since(t0).Milliseconds()))
		return true, false // caller handles the JSON
	}

	// Streaming: parse SSE and translate. Read body in chunks with an idle timeout
	// so a provider that sends headers but then stalls (ghost/half-stream) is cut
	// off and failed over rather than hanging the whole request.
	const idleTimeout = 120 * time.Second
	const dataTimeout = 30 * time.Second
	st := newStreamState()
	var buf bytes.Buffer
	readBuf := make([]byte, 64*1024)
	idleDeadline := time.Now().Add(idleTimeout)

	for {
		// Bound each Read by an idle window so a stalled upstream can't hang forever.
		if rc, ok := resp.Body.(interface{ SetReadDeadline(time.Time) error }); ok {
			_ = rc.SetReadDeadline(time.Now().Add(dataTimeout))
		}
		n, rerr := resp.Body.Read(readBuf)
		if n > 0 {
			idleDeadline = time.Now().Add(idleTimeout)
			buf.Write(readBuf[:n])
			// parse complete SSE lines from buffer
			for {
				idx := bytes.IndexByte(buf.Bytes(), '\n')
				if idx < 0 {
					break
				}
				line := bytes.TrimSpace(buf.Next(idx + 1))
				if !bytes.HasPrefix(line, []byte("data:")) {
					continue
				}
				payload := bytes.TrimSpace(line[5:])
				if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
					continue
				}
				var j map[string]interface{}
				if err := json.Unmarshal(payload, &j); err != nil {
					continue
				}
				st.handleChunk(w, model, j)
			}
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				break
			}
			inet, ok := rerr.(net.Error)
			if ok && inet.Timeout() {
				if time.Now().After(idleDeadline) {
					ReqLog(ctx, "stream idle timeout (%s) from %s@%s", idleTimeout, c.model, c.provider)
					health.penalize(ctx, c, "ghost", 0)
					return false, false
				}
				// transient read timeout; keep waiting for the next chunk
				continue
			}
			ReqLog(ctx, "stream read error: %v", rerr)
			if st.started {
				// partial emission -> deliver what we have (avoid silent drop)
				s := st
				s.closeCur(w)
				for ai := range s.openTools {
					if w != nil {
						w.send("content_block_stop", map[string]interface{}{"type": "content_block_stop", "index": ai})
					}
				}
				if w != nil {
					finalize(w, s)
					w.end()
				}
				return true, false
			}
			health.penalize(ctx, c, "stream", 0)
			return false, false
		}
		if time.Now().After(idleDeadline) {
			ReqLog(ctx, "stream idle timeout (%s) from %s@%s", idleTimeout, c.model, c.provider)
			health.penalize(ctx, c, "ghost", 0)
			return false, false
		}
	}

	if !st.sawAny || !st.sawPayload {
		ReqLog(ctx, "empty stream from %s@%s", c.model, c.provider)
		health.penalize(ctx, c, "ghost", 0)
		return false, false
	}
	if headersMs > time.Duration(cfgSlowHeaderMS)*time.Millisecond {
		health.penalize(ctx, c, "latency", 0)
	} else {
		health.reward(ctx, c)
	}

	st.closeCur(w)
	// close open tools
	for ai := range st.openTools {
		if w != nil {
			w.send("content_block_stop", map[string]interface{}{"type": "content_block_stop", "index": ai})
		}
	}
	var finalContent []interface{}
	for _, x := range st.agg {
		if x["__tool"] == true {
			finalContent = append(finalContent, map[string]interface{}{"type": "tool_use", "id": toString(x["id"]), "name": toString(x["name"]), "input": safeParseArgs(toString(x["args"]))})
		} else {
			finalContent = append(finalContent, x)
		}
	}
	if len(finalContent) == 0 {
		finalContent = append(finalContent, map[string]interface{}{"type": "text", "text": ""})
	}
	// Some upstreams never populate usage at all (observed on some providers'
	// SSE streams). If we report 0/0 tokens on a turn that plainly wasn't
	// free, Claude Code's own context/auto-compact accounting sees zero
	// consumption and never compacts — until the real model's actual context
	// window overflows and hard-fails. Fall back to the estimator so the
	// client always gets a realistic, non-zero signal to compact against.
	if st.uIn == 0 {
		st.uIn = estIn
	}
	if st.uOut == 0 {
		st.uOut = estimateTokens(finalContent)
	}
	if cap != nil {
		cap.Content = finalContent
		cap.StopReason = st.stop
		cap.InTokens = st.uIn
		cap.OutTokens = st.uOut
		cap.Done = true
	}
	if w != nil {
		finalize(w, st)
		w.end()
	}
	ReqLog(ctx, "usage at=%s model=%s uIn=%d uOut=%d ms=%d attempt=1", c.provider, c.model, st.uIn, st.uOut, int(time.Since(t0).Milliseconds()))
	return true, false
}

// redact strips known API keys from logged strings
var redactReplacer *strings.Replacer

func initRedact() {
	parts := []string{}
	for _, p := range cfgProviders {
		for _, k := range p.Keys {
			if k != "" && len(k) >= 8 {
				parts = append(parts, k, "[REDACTED]")
			}
		}
	}
	if len(parts) > 0 {
		redactReplacer = strings.NewReplacer(parts...)
	} else {
		redactReplacer = strings.NewReplacer("-", "-")
	}
}

func redact(s string) string {
	return redactReplacer.Replace(s)
}

// smartWaitCap bounds a single "everything is cooled" wait-and-retry pause.
// Kept short so the loop rechecks health state often rather than committing
// to one long sleep that could outlast a provider's actual recovery.
const smartWaitCap = 5 * time.Second

// translateStream runs the failover chain until success or exhausted.
// Returns (delivered, exhausted).
//
// If every candidate happens to be in cooldown at the same moment (a common
// case under bursty load: a 429 ramp cools a provider for a couple seconds),
// the loop no longer gives up instantly — it waits for the SHORTEST
// remaining cooldown (bounded, and never past the total budget) and retries
// the pass. This is the difference between a request failing outright versus
// a request finishing a couple seconds later than it otherwise would have.
func translateStream(ctx context.Context, w *emitter, model string, oaiBody map[string]interface{}, cands []cand, isStream bool, thinkingDisabled bool, estIn int, cap *respCapture) (delivered, exhausted bool) {
	deadline := time.Now().Add(cfgTotalBudget)
	attempts := 0
	start := time.Now()
	// Per-provider serial attempt budget: a provider that is saturated/network-flaky
	// should not exhaust all its keys serially (5 dahl keys x timeout). After this many
	// consecutive failed attempts on one provider, move to the next provider.
	const maxSameProvider = 2
	for {
		provFails := map[string]int{}
		madeAttempt := false
		minWait := time.Duration(-1)
		for i := 0; i < len(cands); i++ {
			if time.Now().After(deadline) {
				ReqLog(ctx, "total budget (%s) exceeded after %s; %d/%d attempts", cfgTotalBudget, time.Since(start), attempts, len(cands))
				return false, true
			}
			c := cands[i]
			// If this provider already blew its serial-attempt budget, skip its remaining keys.
			if provFails[c.provider] >= maxSameProvider {
				ReqLog(ctx, "skip %s@%s#k%d (provider blew %d-key budget on failures)", c.model, c.provider, c.keyIdx, maxSameProvider)
				continue
			}
			if wait, cooled := health.cooldownRemaining(&c); cooled {
				ReqLog(ctx, "skip %s@%s#k%d (cooled, %s left)", c.model, c.provider, c.keyIdx, wait)
				if minWait < 0 || wait < minWait {
					minWait = wait
				}
				continue
			}
			madeAttempt = true
			attempts++
			ok, _ := attemptOne(ctx, w, &c, model, oaiBody, isStream, thinkingDisabled, estIn, cap)
			if ok {
				return true, false
			}
			provFails[c.provider]++
			select {
			case <-ctx.Done():
				return true, false // client gone; emit may have started
			default:
			}
		}
		if madeAttempt || minWait < 0 {
			// Either we genuinely tried candidates and all failed, or nothing
			// is even in cooldown (config/provider issue) — no point waiting.
			break
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || minWait > remaining {
			break // waiting would exceed (or already exceeds) the total budget
		}
		waitFor := minWait
		if waitFor > smartWaitCap {
			waitFor = smartWaitCap
		}
		ReqLog(ctx, "all %d candidates cooled; smart-wait %s before retry (%s left in budget)", len(cands), waitFor, remaining)
		select {
		case <-time.After(waitFor):
			continue
		case <-ctx.Done():
			return true, false
		}
	}
	ReqLog(ctx, "all candidates exhausted after %d/%d attempts in %s", attempts, len(cands), time.Since(start))
	return false, true
}

// ---------------------------------------------------------------------------
// HTTP handlers
// ---------------------------------------------------------------------------

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	type cool struct {
		Target  string `json:"target"`
		MsLeft  int64  `json:"ms_left"`
		Kind    string `json:"kind"`
		Strikes int    `json:"strikes"`
	}
	var cooling []cool
	health.mu.Lock()
	for k, d := range health.cooldown {
		if time.Now().After(d.until) {
			continue
		}
		cooling = append(cooling, cool{k.provider + "#" + fmt.Sprint(k.keyIdx), int64(time.Until(d.until).Milliseconds()), d.kind, d.strikes})
	}
	for p, d := range health.pbox {
		if time.Now().After(d.until) {
			continue
		}
		cooling = append(cooling, cool{p + " (provider-wide)", int64(time.Until(d.until).Milliseconds()), d.kind, d.strikes})
	}
	health.mu.Unlock()
	cfgChain := strings.Join(cfgFallbackChain, ",")
	if cfgChain == "" {
		cfgChain = "none"
	}
	cache.mu.RLock()
	cacheSize := len(cache.m)
	cacheHits := cache.hits
	cacheMisses := cache.misses
	cacheEvicted := cache.evicted
	cache.mu.RUnlock()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "ok",
		"uptime_s":      int(time.Since(startTime).Seconds()),
		"chain":         cfgChain,
		"providers":     cfgProviderOrder,
		"cooling":       cooling,
		"default_model": cfgFallbackModel,
		"cache": map[string]interface{}{
			"enabled": cfgCacheEnabled,
			"entries": cacheSize,
			"max":     cfgCacheMaxEntries,
			"hits":    cacheHits,
			"misses":  cacheMisses,
			"evicted": cacheEvicted,
			"ttl_ms":  int(cfgCacheTTL / time.Millisecond),
			"persist": cfgCacheDir + "/cache.json",
			"temp0":   cfgCacheTempZeroOnly,
		},
	})
}

var startTime = time.Now()

func modelsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	seen := map[string]bool{}
	var ids []string
	for _, name := range cfgProviderOrder {
		for _, m := range cfgProviders[name].Models {
			if !seen[m] {
				seen[m] = true
				ids = append(ids, m)
			}
		}
	}
	data := make([]map[string]interface{}, 0, len(ids))
	for _, id := range ids {
		data = append(data, map[string]interface{}{"id": id, "object": "model", "created": 0, "owned_by": providerFor(id)})
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"object": "list", "data": data})
}

// Rough token estimate for /v1/messages/count_tokens (Claude Code uses this to
// budget context). Not byte-exact vs Anthropic's tokenizer but close enough for
// context management; fields mirror the Anthropic response shape.
func estimateTokens(v interface{}) int {
	tok := func(s string) int {
		if s == "" {
			return 0
		}
		// ~4 chars/token heuristic
		return (len([]rune(s)) + 3) / 4
	}
	switch t := v.(type) {
	case string:
		return tok(t)
	case map[string]interface{}:
		n := 0
		for k, val := range t {
			switch k {
			case "text", "name", "id", "tool_use_id", "description", "thinking":
				n += tok(toString(val))
			case "content":
				n += estimateTokens(val)
			case "cache_control", "source", "input_schema", "input":
				n += estimateTokens(val)
			}
		}
		return n
	case []interface{}:
		n := 0
		for _, item := range t {
			n += estimateTokens(item)
		}
		return n
	}
	return 0
}

func countTokensHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, cfgMaxBodyBytes+1))
	if err != nil {
		http.Error(w, "read error", 400)
		return
	}
	var in map[string]interface{}
	if err := json.Unmarshal(body, &in); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error": map[string]interface{}{"type": "invalid_request_error", "message": "bad json"}})
		return
	}
	n := estimateTokens(in["system"]) + estimateTokens(in["messages"]) + estimateTokens(in["tools"])
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"input_tokens": n})
}

// finalError chooses a Claude-Code-friendly error type + retry_after when the
// whole chain is exhausted. If upstream cooldowns are warm (we were hitting
// 429/overload), tell the client to back off via overloaded_error instead of a
// hard api_error — native clients retry with backoff on those.
func finalError() (etype string, retryAfter int) {
	etype = "api_error"
	retryAfter = 0
	health.mu.Lock()
	defer health.mu.Unlock()
	// Consider a provider "down" if any key is cooled OR the whole provider is
	// in a provider-wide cooldown (pbox). Without checking pbox, a saturated
	// provider that got ramp-cooled would report api_error + retry_after=0 when
	// the chain finally exhausts, making Claude Code hammer immediately instead
	// of backing off via overloaded_error.
	if len(health.cooldown) > 0 || len(health.pbox) > 0 || len(health.rl) > 0 {
		etype = "overloaded_error"
		retryAfter = 10
	}
	return
}

// ---------------------------------------------------------------------------
// Server-side web search (native-Anthropic emulation)
//
// Claude Code's WebSearch is a server-side tool: it sends a search-turn request
// (tool_choice auto + sole tool "web_search") and expects the *API backend* to
// run the search and hand back server_tool_use + web_search_tool_result blocks.
// Third-party model providers cannot do this, so with a shim the search used to
// come back empty. Here we implement the search ourselves (Tavily keyless) and
// emit Anthropic-native blocks so Claude Code behaves as if talking to Anthropic.
// ---------------------------------------------------------------------------

type braveResult struct {
	Title       string
	URL         string
	Description string
	PageAge     string
}

// webProviderSearch runs a web search and returns normalized results.
// Backend is Tavily's keyless mode (no account, no API key, no card): a single
// X-Tavily-Access-Mode: keyless header grants free, rate-limited search. This
// is what powers Claude Code's forced server-side web_search turn natively.
func webProviderSearch(ctx context.Context, query string, count int) ([]braveResult, bool) {
	if count <= 0 {
		count = 8
	}
	body, _ := json.Marshal(map[string]interface{}{
		"query":               query,
		"max_results":         count,
		"include_answer":      false,
		"search_depth":        "basic",
		"include_raw_content": false,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", cfgBraveURL, bytes.NewReader(body))
	if err != nil {
		return nil, false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// Keyless free tier: no Authorization needed. If a real key is set later it
	// takes precedence transparently (Tavily accepts both).
	if cfgBraveKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfgBraveKey)
	} else {
		req.Header.Set("X-Tavily-Access-Mode", "keyless")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		ReqLog(ctx, "web search HTTP %d: %s", resp.StatusCode, redact(string(body)))
		return nil, false
	}
	dec := json.NewDecoder(io.LimitReader(resp.Body, 4<<20))
	// Tavily returns results directly; each carries title/url/content + a
	// published_date we can surface as the age.
	var payload struct {
		Results []struct {
			Title     string `json:"title"`
			URL       string `json:"url"`
			Content   string `json:"content"`
			Published string `json:"published_date"`
		} `json:"results"`
	}
	if err := dec.Decode(&payload); err != nil {
		ReqLog(ctx, "web search decode failed: %v", err)
		return nil, false
	}
	out := make([]braveResult, 0, len(payload.Results))
	for _, res := range payload.Results {
		out = append(out, braveResult{Title: res.Title, URL: res.URL, Description: res.Content, PageAge: res.Published})
	}
	return out, true
}

// lastUserText returns the concatenated text of the final user-role message,
// ignoring tool_result blocks. Claude Code sends the (pre-extracted) search
// query as its last user text during a search turn. Falls back to the last
// assistant tool_use.input.query when no user text is present.
func lastUserText(body map[string]interface{}) string {
	var sb strings.Builder
	for _, m := range toSlice(body["messages"]) {
		mm, _ := m.(map[string]interface{})
		switch toString(mm["role"]) {
		case "user":
			sb.Reset()
			switch c := mm["content"].(type) {
			case string:
				sb.WriteString(c)
			case []interface{}:
				for _, b := range c {
					bm, _ := b.(map[string]interface{})
					switch bm["type"] {
					case "text":
						if sb.Len() > 0 {
							sb.WriteString("\n")
						}
						sb.WriteString(toString(bm["text"]))
					case "tool_result":
						sb.Reset() // trailing tool_results belong to a prior turn, ignore
					}
				}
			}
		case "assistant":
			// If the last relevant content is a web_search tool_use, keep its query.
			if c, ok := mm["content"].([]interface{}); ok {
				for _, b := range c {
					bm, _ := b.(map[string]interface{})
					if bm["type"] == "tool_use" && (toString(bm["name"]) == "web_search" || toString(bm["name"]) == "WebSearch") {
						if in, ok := bm["input"].(map[string]interface{}); ok {
							if q := strings.TrimSpace(toString(in["query"])); q != "" {
								sb.Reset()
								sb.WriteString(q)
							}
						}
					}
				}
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

// isWebSearchRequest reports whether this is Claude Code's server-side web
// search turn. Two distinct signatures are recognized:
//
//  1. tool_choice pinned to web_search: {"type":"tool","name":"web_search"}
//     (older / some builds).
//  2. The tool list is EXACTLY ["web_search"] (lowercase, sole tool) with
//     tool_choice auto/none — observed in current Claude Code: after the model
//     emits a WebSearch tool_use, the client fires a secondary request whose
//     ONLY tool is web_search and expects the backend to run the search.
//
// Ordinary conversations list web_search amongst MANY tools and let the model
// decide — those must NOT be intercepted here and go through the normal chain.
func isWebSearchRequest(body map[string]interface{}) bool {
	if tc, ok := body["tool_choice"].(map[string]interface{}); ok {
		if tc["type"] == "tool" && toString(tc["name"]) == "web_search" {
			return true
		}
	}
	// Signature 2: sole tool is "web_search".
	if arr := toSlice(body["tools"]); len(arr) == 1 {
		if tm, ok := arr[0].(map[string]interface{}); ok {
			if toString(tm["name"]) == "web_search" {
				return true
			}
		}
	}
	return false
}

// emitWebSearchResult writes an Anthropic-native response that runs the search
// and hands back server_tool_use + web_search_tool_result blocks, so Claude Code
// sees real results instead of an empty envelope. Handles both streaming and
// non-streaming frontends. Returns true if it handled the request.
func emitWebSearchResult(w http.ResponseWriter, r *http.Request, ctx context.Context, body map[string]interface{}, isStream bool) bool {
	query := lastUserText(body)
	ReqLog(ctx, "web_search request intercept (stream=%v) query=%q", isStream, truncate(query, 120))
	if query == "" {
		// Nothing to search: fail forward to the normal chain, which at least
		// produces a model answer (better than empty).
		return false
	}

	id := "srvtoolu_" + hex.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))[:8])
	var results []braveResult
	var ok bool
	bctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	results, ok = webProviderSearch(bctx, query, 8)
	toolResult := map[string]interface{}{
		"type":        "web_search_tool_result",
		"tool_use_id": id,
	}
	if !ok || len(results) == 0 {
		// Empty (or failed) search -> empty content list is valid (vs an error).
		toolResult["content"] = []interface{}{}
	} else {
		var rs []interface{}
		for _, r := range results {
			hit := map[string]interface{}{
				"type":  "web_search_result",
				"title": r.Title,
				"url":   r.URL,
			}
			// Native Anthropic uses encrypted_content; a plain snippet content
			// field is accepted by Claude Code's citation layer and other
			// providers (e.g. MiniMax) ship this same shape.
			if r.Description != "" {
				hit["content"] = r.Description
			}
			if r.PageAge != "" {
				hit["page_age"] = r.PageAge
			}
			rs = append(rs, hit)
		}
		toolResult["content"] = rs
	}

	leadIn := "I'll search the web for that."
	blocks := []interface{}{
		map[string]interface{}{"type": "text", "text": leadIn},
		map[string]interface{}{"type": "server_tool_use", "id": id, "name": "web_search", "input": map[string]interface{}{"query": query}},
	}

	// If we got results, also emit the result block so Claude Code has the data.
	if ok {
		blocks = append(blocks, toolResult)
	}

	stop := "tool_use"
	if ok {
		stop = "end_turn"
	}
	uIn := len(query)/4 + 100
	uOut := 50 + len(results)*40

	if isStream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		em := &emitter{w: w, flusher: flusher}
		em.send("message_start", map[string]interface{}{
			"type": "message_start",
			"message": map[string]interface{}{
				"id":   "msg_" + hex.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))[:8]),
				"type": "message", "role": "assistant", "model": toString(body["model"]),
				"content": []interface{}{}, "stop_reason": nil, "stop_sequence": nil,
				"usage": map[string]interface{}{"input_tokens": uIn, "output_tokens": uOut, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0},
			},
		})
		for i, b := range blocks {
			em.send("content_block_start", map[string]interface{}{"type": "content_block_start", "index": i, "content_block": b})
			em.send("content_block_stop", map[string]interface{}{"type": "content_block_stop", "index": i})
		}
		em.send("message_delta", map[string]interface{}{"type": "message_delta", "delta": map[string]interface{}{"stop_reason": stop, "stop_sequence": nil}, "usage": map[string]interface{}{"output_tokens": uOut}})
		em.send("message_stop", map[string]interface{}{"type": "message_stop"})
		em.end()
		return true
	}

	out := map[string]interface{}{
		"id":            "msg_" + hex.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))[:8]),
		"type":          "message",
		"role":          "assistant",
		"model":         toString(body["model"]),
		"content":       blocks,
		"stop_reason":   stop,
		"stop_sequence": nil,
		"usage":         map[string]interface{}{"input_tokens": uIn, "output_tokens": uOut, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(out)
	return true
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func messagesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rid := newRID()
	ctx := withRID(r.Context(), rid)

	body, err := io.ReadAll(io.LimitReader(r.Body, cfgMaxBodyBytes+1))
	if err != nil {
		ReqLog(ctx, "read body failed: %v", err)
		http.Error(w, "read error", 500)
		return
	}
	if int64(len(body)) > cfgMaxBodyBytes {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(413)
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error": map[string]interface{}{"type": "request_too_large", "message": "request body exceeds limit"}})
		return
	}
	var in map[string]interface{}
	if err := json.Unmarshal(body, &in); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error": map[string]interface{}{"type": "invalid_request_error", "message": "bad json"}})
		return
	}

	model := toString(in["model"])
	isStream := false
	if v, ok := in["stream"].(bool); ok {
		isStream = v
	}
	oaiBody, thinkingDisabled := convertBody(in)
	// Precomputed once per request: the fallback input-token estimate used
	// whenever an upstream provider never populates usage at all (see
	// toAnthropicNonStream / the streaming finalize path) so Claude Code's own
	// context/auto-compact accounting is never silently starved to zero.
	estIn := estimateTokens(in["system"]) + estimateTokens(in["messages"]) + estimateTokens(in["tools"])
	cands := candidates(model)

	// Reject a semantically-empty request up front. A body with no messages and
	// no system prompt has nothing to answer; forwarding it would only make every
	// provider in the chain reject it (400/422 "messages is required"), burn their
	// cooldowns and generate fallback noise. This is what Claude Code's small-fast
	// model (openai/gpt-oss-20b) fires for some background ops.
	if len(toSlice(oaiBody["messages"])) == 0 && (oaiBody["system"] == nil) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error": map[string]interface{}{"type": "invalid_request_error", "message": "request has no messages or system prompt to answer"}})
		return
	}

	// Server-side web search interception: Claude Code's WebSearch is a server
	// tool. When it sends a forced web_search request, run the search here
	// and return native server_tool_use + web_search_tool_result blocks
	// instead of letting a model provider emit an unserivable tool_use.
	if isWebSearchRequest(in) {
		if emitWebSearchResult(w, r, ctx, in, isStream) {
			ReqLog(ctx, "web_search handled natively (query via tavily)")
			return
		}
		// Not configured / empty query: fall through to the model chain.
	}

	// Resolve the model that will actually answer (first viable candidate) and
	// check the exact-match response cache. On a live, unexpired hit we replay
	// without touching any provider. The key includes the full forwarded body,
	// so only a byte-identical re-send matches — never a "similar" prompt.
	respModel := model
	if len(cands) > 0 {
		respModel = cands[0].model
	}
	cacheKey := cache.key(respModel, oaiBody)
	cacheOK := cacheableBody(oaiBody)
	if cacheOK {
		if e, ok := cache.get(cacheKey); ok {
			ReqLog(ctx, "CACHE-HIT model=%s stream:%v (in=%d out=%d)", respModel, isStream, e.InTokens, e.OutTokens)
			if isStream {
				flusher, _ := w.(http.Flusher)
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				w.Header().Set("anthropic-version", "2023-06-01")
				w.WriteHeader(200)
				em := &emitter{w: w, flusher: flusher}
				emitCachedStream(em, respModel, e)
				em.end()
				return
			}
			an := map[string]interface{}{
				"id": "msg_" + strconv.FormatInt(time.Now().UnixNano(), 36), "type": "message", "role": "assistant", "model": respModel,
				"content": e.Content, "stop_reason": e.StopReason, "stop_sequence": nil,
				"usage": map[string]interface{}{"input_tokens": e.InTokens, "output_tokens": e.OutTokens, "cache_creation_input_tokens": 0, "cache_read_input_tokens": e.InTokens},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(an)
			return
		}
	}

	// build chain string for logging
	var cs []string
	for _, c := range cands {
		cs = append(cs, c.model+"@"+c.provider)
	}
	ReqLog(ctx, "-> %s => %s tools:%d msgs:%d stream:%v", model, strings.Join(cs, "|"), len(toSlice(oaiBody["tools"])), len(toSlice(oaiBody["messages"])), isStream)

	if isStream {
		flusher, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("anthropic-version", "2023-06-01")
		w.WriteHeader(200)
		em := &emitter{w: w, flusher: flusher}

		ctx, cancel := context.WithCancel(withRID(r.Context(), rid))
		defer cancel()
		var cap respCapture
		delivered, exhausted := translateStream(ctx, em, model, oaiBody, cands, true, thinkingDisabled, estIn, &cap)
		if delivered && cap.Done && cacheOK {
			cache.put(cacheKey, cacheEntry{Content: cap.Content, StopReason: cap.StopReason, InTokens: cap.InTokens, OutTokens: cap.OutTokens, StoredAt: time.Now()})
		}
		// If the whole chain failed before any block was delivered, surface a
		// proper Anthropic error event so the client doesn't hang on a silent
		// empty 200 (headers were already flushed, so we can't change status).
		if !delivered && exhausted {
			etype, ra := finalError()
			errObj := map[string]interface{}{"type": etype, "message": "All fallback models failed"}
			if ra > 0 {
				errObj["retry_after"] = ra
			}
			em.send("error", map[string]interface{}{"type": "error", "error": errObj})
		}
		return
	}

	// Non-streaming: drive the chain, but we need to return consolidated JSON.
	// We attempt each candidate without an active stream writer, and on success
	// convert to Anthropic non-stream format. Mirrors translateStream's smart-wait:
	// if every candidate is merely in a brief cooldown, wait for the shortest one
	// (bounded by the remaining budget) instead of failing the request outright.
	const maxSameProvider = 2
	deadline := time.Now().Add(cfgTotalBudget)
nonStreamLoop:
	for {
		provFails := map[string]int{}
		madeAttempt := false
		minWait := time.Duration(-1)
		for i := 0; i < len(cands); i++ {
			if time.Now().After(deadline) {
				break nonStreamLoop
			}
			c := cands[i]
			// Per-provider serial attempt budget (same cap as translateStream): a
			// saturated provider should not exhaust all its keys serially.
			if provFails[c.provider] >= maxSameProvider {
				continue
			}
			if wait, cooled := health.cooldownRemaining(&c); cooled {
				if minWait < 0 || wait < minWait {
					minWait = wait
				}
				continue
			}
			prov := cfgProviders[c.provider]
			if prov == nil {
				continue
			}
			madeAttempt = true
			an, ok, clientGone := nonStreamAttempt(ctx, r.Context(), &c, model, oaiBody, thinkingDisabled, estIn)
			if clientGone {
				return
			}
			if !ok {
				provFails[c.provider]++
				continue
			}
			if cacheOK {
				content, _ := an["content"].([]interface{})
				if len(content) > 0 {
					stop := "end_turn"
					if s, ok := an["stop_reason"].(string); ok && s != "" {
						stop = s
					}
					uIn, uOut := 0, 0
					if u, ok := an["usage"].(map[string]interface{}); ok {
						uIn = asInt(u["input_tokens"])
						uOut = asInt(u["output_tokens"])
					}
					cache.put(cacheKey, cacheEntry{Content: content, StopReason: stop, InTokens: uIn, OutTokens: uOut, StoredAt: time.Now()})
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(an)
			return
		}
		if madeAttempt || minWait < 0 {
			break
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || minWait > remaining {
			break
		}
		waitFor := minWait
		if waitFor > smartWaitCap {
			waitFor = smartWaitCap
		}
		ReqLog(ctx, "nonStream: all %d candidates cooled; smart-wait %s before retry (%s left in budget)", len(cands), waitFor, remaining)
		select {
		case <-time.After(waitFor):
			continue
		case <-r.Context().Done():
			return
		}
	}
	ReqLog(ctx, "nonStream: all candidates exhausted")
	etype, ra := finalError()
	errObj := map[string]interface{}{"type": etype, "message": "All fallback models failed"}
	if ra > 0 {
		errObj["retry_after"] = ra
		w.Header().Set("Retry-After", strconv.Itoa(ra))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(502)
	json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error": errObj})
}

// nonStreamAttempt performs a single non-streaming candidate attempt and
// returns the translated Anthropic-shaped response on success. Unlike the
// original inline loop, the per-attempt timeout context is scoped (and
// canceled) to exactly this call via defer — not deferred to the end of the
// whole request handler — so a long candidate chain doesn't accumulate live
// timers/contexts for every already-finished attempt.
func nonStreamAttempt(ctx context.Context, parentCtx context.Context, c *cand, model string, oaiBody map[string]interface{}, thinkingDisabled bool, estIn int) (an map[string]interface{}, ok bool, clientGone bool) {
	prov := cfgProviders[c.provider]
	if prov == nil {
		return nil, false, false
	}
	// zen (relay path): the free-tier gate forces opencode's slow title-generator
	// conversation upstream — TTFB observed at 15-40s. The adaptive timeout tracks
	// the (much faster) live baselines and would abort zen falsely; give zen the
	// full total budget as its TTFB window instead.
	to := cfgTotalBudget
	if c.provider != "zen" {
		to = health.adaptTimeout(c, cfgTotalBudget)
	}
	client := newTTFBClient(to)
	key := ""
	if c.keyIdx < len(prov.Keys) {
		key = prov.Keys[c.keyIdx]
	}
	ob := copyMap(oaiBody)
	ob["model"] = c.model
	// zen requires stream:true (free-tier gate — MITM-confirmed 2026-10-04);
	// the SSE reply is reassembled into a full JSON response below.
	if c.provider == "zen" {
		ob["stream"] = true
		// body fingerprint prepend (same as attemptOne): opencode's system prompt
		if zsp := zenSysprompt(); zsp != "" {
			existing := toSlice(ob["messages"])
			if len(existing) == 0 || !strings.EqualFold(toString(existing[0].(map[string]interface{})["content"]), zsp) {
				combined := append([]interface{}{map[string]interface{}{"role": "system", "content": zsp}}, existing...)
				ob["messages"] = combined
			}
		}
	} else {
		ob["stream"] = false
	}
	// Deduce per-provider thinking shape dynamically (see attemptOne), unless
	// the client explicitly asked for thinking:disabled.
	applyThinking(ob, prov, thinkingDisabled)
	// Clamp max_tokens to the provider's ceiling (non-stream path). Same as
	// attemptOne: the fetch/web summarization helper (compound-mini) sends a
	// large max_tokens that exceeds groq's 8192 cap and would 400+waste the
	// fallback chain otherwise.
	if prov.MaxOutput > 0 {
		if mt, ok := ob["max_tokens"].(float64); ok && int(mt) > prov.MaxOutput {
			ob["max_tokens"] = prov.MaxOutput
		} else if mt, ok := ob["max_tokens"].(int64); ok && int(mt) > prov.MaxOutput {
			ob["max_tokens"] = prov.MaxOutput
		} else if _, ok := ob["max_tokens"]; !ok {
			ob["max_tokens"] = prov.MaxOutput
		}
	}
	bb, _ := json.Marshal(ob)
	// Bound the WHOLE attempt (headers + body) by the adapted per-attempt
	// budget. newTTFBClient only caps time-to-first-byte; without this an
	// upstream that returns headers then stalls mid-body hangs the request
	// forever (observed: gpt-oss-20b non-stream to groq stuck 20+ min with
	// no error line and no resolution), leaking a connection+goroutine and
	// silently blocking the client.
	actx, acancel := context.WithTimeout(parentCtx, to)
	defer acancel()
	req, _ := http.NewRequestWithContext(actx, "POST", prov.URL, bytes.NewReader(bb))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	ua := prov.UA
	if ua == "" {
		ua = "opencode/latest/2.0.21/cli"
	}
	req.Header.Set("User-Agent", ua)
	// opencode v2 free-tier gate (confirmed by MITM-intercepting real opencode
	// traffic, 2026-10-04): zen validates the pair (x-opencode-session id,
	// x-opencode-project id) against its backend — both must be REAL ids created
	// by opencode itself; ids are stored in ZEN_SESSION_ID/ZEN_PROJECT_ID (conf.d).
	// The zen path also REQUIRES stream:true in the body — a stream:false body
	// still gets FreeTierError even with valid ids (nonStream requests are
	// converted to stream upstream; the SSE reply is reassembled below).
	sid := genSessionID()
	if c.provider == "zen" {
		if zsid := os.Getenv("ZEN_SESSION_ID"); zsid != "" {
			sid = zsid
		}
		if zproj := os.Getenv("ZEN_PROJECT_ID"); zproj != "" {
			req.Header.Set("X-Opencode-Project", zproj)
		}
	}
	req.Header.Set("X-Opencode-Session", sid)
	req.Header.Set("X-Opencode-Session-Id", sid)
	req.Header.Set("X-Opencode-Client", "cli")
	req.Header.Set("X-Session-Id", sid)
	req.Header.Set("X-Session-Affinity", sid)
	t0 := time.Now()
	ReqLog(ctx, "nonStream attempt model=%s@%s#k%d", c.model, c.provider, c.keyIdx)
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(parentCtx.Err(), context.Canceled) {
			ReqLog(ctx, "client gone during nonStream attempt")
			return nil, false, true
		}
		ReqLog(ctx, "nonStream connect fail: %v", err)
		health.penalize(ctx, c, "connect", 0)
		return nil, false, false
	}
	headersMs := time.Since(t0)
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		ReqLog(ctx, "nonStream upstream %d: %s", resp.StatusCode, redact(string(b)))
		kind := "http"
		ra := time.Duration(0)
		if resp.StatusCode == 429 {
			kind = "ratelimit"
			if v, e := strconv.Atoi(resp.Header.Get("Retry-After")); e == nil && v > 0 {
				ra = time.Duration(v) * time.Second
			}
		} else if shouldAbort(resp.StatusCode) {
			kind = "auth"
		}
		health.penalize(ctx, c, kind, ra)
		resp.Body.Close()
		return nil, false, false
	}
	health.recordLatency(c, headersMs)
	var j map[string]interface{}
	// zen path (stream-forced): the upstream replies with SSE ("data: {...}" lines)
	// because the free-tier gate requires stream:true. Reassemble the SSE chunks
	// into a chat.completion JSON shape so the normal toAnthropicNonStream
	// conversion below handles the reply like any non-streaming response.
	if c.provider == "zen" {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		resp.Body.Close()
		txt := string(raw)
		if strings.HasPrefix(strings.TrimSpace(txt), "data:") {
			var content strings.Builder
			var usage interface{}
			var modelID, finish string
			for _, line := range strings.Split(txt, "\n") {
				line = strings.TrimSpace(line)
				if !strings.HasPrefix(line, "data: ") {
					continue
				}
				payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
				if payload == "" || payload == "[DONE]" {
					continue
				}
				var chunk map[string]interface{}
				if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
					continue
				}
				if m, ok := chunk["model"].(string); ok && m != "" {
					modelID = m
				}
				if u, ok := chunk["usage"]; ok && u != nil {
					usage = u
				}
				if chs, ok := chunk["choices"].([]interface{}); ok {
					for _, chc := range chs {
						cm, _ := chc.(map[string]interface{})
						if cm == nil {
							continue
						}
						if f, ok := cm["finish_reason"].(string); ok && f != "" {
							finish = f
						}
						if delta, ok := cm["delta"].(map[string]interface{}); ok {
							if dc, ok := delta["content"].(string); ok {
								content.WriteString(dc)
							}
						}
					}
				}
			}
			msg := map[string]interface{}{"role": "assistant"}
			if content.Len() > 0 {
				msg["content"] = content.String()
			}
			j = map[string]interface{}{
				"id": modelID, "object": "chat.completion", "created": time.Now().Unix(),
				"model": modelID, "choices": []interface{}{map[string]interface{}{
					"index": float64(0), "message": msg, "finish_reason": finish,
				}},
			}
			if usage != nil {
				j["usage"] = usage
			}
			ReqLog(ctx, "zen SSE reassembled: %d chars", content.Len())
		} else {
			var j2 map[string]interface{}
			if err := json.Unmarshal([]byte(txt), &j2); err != nil {
				ReqLog(ctx, "zen parse failed: %v", err)
				return nil, false, false
			}
			j = j2
		}
	} else {
		if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				ReqLog(ctx, "nonStream body timeout after %s from %s@%s", to, c.model, c.provider)
			} else {
				ReqLog(ctx, "nonStream decode failed: %v", err)
			}
			resp.Body.Close()
			return nil, false, false
		}
		resp.Body.Close()
	}
	ch0, _ := j["choices"].([]interface{})
	if len(ch0) == 0 {
		ReqLog(ctx, "nonStream ghost payload from %s@%s", c.model, c.provider)
		health.penalize(ctx, c, "ghost", 0)
		return nil, false, false
	}
	an = toAnthropicNonStream(model, j, estIn)
	// Emit a completion line on success so every request has a logged outcome
	// (matches the streaming/attemptOne paths). Without this a successful
	// non-stream response looks like a silent hang in the log.
	uIn, uOut := 0, 0
	if u, ok := an["usage"].(map[string]interface{}); ok {
		uIn = asInt(u["input_tokens"])
		uOut = asInt(u["output_tokens"])
	}
	ReqLog(ctx, "usage at=%s model=%s uIn=%d uOut=%d ms=%d attempt=%d", c.provider, c.model, uIn, uOut, int(time.Since(t0).Milliseconds()), 1)
	return an, true, false
}

func copyMap(m map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func toAnthropicNonStream(model string, j map[string]interface{}, estIn int) map[string]interface{} {
	ch0, _ := j["choices"].([]interface{})
	msg := map[string]interface{}{}
	if len(ch0) > 0 {
		if c, ok := ch0[0].(map[string]interface{}); ok {
			msg, _ = c["message"].(map[string]interface{})
			if msg == nil {
				msg = map[string]interface{}{}
			}
			// finish_reason
			_ = c
		}
	}
	var content []interface{}
	reason, _ := msg["reasoning_content"].(string)
	if reason == "" {
		reason, _ = msg["reasoning"].(string)
	}
	if reason != "" {
		content = append(content, map[string]interface{}{"type": "thinking", "thinking": reason})
	}
	if mc, ok := msg["content"].(string); ok && mc != "" {
		content = append(content, map[string]interface{}{"type": "text", "text": mc})
	} else if arr, ok := msg["content"].([]interface{}); ok {
		for _, p := range arr {
			if pm, ok := p.(map[string]interface{}); ok && pm["type"] == "text" {
				if t, ok := pm["text"].(string); ok && t != "" {
					content = append(content, map[string]interface{}{"type": "text", "text": t})
				} else if tmap, ok := pm["text"].(map[string]interface{}); ok {
					if t, ok := tmap["value"].(string); ok {
						content = append(content, map[string]interface{}{"type": "text", "text": t})
					}
				}
			}
		}
	}
	ti := 0
	for _, tc := range toSlice(msg["tool_calls"]) {
		tcm, _ := tc.(map[string]interface{})
		fn, _ := tcm["function"].(map[string]interface{})
		name := toString(fn["name"])
		if name == "" {
			continue
		}
		id := toString(tcm["id"])
		if id == "" {
			id = "toolu_" + strconv.Itoa(ti)
		}
		content = append(content, map[string]interface{}{"type": "tool_use", "id": id, "name": name, "input": safeParseArgs(toString(fn["arguments"]))})
		ti++
	}
	if len(content) == 0 {
		content = append(content, map[string]interface{}{"type": "text", "text": ""})
	}
	stop := "end_turn"
	if len(ch0) > 0 {
		if c, ok := ch0[0].(map[string]interface{}); ok {
			if fr, ok := c["finish_reason"].(string); ok {
				if fr == "tool_calls" {
					stop = "tool_use"
				} else if fr == "length" {
					stop = "max_tokens"
				}
			}
		}
	}
	var uIn, uOut int
	if u, ok := j["usage"].(map[string]interface{}); ok {
		if v, ok := u["prompt_tokens"].(float64); ok {
			uIn = int(v)
		}
		if v, ok := u["completion_tokens"].(float64); ok {
			uOut = int(v)
		}
	}
	// Some upstreams never populate usage. Reporting 0/0 tokens on a turn that
	// plainly wasn't free starves Claude Code's own context/auto-compact
	// accounting of any signal to compact against — it just keeps growing
	// until the real model's actual context window hard-fails. Fall back to
	// the estimator so the client always gets a realistic, non-zero number.
	if uIn == 0 {
		uIn = estIn
	}
	if uOut == 0 {
		uOut = estimateTokens(content)
	}
	return map[string]interface{}{
		"id": "msg_" + strconv.FormatInt(time.Now().UnixNano(), 36), "type": "message", "role": "assistant", "model": model,
		"content": content, "stop_reason": stop, "stop_sequence": nil,
		"usage": map[string]interface{}{"input_tokens": uIn, "output_tokens": uOut, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0},
	}
}

// ---------------------------------------------------------------------------
// Response cache (exact-match, safe, in-memory + disk snapshot)
// ---------------------------------------------------------------------------
//
// Third-party upstreams (dahl/zen/... via the OpenAI-compat path) do not offer
// Anthropic prompt-cache billing, so we cache complete assistant responses and
// replay them on the genuinely common pattern: the client re-sending a byte-
// identical request (retries, resume/continue replays, repeated background ops
// from the small-fast model). Cache key = hash of the canonical forwarded
// request (messages chain + system + tools + params + resolved model). Exact
// match only — no semantic matching, so a cached answer is only ever returned
// for a request identical to one already answered. TTL bounds staleness, and by
// default we only store deterministic (temperature 0/absent) responses.

type cacheEntry struct {
	Content    []interface{} `json:"content"`
	StopReason string        `json:"stop_reason"`
	InTokens   int           `json:"in_tokens"`
	OutTokens  int           `json:"out_tokens"`
	StoredAt   time.Time     `json:"stored_at"`
}

type responseCache struct {
	mu      sync.RWMutex
	m       map[string]cacheEntry
	max     int
	ttl     time.Duration
	hits    int64
	misses  int64
	evicted int64
}

var cache = &responseCache{m: map[string]cacheEntry{}}

func (rc *responseCache) initLockedFromDisk() {
	if !cfgCacheEnabled {
		return
	}
	data, err := os.ReadFile(cfgCacheDir + "/cache.json")
	if err != nil {
		return
	}
	var disk map[string]cacheEntry
	if err := json.Unmarshal(data, &disk); err != nil {
		log.Print("cache load failed: ", err)
		return
	}
	now := time.Now()
	loaded := 0
	for k, e := range disk {
		if !e.StoredAt.IsZero() && now.Sub(e.StoredAt) > rc.ttl {
			continue // expired already
		}
		if len(rc.m) >= rc.max {
			break
		}
		rc.m[k] = e
		loaded++
	}
	if loaded > 0 {
		log.Printf("cache loaded %d entries from %s", loaded, cfgCacheDir+"/cache.json")
	}
}

func (rc *responseCache) persist() {
	if !cfgCacheEnabled {
		return
	}
	rc.mu.RLock()
	now := time.Now()
	disk := map[string]cacheEntry{}
	for k, e := range rc.m {
		if now.Sub(e.StoredAt) > rc.ttl {
			continue
		}
		disk[k] = e
	}
	n := len(disk)
	rc.mu.RUnlock()
	if err := os.MkdirAll(cfgCacheDir, 0o700); err != nil {
		return
	}
	b, err := json.Marshal(disk)
	if err != nil {
		return
	}
	tmp := cfgCacheDir + "/cache.json.tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return
	}
	os.Rename(tmp, cfgCacheDir+"/cache.json")
	log.Printf("cache persisted %d entries to %s", n, cfgCacheDir+"/cache.json")
}

// deterministic canonical JSON for keying: sorted map keys so re-serialization
// is stable regardless of Go map iteration order.
func canonicalJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (rc *responseCache) key(model string, oaiBody map[string]interface{}) string {
	kb := map[string]interface{}{
		"model":      model,
		"messages":   oaiBody["messages"],
		"system":     oaiBody["system"],
		"tools":      oaiBody["tools"],
		"max_tokens": oaiBody["max_tokens"],
		"temp":       oaiBody["temperature"],
		"top_p":      oaiBody["top_p"],
		"stop":       oaiBody["stop"],
	}
	raw := canonicalJSON(kb)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// cacheableBody reports whether we may store a response for this request.
// Default: only deterministic bodies (temperature 0 or absent) unless
// CACHE_TEMP_ZERO_ONLY is disabled.
func cacheableBody(oaiBody map[string]interface{}) bool {
	if !cfgCacheEnabled {
		return false
	}
	if cfgCacheTempZeroOnly {
		if t, ok := oaiBody["temperature"].(float64); ok && t > 0 {
			return false
		}
	}
	if len(toSlice(oaiBody["messages"])) == 0 {
		return false
	}
	return true
}

func (rc *responseCache) get(key string) (cacheEntry, bool) {
	rc.mu.RLock()
	e, ok := rc.m[key]
	rc.mu.RUnlock()
	if !ok {
		rc.mu.Lock()
		rc.misses++
		rc.mu.Unlock()
		return cacheEntry{}, false
	}
	if time.Since(e.StoredAt) > rc.ttl {
		rc.mu.Lock()
		delete(rc.m, key)
		rc.evicted++
		rc.mu.Unlock()
		return cacheEntry{}, false
	}
	rc.mu.Lock()
	rc.hits++
	rc.mu.Unlock()
	return e, true
}

func (rc *responseCache) put(key string, e cacheEntry) {
	if !cfgCacheEnabled || len(e.Content) == 0 {
		return
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if _, exists := rc.m[key]; !exists && len(rc.m) >= rc.max {
		// simple LRU eviction: drop the oldest-stored entry
		oldest := ""
		var oldT time.Time
		for k, ce := range rc.m {
			if oldest == "" || ce.StoredAt.Before(oldT) {
				oldest = k
				oldT = ce.StoredAt
			}
		}
		if oldest != "" {
			delete(rc.m, oldest)
			rc.evicted++
		}
	}
	rc.m[key] = e
}

// respCapture carries the final assistant content out of the stream path so it
// can be cached. It is optional (nil = no capture).
type respCapture struct {
	Content    []interface{}
	StopReason string
	InTokens   int
	OutTokens  int
	Done       bool
}

// emitCachedStream replays a cached entry as a full Anthropic SSE stream.
func emitCachedStream(w *emitter, model string, e cacheEntry) {
	em := w
	em.send("message_start", map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id": "msg_" + strconv.FormatInt(time.Now().UnixNano(), 36), "type": "message", "role": "assistant",
			"model": model, "content": []interface{}{}, "stop_reason": nil, "stop_sequence": nil,
			"usage": map[string]interface{}{"input_tokens": e.InTokens, "output_tokens": e.OutTokens, "cache_creation_input_tokens": 0, "cache_read_input_tokens": e.InTokens},
		},
	})
	for i, block := range e.Content {
		bm, _ := block.(map[string]interface{})
		typ := toString(bm["type"])
		em.send("content_block_start", map[string]interface{}{"type": "content_block_start", "index": i, "content_block": bm})
		switch typ {
		case "thinking":
			txt := toString(bm["thinking"])
			em.send("content_block_delta", map[string]interface{}{"type": "content_block_delta", "index": i, "delta": map[string]interface{}{"type": "thinking_delta", "thinking": txt}})
			em.send("signature_delta", map[string]interface{}{"type": "content_block_delta", "index": i, "delta": map[string]interface{}{"type": "signature_delta", "signature": "sig-shim-cache-" + strconv.Itoa(i)}})
		case "tool_use":
			em.send("content_block_delta", map[string]interface{}{"type": "content_block_delta", "index": i, "delta": map[string]interface{}{"type": "input_json_delta", "partial_json": canonicalJSON(bm["input"])}})
		default:
			txt := toString(bm["text"])
			em.send("content_block_delta", map[string]interface{}{"type": "content_block_delta", "index": i, "delta": map[string]interface{}{"type": "text_delta", "text": txt}})
		}
		em.send("content_block_stop", map[string]interface{}{"type": "content_block_stop", "index": i})
	}
	em.send("message_delta", map[string]interface{}{
		"type":  "message_delta",
		"delta": map[string]interface{}{"stop_reason": e.StopReason, "stop_sequence": nil},
		"usage": map[string]interface{}{"input_tokens": e.InTokens, "output_tokens": e.OutTokens, "cache_creation_input_tokens": 0, "cache_read_input_tokens": e.InTokens},
	})
	em.send("message_stop", map[string]interface{}{"type": "message_stop"})
}

func main() {
	loadConfig()
	initRedact()
	cache.mu.Lock()
	cache.max = cfgCacheMaxEntries
	cache.ttl = cfgCacheTTL
	cache.initLockedFromDisk()
	cache.mu.Unlock()
	if cfgCacheEnabled {
		go func() {
			t := time.NewTicker(cfgCachePersistMs)
			defer t.Stop()
			for range t.C {
				cache.persist()
			}
		}()
	}
	if len(cfgFallbackChain) == 0 {
		log.Print("warning: FALLBACK_CHAIN empty; will only try primary model")
	}
	log.Printf("anthropic-shim starting on :%s | providers: %s | default: %s", cfgPort, strings.Join(cfgProviderOrder, ","), cfgFallbackModel)

	mux := http.NewServeMux()
	// NOTE: use plain patterns, NOT the "METHOD /path" form. This host's Go 1.26.3
	// toolchain builds binaries whose ServeMux silently drops method-prefix patterns
	// (GET "/" and GET "/v1/models" both 404 in a minimal repro). Plain "/" patterns
	// match correctly; handlers below enforce the HTTP method explicitly.
	mux.HandleFunc("/", healthHandler)
	mux.HandleFunc("/v1/models", modelsHandler)
	mux.HandleFunc("/v1/messages", messagesHandler)
	mux.HandleFunc("/v1/messages/count_tokens", countTokensHandler)

	server := &http.Server{
		Addr:              "127.0.0.1:" + cfgPort,
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		log.Print("shutting down; persisting cache ...")
		cache.persist()
		os.Exit(0)
	}()

	log.Fatal(server.ListenAndServe())
}
