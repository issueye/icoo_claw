# Codex Desktop Chat Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Validate Wails3 first, then build a Wails3 desktop chat client for `icoo_claw` that connects to the existing gateway in gateway-only mode, with WebSocket-based chat streaming and TOML-backed local settings.

**Architecture:** Start with a narrow Wails3 technical spike that proves Windows desktop packaging, Go/JS bindings, file dialogs, and frontend hot reload work in this repo. Then create a new top-level `desktop/` Wails3 module so the app stays isolated from `server/*`. Keep system integration and TOML persistence in Go services exposed through Wails bindings, while the Vue3 frontend owns UI, state, routing, and gateway HTTP plus WebSocket calls. Gateway should expose a new WebSocket chat endpoint for the desktop app; `gateway -> claw` can continue using the existing internal SSE/channel pipeline in phase 1. Conversation creation should be draft-first: the first user prompt generates the local title, then the frontend creates the conversation with that title before sending the first message.

**Tech Stack:** Go 1.25, Wails v3 (`wails3` CLI), Vue3, Vite, JavaScript, Pinia, Vue Router, Tailwind CSS, Vitest, TOML via Go, WebSocket on Gateway.

---

## Preconditions

- Use the existing gateway APIs in [server/gateway/internal/router/router.go](/E:/code/issueye/icoo_claw/server/gateway/internal/router/router.go) and DTOs in [server/gateway/internal/dto/conversation.go](/E:/code/issueye/icoo_claw/server/gateway/internal/dto/conversation.go) and [server/gateway/internal/dto/agent.go](/E:/code/issueye/icoo_claw/server/gateway/internal/dto/agent.go).
- Do not add a local-mode path in P0.
- Do not add agent create/edit UI in P0; only consume `GET /v1/agents` and persist the selected default agent locally.
- Replace the gateway external streaming contract with WebSocket for the desktop app.
- Keep `gateway -> claw` on the existing internal stream path in phase 1 unless a later task explicitly upgrades it too.

## Target Directory Layout

```text
desktop/
  go.mod
  main.go
  Taskfile.yml
  build/config.yaml
  app.go
  internal/
    app/
      app.go
    config/
      config.go
      store.go
      store_test.go
    service/
      config_service.go
      system_service.go
  frontend/
    package.json
    vite.config.js
    postcss.config.js
    tailwind.config.js
    index.html
    src/
      main.js
      App.vue
      assets/tailwind.css
      router/index.js
      layouts/AppShell.vue
      components/
        chrome/AppTitlebar.vue
        chrome/AppSidebar.vue
        chat/ChatComposer.vue
        chat/ChatMessageList.vue
        chat/ChatMessageItem.vue
        chat/ChatEmptyState.vue
        chat/ChatStatusBar.vue
        conversation/ConversationList.vue
        project/ProjectSwitcher.vue
        common/EmptyPlaceholder.vue
      views/
        ChatHomeView.vue
        ChatConversationView.vue
        SearchView.vue
        SkillsView.vue
        PluginsView.vue
        AutomationsView.vue
        SettingsView.vue
      stores/
        app.js
        settings.js
        agents.js
        projects.js
        conversations.js
        chat.js
      services/
        gateway/http.js
        gateway/ws.js
        gateway/agents.js
        wails/config.js
        utils/title.js
      tests/
        title.spec.js
        ws.spec.js
        settings.spec.js
```

## Implementation Notes

- Prefer a new `desktop` Go module and add it to [go.work](/E:/code/issueye/icoo_claw/go.work).
- Start from Wails v3 official CLI conventions: `wails3 dev` for hot reload and `wails3 build` for production builds. Wails v3 docs currently show `build/config.yaml`, `Taskfile.yml`, and generated frontend bindings under `frontend/bindings/`.
- Keep the Wails Go services small: config IO, directory picking, app info. Gateway requests remain in frontend JavaScript.
- Use Vue Router for the main content views; keep project list and conversation list in the persistent left sidebar.
- Use a draft conversation model in the frontend. A conversation record should only be created remotely when the user sends the first prompt.
- Add a gateway WebSocket endpoint dedicated to desktop chat. Recommended path: `GET /v1/ws/chat`.
- Define a compact WebSocket event protocol instead of tunneling raw SSE lines to the client.

