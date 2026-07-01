## ADDED Requirements

### Requirement: Remote service manages Bilibili game sessions
The remote service SHALL expose HTTP JSON APIs, documented in `docs/wolfy_api.thrift`, to start, inspect, and stop a Bilibili live-game session by `anchor_code`.

#### Scenario: Start game session
- **WHEN** a client sends `POST /openapi/games` with `app_id` and `anchor_code`
- **THEN** the remote service SHALL start the Bilibili Open Platform app, connect the danmu websocket, and return a `GameSession` with `status` set to `running`

#### Scenario: Get existing game session
- **WHEN** a client sends `GET /openapi/games/:anchor_code` for a running session
- **THEN** the remote service SHALL return the current `GameSession` including `game_id`, `status`, and `last_seq`

#### Scenario: Stop game session
- **WHEN** a client sends `DELETE /openapi/games/:anchor_code` for a running session
- **THEN** the remote service SHALL close the danmu websocket, call the Bilibili app end API, and return a `GameSession` with `status` set to `stopped`

### Requirement: Remote service buffers danmu events with cursors
The remote service SHALL assign a monotonically increasing `seq` to every accepted danmu event and allow clients to pull events by cursor.

#### Scenario: Pull events after sequence
- **WHEN** a client sends `GET /openapi/games/:anchor_code/danmu?after_seq=10&limit=100`
- **THEN** the remote service SHALL return events with `seq` greater than `10`, a `next_seq` cursor, and `has_more` when more buffered events remain

#### Scenario: Long poll when no event is available
- **WHEN** a client sends `GET /openapi/games/:anchor_code/danmu?after_seq=<latest>&wait_ms=30000`
- **THEN** the remote service SHALL wait until a new event arrives or the wait timeout expires before returning an empty event list

### Requirement: Remote service parses Wolfy danmu commands
The remote service SHALL hardcode command parsing for danmu messages prefixed with `点歌`, `换歌`, `换谱`, and `删除`, and SHALL include an optional parsed `Task` in each returned danmu event.

#### Scenario: Parse pick command
- **WHEN** the remote service receives danmu `点歌 sky` from caller `alice`
- **THEN** the returned event SHALL include `task.command` as `pick`, `task.caller` as `alice`, `task.content` as `sky`, and `task.index` as `-1`

#### Scenario: Parse rank switch command
- **WHEN** the remote service receives danmu `换歌 2` from caller `alice`
- **THEN** the returned event SHALL include `task.command` as `next_rank`, `task.caller` as `alice`, `task.content` as an empty string, and `task.index` as `1`

#### Scenario: Parse level switch command
- **WHEN** the remote service receives danmu `换谱 3` from caller `alice`
- **THEN** the returned event SHALL include `task.command` as `next_level`, `task.caller` as `alice`, `task.content` as an empty string, and `task.index` as `2`

#### Scenario: Parse delete command
- **WHEN** the remote service receives danmu `删除 1` from caller `alice`
- **THEN** the returned event SHALL include `task.command` as `finish`, `task.caller` as `alice`, `task.content` as an empty string, and `task.index` as `0`

#### Scenario: Preserve non-command danmu
- **WHEN** the remote service receives danmu without a supported command prefix
- **THEN** the returned event SHALL include the raw caller and message and SHALL omit `task`

### Requirement: Remote service replaces signing API
The remote service SHALL NOT expose the legacy `/sign` endpoint as part of the new contract.

#### Scenario: Legacy sign endpoint is unavailable
- **WHEN** a client calls `POST /sign`
- **THEN** the remote service SHALL return a non-success response or no route, and clients SHALL use the remote game/danmu APIs instead
