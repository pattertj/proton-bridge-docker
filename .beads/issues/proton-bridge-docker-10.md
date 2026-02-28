---
type: task
title: "Bridge CLI interaction layer"
status: done
priority: high
epic: proton-bridge-docker-8
blocked_by: [proton-bridge-docker-9]
created: 2026-02-27
---

# Bridge CLI interaction layer

Create `webui/handlers/bridge.go` — the core technical challenge:

## BridgeCLI struct
- Manages `bridge --cli` subprocess via stdin/stdout pipes
- `sync.Mutex` for serialized access
- State machine: RUNNING / SETUP_MODE / TRANSITIONING

## Methods
- `NewBridgeCLI()` — kill non-interactive bridge, spawn CLI, wait for `>>>` prompt
- `Execute(command)` — send command, read until `>>>`, return output
- `Login(username, password)` — multi-step: send login, handle Username/Password prompts, detect 2FA requirement
- `SubmitTwoFactor(code)` — send 2FA code, parse success/failure
- `GetAccountInfo()` — send `info`, parse into Account structs (email, ports, username, password, security)
- `Close()` — send `exit`, wait for termination, restart non-interactive bridge

## Critical details
- Prompt detection: `>>> ` delimiter, 30s timeout
- Parse patterns: "Username:", "Password:", "Two factor code:", "was added successfully", "Incorrect login credentials"
- Sync progress: parse `Sync (account): XX.X%` lines
- Goroutine for continuous stdout reading with buffering
