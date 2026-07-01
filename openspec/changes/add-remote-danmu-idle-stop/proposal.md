## Why

Remote danmu sessions can continue running even after the local bridge or other clients stop polling for events. This can leave Bilibili game sessions and websocket resources active with no consumer.

The remote service also currently bounds buffered danmu events near the intended limit, but the contract should explicitly cap retained events at 1000 per session.

## What Changes

- Automatically stop a remote game session when its `anchor_code` has no danmu pull API activity for 30 seconds.
- Treat an in-flight danmu long-poll request as active so a valid `wait_ms=30000` request is not stopped while it is waiting.
- Return an error response from `GET /openapi/games/:anchor_code/danmu` after a session has been auto-stopped.
- Cap retained remote danmu events at the latest 1000 events per session while keeping `seq` monotonically increasing.
- Preserve `GET /openapi/games/:anchor_code` as the way to inspect the stopped session snapshot and stop reason.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `remote-danmu-service`: add idle auto-stop behavior, define pull behavior after auto-stop, and tighten the event retention limit.

## Impact

- Affected code: `server/remote.go`, `server/remote_test.go`.
- Affected API behavior: `GET /openapi/games/:anchor_code/danmu` returns a non-success response for sessions stopped by idle timeout.
- Affected API contract/docs: `docs/wolfy_api.thrift` and `openspec/specs/remote-danmu-service/spec.md`.
- No new external dependencies are expected.