### Task 0: Validate Wails3 in this repo

**Files:**
- Create: `docs/plans/2026-05-19-wails3-technical-spike.md`
- Create: `desktop_spike/go.mod`
- Create: `desktop_spike/main.go`
- Create: `desktop_spike/app.go`
- Create: `desktop_spike/frontend/package.json`
- Create: `desktop_spike/frontend/src/main.js`
- Create: `desktop_spike/frontend/src/App.vue`
- Modify: [go.work](/E:/code/issueye/icoo_claw/go.work)

**Step 1: Create a minimal Wails3 spike**

- Build the smallest possible Wails3 app in a temporary `desktop_spike/` module.
- Validate:
  - app boots on Windows
  - Vue3 frontend renders
  - Go method binding works
  - directory picker works
  - `wails3 dev` hot reload works
  - `wails3 build` can package a Windows binary

**Step 2: Record exact findings**

- Write down:
  - Wails3 version used
  - install steps
  - any Windows-specific caveats
  - any binding generation quirks
  - whether the final implementation should reuse or replace the spike directory

**Step 3: Decision gate**

- If the spike fails on a blocking issue, stop and revise the desktop stack before touching gateway or frontend implementation.
- If it passes, fold the proven structure into the real `desktop/` module.

**Step 4: Verify**

- Run: `go test ./desktop_spike/...`
- Run: `cd desktop_spike/frontend; npm install`
- Run: `cd desktop_spike/frontend; npm run build`
- Run: `cd desktop_spike; wails3 build`

**Step 5: Commit**

- Commit message: `chore: validate wails3 spike`

### Task 1: Refactor gateway external streaming from SSE to WebSocket

**Files:**
- Modify: [server/gateway/go.mod](/E:/code/issueye/icoo_claw/server/gateway/go.mod)
- Modify: [server/gateway/internal/router/router.go](/E:/code/issueye/icoo_claw/server/gateway/internal/router/router.go)
- Modify: [server/gateway/internal/controller/chat_controller.go](/E:/code/issueye/icoo_claw/server/gateway/internal/controller/chat_controller.go)
- Modify: [server/gateway/internal/service/chat_service.go](/E:/code/issueye/icoo_claw/server/gateway/internal/service/chat_service.go)
- Modify: [server/gateway/internal/client/claw_client.go](/E:/code/issueye/icoo_claw/server/gateway/internal/client/claw_client.go)
- Create: `server/gateway/internal/dto/chat_ws.go`
- Create: `server/gateway/internal/service/chat_ws_service.go`
- Create: `server/gateway/internal/controller/chat_ws_controller.go`
- Create: `server/gateway/internal/service/chat_ws_service_test.go`
- Modify: `docs/ai-agent-platform/01-requirements.md`
- Modify: `docs/ai-agent-platform/02-architecture.md`
- Modify: `docs/ai-agent-platform/03-technical-spec.md`
- Modify: `docs/ai-agent-platform/05-mvp.md`

**Step 1: Introduce the WebSocket contract**

- Add a new gateway endpoint:
  - `GET /v1/ws/chat`
- Recommended client -> gateway messages:
  - `chat.start`
  - `chat.cancel`
  - `ping`
- Recommended gateway -> client messages:
  - `session.accepted`
  - `message.delta`
  - `message.completed`
  - `message.error`
  - `pong`

**Step 2: Keep gateway to claw simple in phase 1**

- Reuse the existing `ClawClient.Stream()` and gateway service channel pipeline.
- Gateway should translate claw stream events into the new WebSocket event format.
- Do not refactor `server/claw` transport in this task unless required by tests.

**Step 3: Move stream orchestration out of the HTTP controller**

