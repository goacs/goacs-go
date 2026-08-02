# testscenarios

Black-box scenario tests that drive a real `goacs-go` server with the
`goacs-client` CPE simulator, exercising the same CWMP wire protocol and REST
API a real deployment would, then asserting results through goacs-go's own
REST API. See `AGENTS.md` at the repo root for the codebase's general
conventions - this covers what's specific to this test package.

These tests cover four areas that had no end-to-end coverage before:

- every Lua function scripts can call (`acs/scripts/`) - `scripts_test.go`
- template application, including the priority-override and gating
  nuances - `templates_test.go`
- provisioning from already-stored parameters without any script logic, and
  the admin panel's "provision now" trigger - `provision_stored_params_test.go`
- provision rule matching by event, request type, and parameter conditions
  (vendor/firmware/product class) - `provision_matching_test.go`

## Running

They're gated behind the `scenario` build tag so `go test ./...` (the check
in the repo root `AGENTS.md`) never touches them and never needs a database:

```bash
docker compose up -d goacs-db     # if not already running
go test -tags scenario ./testscenarios/... -v
```

Requirements:

- `.env` must exist at the repo root (`cp .env.example .env` if you haven't
  already) - the harness reads `MYSQL_HOST`/`MYSQL_PORT`/`MYSQL_ROOT_PASSWORD`
  from it to reach the same MariaDB `docker compose up -d goacs-db` starts.
- The sibling `goacs-client` checkout must be reachable - by default the
  harness assumes it's at `../goacs-client` next to this repo; override with
  `GOACS_CLIENT_DIR=/path/to/goacs-client` if yours lives elsewhere.
- Both `goacs-go` and `goacs-client` must `go build` cleanly - the harness
  compiles a fresh binary of each once per test run (`testscenarios/harness`)
  and reuses it across every scenario in that run.

Each top-level `Test...` function starts its own goacs-go server against a
freshly created, uniquely-named database (`goacs_scenario_<n>`), migrated
from scratch, and tears both down in `t.Cleanup`. Subtests within one
top-level test share that server, scoping themselves to unique serial
numbers/provision/template names so they never collide - there is no
truncate-between-tests step.

To run just one area: `go test -tags scenario ./testscenarios/... -run TestTemplates -v`.
All four files pass reliably back-to-back and under repeated runs
(`-count=N`); none of the scenarios are expected to be flaky.

## How a scenario is built

The common shape, see `helpers_test.go`:

1. `newEnv(t)` starts a server, logs in (seeded `admin`/`admin` account,
   `contrib/database/01_initial.sql`), and deletes the migrated demo
   provision ("Multiplay WiFi + ACS credentials",
   `contrib/database/04_multiplay_wifi_provision.sql`) - see "Why the demo
   provision is removed" below.
2. `scopedRule(t)` generates a temporary goacs-client profile with a unique
   `DeviceInfo.HardwareVersion` value and a matching `ProvisionRule` keyed on
   it - this is how a provision gets scoped to exactly one simulated device
   without depending on a device's serial number, manufacturer, or product
   class, none of which are resolvable in a rule until a later session (see
   below).
3. `mustCreateProvision(t, client, ...)` seeds a provision via the REST API
   (exercising real validation) and registers its own cleanup - provisions
   are the one resource that's genuinely global (matched against every
   session that satisfies its events/requests/rules), so every scenario that
   creates one must clean it up before the next subtest runs. Where a helper
   creates a provision several times within one subtest (not nested under
   separate `t.Run`s), it deletes it immediately instead of relying on the
   deferred cleanup - see `provision_matching_test.go`'s `warmUpThenTest`.
4. `runDevice(t, srv, harness.DeviceOpts{...})` runs one `goacs-client device`
   session (which may itself span several CWMP round-trips - e.g. a
   Download -> TransferComplete exchange completes within a single call).
