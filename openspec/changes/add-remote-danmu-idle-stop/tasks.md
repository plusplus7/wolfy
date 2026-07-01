## 1. Impact And Contract Review

- [ ] 1.1 Run GitNexus impact analysis for the remote session symbols that will be edited, including `RemoteDanmuManager.Poll`, `remoteDanmuSession.poll`, `remoteDanmuSession.stop`, and `remoteDanmuSession.appendEvent`.
- [ ] 1.2 Confirm the API behavior in `docs/wolfy_api.thrift`: unknown `anchor_code` remains not found, while known stopped sessions return a pull error.

## 2. Remote Session Lifecycle

- [ ] 2.1 Change the remote danmu event retention capacity to exactly 1000 retained events per session.
- [ ] 2.2 Add per-session pull activity tracking so each danmu pull marks activity and in-flight long polls count as active.
- [ ] 2.3 Add idle monitoring for running sessions that auto-stops after 30 seconds with no pull activity and records an idle timeout reason.
- [ ] 2.4 Ensure idle auto-stop reuses the existing session cancel/stop path so websocket and Bilibili app lifecycle cleanup still happen once.
- [ ] 2.5 Make danmu pulls for known stopped sessions return a non-success response, while preserving existing not-found behavior for unknown `anchor_code`.

## 3. Tests And Documentation

- [ ] 3.1 Add remote server/session tests for auto-stop after no pull activity.
- [ ] 3.2 Add tests proving an in-flight long poll prevents idle auto-stop until it returns or is canceled.
- [ ] 3.3 Add tests proving pulls after idle auto-stop return an error response.
- [ ] 3.4 Update the bounded-buffer test to assert the 1000 event retention contract or a test-configurable equivalent.
- [ ] 3.5 Update `docs/wolfy_api.thrift` to document idle auto-stop and stopped-session pull errors.

## 4. Verification

- [ ] 4.1 Run `go test ./server ./components/danmu`.
- [ ] 4.2 Run `go test ./components/danmu/bilibili` with a writable `GOCACHE` and required local-port permissions.
- [ ] 4.3 Run `openspec status --change add-remote-danmu-idle-stop` and confirm the change is apply-ready.
- [ ] 4.4 Run `gitnexus_detect_changes()` before any commit to verify affected symbols and execution flows are expected.