- Add a dedicated WebSocket service that:
  - upgrades the connection
  - validates incoming payloads
  - starts chat runs
  - forwards incremental events
  - handles cancel and connection close

**Step 4: Remove deprecated REST stream route**

- Remove the deprecated REST stream endpoint entirely.
- Document the chosen policy in the platform docs.

**Step 5: Test**

- Add tests for:
  - successful WebSocket connect
  - valid `chat.start`
  - event forwarding from fake claw stream
  - malformed client payload
  - session busy propagation
  - clean disconnect

**Step 6: Commit**

- Commit message: `feat: add gateway websocket chat streaming`

### Task 2: Scaffold the desktop workspace

**Files:**
- Modify: [go.work](/E:/code/issueye/icoo_claw/go.work)
- Create: `desktop/go.mod`
- Create: `desktop/main.go`
- Create: `desktop/app.go`
- Create: `desktop/Taskfile.yml`
- Create: `desktop/build/config.yaml`
- Create: `desktop/frontend/package.json`
- Create: `desktop/frontend/vite.config.js`
- Create: `desktop/frontend/postcss.config.js`
- Create: `desktop/frontend/tailwind.config.js`
- Create: `desktop/frontend/index.html`
- Create: `desktop/frontend/src/main.js`
- Create: `desktop/frontend/src/App.vue`
- Create: `desktop/frontend/src/assets/tailwind.css`

**Step 1: Initialize the Wails v3 app shell**

- Create the `desktop/` module and wire it into `go.work`.
- Use the Wails v3 CLI structure rather than inventing a custom bootstrap layout.
- Set the frontend working directory to `desktop/frontend`.

**Step 2: Add the Vue/Vite/Tailwind frontend skeleton**

- Create `package.json` scripts for `dev`, `build`, and `test`.
- Add Vue3, Pinia, Vue Router, Tailwind CSS, PostCSS, and Vitest.
- Create a minimal `src/main.js` that mounts Vue, Pinia, Router, and imports Tailwind.

**Step 3: Add a smoke-level desktop entrypoint**

- `main.go` should create the application and register Go services.
- `App.vue` should render a temporary “desktop shell loading” marker until layout work begins.

**Step 4: Verify the scaffold**

- Run: `go test ./desktop/...`
- Run: `cd desktop/frontend; npm install`
- Run: `cd desktop/frontend; npm run build`
- Run: `cd desktop; wails3 build`

**Step 5: Commit**

- Commit message: `chore: scaffold desktop wails app`

### Task 3: Implement TOML config storage and Wails bindings

**Files:**
- Create: `desktop/internal/config/config.go`
- Create: `desktop/internal/config/store.go`
- Create: `desktop/internal/config/store_test.go`
- Create: `desktop/internal/service/config_service.go`
- Create: `desktop/internal/service/system_service.go`
- Modify: `desktop/main.go`

**Step 1: Define the config schema**

- Add `AppConfig`, `GatewayConfig`, `ChatConfig`, `ProjectConfig`, and `UIConfig`.
- Include these confirmed fields:
  - gateway base URL
  - timeout
  - default agent id
  - default model label
  - theme
  - project list
  - current project id

**Step 2: Build a file-backed store**

- Resolve the config path under the user config directory, for example:
  - `.../icoo_claw_desktop/config.toml`
- Add `Load()`, `Save(AppConfig)`, and `Default()` methods.
- Handle first-run creation and malformed TOML with explicit errors.

**Step 3: Expose Wails services**

- `ConfigService`:
  - `GetConfig()`
  - `SaveConfig(payload)`
  - `ResetConfig()`
- `SystemService`:
  - `GetAppInfo()`
  - `ChooseDirectory()`
  - `OpenConfigDirectory()`

**Step 4: Test config behavior**

- Write unit tests for:
  - default config creation
  - save -> reload roundtrip
  - malformed TOML error handling

**Step 5: Commit**

- Commit message: `feat: add desktop config services`

### Task 4: Build the app shell, router, and placeholder views

