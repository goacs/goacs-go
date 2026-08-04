package scripts

import (
	acshttp "goacs/acs/http"
	acsxml "goacs/acs/types"
	dlog "goacs/models/log"
	"goacs/models/tasks"
	"goacs/repository"
	"goacs/repository/mysql"
	"log"
	"math/rand"
	"strconv"

	lua "github.com/yuin/gopher-lua"
)

// registerFunctions binds the provisioning script API. Functions here never trigger a
// CWMP round-trip to the CPE - they only read/write the session's local parameter
// cache and the DB. Blocking calls that actually wait on a device response (addObject,
// reboot, ...) land in a later phase behind the goroutine+channel bridge.
func (se *ScriptEngine) registerFunctions(L *lua.LState) {
	L.SetGlobal("setParameter", L.NewFunction(se.luaSetParameter))
	L.SetGlobal("getParameterValue", L.NewFunction(se.luaGetParameterValue))
	L.SetGlobal("parameterExist", L.NewFunction(se.luaParameterExist))
	L.SetGlobal("parameterNotExist", L.NewFunction(se.luaParameterNotExist))
	L.SetGlobal("deleteParameter", L.NewFunction(se.luaDeleteParameter))
	L.SetGlobal("saveDevice", L.NewFunction(se.luaSaveDevice))
	L.SetGlobal("log", L.NewFunction(se.luaLog))
	L.SetGlobal("piiValue", L.NewFunction(se.luaPiiValue))
	L.SetGlobal("assignTemplateByName", L.NewFunction(se.luaAssignTemplateByName))
	L.SetGlobal("assignTemplateById", L.NewFunction(se.luaAssignTemplateById))
	L.SetGlobal("unassignTemplateByName", L.NewFunction(se.luaUnassignTemplateByName))
	L.SetGlobal("unassignTemplateById", L.NewFunction(se.luaUnassignTemplateById))
	L.SetGlobal("kick", L.NewFunction(se.luaKick))
	L.SetGlobal("provision", L.NewFunction(se.luaKick)) // alias, same as goacs-php
	L.SetGlobal("uploadFirmware", L.NewFunction(se.luaUploadFirmware))
	L.SetGlobal("safe", L.NewFunction(se.luaSafe))
}

func (se *ScriptEngine) luaSetParameter(L *lua.LState) int {
	path := L.CheckString(1)
	value := L.CheckString(2)
	flags := L.OptString(3, "RWS")

	flag, err := acsxml.FlagFromString(flags)
	if err != nil {
		L.RaiseError("setParameter(%q): invalid flags %q: %v", path, flags, err)
		return 0
	}

	parameter := acsxml.ParameterValueStruct{
		Name:        path,
		ValueStruct: acsxml.ValueStruct{Value: value},
		Flag:        flag,
	}

	if current := se.ReqRes.Session.CPE.GetParameter(path); current != nil {
		parameter.ValueStruct.Type = current.ValueStruct.Type
	}

	se.ReqRes.Session.CPE.AddParameter(parameter)

	cpeRepository := mysql.NewCPERepository(repository.GetConnection())
	if _, err := cpeRepository.UpdateParameter(&se.ReqRes.Session.CPE, parameter); err != nil {
		log.Println("setParameter: update error:", err)
	}

	if !flag.System {
		se.ReqRes.Session.AddParameterToAdd(parameter)
	}

	return 0
}

func (se *ScriptEngine) luaGetParameterValue(L *lua.LState) int {
	path := L.CheckString(1)
	L.Push(lua.LString(se.getParameterValue(path)))
	return 1
}

func (se *ScriptEngine) getParameterValue(path string) string {
	if value, err := se.ReqRes.Session.CPE.GetParameterValue(path); err == nil {
		return value
	}

	cpeRepository := mysql.NewCPERepository(repository.GetConnection())
	cpeParameters, _ := cpeRepository.GetCPEParameters(&se.ReqRes.Session.CPE)
	se.ReqRes.Session.CPE.AddParameterValues(cpeParameters)

	value, err := se.ReqRes.Session.CPE.GetParameterValue(path)
	if err != nil {
		return ""
	}

	return value
}

func (se *ScriptEngine) luaParameterExist(L *lua.LState) int {
	path := L.CheckString(1)
	L.Push(lua.LBool(se.ReqRes.Session.CPE.ParameterValueExist(path)))
	return 1
}

func (se *ScriptEngine) luaParameterNotExist(L *lua.LState) int {
	path := L.CheckString(1)
	L.Push(lua.LBool(!se.ReqRes.Session.CPE.ParameterValueExist(path)))
	return 1
}

func (se *ScriptEngine) luaDeleteParameter(L *lua.LState) int {
	path := L.CheckString(1)

	cpeRepository := mysql.NewCPERepository(repository.GetConnection())
	if _, err := cpeRepository.DeleteParameter(&se.ReqRes.Session.CPE, path); err != nil {
		log.Println("deleteParameter error:", err)
	}

	return 0
}

func (se *ScriptEngine) luaSaveDevice(L *lua.LState) int {
	cpeRepository := mysql.NewCPERepository(repository.GetConnection())
	// preserveServerControlled=true: saveDevice() bulk-flushes the whole in-memory
	// parameter set, which mixes in whatever the CPE most recently reported - must not
	// clobber a Send/System-flagged parameter the ACS itself controls.
	_ = cpeRepository.BulkInsertOrUpdateParameters(&se.ReqRes.Session.CPE, se.ReqRes.Session.CPE.ParameterValues, true)
	return 0
}

func (se *ScriptEngine) luaLog(L *lua.LState) int {
	title := L.CheckString(1)
	details := L.OptString(2, "")
	log.Printf("[script:%s] %s: %s\n", se.ReqRes.Session.CPE.SerialNumber, title, details)
	return 0
}

