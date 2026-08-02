# Provisioning scripts

Provisioning scripts are Lua ([gopher-lua](https://github.com/yuin/gopher-lua), roughly
Lua 5.1) programs that run against one CPE session. They are attached to a `Provision`
(events + requests + parameter conditions, see `models/provisions`) and queued as
`RunScript` tasks whenever a matching CWMP request comes in (`acs/logic/provision.go`).
The engine is a real sandboxed interpreter, not `eval()` against the host process - a
script can only touch the API described below.

This document is a reference for the Lua-facing API. The Go side lives in
`engine.go` (globals), `functions.go` (non-blocking functions) and `bridge.go`
(blocking functions).

## Globals

Two read-only tables are injected into every script:

```lua
local _ = device.serialNumber      -- string
local _ = device.oui               -- string
local _ = device.productClass      -- string
local _ = device.manufacturer      -- string
local _ = device.softwareVersion   -- string
local _ = device.hardwareVersion   -- string
local _ = device.root              -- string, e.g. "InternetGatewayDevice" or "Device"

local _ = context.isNewDevice      -- bool, true the first time this CPE has ever informed
local _ = context.isBoot           -- bool, true on a "1 BOOT" event
local _ = context.isBootstrap      -- bool, true on a "0 BOOTSTRAP" event
local _ = context.requestType      -- string, e.g. "inform"
```

Always build parameter paths off `device.root` instead of hardcoding
`"InternetGatewayDevice"`, so the same script works against both TR-098 and TR-181
devices:

```lua
local ssidPath = device.root .. ".LANDevice.1.WLANConfiguration.1.SSID"
```

A `ProvisionRule`'s `Parameter` column (evaluated *before* deciding whether to run the
script at all - see `acs/logic/provision.go`) uses the same `device.root` spelling: a
`"device.root."` prefix there is resolved to the session's actual root the same way it
is inside the script, so a rule's `Parameter` and a script's parameter paths read
identically. This is a plain string substitution done by the Go matcher, not Lua - it
is never evaluated as a table access there, just textually replaced before matching.

## Non-blocking functions

These only touch the session's local parameter cache and the database - no CWMP
round-trip, no waiting for the CPE.