**Files:**
- Create: `desktop/frontend/src/router/index.js`
- Create: `desktop/frontend/src/layouts/AppShell.vue`
- Create: `desktop/frontend/src/components/chrome/AppTitlebar.vue`
- Create: `desktop/frontend/src/components/chrome/AppSidebar.vue`
- Create: `desktop/frontend/src/components/common/EmptyPlaceholder.vue`
- Create: `desktop/frontend/src/views/ChatHomeView.vue`
- Create: `desktop/frontend/src/views/SearchView.vue`
- Create: `desktop/frontend/src/views/SkillsView.vue`
- Create: `desktop/frontend/src/views/PluginsView.vue`
- Create: `desktop/frontend/src/views/AutomationsView.vue`
- Create: `desktop/frontend/src/views/SettingsView.vue`
- Create: `desktop/frontend/src/stores/app.js`
- Modify: `desktop/frontend/src/App.vue`

**Step 1: Define route structure**

- Add these routes:
  - `/chat`
  - `/chat/:conversationId`
  - `/search`
  - `/skills`
  - `/plugins`
  - `/automations`
  - `/settings`

**Step 2: Implement a Codex-style shell**

- Left sidebar with:
  - quick chat entry
  - search
  - skills
  - plugins
  - automations
  - project section
  - conversation section
  - settings
- Center content area rendered by router view.

**Step 3: Add placeholder pages**

- Non-chat pages should be visually complete but explicitly placeholder-only in P0.
- Use one reusable `EmptyPlaceholder` component with title, description, and status badge.

**Step 4: Test the shell**

- Add a lightweight router smoke test if setup time is small.
- Otherwise verify manually with `wails3 dev`.

**Step 5: Commit**

- Commit message: `feat: add desktop shell and placeholder views`

### Task 5: Wire settings, connection testing, and agent bootstrap

**Files:**
- Create: `desktop/frontend/src/stores/settings.js`
- Create: `desktop/frontend/src/stores/agents.js`
- Create: `desktop/frontend/src/services/wails/config.js`
- Create: `desktop/frontend/src/services/gateway/http.js`
- Create: `desktop/frontend/src/services/gateway/agents.js`
- Create: `desktop/frontend/src/tests/settings.spec.js`
- Modify: `desktop/frontend/src/views/SettingsView.vue`

**Step 1: Build a frontend wrapper for Wails bindings**

- Wrap generated Go binding calls so stores do not import raw generated files directly.
- Methods should cover:
  - load config
  - save config
  - choose directory
  - open config directory
  - get app info

**Step 2: Implement gateway HTTP primitives**

- Centralize:
  - base URL resolution
  - timeout
  - JSON response parsing
  - error normalization
- The error shape should preserve backend `code` values from [server/gateway/internal/controller/agent_controller.go](/E:/code/issueye/icoo_claw/server/gateway/internal/controller/agent_controller.go).

**Step 3: Implement agent discovery**

- `GET /v1/agents`
- Filter to enabled agents in the frontend.
- Persist the selected default agent id in local config.
- If no agents are available, show a blocking setup empty state in settings and chat.

**Step 4: Add connection testing**

- Test the gateway through:
  - `GET /health`
  - `GET /v1/agents`
- Surface friendly states:
  - reachable
  - auth/permission issue
  - no agents found
  - timeout
  - network failure

**Step 5: Test**

- Unit test settings store save/load normalization.
- Manually verify changing gateway URL and default agent updates the UI state.

**Step 6: Commit**

- Commit message: `feat: add settings and gateway bootstrap`

### Task 6: Implement project management and persistent sidebar sections

**Files:**
- Create: `desktop/frontend/src/stores/projects.js`
- Create: `desktop/frontend/src/components/project/ProjectSwitcher.vue`
- Modify: `desktop/frontend/src/components/chrome/AppSidebar.vue`
- Modify: `desktop/frontend/src/views/SettingsView.vue`

**Step 1: Add project data model**

- Each project needs:
  - `id`
  - `name`
  - `root`
  - `description`
  - `defaultAgentId`
  - `defaultModel`

