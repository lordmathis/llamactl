# Spec: Custom Configurable Backend

**Version:** 1.0 (MVP)
**Status:** Approved design, ready for implementation

## Overview

Add a `custom` backend type that runs any user-supplied command as a managed instance. Instead of encoding another engine's flag grammar in Go (as done for llama.cpp, vLLM, and MLX), the operator defines **named custom backends** in the config file — each is a command plus optional global args/env — and instances reference one by name. There can be any number of them. llamactl provides lifecycle, port allocation, health checking, and the OpenAI-compatible reverse proxy it already has. The motivating use case is [Splash](https://github.com/incoai/splash), but the feature is engine-agnostic: any server speaking OpenAI `/v1/*` HTTP can be managed this way.

## Tech Stack

- Go backend (existing patterns in `pkg/backends`, `pkg/config`, `pkg/instance`)
- React + TypeScript webui (zod schemas, `InstanceDialog`)
- No new dependencies

## Core Principles

1. **No flag grammar.** The user already knows their command. Options are a raw arg list plus connection metadata — no typed flags, no reflection builder, no CLI parser.
2. **Contract, not detection.** The custom server must serve OpenAI-compatible `/v1/*` endpoints and an HTTP health endpoint. Documented, never probed.
3. **Fail fast at creation.** Misconfiguration (missing name, unknown name, missing command, missing `{port}`) is rejected when the instance is created, not at start time.
4. **Existing machinery comes free.** Port allocation, idle timeout, LRU eviction, auto-restart, Docker toggle, multi-node, persistence — all backend-agnostic and untouched.

## The User-Facing Contract

Configuration flow:

1. The operator defines one or more **named custom backends** statically in the config file, under `backends.custom.<name>`: the command and optionally global args/env/docker settings. The per-instance `command_override` (already supported for all backends) also works.
2. When creating an instance, the backend dropdown lists the built-in backends **plus every configured custom backend by its name**. Selecting one creates a `custom`-type instance whose options reference that name. Per-instance options are the same fields other backends expose (args, model, host/port, health path) — nothing else; there is no custom-specific UI beyond the args input.

A custom backend server must:

1. Bind a TCP port (the port is communicated via the `{port}` placeholder in args).
2. Respond with HTTP 200 on a configurable health path (default `/health`).
3. Serve OpenAI-compatible endpoints (`/v1/chat/completions`, etc.) — llamactl proxies requests verbatim.

Config example (Splash):

```yaml
backends:
  custom:
    splash:
      command: splash
      args: ["serve"]
    # ...any number of additional named custom backends
```

Instance creation:

```json
{
  "name": "splash-qwen",
  "backend_options": {
    "backend_type": "custom",
    "backend_options": {
      "name": "splash",
      "model": "incoai/Qwen3.8-27B-Splash",
      "args": ["--no-webui", "--port", "{port}"],
      "health_path": "/ready"
    }
  }
}
```

Resulting process: `splash serve --no-webui --port 8456` (config args first, then instance args).

Wire-format decision: `backend_type` stays the closed value `"custom"` and the configured backend is referenced by the `name` field inside `backend_options`. This keeps `backend_type` a closed enum (swagger, webui types, and typo detection at the JSON boundary all stay boring); the friendly name is presentation, surfaced by the UI.

---

## Phase 1: Go Core

### Backend type

**Create `pkg/backends/custom.go`** — `CustomServerOptions`:

```go
type CustomServerOptions struct {
    Name       string   `json:"name,omitempty"`        // references backends.custom.<name> in config
    Model      string   `json:"model,omitempty"`        // proxy model rewriting (GetModel)
    Host       string   `json:"host,omitempty"`         // host the proxy/health check connects to
    Port       int      `json:"port,omitempty"`         // written by port allocator
    Args       []string `json:"args,omitempty"`         // instance-level args, appended after config args
    HealthPath string   `json:"health_path,omitempty"`  // default "/health"
}
```

Implementation notes:

- **Do NOT use the reflection builder** (`BuildCommandArgs(o, multipleFlags)` from `builder.go`). That machinery converts struct fields to `--kebab-case` flags; custom args are literal strings. `BuildCommandArgs()` returns `o.Args` with placeholder substitution only.
- Placeholder substitution: replace `{port}` with `o.Port` and `{model}` with `o.Model` in every element of `o.Args`. These two are the complete templating language.
- `GetHealthPath()`: return `o.HealthPath`; normalize by prepending `/` if the value is non-empty and lacks a leading slash.
- No custom `UnmarshalJSON` — unknown fields in `backend_options` are silently ignored (standard `encoding/json` behavior). There is no `ExtraArgs` field; args are already free-form.
- `Validate()`: reject port outside 0–65535; error if `Name` is empty; error if any arg contains `{model}` but `Model` is empty. The name's *existence in config* is checked at creation time (config is not available here).
- `ParseCommand(command string)`: return an error — `custom` has no grammar to parse.
- `BuildDockerArgs()`: same output as `BuildCommandArgs()` (docker mode composes `docker <docker args> <image> <backend args>`, which works for custom as-is).