// luaPiiValue picks a random Periodic Inform Interval within a range, defaulting to the
// pii_min/pii_max config keys (falling back to 300-900 seconds) unless the script
// passes explicit bounds.
func (se *ScriptEngine) luaPiiValue(L *lua.LState) int {
	configRepository := mysql.NewConfigRepository(repository.GetConnection())

	defaultMin, defaultMax := 300, 900
	if v, err := configRepository.GetValue("pii_min"); err == nil {
		if parsed, perr := strconv.Atoi(v); perr == nil {
			defaultMin = parsed
		}
	}
	if v, err := configRepository.GetValue("pii_max"); err == nil {
		if parsed, perr := strconv.Atoi(v); perr == nil {
			defaultMax = parsed
		}
	}

	min := L.OptInt(1, defaultMin)
	max := L.OptInt(2, defaultMax)
	if max <= min {
		max = min + 1
	}

	L.Push(lua.LNumber(min + rand.Intn(max-min)))
	return 1
}

func (se *ScriptEngine) luaAssignTemplateByName(L *lua.LState) int {
	name := L.CheckString(1)
	priority := L.OptInt(2, 100)

	templateRepository := mysql.NewTemplateRepository(repository.GetConnection())
	template, err := templateRepository.FindByName(name)
	if err != nil {
		L.RaiseError("assignTemplateByName(%q): %v", name, err)
		return 0
	}

	if err := templateRepository.AssignTemplateToDevice(&se.ReqRes.Session.CPE, template.Id, int64(priority)); err != nil {
		L.RaiseError("assignTemplateByName(%q): %v", name, err)
	}

	return 0
}

func (se *ScriptEngine) luaAssignTemplateById(L *lua.LState) int {
	id := L.CheckInt64(1)
	priority := L.OptInt(2, 100)

	templateRepository := mysql.NewTemplateRepository(repository.GetConnection())
	if err := templateRepository.AssignTemplateToDevice(&se.ReqRes.Session.CPE, id, int64(priority)); err != nil {
		L.RaiseError("assignTemplateById(%d): %v", id, err)
	}

	return 0
}

func (se *ScriptEngine) luaUnassignTemplateByName(L *lua.LState) int {
	name := L.CheckString(1)

	templateRepository := mysql.NewTemplateRepository(repository.GetConnection())
	template, err := templateRepository.FindByName(name)
	if err != nil {
		L.RaiseError("unassignTemplateByName(%q): %v", name, err)
		return 0
	}

	if err := templateRepository.UnassignTemplateFromDevice(&se.ReqRes.Session.CPE, template.Id); err != nil {
		L.RaiseError("unassignTemplateByName(%q): %v", name, err)
	}

	return 0
}

func (se *ScriptEngine) luaUnassignTemplateById(L *lua.LState) int {
	id := L.CheckInt64(1)

	templateRepository := mysql.NewTemplateRepository(repository.GetConnection())
	if err := templateRepository.UnassignTemplateFromDevice(&se.ReqRes.Session.CPE, id); err != nil {
		L.RaiseError("unassignTemplateById(%d): %v", id, err)
	}

	return 0
}

// luaKick issues a Connection Request to the device outside of the current session -
// same operation as the "kick" button in the admin panel (http/controllers/device.go).
func (se *ScriptEngine) luaKick(L *lua.LState) int {
	acsRequest := acshttp.NewACSRequest(&se.ReqRes.Session.CPE)
	acsRequest.Kick()
	return 0
}

// luaUploadFirmware queues a Download RPC for the given file, sent on a later
// round-trip within this session (not blocking - see engine.go doc comment).
func (se *ScriptEngine) luaUploadFirmware(L *lua.LState) int {
	filename := L.CheckString(1)
	filetype := L.OptString(2, "1 Firmware Upgrade Image")

	dlTask := tasks.NewCPETask(se.ReqRes.Session.CPE.UUID)
	dlTask.AsUploadFirmware(filename, filetype)
	se.ReqRes.Session.AddTask(dlTask)

	return 0
}

// luaSafe is a global, always-available alternative to writing pcall(...) at every call
// site. Every blocking bridge function (addObject, deleteObject, reboot,
// getParameterValues, setParameterValues - see bridge.go) raises a Lua error on a CPE
// fault or protocol violation, so an unprotected call aborts the whole script. safe(fn,
// ...) calls fn(...) in protected mode - same semantics as Lua's built-in pcall, and in
// fact mirrors gopher-lua's own basePCall (baselib.go) - but additionally logs the
// failure through the same [script:<serial>] channel as log(), so scripts get
// fail-soft-with-a-log-line behavior without repeating pcall+log boilerplate at every
// call site:
//
//	local ok, params = safe(getParameterValues, path)
//	if not ok then return end
func (se *ScriptEngine) luaSafe(L *lua.LState) int {
	L.CheckAny(1)
	v := L.Get(1)
	if v.Type() != lua.LTFunction && L.GetMetaField(v, "__call").Type() != lua.LTFunction {
		L.RaiseError("safe(): first argument must be a function, got %s", v.Type().String())
		return 0
	}

	nargs := L.GetTop() - 1
	if err := L.PCall(nargs, lua.MultRet, nil); err != nil {
		message := err.Error()
		if apiErr, ok := err.(*lua.ApiError); ok {
			message = apiErr.Object.String()
		}
		log.Printf("[script:%s] safe() call failed: %s\n", se.ReqRes.Session.CPE.SerialNumber, message)
		logScriptEvent(se.ReqRes, dlog.TypeError, "", message)

		L.Push(lua.LFalse)
		L.Push(lua.LString(message))
		return 2
	}

	L.Insert(lua.LTrue, 1)
	return L.GetTop()
}
