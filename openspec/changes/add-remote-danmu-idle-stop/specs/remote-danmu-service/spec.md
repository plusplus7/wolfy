## ADDED Requirements

### Requirement: Remote service auto-stops idle danmu sessions
The remote service SHALL automatically stop a running Bilibili game session when its `anchor_code` has no danmu pull API activity for 30 seconds.

#### Scenario: Auto-stop after no pull activity
- **GIVEN** a running session exists for `anchor_code`
- **WHEN** no client calls `GET /openapi/games/:anchor_code/danmu` for 30 seconds
- **THEN** the remote service SHALL stop the session and close its Bilibili app lifecycle
- **AND** `GET /openapi/games/:anchor_code` SHALL return a `GameSession` with `status` set to `stopped` and an error or reason indicating idle timeout

#### Scenario: In-flight long poll keeps session active
- **GIVEN** a running session exists for `anchor_code`
- **WHEN** a client has an in-flight `GET /openapi/games/:anchor_code/danmu?after_seq=<latest>&wait_ms=30000` request
- **THEN** the remote service SHALL treat the in-flight request as pull activity until it returns or is canceled
- **AND** the remote service SHALL NOT auto-stop the session solely because the request waited for 30 seconds

## MODIFIED Requirements

### Requirement: Remote service buffers danmu events with cursors
The remote service SHALL assign a monotonically increasing `seq` to every accepted danmu event, retain at most the latest 1000 danmu events per session, and allow clients to pull events by cursor while the session is running.

#### Scenario: Pull events after sequence
- **WHEN** a client sends `GET /openapi/games/:anchor_code/danmu?after_seq=10&limit=100`
- **THEN** the remote service SHALL return events with `seq` greater than `10`, a `next_seq` cursor, and `has_more` when more buffered events remain

#### Scenario: Long poll when no event is available
- **WHEN** a client sends `GET /openapi/games/:anchor_code/danmu?after_seq=<latest>&wait_ms=30000`
- **THEN** the remote service SHALL wait until a new event arrives or the wait timeout expires before returning an empty event list

#### Scenario: Retain latest 1000 events
- **WHEN** a running session receives more than 1000 danmu events
- **THEN** the remote service SHALL retain only the latest 1000 events for that session
- **AND** assigned `seq` values SHALL remain monotonically increasing across discarded and retained events

#### Scenario: Pull after idle auto-stop returns error
- **GIVEN** a session for `anchor_code` was automatically stopped because no danmu pull activity occurred for 30 seconds
- **WHEN** a client calls `GET /openapi/games/:anchor_code/danmu`
- **THEN** the remote service SHALL return a non-success response indicating the session has stopped due to idle timeout
