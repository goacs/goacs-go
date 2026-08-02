# AGENTS.md

Guidance for AI coding agents (and human contributors) working in this repository. Read this before making changes — it captures conventions and gotchas that aren't obvious from the code alone.

## What this project is

GoACS is a TR-069/CWMP Auto Configuration Server, written in Go, with a Vue 3 admin panel in `frontend/`. It manages CPE devices (routers/ONTs/etc.) over the CWMP protocol: reading/writing parameters, queuing tasks (reboot, firmware upload, add/delete object), and applying provisioning rules automatically as devices check in.

It is a from-scratch Go rewrite of an existing Laravel/Vue 2 panel (`goacs-php` in a sibling repo). That repo is a useful reference for intended behavior, but this codebase has its own conventions and its own REST API contract — do not assume the two are wire-compatible.

## Repo layout

```
acs/            CWMP protocol engine (see below)
http/           Gin server: REST API for the frontend, controllers, middleware
lib/            Small helpers (env, files, in-memory cache)
models/         Domain structs (cpe, templates, provisions, tasks, user, log, fault)
repository/     Data access layer (mysql/*.go), one repository per domain
contrib/        SQL migrations (contrib/database/*.sql, run in filename order) + MariaDB conf
frontend/       Vue 3 admin panel (separate npm project, own README)
testrequests/   .http files for manually exercising the API (VS Code REST Client / JetBrains HTTP Client)
_doc/           Reference TR-069/TR-106/TR-181 spec PDFs
```

### The ACS engine (`acs/`)

This is the core protocol logic and the highest-risk part of the codebase to change.

