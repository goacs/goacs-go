-- 01_initial.sql's seed row for the "new device" onboarding RunScript task embedded its
-- JSON payload with single-escaped quotes/newlines (\" and \n) inside a MySQL single-quoted
-- string literal. MySQL's own backslash-escape processing consumes those backslashes before
-- the value is ever stored, so the payload column ends up holding invalid JSON (raw, unescaped
-- " and newline characters where the JSON string needed \" and \n). Every brand-new CPE
-- checking in then fails to load this task (TaskPayload.Scan json.Unmarshal error, logged as
-- "invalid character 'I' after object key:value pair" - the 'I' from "InternetGatewayDevice"
-- right after the JSON string value got cut short), silently skipping the onboarding script.
-- This re-saves the same row with correctly double-escaped backslashes so the stored bytes
-- are valid JSON. Matches the row 01_initial.sql now seeds on a fresh install.
UPDATE tasks
SET payload = '{"script":"local mac = getParameterValue(\\"InternetGatewayDevice.LANDevice.1.LANEthernetInterfaceConfig.1.MACAddress\\")\\nlocal mac4 = string.sub((mac:gsub(\\":\\", \\"\\")), 9, 12)\\nlocal ssid = \\"Multiplay_\\" .. mac4\\n\\nif parameterExist(\\"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.EnableSSIDPrefix\\") then\\n  setParameter(\\"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.EnableSSIDPrefix\\", \\"0\\", \\"RWS\\")\\nend\\n\\nif parameterExist(\\"InternetGatewayDevice.DeviceInfo.X_ZTE-COM_AdminAccount.Password\\") then\\n  setParameter(\\"InternetGatewayDevice.DeviceInfo.X_ZTE-COM_AdminAccount.Password\\", \\"CHANGEME\\", \\"RWS\\")\\nend\\n\\nsetParameter(\\"InternetGatewayDevice.ManagementServer.Password\\", \\"XD\\" .. device.serialNumber, \\"RWS\\")\\nsetParameter(\\"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID\\", ssid, \\"RWS\\")\\nsetParameter(\\"InternetGatewayDevice.ManagementServer.PeriodicInformInterval\\", \\"600\\", \\"RWS\\")\\nsaveDevice()"}'
WHERE for_name = 'global' AND for_id = 'new' AND task = 'RunScript';