**Step 2: Keep projects local-only in P0**

- Read and write through the TOML config service.
- Allow create, edit, delete, and current-project selection from the settings view.

**Step 3: Surface current project in the shell**

- Sidebar shows project list.
- Chat composer and chat header show the active project.
- If no project exists, keep the UI functional and show “No project selected”.

**Step 4: Commit**

- Commit message: `feat: add local project management`

### Task 7: Build conversation list and draft conversation flow

**Files:**
- Create: `desktop/frontend/src/stores/conversations.js`
- Create: `desktop/frontend/src/components/conversation/ConversationList.vue`
- Modify: `desktop/frontend/src/components/chrome/AppSidebar.vue`
- Modify: `desktop/frontend/src/views/ChatHomeView.vue`
- Create: `desktop/frontend/src/views/ChatConversationView.vue`

**Step 1: Implement conversation list queries**

- Call:
  - `GET /v1/conversations`
  - `GET /v1/conversations/:id/messages`
- Cache list and active conversation detail separately.

**Step 2: Add the draft conversation model**

- Keep a local unsaved conversation state for `/chat`.
- Do not call `POST /v1/conversations` until the first prompt is submitted.

**Step 3: Create remote conversations lazily**

- On first send:
  - derive the title from the first user prompt
  - call `POST /v1/conversations`
  - navigate to `/chat/:conversationId`
  - then start the first message flow

**Step 4: Title generation rules**

- Implement a utility that:
  - trims whitespace
  - collapses newlines
  - uses the first 20-30 visible characters
  - falls back to `New conversation`

**Step 5: Test**

- Unit test the title utility with:
  - short prompts
  - multiline prompts
  - empty strings
  - Chinese text

**Step 6: Commit**

- Commit message: `feat: add conversation list and draft flow`

### Task 8: Implement chat composer, message rendering, and WebSocket streaming

**Files:**
- Create: `desktop/frontend/src/stores/chat.js`
- Create: `desktop/frontend/src/components/chat/ChatComposer.vue`
- Create: `desktop/frontend/src/components/chat/ChatMessageList.vue`
- Create: `desktop/frontend/src/components/chat/ChatMessageItem.vue`
- Create: `desktop/frontend/src/components/chat/ChatEmptyState.vue`
- Create: `desktop/frontend/src/components/chat/ChatStatusBar.vue`
- Create: `desktop/frontend/src/services/gateway/ws.js`
- Create: `desktop/frontend/src/services/utils/title.js`
- Create: `desktop/frontend/src/tests/title.spec.js`
- Create: `desktop/frontend/src/tests/ws.spec.js`
- Modify: `desktop/frontend/src/views/ChatHomeView.vue`
- Modify: `desktop/frontend/src/views/ChatConversationView.vue`

**Step 1: Build the composer and message list**

- Enter sends.
- Shift+Enter inserts a newline.
- Disable send when:
  - prompt is empty
  - gateway not configured
  - no default agent available
  - an active request is already running

**Step 2: Render optimistic user messages**

- Insert the user message in the UI immediately.
- Add a pending assistant message shell before the stream begins.

**Step 3: Implement WebSocket chat transport**

- Open a WebSocket connection to `GET /v1/ws/chat`.
- Send `chat.start` payloads with:
  - `conversation_id`
  - `prompt`
  - `request_id`
  - optional metadata
- Append content on `message.delta`.
- Mark completion on `message.completed`.
- Treat `message.error` and socket close during an active run as request failures.

**Step 4: Support cancel and retry**

- Use WebSocket `chat.cancel` for in-flight cancellation.
- Retry should resend the last user prompt to the same conversation.

**Step 5: Reload persisted history after completion**

- After a stream ends successfully, refresh conversation messages from:
  - `GET /v1/conversations/:id/messages`
- This avoids frontend drift from stream parsing assumptions.

**Step 6: Test**

- Unit test WebSocket client behavior with:
  - connect success
  - delta accumulation
  - completion handling
  - malformed frame
  - socket close during active response
