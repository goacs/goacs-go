package scripts

import (
	acshttp "goacs/acs/http"
	acsxml "goacs/acs/types"
	"goacs/models/log"
	"goacs/repository"
	"goacs/repository/mysql"
)

// logScriptEvent persists a script-related entry to the CPE's device log, following the
// same convention as acs/methods/faultmethods.go's ResponseDecision (the reference
// implementation for the equivalent top-level, non-script case). Best-effort: a logging
// failure must never interrupt the CWMP session, so errors are swallowed here, same as
// every other log-writing call site in this codebase.
func logScriptEvent(reqRes *acshttp.CPERequest, logType, code, message string) {
	if !repository.HasConnection() {
		return
	}

	logRepository := mysql.NewLogRepository(repository.GetConnection())
	_ = logRepository.Save(&log.Log{
		CPEUUID:   reqRes.Session.CPE.UUID,
		Code:      code,
		Message:   message,
		Type:      logType,
		From:      log.FromACS,
		SessionId: reqRes.Session.Id,
	})
}

// logDeviceFault persists a CPE-returned Fault received while a script was waiting on a
// blocking RPC reply (addObject, reboot, ...) - the script equivalent of
// acs/methods/faultmethods.go's ResponseDecision, which handles the same event for a
// top-level (non-script) Fault response: a row in "faults" plus a Type: log.TypeFault
// device-log row, so a fault shows up the same way regardless of which path it came
// from.
func logDeviceFault(reqRes *acshttp.CPERequest, fault *acsxml.Fault) {
	if !repository.HasConnection() {
		return
	}

	faultRepository := mysql.NewFaultRepository()
	faultRepository.SaveFault(&reqRes.Session.CPE, fault.DetailFaultCode, fault.DetailFaultString)

	logRepository := mysql.NewLogRepository(repository.GetConnection())
	_ = logRepository.Save(&log.Log{
		CPEUUID:   reqRes.Session.CPE.UUID,
		Code:      fault.DetailFaultCode,
		Message:   fault.DetailFaultString,
		Type:      log.TypeFault,
		From:      log.FromDevice,
		SessionId: reqRes.Session.Id,
	})
}

// scriptFaultLoggedError wraps a script-ending error that already has a corresponding
// Type: log.TypeFault device-log entry, written by logDeviceFault at the moment the
// CPE's fault response was received (see bridgeContext.call). LogScriptError checks for
// this via IsFaultLogged so a script aborted by a CPE fault doesn't also get a second,
// redundant Type: log.TypeError entry for the same event.
type scriptFaultLoggedError struct{ error }

// IsFaultLogged reports whether err (as returned by Start/Resume) already has a
// corresponding device-log Fault entry.
func IsFaultLogged(err error) bool {
	_, ok := err.(scriptFaultLoggedError)
	return ok
}

// LogScriptError persists a script execution error (panic, protocol violation,
// uncaught Lua runtime error, local-step/total timeout, ...) to the device log as
// Type: log.TypeError. Call after Start/Resume returns a non-nil err; a no-op when that
// err already reflects a fault logged by logDeviceFault, to avoid a duplicate entry for
// the same event.
func LogScriptError(reqRes *acshttp.CPERequest, err error) {
	if err == nil || IsFaultLogged(err) {
		return
	}
	logScriptEvent(reqRes, log.TypeError, "", err.Error())
}