5. Assertions go through `harness.Client`'s REST wrappers
   (`GetDeviceParameters`, `GetDeviceLogs`, `GetDeviceTasks`,
   `GetDeviceTemplates`, ...), the wire-level request/response log
   (`DeviceResult.Stdout`, from `goacs-client`'s `--verbose` output), or, for
   the Lua `log()` function specifically, `srv.Output()` - `log()` writes to
   the server process's stdout via `log.Printf`, not to any DB-backed log
   table (see the comment on `luaLog` in `acs/scripts/functions.go`).

## Non-obvious engine behaviors these tests are built around

- **A provisioning rule needs the CPE to have already reported the parameter
  it checks.** `Session.CPE.GetParameterValue` (used by
  `acs/logic/provision.go`'s rule matcher) only searches the session's live
  parameter cache, falling back to the DB. That cache is seeded from Inform's
  own `ParameterList`, which only ever carries `SoftwareVersion`,
  `HardwareVersion`, and `ManagementServer.ConnectionRequestURL` - not
  `Manufacturer`, `OUI`, `ProductClass`, or `SerialNumber` (those arrive only
  via the Inform's `DeviceId` element, which isn't mirrored into the
  parameter cache at all). So a rule on `SoftwareVersion` can match from a
  device's very first session; a rule on `Manufacturer` or `ProductClass`
  needs a prior session's full parameter walk to have persisted it to the DB
  first. `provision_matching_test.go`'s vendor/product-class scenarios run
  two sessions for exactly this reason - see the file's top comment.
- **Pushing an arbitrary stored/template parameter needs the live cache
  seeded first, in the same session.** `PrepareParametersToSend`
  (`acs/methods/parametermethods.go`) diffs the desired value set against
  `Session.CPE.ParameterValues` - the live session cache, not the DB - so an
  arbitrary parameter has to be in that cache before the diff can see it.
  `templates_test.go` and `provision_stored_params_test.go`'s
  `noOpScriptProvision` does this with a single blocking `getParameterValues`
  call on the target path: a real round-trip that fetches the CPE's current
  value with no dependency on a same-session parameter walk happening to
  reach that container first (walk-vs-script task ordering is not reliably
  predictable - see "Bugs found and fixed" below for why blocking calls can
  even reach `PrepareParametersToSend` at all).
- **Each `goacs-client device` invocation is a fresh process with no memory
  of a previous one.** The simulated device's parameter tree always starts
  from the profile's plain YAML defaults; nothing pushed to it in an earlier
  session survives into the next one. Only goacs-go's own database state
  (`cpe`, `cpe_parameters`, ...) persists across separate `RunDevice` calls
  for the same serial number.

## Bugs found and fixed while building this suite

Building black-box scenarios against real session/task-queue timing surfaced
five pre-existing bugs, all fixed as part of this work (see each commit for
the reasoning; the short version is here for anyone wondering why these
particular lines look the way they do):

1. **`acs/logic/dispatcher.go`'s `parseBody` was missing a case for
   `"deleteobjectresponse"`.** Every other RPC response type has one; without
   it, a `DeleteObjectResponse` fell into `default: ... UNKNOWN`, so the
   `deleteObject()` Lua function's blocking call always failed with a
   "protocol violation" error, regardless of what the CPE actually did.
2. **`scripts.Resume`'s continuation (in `acs/logic/dispatcher.go`) never
   called `PrepareParametersToSend`.** A script that used a blocking call
   (`getParameterValues`, `setParameterValues`, `addObject`, ...) finishes via
   a different code path than one that never blocks
   (`acs/logic/taskrunner.go`'s `runScriptTask`, which does call it inline) -
   so the entire template/stored-parameter diff-and-push mechanism was
   unreachable from any script that used a blocking call, even one whose only
   job was fetching a value. Fixed by calling `PrepareParametersToSend` from
   the Resume continuation too, mirroring `runScriptTask`.
3. **`repository/mysql/cperepository.go`'s `parameterRowParser` (used only by
   `FindParameter`) called `row.StructScan` then `row.MapScan` on the same
   `*sqlx.Row`.** A `sqlx.Row` (from `QueryRowx`) closes its underlying result
   set after its first scan call, so the second call always failed silently -
   `FindParameter` returned an empty value/type for every parameter, whether
   or not it actually existed. This broke every provisioning rule that
   depends on the DB-fallback lookup (any rule on a parameter not in the
   session's live cache, e.g. `Manufacturer` or `ProductClass`), and made
   `UpdateOrCreateParameter`'s create-vs-update decision always resolve to
   "exists" (nil error), so it could never actually create a new row. Fixed
   by reading via `MapScan` only and building the struct from that map (the
   same nested-`ValueStruct` workaround already used by the multi-row
   variant, `parametersRowsParser`), and by making `FindParameter` propagate
   a real not-found error instead of the effectively-dead
   `row.Err() == sql.ErrNoRows` check.
4. **A brand-new device's session can end before
   `runSetParamsProvisioningOnce` ever runs.** `acs/logic/taskrunner.go`'s
   `loadDeviceTasks` queues a one-shot global "new device" task on every
   `GetParameterValuesResponse` round-trip while `Session.IsNewInACS` is
   true; if that task has no configured type, `TaskRunner.Run()`'s untyped
   `default:` branch marks it done and returns *without recursing*, ending
   the session right there instead of checking whether the queue is now
   empty. A provision scoped to the synthetic `SetParameterValuesProcessor`
   request type can therefore never fire on a device's very first session.
   Not fixed (see below) - `provision_matching_test.go`'s
   `requests_SetParameterValuesProcessor_runs_after_the_inform_pass` instead
   runs on an already-known device (`warmUpDevice` first), which does no
   parameter walk at all and so never queues that task.
5. **Two of this suite's own fixture profiles used semver-style firmware
   strings** (`"2.5.0"`, `"1.0.0"`) that `acs/logic/provision.go`'s
   `numericCompare` (`strconv.ParseFloat`) can't parse - not a goacs-go bug,
   but worth noting since the symptom (a `>=` rule silently not matching)
   looked identical to bug #3 while debugging. Fixed by using plain floats
   (`"2.5"`, `"1.0"`) in `testscenarios/profiles/acme-router.yaml` and
   `legacy-router.yaml`.

Bug #4 is the one item above left as a known gap rather than fixed - it's a
narrower, lower-impact issue (only matters for a provision scoped to the
`SetParameterValuesProcessor` synthetic request type, and only on a device's
very first-ever session) and fixing it properly means changing
`TaskRunner.Run()`'s recursion structure, which felt like it warranted its
own separate, more careful change rather than folding it into this batch.

## Known limitations

- `kick()`'s live Connection Request delivery is tested via
  `goacs-client device --conn-request`, whose listener only stays up for that
  one foreground session - there's no way to keep a simulated device
  listening across two separate `RunDevice` calls the way a real CPE would.
  `TestScriptFunctions_NonBlocking/kick` checks that the ACS makes a genuine
  Connection Request attempt (it gets Digest-challenged, since goacs-go
  doesn't yet know this brand-new device's connection-request credentials);
  `TestProvisionNowForcesFullWalk` checks provision-now's forcing effect
  directly instead of relying on a live kick reaching a since-exited
  listener.
- `deleteObject`'s success path is only verified by "the blocking call
  completed without a Lua error" (a real DeleteObjectResponse came back) -
  unlike `setParameterValues`, deleting an object doesn't itself update
  `cpe_parameters`, so there's no independent stored-state check available
  without a further GetParameterValues walk revealing the object's removal.
- `CombineTemplateParameters`'s "a parameter present only in a template
  always applies regardless of priority" clause is not exercised end-to-end
  here - constructing a case where a parameter name is genuinely absent from
  `cpeDBParameters` but still lands in the session's live cache at diff time
  turned out to be intricate to set up reliably as a black-box scenario. It's
  better covered by a plain Go unit test directly against
  `models/cpe.CombineTemplateParameters`.
- goacs-client's `GetRPCMethodsResponse` advertises `FactoryReset`, which
  isn't implemented anywhere in goacs-client - not exercised here since
  goacs-go doesn't have a FactoryReset flow to test against it either.
- A prebuilt binary (`goacs-client/goacs-client`, ~11MB) is committed at the
  sibling repo's root - unrelated to this test suite, but worth cleaning up
  separately.
