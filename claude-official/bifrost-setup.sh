#!/bin/bash
# Configure Bifrost gateway from scratch: Zen upstream, opencode UA, PUBLIC key.
# Idempotent-ish: wipes config.db so no stale secrets remain. Run while bifrost stopped.
set -e
PORT=9084
UA="opencode/1.18.18 ai-sdk/provider-utils/4.0.23 runtime/bun/1.3.14"
DB="${BIFROST_DB:-$HOME/.config/bifrost/config.db}"

rm -f "$DB"
rc-service bifrost restart || true
sleep 5

# provider-level config
curl -s -m15 -X POST http://127.0.0.1:$PORT/api/providers \
 -H 'Content-Type: application/json' \
 -d "{\"provider\":\"openai\",\"keys\":[],\"network_config\":{\"base_url\":\"https://opencode.ai/zen\",\"extra_headers\":{\"User-Agent\":\"$UA\"}}}" >/dev/null || {
   curl -s -m15 -X PUT http://127.0.0.1:$PORT/api/providers/openai \
    -H 'Content-Type: application/json' \
    -d "{\"network_config\":{\"base_url\":\"https://opencode.ai/zen\",\"extra_headers\":{\"User-Agent\":\"$UA\"}},\"concurrency_and_buffer_size\":{\"concurrency\":1000,\"buffer_size\":5000}}" >/dev/null; }

# key: literal "public" — Zen free tier. No real secret anywhere.
curl -s -m15 -X POST http://127.0.0.1:$PORT/api/providers/openai/keys \
 -H 'Content-Type: application/json' \
 -d '{"name":"zen","value":"public","models":["big-pickle","moonshotai/Kimi-K2.6","mimo-v2.5-free","nemotron-3-ultra-free","nemotron-3.5-lightning-free","hy3-free"],"weight":1,"use_anthropic_endpoints":true}' >/dev/null

echo "--- verify ---"
curl -s http://127.0.0.1:$PORT/api/providers/openai | grep -o '"base_url":"[^"]*"'
curl -s http://127.0.0.1:$PORT/api/providers/openai/keys | python3 -c "import json,sys;print('keys:',[(k['name'],k['value']['value']) for k in json.load(sys.stdin)['keys']])"
grep -q "sk-" "$DB" && echo "WARN: sk- string found in DB" || echo "DB clean of sk- keys"
