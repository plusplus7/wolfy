## Context

Wolfy currently runs a local service for the frontend and Bilibili danmu listener, plus a remote build that exposes `/sign` so the local listener can sign Bilibili Open Platform requests without storing AK/SK locally. This still leaves the local runtime responsible for starting the Bilibili app session, maintaining heartbeats, connecting to the websocket, and parsing danmu into ticket tasks.

The new direction is to make the remote build a dedicated danmu service. It owns AK/SK, the Bilibili app lifecycle, websocket listener, command parsing, and a small event buffer. The local build becomes a frontend-facing service that configures the remote source, pulls parsed events, and forwards tasks into the existing tickets/messages components.

## Goals / Non-Goals

**Goals:**

- Remove signing from the local runtime contract.
- Replace the remote `/sign` endpoint with a remote HTTP API described in `docs/wolfy_api.thrift`.
- Let the remote service hardcode parsing for `点歌`, `换歌`, `换谱`, and `删除`.
- Keep existing local frontend APIs compatible where possible.
- Keep songs and tickets behavior local so song matching, queue persistence, and frontend behavior remain under the local service.

**Non-Goals:**

- Do not introduce Thrift RPC runtime; the thrift file remains an HTTP JSON contract document.
- Do not move songs storage, ticket matching, ticket persistence, or frontend static hosting to the remote service.
- Do not add multi-tenant account management beyond routing sessions by `anchor_code`.
- Do not redesign the current ticket command model.

## Decisions

1. Remote returns parsed events, not just raw danmu.

   The remote service SHALL return each danmu event with raw caller/message fields and an optional parsed `Task`. This matches the user's preference to parse command prefixes remotely while preserving raw danmu for local messages and debugging. The alternative was to keep parsing local, but that would leave more danmu-specific behavior in the local component.

2. Local still owns ticket execution.

   The remote service SHALL NOT call local ticket APIs or know about songs storage. Parsed tasks are data only. The local service pulls them and feeds the existing `TicketsComponent`, preserving local queue rules such as max queue size, per-user limits, song matching, checkpointing, and manual frontend actions.

3. Use polling with sequence cursors for the first implementation.

   The remote API SHALL expose `GET /openapi/games/:anchor_code/danmu?after_seq=&limit=&wait_ms=`. Long polling is easier to represent in the existing HTTP/thrift documentation and avoids adding a new bidirectional transport. SSE or websocket can be added later if polling becomes insufficient.

4. Keep local API compatibility and add only a small danmu control API.

   The current frontend-compatible endpoints remain: `/api/tickets`, `/api/messages`, `/api/event/:caller/:event/:content`, `/api/sysinfo`, and component management endpoints. New local endpoints are limited to `/api/danmu`, `/api/danmu/start`, and `/api/danmu/stop` so settings UI can manage the remote bridge without using generic component params.

5. Remove AK/SK component params from the local contract.

   The local `danmu` component SHALL only expose `remote_base_url`, `app_id`, and `anchor_code`. Existing `danmu.bilibili_ak_id` and `danmu.bilibili_ak_secret` params become obsolete. The remote build continues reading AK/SK from its deployment environment.

## Risks / Trade-offs

- Remote parser drift -> Keep the parsed task shape identical to `model.Task` and cover prefix parsing with tests for `点歌`, `换歌`, `换谱`, and `删除`.
- Polling duplicates or gaps -> Use monotonically increasing `seq` and persist the local component's in-memory `last_seq` during a running bridge session; remote responses include `next_seq` and `has_more`.
- Remote memory growth -> Bound each `anchor_code` event buffer and return a clear error or cursor reset behavior if the local client falls behind too far.
- Existing saved params contain AK/SK -> Ignore unknown legacy signing params during migration or clean them from the param store when the danmu component registers the new key set.
- Remote `/sign` consumers break -> Treat `/sign` removal as a documented breaking change and update tests/docs together with the new remote API.

## Migration Plan

1. Update `docs/wolfy_api.thrift` to the new remote danmu and local bridge contract.
2. Replace the remote sign server with a remote danmu server in the remote build.
3. Refactor the local danmu component to call the remote game/danmu APIs and forward returned tasks/messages locally.
4. Preserve existing local frontend APIs and component management behavior.
5. Remove or rewrite remote signing tests to cover remote game lifecycle, polling, and command parsing.

Rollback is straightforward before deployment by reverting to the previous remote build and local danmu component. After deployment, clients depending on `/sign` must be migrated because `/sign` is intentionally removed from the new contract.
