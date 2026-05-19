# icoo_claw Parallel Development Plan

## Goal

Move the project from a chat MVP prototype toward a repeatable local development flow by closing two high-impact gaps in parallel:

1. Local project management in the desktop client.
2. Stable local startup and end-to-end chat verification.

## Current Baseline

- Backend unit tests pass for `server/gateway`, `server/claw`, and `server/session_store`.
- Desktop frontend unit tests and production build pass after `npm ci`.
- Desktop Go tests pass after `desktop/frontend/dist` exists.
- Playwright chat flow has an existing failure mode when the gateway is not running or not bootstrapped.
- Search, Skills, Plugins, and Automations remain placeholder modules for P0.

## Parallel Workstreams

### Workstream A: Local Project Management

Owner scope:

- `desktop/internal/config/config.go`
- `desktop/internal/config/store_test.go`
- `desktop/frontend/src/services/settings/schema.js`
- `desktop/frontend/src/stores/projects.js`
- `desktop/frontend/src/components/project/ProjectSwitcher.vue`
- `desktop/frontend/src/views/SettingsView.vue`
- `desktop/frontend/src/components/chrome/AppSidebar.vue`
- `desktop/frontend/src/layouts/AppShell.vue` only if shell display needs it
- focused frontend tests as needed

Deliverables:

- TOML and frontend settings schema support a local `projects` list and `currentProjectId`.
- Existing `workspace.rootDir` remains backward compatible.
- Settings page can create, edit, delete, and select the current project.
- Sidebar or shell shows the active project state.
- Chat remains usable when no project exists.

Acceptance checks:

- `go test ./...` from `desktop` after frontend build.
- `npm run test` from `desktop/frontend`.
- Manual smoke: create project, select it, reload settings, confirm it persists.

### Workstream B: Local Startup And E2E Stability

Owner scope:

- `scripts/`
- `desktop/frontend/e2e/`
- `desktop/frontend/playwright.config.js`
- `desktop/frontend/package.json`
- `README.md`
- `desktop/README.md`
- supporting runbook docs under `docs/plans/`

Deliverables:

- A repeatable local command sequence, preferably scripted, that starts `session_store`, `gateway`, and the fake-agent-ready environment for E2E.
- Playwright chat flow waits for durable UI or network readiness instead of fragile copy text.
- Documentation describes install, build, service startup, preview, and E2E order.

Acceptance checks:

- `npm run test:e2e` from `desktop/frontend` succeeds when the documented local environment is running.
- The script can be rerun without corrupting local DB or leaving confusing state.
- Existing unit tests and frontend build still pass.

## Integration Order

1. Merge Workstream B first if it only touches scripts and test docs, because it improves verification for the rest.
2. Merge Workstream A next, then run frontend unit tests and build.
3. Run backend service tests after both workstreams land.
4. Run Playwright chat flow against the scripted local environment.
5. Update this plan with final verification notes or link to the new runbook.

## Non-Goals For This Parallel Pass

- Do not implement backend project APIs.
- Do not turn Search, Skills, Plugins, or Automations into real product features.
- Do not replace the current Gateway-to-Claw streaming transport.
- Do not add a full Agent create/edit desktop UI.

## Risks

- Project settings schema changes can break existing browser localStorage or TOML settings if defaults are not merged carefully.
- E2E can remain flaky if service startup races with frontend boot.
- The untracked `go_pkg/` directory should not be mixed into these feature changes unless explicitly decided.