| Function | Signature | Notes |
|---|---|---|
| `setParameter(path, value, [flags])` | strings; `flags` optional, default `"RWS"` | Queues the value for the CPE (deferred SPV) unless the `X` flag is set. See [Flags](#flags-for-setparameter). |
| `getParameterValue(path)` | `-> string` | Reads the local cache/DB. Returns `""` if unknown - no error. |
| `parameterExist(path)` | `-> bool` | |
| `parameterNotExist(path)` | `-> bool` | |
| `deleteParameter(path)` | | Deletes from the DB. |
| `saveDevice()` | | Bulk-persists the session's in-memory parameter values to the DB. |
| `log(title, [details])` | | Writes `[script:<serial>] title: details` to the process log. |
| `piiValue([min], [max])` | `-> number` | Random Periodic Inform Interval; defaults come from the `pii_min`/`pii_max` config keys (fallback 300-900). |
| `assignTemplateByName(name, [priority=100])` | | |
| `assignTemplateById(id, [priority=100])` | | |
| `unassignTemplateByName(name)` | | |
| `unassignTemplateById(id)` | | |
| `kick()` / `provision()` | | Issues an out-of-band Connection Request (aliases, same as goacs-php). |
| `uploadFirmware(filename, [filetype="1 Firmware Upgrade Image"])` | | Queues a `Download` RPC for a later round-trip in this session - does not block. |
| `safe(fn, ...)` | `-> ok, result...` | See [Error handling](#error-handling). |

## Blocking functions

These issue a real CWMP RPC and **suspend the script until the CPE actually replies**,
which may be a separate HTTP round-trip entirely (the script goroutine parks; the
current response is the outgoing RPC; execution resumes transparently once the CPE's
answer arrives). This is what makes it possible to write straight-line code like "add
this object, then configure the fields on the instance the CPE just gave me" without
manual callback/continuation handling.

| Function | Signature | Notes |
|---|---|---|
| `addObject(path)` | `-> {instance, status, path}` | `path` must end in `.` (e.g. `"...WLANConfiguration."`). `path` in the result is `path .. instance .. "."`, ready to build child parameter names from. |
| `deleteObject(path)` | `-> status` (number) | `path` is a full instance path ending in `.` (e.g. `"...Host.2."`). |
| `reboot([commandKey=""])` | | No return value. |
| `getParameterValues(path1, path2, ...)` | `-> {[path] = value}` | One or more full parameter names, or partial paths ending in `.` to fetch a whole branch. Every value returned is also written into the session's local cache, so it's immediately visible to `getParameterValue`/`parameterExist` for the rest of the script. |
| `setParameterValues({[path] = value, ...})` | `-> status` (number) | Single table argument, keys must be strings. On a confirmed reply, also updates the local cache + DB, same as `setParameter`. |

**On a CPE Fault, or a protocol violation (the CPE replied with something unexpected),
every blocking function raises a Lua error** - it does not return an error value. An
unprotected call aborts the whole script. Use `safe()` (below) to survive a fault and
keep running, or `pcall` directly for one-off custom handling.

```lua
local obj = addObject(device.root .. ".LANDevice.1.WLANConfiguration.")
-- obj.instance, obj.status, obj.path are only reached if addObject succeeded;
-- a CPE fault here aborts the script before this line runs.
```

## Error handling

`safe(fn, ...)` calls `fn(...)` in protected mode - identical semantics to Lua's
built-in `pcall(fn, ...)` (in fact it's implemented the same way gopher-lua implements
`pcall` itself) - **and additionally logs the failure automatically** through the same
`[script:<serial>]` channel as `log()`. It is a global, registered once in the engine
(`functions.go`), so no script needs to define its own wrapper.

```lua
local ok, params = safe(getParameterValues, path1, path2)
if not ok then
    return  -- failure already logged by safe(); nothing more to do here
end

-- multiple return values pass through, just like pcall:
local ok, status = safe(setParameterValues, { [path] = "value" })
```

Use plain `pcall` instead only when you need to react to the specific error *without*
it being logged as a generic "safe() call failed" line, or want a custom message.
Calling a blocking function completely unprotected is a deliberate choice too: it means
"a fault here should abort the entire script", which is sometimes exactly what you want
(e.g. there's no sensible way to continue after this specific write fails).

## Flags for `setParameter`

The optional third argument to `setParameter` (default `"RWS"`) is any combination of:

| Flag | Meaning |
|---|---|
| `R` | Read |
| `W` | Write |
| `A` | AddObject-capable |
| `X` | System - local-only; the value is **not** queued to be pushed to the CPE |
| `P` | Periodic read |
| `I` | Important |
| `S` | Send - queue the value to be pushed to the CPE on this session |

An unknown letter raises a Lua error (`setParameter(...): invalid flags ...`).

## How a script gets run

A `Provision` (`models/provisions`) bundles:

- `Events` - CSV of CWMP event codes, e.g. `"0 BOOTSTRAP,1 BOOT"`. Empty matches any event.
- `Requests` - CSV of request types. Empty matches any request type.
- `Rules` - zero or more `{Parameter, Operator, Value}` conditions, ANDed together
  (operators: `==`, `!=`, `in`, `not in`, `>`, `>=`, `<`, `<=`). `Parameter` may use the
  `device.root.` prefix described above, e.g. `"device.root.DeviceInfo.ProductClass"`.
- `Script` - one or more Lua bodies; each is queued as its own `RunScriptTask`.

Whenever a CWMP request comes in, every provision whose events/requests/rules all match
gets its scripts queued. See `contrib/database/04_multiplay_wifi_provision.sql` for a
worked example of seeding one directly via SQL, or use the admin panel's Provisioning
screen.

## Examples

### Read a value, write another

```lua
local swVersion = getParameterValue(device.root .. ".DeviceInfo.SoftwareVersion")
log("current firmware", swVersion)

setParameter(device.root .. ".ManagementServer.PeriodicInformInterval", "600", "RWS")
saveDevice()
```

### Rename the first SSID from the last 4 bytes of a MAC address

```lua
local mac = getParameterValue(device.root .. ".WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.MACAddress")
local last4Bytes = mac:gsub(":", ""):gsub("-", ""):upper():sub(-8)
local ssid = "Multiplay_" .. last4Bytes

setParameter(device.root .. ".LANDevice.1.WLANConfiguration.1.SSID", ssid, "RWS")
saveDevice()
```

(the full provisioning script in `contrib/database/04_multiplay_wifi_provision.sql`
does the same thing via the blocking `getParameterValues`, since it needs to fetch this
value from the CPE rather than assume it's already in the local cache)

### Force ACS connection-request credentials via a real round-trip

Unlike `setParameter` (deferred, fire-and-forget until the next SPV batch),
`setParameterValues` blocks until the CPE has actually confirmed the write:

```lua
local mgmtUserPath = device.root .. ".DeviceInfo.ManagementServer.ConnectionRequestUsername"
local mgmtPassPath = device.root .. ".DeviceInfo.ManagementServer.ConnectionRequestPassword"

local ok, status = safe(setParameterValues, {
    [mgmtUserPath] = "ACS",
    [mgmtPassPath] = "ACS",
})
if ok then
    log("ConnectionRequest credentials confirmed", tostring(status))
end
```

### `addObject` / `deleteObject` with rollback on failure

This is the pattern used in `contrib/database/04_multiplay_wifi_provision.sql` to add a
port-forwarding rule idempotently, rolling back the newly created object if configuring
it fails partway through:

```lua
local base = device.root .. ".WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.PortMapping."

local addOk, obj = safe(addObject, base)
if addOk then
    log("created PortMapping instance", tostring(obj.instance))

    local ok = safe(setParameterValues, {
        [obj.path .. "PortMappingDescription"] = "ACS-managed-rule",
        [obj.path .. "PortMappingEnabled"] = "1",
        [obj.path .. "PortMappingProtocol"] = "TCP",
        [obj.path .. "ExternalPort"] = "8080",
        [obj.path .. "InternalPort"] = "80",
        [obj.path .. "InternalClient"] = "192.168.1.100",
    })

    if not ok then
        log("configuration failed, rolling back", obj.path)
        safe(deleteObject, obj.path)
    end
end
```

Before creating a new instance, check whether one already exists (via
`getParameterValues` on the table's partial path) so the script stays idempotent across
every BOOT/periodic Inform it runs on - see the full example in
`contrib/database/04_multiplay_wifi_provision.sql`.

### Assign a template conditionally, then kick the device

```lua
if getParameterValue(device.root .. ".DeviceInfo.ProductClass") == "ONT-5G" then
    assignTemplateByName("5g-ont-defaults")
else
    assignTemplateByName("default-router")
end

kick()
```

### Reboot after a firmware upload

```lua
uploadFirmware("firmware-v2.1.bin")
-- Download is queued for this session; once it completes and TransferComplete is
-- received on a later round-trip, a separate provision/script can call reboot() if the
-- device doesn't reboot on its own.
```

## Limits and gotchas

All three limits below are configurable from the admin panel's Settings screen (they
persist to the `config` table, same mechanism as `pii_min`/`pii_max` - see
`repository/mysql/configrepository.go`). Each is re-read live on every use, so a change
takes effect on the next script/round-trip without restarting the server. Raise
`script_total_timeout_seconds` (and, if needed, `script_local_step_timeout_seconds`) for
devices that are known to take a long time mid-script - e.g. a firmware upgrade whose
reboot/`TransferComplete` round-trip is slow.

| Config key | Default | Purpose |
|---|---|---|
| `script_total_timeout_seconds` | 300 (5 min) | Bounds the entire script lifetime, including every blocking RPC round-trip it makes. If the CPE never answers, the script is aborted. |
| `script_local_step_timeout_seconds` | 5 | Guards against a script that neither finishes nor calls a blocking function (e.g. an accidental infinite loop) - a local CPU-bound wait, so it can usually stay short even when the total timeout above is raised. |
| `run_script_max_count` | 30 | Scripts per session (`acs/logic/taskrunner.go`) - a safety cap against a provisioning rule that would otherwise re-queue itself forever. |

Each falls back to its default if the key is unset, non-numeric, or non-positive - see
`configuredTimeoutSeconds` (`acs/scripts/bridge.go`) and `runScriptMaxCount`
(`acs/logic/taskrunner.go`).
- There is no CPE simulator built into this repo; use the sibling `goacs-client` project
  to exercise scripts that call `addObject`, `deleteObject`, `reboot`,
  `getParameterValues` or `setParameterValues` against a real CWMP round-trip.
- Standard Lua 5.1 `string`/`table`/`math` libraries are fully available (gopher-lua,
  unrestricted) - no custom MAC/string helpers exist, plain `string.gsub`/`sub`/`match`
  is the idiom (see the SSID example above).