**Modify `pkg/backends/backend.go`:**

- Add `BackendTypeCustom BackendType = "custom"` to the constants block (`:13-18`).
- Add a `GetHealthPath() string` method to the `backend` interface (`:20-29`). Implement on all four backends: llama.cpp/MLX/vLLM return `"/health"`, custom returns the configured value. Add an `Options.GetHealthPath()` wrapper following the `GetHost()` pattern (`:268-274`), defaulting to `"/health"` when the backend is nil.
- Register in `backendConstructors` (`:31-35`), add the typed field `CustomServerOptions *CustomServerOptions` to `Options` (`:42-44`), and add cases in `setBackendOptions` (`:112-121`) and `getBackend` (`:137-148`).
- `getBackendSettings` (`:123-134`) — the custom case resolves the **named entry**: `backendConfig.Custom[o.CustomServerOptions.Name]`. Guard both failure modes to avoid a nil-deref in `GetCommand`: if `CustomServerOptions` is nil or the name is absent from the map, return a zero-value `&BackendSettings{}`. (Creation-time validation is the real gate; this guard only covers config edits made after an instance was created.)
- **Docker: allowed, not default.** Do not add a custom-backend rejection in `isDockerEnabled` (`:151-169`) the way MLX is rejected — the generic docker path works for custom servers, and each named backend can carry its own `Docker` section. Config entries simply default to no `Docker` section (nil → disabled).

### Health path plumbing

**Modify `pkg/instance/process.go:282`** — the only place `/health` is hardcoded:

```go
healthURL := fmt.Sprintf("http://%s:%d%s", host, port, paths.Normalize(p.instance.options.GetHealthPath()))
```

Use `Options.GetHealthPath()` (added above). Keep everything else in `waitForHealthy` (`:267-325`) unchanged.

### Configuration

- `pkg/config/types.go:24-29` — add `Custom map[string]BackendSettings \`yaml:"custom,omitempty" json:"custom,omitempty"\`` to `BackendConfig`.
- `pkg/config/defaults.go:22-56` — add `Custom: map[string]BackendSettings{}` (empty, non-nil map).
- `pkg/config/env.go` — **no changes.** The existing env pattern is one flat block per backend; with an arbitrary number of named entries there is no sensible `LLAMACTL_CUSTOM_*` naming scheme. Custom backends are config-file only (see Out of Scope).

### Creation-time validation

**Modify `pkg/manager/operations.go`** (`CreateInstance`, currently validates at `:59`): add a small check when `BackendType == custom`:

- `Name` must be defined in `cfg.Backends.Custom` — error "custom backend '<name>' is not defined in config" (also covers the empty-name case).
- Merged args (the named entry's config `Args` + instance `Args`) must contain at least one element with `{port}` — otherwise the server cannot learn its port and the proxy will target the wrong one. Error message should name the fix ("add `{port}` to custom backend args").
- A command must resolve: the named entry's `Command` (or the per-instance `command_override`) must be non-empty.

One helper function, all three checks, called from `CreateInstance` where config is in scope. `Validate()` on the options struct cannot perform these checks (no access to config or command override).

### Tests

Create `pkg/backends/custom_test.go` (table-driven, `package backends_test`, following `mlx_test.go`):

- `BuildCommandArgs`: literal passthrough, `{port}`/`{model}` substitution, mixed placeholders, no placeholders.
- `GetHealthPath`: default empty → handled at wrapper, normalization of `ready` → `/ready`, leading-slash passthrough.
- `Validate`: empty name, port bounds, `{model}` without model set, happy path.
- `ParseCommand` returns error.
- `Options` JSON round-trip: marshal → unmarshal preserves `backend_type: custom` and all fields including `name` (constructor map dispatch).
- `getBackendSettings`/`GetCommand` with a populated `Custom` map: name found (command/args resolve from the named entry), name missing (zero-value settings, no panic), nil options (no panic).

Extend instance-level tests (`pkg/instance/instance_test.go` patterns): `waitForHealthy` against an `httptest` server that returns 200 on a custom path and 404 on `/health`, proving the path flows through.

Config tests: YAML unmarshal of multiple named custom entries into `BackendConfig.Custom`; defaults produce an empty non-nil map.

**Verify with:** `go test ./... -race` and `go vet ./...`.

### Deliverable

A user can create, start, stop, and proxy to a custom-backend instance entirely via the REST API (curl), with configurable health path and automatic port injection — e.g. a locally installed `splash` or any OpenAI-compatible server.

---

## Phase 2: WebUI

- `webui/src/types/instance.ts:5-10` — add `CUSTOM: 'custom'` to the `BackendType` map.
- `webui/src/schemas/backends/custom.ts` — hand-written zod schema (do not extend zodgen; it parses `llama.go`'s AST): `name` (string, required — references a configured custom backend), `model` (string, optional), `host` (string, optional), `port` (number, optional), `args` (array of strings — render as a textarea, one arg per line, to preserve args containing spaces), `health_path` (string, optional, placeholder "/health"). Re-export from `webui/src/schemas/backends/index.ts`.
- `webui/src/components/InstanceDialog.tsx` — the backend dropdown lists the built-in backends plus one entry per configured custom backend, labeled with its configured name (source: the config already exposed to the settings UI). Selecting a custom entry sets `backend_type: custom` with `name` filled in. The per-instance form shows exactly the fields defined in the options struct (model, host, port, args, health path) — no new UI concepts beyond what other backends already render. Hide the "parse command" affordance for custom (no endpoint exists).
- `webui/src/types/config.ts` — the custom block becomes a record type (`Record<string, BackendSettings>`-shaped); `webui/src/components/settings/BackendTab.tsx` — render each named custom backend's settings (command, args, env) as its own block, following the existing per-backend block pattern.
- `BackendBadge.tsx` — display the configured custom backend's name (from instance options), not the literal string "custom".
- Update zod/instance schema tests under `webui/src/components/__tests__` and schema tests if they enumerate backend types.

**Verify with:** `npm run check`, `npm run type-check`, `npm run test:run` in `webui/`.

### Deliverable

A user can select a configured custom backend by name in the instance dialog and manage it with the same UX as built-in backends.

---

## Phase 3: Docs

- `docs/configuration.md` — new "Custom backends" section: the contract (OpenAI `/v1/*`, health endpoint returning 200, `{port}` placeholder), config reference (`backends.custom.<name>.*` — command required, args/env/docker optional; any number of entries), instance creation via the name, and the arg-merging order (config args, then instance args). Include a generic operational note: servers that download models or run long setup on first start may exceed `instances.on_demand_start_timeout` — raise it or pre-warm caches. Keep it engine-agnostic; no specific servers are named.
- `README.md` — extend the supported-backends sentence generically (e.g. "or any OpenAI-compatible server via the custom backend").
- Regenerate swagger annotations if instance/options schemas changed in swag's view: `swag init -g cmd/server/main.go` (per `CONTRIBUTING.md`).

### Deliverable

A user can wire up any OpenAI-compatible server by copying documented config, and the feature is officially supported.

---

## Out of Scope (explicitly)

- Env-var overrides (`LLAMACTL_CUSTOM_*`) for named custom backends — config-file only; revisit if a concrete need appears.
- Typed flag schemas, `parse-command` support, or a zodgen extension for custom backends.
- Endpoint/capability auto-discovery or health response body parsing (status code 200 only, as today).
- `{host}` or env-value templating; arbitrary placeholder variables beyond `{port}` and `{model}`.
- Per-instance health timeout (the existing global `on_demand_start_timeout` covers the on-demand path; revisit only with a concrete complaint).
- Per-custom-backend config defaults for health path or model (instance options only; add config defaults if duplication hurts in practice).
- A catalog/recipe system of preconfigured custom backends. YAML is the recipe.
- Native (bespoke) Splash backend — superseded by this feature.

## Gotchas

- Arg merge order is config-args-then-instance-args (`backend.go:213-217`); document it, don't change it.
- The OpenAI proxy rewrites the request `model` to `GetModel()` only when the requested model has no `/` (`handlers_openai.go:216-224`). Splash package IDs (`owner/repo`) pass through untouched; plain model names resolve to the configured `Model` — which is exactly why `Model` exists on the options struct.
- `{port}` substitution reads `o.Port` at build time, i.e. after `SetPort` — same lifecycle as the flag-based backends. A user-specified fixed port (non-zero `Port` reserved via `allocateSpecific`) flows through the placeholder identically.
- Commands must exist on the executing node — relevant for multi-node setups; note in docs, no code change.
- Renaming or deleting a config entry after instances reference it leaves those instances resolvable but unrunnable — the `getBackendSettings` zero-value guard prevents a panic, and start fails with an empty command. Acceptable; the error should be legible, not papered over.
- `getBackendSettings` returning nil for an unregistered type would make `GetCommand` nil-deref; the constructor map + switches registration in Phase 1 prevents this for `custom` — just don't miss one of the switches.
