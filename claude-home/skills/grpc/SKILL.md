---
name: grpc
description: gRPC API security testing — endpoint discovery, proto reflection, injection/authorization testing, and message fuzzing over HTTP/2. Use when target exposes :8080/:443 gRPC (protobuf) services or gateway on /v1.
category: api
---

# gRPC Testing

## Detection
- Fingerprint: HTTP/2 `content-type: application/grpc`, `grpc-status` headers, `grpc-timeout`, `te: trailers`, `X-Grpc-Web`.
- Tooling: grpcurl / grpc-web / postman gRPC. Try server reflection first: `grpcurl -plaintext $U list`.

## Exploitation
1. **Reflection enabled** → enumerate all services + methods (`grpcurl $U describe <svc>`), then test each.
2. **No reflection** → recover protos from: JS bundles (`grpc-web`), swagger, sourcemaps, mobile APK, `google.protobuf.Any` unknown.
3. **Authorization gaps**: methods lack per-user checks (BFLA), unauthenticated `Admin`/`Internal` services.
4. **Injection**: message fields → SQL/NoSQL/command injection same as HTTP.
5. **Metadata smuggling**: gRPC metadata (`authorization`, `x-user-id`) influences backend decision → tamper.
6. **Input validation fuzz**: invalid enum/int fields, missing required, oversized → error/panic leaks (stack traces in trailers).

## Tool Commands (Windows)
```powershell
# if grpcurl is installed
grpcurl.exe -plaintext "$U:443" list
grpcurl.exe -plaintext "$U:443" describe package.Service
grpcurl.exe -plaintext -d '{"id":"1 OR 1=1"}' "$U:443" package.Service/GetUser
grpcurl.exe -plaintext -H 'authorization: Bearer X' "$U:443" package.Admin/ListUsers
```

## Verification & Evidence
- Unauthorized data access/injection proof via gRPC call with request/response (grpcurl -v captures).
- Save request.json + response + reproduce.ps1 using grpcurl.