## Context

The remote danmu service owns Bilibili app lifecycle, websocket consumption, and the per-`anchor_code` event buffer. Local clients consume events through `GET /openapi/games/:anchor_code/danmu`, usually in a loop.

Today a remote session stops only through an explicit stop request or replacement start. If the local bridge exits or loses connectivity without calling stop, the remote session can keep the Bilibili app and websocket alive with no consumer. The event buffer is already bounded, but its current capacity is not the requested contract limit of 1000 retained events.

## Goals / Non-Goals

**Goals:**

- Stop remote sessions automatically when no danmu pull activity is observed for 30 seconds.
- Keep valid long-poll requests active for their full wait window, including `wait_ms=30000`.
- Return a non-success response for danmu pulls after idle auto-stop.
- Retain at most the latest 1000 danmu events per remote session.
- Preserve existing explicit start, get, stop, cursor, and command parsing behavior.

**Non-Goals:**

- Do not stop sessions merely because no new danmu messages arrive.
- Do not change the local bridge polling interval in this change.
- Do not introduce persistent storage for remote danmu events.
- Do not add external dependencies or background job infrastructure.

## Decisions

1. Track client pull activity in the remote session, not in the local bridge.

   Rationale: the remote service is the owner of Bilibili app lifecycle and can protect itself from any client that disappears, not just the local Wolfy bridge.

   Alternative considered: have the local bridge explicitly stop on shutdown. That is still useful behavior, but it does not protect the remote service from crashes, network loss, or non-Wolfy clients.

2. Treat an in-flight pull request as active.

   Rationale: the remote API already supports long polling with `wait_ms` up to 30000. A request waiting for new events must not be stopped exactly because it has not returned yet.

   Implementation direction: maintain session activity state such as `lastPullAt` and `activePolls`. The idle monitor should only auto-stop when `activePolls == 0` and `now - lastPullAt >= 30s`.

3. Reuse the existing session stop path for idle timeout.

   Rationale: `remoteDanmuSession.stop` cancels the session context and wakes long-poll waiters, and the app service already calls Bilibili app end when the session context is canceled. Reusing this path avoids duplicating lifecycle shutdown logic.

   The stop reason should be visible in the session snapshot, for example `idle timeout`.

4. Return a pull error after idle auto-stop.

   Rationale: after auto-stop, continuing to return empty pull responses makes clients look healthy while the remote game is no longer running. A non-success response lets the local bridge enter its existing error path.

   Suggested behavior: unknown `anchor_code` remains `404`; known but stopped sessions return `409 Conflict` with a clear message.

5. Set remote event retention to exactly 1000 events per session.

   Rationale: the service only needs a recent cursor window for reconnecting clients. Keeping `seq` monotonic while retaining only the latest 1000 events preserves cursor semantics and bounds memory.

## Risks / Trade-offs

- Idle monitor timing can be flaky in tests -> inject or make timeout/check interval configurable inside tests where needed.
- A long-poll request near the 30 second boundary could race with the idle monitor -> count active polls under the same session lock used for stop/status updates.
- Returning `409` for stopped sessions changes API behavior -> update tests and API documentation so clients can handle it intentionally.
- Retaining only the latest 1000 events means a very stale cursor can miss older events -> this is already the bounded-buffer behavior, now with a precise limit.

## Migration Plan

Deploying the remote service with this change is sufficient. Existing local bridge behavior should continue polling every second, so active sessions should not auto-stop during normal operation.

Rollback is to restore the previous remote session lifecycle and buffer capacity if clients cannot handle stopped-session pull errors.

## Open Questions

None.
