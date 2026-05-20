# @icoo-claw/d-page

Vue 3 dynamic page renderer package for JSON/Card Schema driven UI.

The package is intentionally built as a reusable rendering kernel, not as a desktop-only feature. The desktop chat client can later consume it through message metadata, while host applications remain responsible for business actions and permissions.

## Install Locally

From a consuming package during early development:

```json
{
  "dependencies": {
    "@icoo-claw/d-page": "file:../../d_page"
  }
}
```

## Commands

```powershell
npm install
npm run demo
npm run test
npm run build
```

`npm run demo` starts a local playground at [http://127.0.0.1:9360](http://127.0.0.1:9360). It lets you switch between the bundled schemas and inspect runtime state, data, host events, and adapter calls.

## Basic Usage

```js
import { DPageRenderer } from '@icoo-claw/d-page'
import '@icoo-claw/d-page/style.css'
```

```vue
<DPageRenderer :schema="schema" :context="context" />
```

`DPageRenderer` can create its own runtime from the schema. Use an explicit runtime when the host needs to inject actions, adapters, initial state, or telemetry.

```js
import { createDPageRuntime } from '@icoo-claw/d-page'

const runtime = createDPageRuntime({
  schema,
  context: { messageId: 'msg_1' },
  actions: {
    copyToComposer(action, runtime, eventContext) {
      const payload = runtime.resolveBinding(action.payload || {}, eventContext)
      return { ok: true, type: action.type, value: payload }
    },
  },
  adapters: {
    copyText: (text) => navigator.clipboard.writeText(text),
    openUrl: (url, options) => window.open(url, options?.target || '_blank'),
  },
})
```

```vue
<DPageRenderer :schema="schema" :runtime="runtime" />
```

## Public API

- `DPageRenderer`: renders a full schema from `root`.
- `DCardRenderer`: renders one card and its nested `children` or `slots`.
- `createDPageRuntime`: creates schema, state, data, context and action runtime.
- `createComponentRegistry`: lets hosts register or replace renderable components.
- `createActionRegistry`: lets hosts register safe action handlers.
- `defaultComponents`: built-in `text`, `heading`, `button`, `input`, `textarea`, `select`, `checkbox`, `switch`, `cardSurface`, `table`, `alert`, `stat`, `list`, `tag`, `divider`, `image`.
- `defaultActions`: built-in `setState`, `emit`, `copyText`, `openUrl`, `chain`.
- `normalizeSchema`, `validateSchema`, `resolveBinding`, `executeAction`: runtime utilities.

## Example Schemas

The package includes MVP examples under `src/schemas/examples`:

- `chat-tool-result.json`: a chat-friendly result card with copy and detail actions.
- `simple-form.json`: an input and button flow backed by `setState`.
- `table-card.json`: a table bound to `data.rows` with row selection.
- `live-input-preview.json`: a live editing demo using input, select, textarea, alert, stat and list.
- `component-gallery.json`: a compact gallery for tag, divider, checkbox, switch and image fallbacks.

## Built-in Components

The current MVP component set includes:

- Display: `text`, `heading`, `alert`, `stat`, `list`, `cardSurface`, `tag`, `divider`, `image`.
- Controls: `button`, `input`, `textarea`, `select`, `checkbox`, `switch`.
- Data: `table`.

Controls emit interaction events such as `input`, `change`, `click`, and `rowSelect`. They do not mutate global state directly; schema actions decide how events update runtime state. Image and display components provide empty or failure fallback text through props such as `emptyText` and `errorText`.

The schema entry point is always `root`, and every rendered node is a card:

```json
{
  "schemaVersion": "0.1.0",
  "state": { "name": "" },
  "actions": {},
  "root": {
    "id": "root",
    "type": "card",
    "kind": "page",
    "children": []
  }
}
```

Bindings are intentionally limited to safe path reads:

```json
{
  "text": "Hello {{ state.name }}"
}
```

## Chat Integration Direction

Chat should not parse dynamic schemas itself. When mature, chat can render messages with:

```json
{
  "metadata": {
    "render_type": "d_page",
    "d_page_schema": {}
  }
}
```

The host chat app should inject actions such as `copyToComposer`, `sendChatPrompt`, and `saveArtifact` through runtime options. This package must not import desktop stores directly.

Default actions do not include arbitrary network requests. Hosts should inject request-like behavior only when they can enforce their own permission and audit rules.