- Manual test against a live gateway instance.

**Step 7: Commit**

- Commit message: `feat: add websocket chat workflow`

### Task 9: Polish P0 states, placeholders, and error handling

**Files:**
- Modify: `desktop/frontend/src/stores/app.js`
- Modify: `desktop/frontend/src/stores/settings.js`
- Modify: `desktop/frontend/src/stores/conversations.js`
- Modify: `desktop/frontend/src/stores/chat.js`
- Modify: `desktop/frontend/src/components/chrome/AppSidebar.vue`
- Modify: `desktop/frontend/src/views/SearchView.vue`
- Modify: `desktop/frontend/src/views/SkillsView.vue`
- Modify: `desktop/frontend/src/views/PluginsView.vue`
- Modify: `desktop/frontend/src/views/AutomationsView.vue`

**Step 1: Standardize UI state**

- Every view/store should expose:
  - `isLoading`
  - `error`
  - `isEmpty`
- Use a consistent visual language for empty and error states.

**Step 2: Add placeholder copy for non-chat sections**

- Search: “P0 only indexes local conversations and projects later.”
- Skills/Plugins/Automations: “Reserved for gateway-backed integration.”

**Step 3: Handle gateway edge cases**

- Missing gateway URL
- Gateway offline
- No agents configured
- Conversation not found
- Session busy / socket failure

**Step 4: Commit**

- Commit message: `feat: polish desktop p0 states`

### Task 10: Final verification, packaging, and docs

**Files:**
- Modify: [README.md](/E:/code/issueye/icoo_claw/README.md)
- Create: `desktop/README.md`
- Modify: `docs/plans/2026-05-19-codex-desktop-chat-requirements.md`

**Step 1: Add run instructions**

- Document:
  - desktop prerequisites
  - `wails3` install
  - frontend install
  - local gateway startup order
  - `wails3 dev`
  - `wails3 build`

**Step 2: Verify end-to-end manually**

1. Start `gateway`.
2. Start `gateway`.
3. Ensure at least one agent exists in gateway.
4. Launch `desktop` with `wails3 dev`.
5. Save settings.
6. Load agents.
7. Send the first prompt from `/chat`.
8. Confirm the conversation title is generated from the first user prompt.
9. Confirm message flow runs over WebSocket.
10. Reload the app and confirm config persistence.

**Step 3: Run checks**

- Run: `go test ./desktop/...`
- Run: `cd desktop/frontend; npm run test`
- Run: `cd desktop/frontend; npm run build`
- Run: `cd desktop; wails3 build`

**Step 4: Commit**

- Commit message: `docs: add desktop app runbook`

## Delivery Sequence

Build in this order:

1. Task 0
1. Task 1
2. Task 2
3. Task 3
4. Task 4
5. Task 5
6. Task 6
7. Task 7
8. Task 8
9. Task 9
10. Task 10

Do not start desktop chat work before Wails3 validation and gateway WebSocket transport are in place.

## Risks To Watch

- Wails v3 is still in alpha, so keep the desktop module isolated and avoid spreading Wails-specific assumptions into shared backend code.
- The gateway currently requires `agent_id` on conversation creation, so the client must not create empty remote conversations before the first prompt unless it already has a valid default agent.
- WebSocket introduces connection lifecycle, heartbeat, and cancel semantics that must be designed explicitly.
- The gateway returns only basic message content today, so treat stream output as display-first and re-fetch final history after completion.

## Definition of Done

- Desktop app starts on Windows through `wails3 dev`.
- User can save gateway settings and choose a default agent.
- User can create a first conversation from `/chat` with lazy remote creation.
- Conversation title comes from the first user prompt.
- User can receive and read WebSocket streaming assistant output.
- Conversation list persists remotely through the existing gateway.
- Non-chat modules render as stable placeholders and do not break the shell.

Plan complete and saved to `docs/plans/2026-05-19-codex-desktop-chat-implementation-plan.md`. Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?

