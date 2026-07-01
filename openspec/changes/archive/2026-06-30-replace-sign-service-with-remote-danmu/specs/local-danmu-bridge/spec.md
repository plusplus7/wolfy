## ADDED Requirements

### Requirement: Local service preserves frontend-compatible APIs
The local service SHALL preserve the existing frontend-compatible HTTP APIs for tickets, messages, manual frontend events, system info, and component management.

#### Scenario: Existing ticket and message APIs still work
- **WHEN** the frontend calls `GET /api/tickets` or `GET /api/messages`
- **THEN** the local service SHALL return the same JSON envelope shape as the current API

#### Scenario: Existing manual event API still works
- **WHEN** the frontend calls `GET /api/event/:caller/:event/:content`
- **THEN** the local service SHALL convert the event to a local ticket task and execute it through the tickets component

### Requirement: Local danmu configuration uses remote source only
The local danmu component SHALL expose only `remote_base_url`, `app_id`, and `anchor_code` as danmu configuration in the intended API contract.

#### Scenario: Read local danmu status
- **WHEN** a client calls `GET /api/danmu`
- **THEN** the local service SHALL return danmu bridge status, `remote_base_url`, `app_id`, `anchor_code`, `last_seq`, and any current error

#### Scenario: Update local danmu configuration
- **WHEN** a client calls `PATCH /api/danmu` with `remote_base_url`, `app_id`, and `anchor_code`
- **THEN** the local service SHALL persist those values and SHALL NOT require or expose Bilibili AK/SK values locally

### Requirement: Local bridge controls remote danmu session
The local service SHALL provide concise control endpoints that start and stop the remote danmu bridge using the saved local danmu configuration.

#### Scenario: Start local bridge
- **WHEN** a client calls `POST /api/danmu/start`
- **THEN** the local service SHALL call the remote start game API, begin pulling remote danmu events, and report bridge status as `running` when the pull loop starts successfully

#### Scenario: Stop local bridge
- **WHEN** a client calls `POST /api/danmu/stop`
- **THEN** the local service SHALL stop the local pull loop, call the remote stop game API for the configured `anchor_code`, and report bridge status as `stopped` or `waiting`

### Requirement: Local bridge delivers remote tasks to tickets and messages
The local bridge SHALL consume remote danmu events, store raw danmu messages in the local messages component, and deliver parsed tasks to the local tickets component.

#### Scenario: Remote event contains parsed task
- **WHEN** the local bridge pulls a remote danmu event containing a `Task`
- **THEN** the local service SHALL push an informational danmu message to messages and pass the `Task` to the tickets component

#### Scenario: Remote event has no task
- **WHEN** the local bridge pulls a remote danmu event without a `Task`
- **THEN** the local service SHALL push an informational danmu message to messages and SHALL NOT call the tickets component for that event

#### Scenario: Remote event is delivered once per cursor
- **WHEN** the local bridge successfully handles events up to `next_seq`
- **THEN** subsequent pulls SHALL use the updated cursor and SHALL NOT redeliver already handled events during the same running bridge session

### Requirement: Local service removes signing dependency
The local service SHALL NOT create local Bilibili signatories or call the legacy remote `/sign` endpoint for danmu operation.

#### Scenario: Local start does not require AK/SK
- **WHEN** the local danmu bridge starts with valid `remote_base_url`, `app_id`, and `anchor_code`
- **THEN** the bridge SHALL operate without reading `danmu.bilibili_ak_id` or `danmu.bilibili_ak_secret`
