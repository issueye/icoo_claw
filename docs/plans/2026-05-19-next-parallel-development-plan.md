# icoo_claw Next Parallel Development Plan

## Goal

Build on the desktop project-management and fake-stack E2E work by turning the new local project concept into useful chat context, replacing the Search placeholder with a local search MVP, and preparing the working tree for a clean deliverable.

## Baseline

The previous parallel pass delivered:

- Local desktop project settings: `projects`, `currentProjectId`, project CRUD, sidebar project switcher.
- Fake-stack startup: `session_store`, `gateway`, default fake agent, ready fake agent instance, Vite preview, Playwright smoke.
- Passing checks: frontend unit tests, frontend build, desktop Go tests, three backend service test suites, UI smoke.

Open context to respect:

- `go_pkg/redka/` is still untracked and should not be mixed into these feature changes.
- Generated project completion images under `docs/generated/` are optional documentation artifacts.
- Search, Skills, Plugins, and Automations are still placeholder modules; this plan only promotes Search to a small local MVP.

## Workstream A: Project Context In Chat

Owner scope:

- `desktop/frontend/src/stores/chat.js`
- `desktop/frontend/src/views/ChatHomeView.vue`
- `desktop/frontend/src/views/ChatConversationView.vue`
- `desktop/frontend/src/components/chat/*`
- `desktop/frontend/src/stores/projects.js`
- `desktop/frontend/src/tests/*` focused on project metadata/title/chat helpers
- Minimal docs if behavior changes need explanation

Deliverables:

- Chat composer/status area shows the active project name and root directory when available.
- New chat sends project metadata through the existing WebSocket `chat.start` payload, without requiring backend schema changes.
- Existing no-project flow remains fully usable.
- If a project has a default Agent/model field already available from settings, use it; otherwise keep current default Agent behavior.

Acceptance checks:

- Frontend unit tests cover metadata creation with and without an active project.
- `npm run test` and `npm run build` pass from `desktop/frontend`.
- UI smoke still passes.

## Workstream B: Local Search MVP

Owner scope:

- `desktop/frontend/src/router/index.js`
- `desktop/frontend/src/views/SearchView.vue` or replacement of the current search placeholder route
- `desktop/frontend/src/stores/search.js`
- `desktop/frontend/src/stores/conversations.js` only if a small read-only helper is needed
- `desktop/frontend/src/components/search/*`
- `desktop/frontend/src/tests/*` focused on search filtering

Deliverables:

- `/search` becomes a functional local search page.
- Search covers loaded conversation titles and cached messages.
- Empty, no-results, loading, and gateway-offline states are visible and stable.
- No new backend API is required.

Acceptance checks:

- Unit tests cover case-insensitive title/message filtering and empty query behavior.
- `npm run test` and `npm run build` pass from `desktop/frontend`.

## Workstream C: Delivery Cleanup And Verification

Owner scope:

- `.gitignore`
- `README.md`
- `desktop/README.md`
- `docs/plans/*runbook*.md`
- `desktop/frontend/test-results/*`
- package scripts and Playwright config only if needed for clean reruns

Deliverables:

- Decide whether Playwright `test-results` should be ignored, retained as status, or cleaned from this delivery.
- Add or update ignore rules for generated runtime/test artifacts without hiding source files.
- Update docs with the final command set for unit tests, build, and UI smoke.
- Produce a concise verification checklist for the final handoff.

Acceptance checks:

- `git status --short --untracked-files=all` clearly separates source changes from intentionally untracked external/vendor content.
- No generated runtime files under `.local/`, `dist/`, or Playwright transient folders are accidentally introduced.

## Integration Rules

- Workers must not revert or overwrite edits from other workers.
- Keep write scopes disjoint unless a small integration edit is unavoidable.
- Avoid broad UI redesign. This pass is about useful behavior, not visual overhaul.
- Keep backend API unchanged unless a bug prevents the existing frontend metadata path from working.
- Main integrator runs final checks after workers finish.

## Final Verification Target

Run these before calling the phase complete:

```powershell
Push-Location .\desktop\frontend
npm run test
npm run build
Pop-Location

Push-Location .\desktop
go test ./...
Pop-Location

Push-Location .\server\gateway
go test ./...
Pop-Location

Push-Location .\server\claw
go test ./...
Pop-Location

Push-Location .\server\session_store
go test ./...
Pop-Location

.\scripts\dev\run-ui-smoke.ps1
```

