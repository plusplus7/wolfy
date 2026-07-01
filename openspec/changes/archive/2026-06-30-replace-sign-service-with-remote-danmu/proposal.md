## Why

The current architecture keeps Bilibili AK/SK out of most local setups by using a remote signing service, but the local service still owns the Bilibili Open Platform lifecycle and websocket listener. Moving the live-game session and danmu listener to the remote service removes signing from the local runtime entirely and gives the local service a smaller, frontend-focused role.

## What Changes

- Replace the remote `/sign` service with a remote danmu HTTP service that starts, reports, stops, and polls Bilibili live-game sessions by `anchor_code`.
- Hardcode remote parsing for danmu prefixes `点歌`, `换歌`, `换谱`, and `删除`, returning parsed `Task` payloads alongside raw danmu events.
- Change the local danmu component to store only `remote_base_url`, `app_id`, and `anchor_code`, then pull parsed danmu tasks from the remote service and deliver them to the existing tickets/messages components.
- Keep the current frontend-compatible local APIs: `GET /api/tickets`, `GET /api/messages`, `GET /api/event/:caller/:event/:content`, `GET /api/sysinfo`, and component management endpoints.
- Add a small local `/api/danmu` control surface for reading/updating remote danmu config and starting/stopping the bridge.
- **BREAKING**: Remove local and remote signing APIs/parameters from the intended contract: `/sign`, `danmu.bilibili_ak_id`, and `danmu.bilibili_ak_secret`.
- Update `docs/wolfy_api.thrift` to describe the new remote danmu service and refined local service contract.

## Capabilities

### New Capabilities

- `remote-danmu-service`: Remote HTTP API for Bilibili live-game lifecycle, danmu buffering, and hardcoded command parsing.
- `local-danmu-bridge`: Local HTTP/API and component behavior for configuring the remote danmu source, pulling parsed tasks, and preserving existing frontend compatibility.

### Modified Capabilities

- None.

## Impact

- Remote build entrypoint and server code currently serving `/sign`.
- Bilibili danmu/signatory code under `components/danmu/bilibili`.
- Local `danmu` component parameters and startup behavior.
- Local HTTP handlers and API documentation in `server/local.go` and `docs/wolfy_api.thrift`.
- Tests around remote signing, danmu parsing/listening, local component params, and local HTTP compatibility.