- `acs/acssession.go` — in-memory session store, keyed by CWMP session ID (not HTTP session/cookie — CPEs don't reliably support cookies).
- `acs/logic/dispatcher.go` — single entry point (`HandleCPERequest`) for the `/acs` endpoint. Parses the SOAP envelope, dispatches to a processor per RPC type, then hands off to the task runner.
- `acs/logic/taskrunner.go` — drains the session's task queue for one HTTP round-trip (CWMP is a slow, poll-driven protocol: one task per request/response pair, not fire-and-forget).
- `acs/logic/provision.go` — matches provisioning rules (`models/provisions`) against the current CWMP event + request type + parameter conditions, queues `RunScript` tasks for matches.
- `acs/methods/*.go` — one file per RPC family (Inform, GetParameterValues, Fault, ...), each implementing the state transitions for that RPC.
- `acs/scripts/` — Lua sandbox (gopher-lua) that runs provisioning scripts against the CPE's parameter tree.
- `acs/types/` — CWMP XML wire structs, TR-069 flag semantics, parameter path helpers.

If you touch anything under `acs/`, know that there is no real CPE simulator in this repo to validate against — reason carefully through session state transitions, and prefer additive changes (new cache-gated hooks) over altering existing control flow. See "Known gotchas" below for two examples of exactly this kind of careful, additive change.

### REST API (`http/`)

- Routes are all registered in one place: `http/api_routes.go`. Look there first to find/add an endpoint.
- Controllers are plain functions `func Xxx(ctx *gin.Context)` in `http/controllers/<domain>.go` — no DI container, no controller structs. Repositories are constructed inline: `mysql.NewCPERepository(repository.GetConnection())`.
- Request DTOs are structs defined next to the controller that uses them, validated via `request.NewApiValidator(ctx, dto)` + `validator.Validate()` (go-playground/validator under the hood).
- **Response envelope** (this is NOT the same as the original Laravel API — don't assume Laravel conventions):
  - `response.ResponseData(ctx, data)` → `{"message": "Ok", "data": <data>}`
  - `response.ResponsePaginatior(ctx, repository.NewPaginatorResponse(...))` → `{"page", "per_page", "filter", "sort", "next_page", "prev_page", "total", "data"}`
  - `response.ResponseValidationErrors(ctx, validator)` → 422, `{"message": "Validation error", "data": {field: message}}`
  - `response.Response500(ctx, message, data)` / `response.ResponseError(ctx, code, message, data)` for other errors
  - A few endpoints predate this convention and return bare bodies instead (e.g. `PUT /device/:uuid/parameters` returns a literal `204 ""`, `POST /user/create` writes the raw `User` JSON with no envelope). Check the controller before assuming the envelope shape — the frontend's `device.api.ts`/`user.api.ts` have comments at each spot where this bites.
- **Pagination**: `repository.PaginatorRequestFromContext(ctx)` reads `page`, `per_page`, `filter[key]=value`, `sort[key]=asc|desc` from the query string. Sortable/filterable columns are allow-listed per repository (e.g. `cpeSortableColumns` in `cperepository.go`) to avoid arbitrary-column SQL and keep sorting meaningful — add new columns to the allow-list when you add them to a `SELECT *`.
- Auth: JWT (`dgrijalva/jwt-go`), `http/middleware/jwt/jwt.go`. On success it sets `user_uuid` in the Gin context (`controllers.CurrentUserUUID(ctx)` reads it back) — needed for self-delete guards and token refresh.
- CORS is configured in `http/server.go` from `CORS_ALLOWED_ORIGINS` (comma-separated), since the frontend is a separately-deployed app, not server-rendered.
- Real-time: Socket.IO (`http/socket.go`), see "Socket.IO" below.

### Repository pattern (`repository/mysql/`)

- One struct per domain, holding `db *sqlx.DB`, constructed via `NewXRepository(connection)`.
- Query building is a mix of raw SQL (`sqlx`) and `goqu` (github.com/doug-martin/goqu/v9) query builder — follow whatever the file you're editing already uses.
- `repository.GetConnection()` returns the global `*sqlx.DB` (set up in `main.go` via `repository.InitConnection()`). It panics if called before `InitConnection()` — see `repository.HasConnection()` for code paths (like conversation logging) that also run in unit tests without a DB and need to check first.
- `repository/hooks.go` has `repository.OnLogSaved func(interface{})`, set by `http/server.go` at startup to `http.EmitDeviceLogged`. This exists so `repository/mysql` can notify Socket.IO clients without importing `http` (which would create an import cycle, since `http` already imports `repository/mysql`). If you add another cross-cutting notification, follow this same hook pattern rather than reaching for a new import.

### Socket.IO

`go-socket.io` v1.4.4 (server, in `http/socket.go`) only speaks **Engine.IO protocol v3**. Any client must be pinned accordingly:
- Frontend: `socket.io-client` is deliberately pinned to `^2.x` in `frontend/package.json` — v3/v4 dropped EIO3 support and simply cannot connect (fails with a 403→502 handshake loop, not a clean error). Do not "upgrade" this dependency without re-verifying the handshake against the running Go server.
- `CheckOrigin` is set explicitly on both the polling and websocket transports in `NewSocketIO`, reusing the same `CORS_ALLOWED_ORIGINS` allow-list as the REST API — gorilla/websocket's default `CheckOrigin` rejects any cross-origin upgrade, which every request from the separately-deployed frontend is.
- Auth model: there's no per-channel auth like Pusher's private channels. The client sends its JWT in the `join-device`/`leave-device` event payload; the server validates it in the handler before joining the `device.<uuid>` room. See `validSocketToken` in `http/socket.go`.
- Event emitted on new logs: `device.logged`, broadcast to room `device.<cpe_uuid>`, wired via the `OnLogSaved` hook above.

## Known gotchas (found while building the frontend against this API — read before you hit them again)

1. **sqlx can't auto-map a named (non-embedded) nested struct field.** `types.ParameterValueStruct.ValueStruct` (holding `Value`/`Type`) has no `db:"..."` tag on the `ValueStruct` field itself, so plain `sqlx.Select`/`StructScan` silently leaves it zeroed — the SQL columns `value`/`type` never make it in. The established workaround (see `parametersRowsParser` in `cperepository.go`, and the same pattern in `templaterepository.go`'s `ListTemplateParameters`/`GetPrioritizedParametersForCPE`) is: `Queryx` the rows, then for each row call both `rows.StructScan(&dest)` *and* `rows.MapScan(mapScan)`, then manually copy `mapScan["value"]`/`mapScan["type"]` (as `[]byte`) into `dest.ValueStruct`. If you add a new query returning `ParameterValueStruct`-shaped rows, use this pattern, not a plain `Select`.
2. **`ScriptList.Value()` has a pointer receiver.** `provisions.ScriptList` (a `[]string` alias) implements `driver.Valuer` on `*ScriptList`, not `ScriptList`. Passing a value (`p.Script`) as a query arg to `db.Exec`/`tx.Exec` does *not* satisfy `driver.Valuer` and fails silently into a generic-slice conversion error — always pass `&p.Script`. (This does not affect `sqlx.Select`/`Get` reading the column back via `Scan`, which sqlx already calls on a pointer.)
3. **JWT parsing can return a nil `*jwt.Token` alongside a non-nil error** (e.g. malformed base64). Always check `err != nil || token == nil` before touching `token.Valid` — see `http/middleware/jwt/jwt.go`.
4. Test carefully after touching `acs/` or `repository` — restarting `go run main.go` picks up code changes, but check for stray already-running `go run`/compiled binaries on port 8085 first (`ss -tlnp | grep 8085`), otherwise you'll be testing against stale code without realizing it.

## Frontend (`frontend/`)

Separate Vite + Vue 3 + TypeScript + PrimeVue + Pinia project, deployed independently from the Go backend (hence the CORS setup above). See `frontend/README.md` for the file-by-file layout; in short:

- `src/api/http.ts` — single axios instance; JWT is read from a module-level `authState` object (not imported from the Pinia store, to avoid a store↔http circular import); 401 triggers a configurable `unauthorizedHandler`; 422 is normalized into `ApiValidationError`.
- `src/api/endpoints/*.api.ts` — one file per backend domain, thin wrappers around `http` calls. Check the comments in `device.api.ts` and `user.api.ts` for the handful of endpoints that don't use the standard `{message,data}` envelope (see gotcha above).
- `src/composables/useServerTable.ts` — drives PrimeVue `DataTable` against the Go paginator shape. Sorting is wired but only takes effect on repositories whose allow-list includes the column (see backend section above).
- `src/stores/*.store.ts` — Pinia, Composition-API "setup store" style throughout.
- `src/sockets/socket.ts` + `src/composables/useDeviceLogsSocket.ts` — Socket.IO client wiring, paired with the backend section above.

## Dev workflow

```bash
cp .env.example .env            # adjust MYSQL_PORT etc. if 3306 is already taken locally
docker compose up -d goacs-db   # MariaDB only - schema is NOT auto-applied, see below
go run main.go migrate          # applies contrib/database/*.sql, tracked in schema_migrations
go run main.go                  # backend on :8085 (HTTP_PORT in .env)

cd frontend
npm install
npm run dev                     # frontend on :5173, talks to the backend via VITE_API_URL
```

`contrib/database/*.sql` is applied **only** via `go run main.go migrate`, never automatically -
the MariaDB service no longer bulk-loads that directory on container init
(`docker-entrypoint-initdb.d` was deliberately removed from `docker-compose.yml`). Every
environment, fresh or existing, goes through this same tracked path (`schema_migrations`
table, one row per applied filename - same idea as Laravel's migrations table). Run it
again any time an update adds a new numbered file there; already-applied files are
skipped automatically. See the doc comment on `repository.RunMigrations` for why this
doesn't try to guess "already applied" from SQL error codes (that only works for
CREATE-TABLE-style DDL and silently breaks for column modifications or plain inserts) -
any error during a normal `migrate` run is a real error, full stop.

For a database that already has schema from before this tool existed (e.g. was
bootstrapped by the old `docker-entrypoint-initdb.d` mechanism), run **once**, naming
exactly the files that database already has (not the whole directory - an existing
deployment is rarely caught up to the newest file, and baselining one that hasn't
actually run yet would silently skip it forever):

```bash
go run main.go migrate --baseline 01_initial.sql 02_provisioning.sql 03_logs_and_settings.sql
```

then use plain `go run main.go migrate` from then on to apply anything newer for real.

Backend checks before committing:
```bash
gofmt -l .          # should print nothing (pre-existing exception: acs/types/xmlstruct.go)
go build ./...
go test ./...        # acs/types has two known pre-existing failures (TestParser, TestIsObjectParameter) unrelated to API/frontend work
```

Frontend checks before committing:
```bash
npx vue-tsc -b --noEmit
npm run build
```

There's no CPE simulator in this repo — anything touching `/acs` request handling should be reasoned through carefully and, where possible, tested against the paginated `logs`/`faults` API afterward to confirm session state changed as expected.
