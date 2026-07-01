## 1. Contract and Impact Review

- [x] 1.1 Run GitNexus impact analysis for the symbols that will be edited, including `RemoteSignatoryServer`, `DanmuComponent`, `LocalServer`, `AppService`, and websocket handling symbols; report any HIGH or CRITICAL risk before editing.
- [x] 1.2 Update `docs/wolfy_api.thrift` to remove the `/sign` contract and document `WolfyRemoteDanmuHTTPAPI`, `DanmuEvent`, `GameSession`, `Task`, and the refined local `/api/danmu` contract.
- [x] 1.3 Add or update contract-focused tests or snapshots that verify the documented JSON field names match the implementation structs.

## 2. Remote Danmu Service

- [x] 2.1 Replace the remote build entrypoint wiring so `go run -tags remote .` starts a remote danmu service instead of the remote signing service.
- [x] 2.2 Implement remote game session management keyed by `anchor_code`, including start, get status, stop, cancellation, and Bilibili app end behavior.
- [x] 2.3 Implement bounded per-session danmu event buffering with monotonically increasing `seq`, `next_seq`, `has_more`, `limit`, and `wait_ms` polling behavior.
- [x] 2.4 Move websocket danmu delivery into the remote service so incoming Bilibili messages are converted into buffered `DanmuEvent` values.
- [x] 2.5 Implement hardcoded remote parsing for `点歌`, `换歌`, `换谱`, and `删除` into the existing `model.Task` command shape.
- [x] 2.6 Remove or disable the legacy `/sign` route from the remote server.

## 3. Local Danmu Bridge

- [x] 3.1 Refactor the local danmu component params to `remote_base_url`, `app_id`, and `anchor_code`, removing local AK/SK params from the intended contract.
- [x] 3.2 Implement a local remote-danmu HTTP client for start, stop, status, and poll operations using the thrift-documented JSON shapes.
- [x] 3.3 Replace local Bilibili open-platform startup with a pull loop that polls remote danmu events by cursor.
- [x] 3.4 Deliver each remote event to the messages component and deliver only events with parsed tasks to the tickets component.
- [x] 3.5 Ensure the bridge updates `last_seq` only after successfully handling the corresponding remote events in the current running session.

## 4. Local HTTP Compatibility

- [x] 4.1 Preserve current local frontend routes and response envelopes for `/api/tickets`, `/api/messages`, `/api/event/:caller/:event/:content`, `/api/sysinfo`, and component management endpoints.
- [x] 4.2 Add `GET /api/danmu`, `PATCH /api/danmu`, `POST /api/danmu/start`, and `POST /api/danmu/stop` handlers backed by the local danmu bridge.
- [x] 4.3 Ensure legacy saved signing params do not break startup after the danmu component registers the new param set.

## 5. Tests and Verification

- [x] 5.1 Replace remote signing tests with remote danmu service tests for start/status/stop routing and `/sign` removal.
- [x] 5.2 Add parser tests covering `点歌`, `换歌`, `换谱`, `删除`, invalid indices, and non-command danmu events.
- [x] 5.3 Add remote buffer polling tests for cursor advancement, empty long-poll timeout, `has_more`, and bounded-buffer fallback behavior.
- [x] 5.4 Add local bridge tests verifying remote events become messages and parsed tasks are delivered to tickets exactly once per cursor.
- [x] 5.5 Add local HTTP tests for `/api/danmu` handlers and compatibility checks for existing frontend routes.
- [x] 5.6 Run the full Go test suite for normal and remote build tags, then run `gitnexus_detect_changes()` before any commit.
