# AGENTS.md

Project-specific guidance for ZCode agents working in this repository.

## Project Overview

**KvirNet** — a lightweight TUN-based virtual LAN that interconnects devices across networks via a centralized relay server. Go 1.25 backend, Wails v3 (alpha) desktop client.

- Module path in `go.mod` is **`virtualnet`** (the directory is `virnet`, the desktop app is `kvirnet` — import paths always use `virtualnet/...`).
- Virtual subnet is **`192.168.32.0/24`**; IP pool allocates `192.168.32.2`–`254` (253 addresses), see `server/client.go:initIPPool`.
- All traffic is relayed through the server (not P2P). The TCP data channel is **unencrypted**, and there is **no auto-reconnect** on disconnect.

## Directory Layout

| Path | Role |
|------|------|
| `server/` | Relay server: TCP (`:8081`) + WebSocket (`:8080`) — IP pool, connection coordination, `io.Copy` data forwarding. |
| `client/` | Client core library (used by both the CLI and the Wails desktop app). |
| `client/tun.go` | Shared TUN init via `golang.zx2c4.com/wireguard/tun`. |
| `client/tun_linux.go` / `client/tun_windows.go` | Platform-specific TUN config (build-tag split). |
| `common/common.go` | Shared WebSocket message types and protocol structs — the contract between server and client. |
| `kvirnet/` | Wails v3 desktop app wrapping `client/`; `frontend/` is React + TypeScript + Ant Design. |
| `sh/ci_server.sh` | Server deploy script (clones, builds, restarts process). |

## Build & Run

```bash
# Server (TCP :8081, WebSocket :8080)
cd server && go build && ./server

# CLI client
cd client && go run . -server ws://<server>:8080/ws

# Desktop client — uses Taskfile.yml (requires `wails3` CLI + `task`)
cd kvirnet
wails3 dev                              # or: wails3 task dev
wails3 task build | run | package       # build / run / package current OS

# Frontend only (Vite)
cd kvirnet/frontend && npm run dev      # dev server
cd kvirnet/frontend && npm run build    # tsc + vite build (output embedded via //go:embed all:frontend/dist)

# Regenerate Wails TS bindings after changing bound Go methods
cd kvirnet && wails3 generate bindings -f '{{.BUILD_FLAGS}}' -clean=true -ts
```

There is no project-wide `go vet`/lint config; run `go build ./...` from the repo root to typecheck.

## Protocol

- **Data channel (TCP):** 4-byte big-endian length prefix followed by the raw IPv4 packet. Max packet length is gated by MTU 1500 in `client/network.go`.
- **Signaling channel (WebSocket):** JSON `WSMessage{Type, Content}` where `Content` is `json.RawMessage`. All message-type constants and content structs live in `common/common.go` — edit there to extend the protocol on both sides at once.
- Only IPv4 packets destined for `192.168.32.0/24` are forwarded; broadcast (`192.168.32.255`) fans out via `io.MultiWriter`.

## Platform Handling (TUN)

TUN setup is split by Go build tags — **do not put platform code in `tun.go`**:

- `tun_linux.go` (`//go:build linux`): device `kvirnet0`, configured via `ip addr` / `ip link`.
- `tun_windows.go` (`//go:build windows`): device `wintun0`, configured via `netsh`.
- Creating TUN devices and running `ip`/`netsh` requires **root/administrator** privileges.

> Note: `README.md` still claims "Windows-only" — this is **outdated**. Linux is supported as of the `支持Linux客户端` commit. Trust the build-tagged source over the README.

## Conventions & Gotchas

- **Comments and log messages are in Chinese** — match this style when editing existing files.
- **Server address is hardcoded in two places** and must be kept in sync: the `-server` flag default in `client/client.go` (`ws://43.138.247.132:8080/ws`) and `tcpAddr` in `server/main.go` (`43.138.247.132:8081`).
- Wails v3 is **alpha** (`v3.0.0-alpha.37`); APIs and Taskfile tasks may differ from v2 examples online. Generated TS bindings live in `kvirnet/frontend/bindings/virtualnet/` — import bound methods from there in React, and regenerate after changing `kvirnet/client_service.go`.
- `client/` is a **library**, not a standalone binary — its `Run()` is invoked by `kvirnet/main.go` and the CLI entrypoint.
- Shared mutable state (`tcpConns`, `clients`) is guarded by `sync.Mutex`/`sync.RWMutex`; preserve locking when touching these maps.
