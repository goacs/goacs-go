-- Replaces the old "read the whole parameter tree for every brand-new device" behavior
-- with a curated script, mirroring goacs-php's EmptyResponseProcessor::requestBasicParams()
-- fix (commit f6be100, "read only wanted params/tree"). acs/methods/informmethods.go no
-- longer queues an automatic GetParameterNames walk for brand-new devices (IsNewInACS) -
-- see the comment there - so this "init" Provision takes over: it matches a device's very
-- first Inform (0 BOOTSTRAP / 1 BOOT) and runs a Lua script that fetches just DeviceInfo.
-- and ManagementServer. via the blocking getParameterValues() bridge function, instead of
-- walking the entire device model one leaf parameter at a time.
--
-- The old hardcoded "global new device" onboarding task (seeded in 01_initial.sql, fixed
-- up in 05_fix_new_device_task_payload.sql) is retired here: it triggered on
-- event = GetParameterValuesResponse, which only ever happened after the old automatic
-- full walk completed. Now that the walk is gone, that trigger would instead fire the
-- moment THIS provision's own getParameterValues() call produces a
-- GetParameterValuesResponse round-trip, running the legacy MAC/SSID-renaming script a
-- second time inside the same onboarding flow. Deleting the row removes that conflict;
-- acs/logic/taskrunner.go's loadDeviceTasks no longer has any code path that loads it.
DELETE FROM tasks WHERE for_name = 'global' AND for_id = 'new' AND task = 'RunScript';

insert into provisions (name, events, requests, script)
values ('init', '0 BOOTSTRAP,1 BOOT', 'inform', '["-- init: read only the basic/core device parameters on first contact instead of\\n-- walking the entire tree (previously one GetParameterValues round-trip per leaf\\n-- parameter across the whole device model). Mirrors goacs-php EmptyResponseProcessor\\n-- requestBasicParams().\\nlocal root = device.root\\n\\nlocal ok = safe(getParameterValues,\\n    root .. \\".DeviceInfo.\\",\\n    root .. \\".ManagementServer.\\"\\n)\\n\\nif not ok then\\n    log(\\"init: failed to read basic parameters\\")\\n    return\\nend\\n\\nsaveDevice()\\nlog(\\"init: basic parameters read and saved\\")"]');
