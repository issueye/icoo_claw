# Gateway Session API GORM Model Design

Gateway Session API now uses GORM + `github.com/glebarez/sqlite`, which runs on pure Go SQLite and does not require CGO.

## Tables

### `sessions`

Stores session metadata and optimistic concurrency state.

- Primary key: `session_id`
- Indexed filters: `user_id + updated_at`, `agent_id + updated_at`, `status + updated_at`
- `revision` is incremented when messages are appended or replaced, and is used as the CAS version for snapshot replacement.
- `metadata` is JSON text for agent-specific extensibility.

### `session_messages`

Stores normalized conversation messages.

- Primary key: message `id`
- Unique order key: `session_id + position`
- `position` is the canonical ordering column for tail, before, and after pagination.
- `content_blocks`, `tool_calls`, and `metadata` are JSON text fields so multimodal content and tool calls can evolve without schema churn.

### `session_runs`

Stores one agent/model execution run per row.

- Primary key: run `id`
- Unique order key: `session_id + position`
- Indexed trace field: `request_id`
- `usage` and `metadata` are JSON text fields for provider-specific token, latency, cost, and runtime attributes.

### `session_run_events`

Stores streaming or lifecycle events for each run.

- Primary key: event `id`
- Main replay index: `session_id + run_id + sequence`
- `payload` and `metadata` are JSON text fields, allowing delta, tool call, usage, and error events to share one table.

## Repository Semantics

- Session list reads are SQL-filtered and sorted by `updated_at DESC`.
- Message append and replace run in transactions and update the parent session timestamp.
- Message append and replace increment `revision`; run and event writes only touch `updated_at`.
- Session delete remains idempotent and cascades manually through events, runs, and messages.

