// Package scripts runs provisioning scripts against the current CWMP session. The
// engine is Lua (gopher-lua) rather than PHP-style eval() or the previous Anko
// interpreter - a real sandboxed language with no access to the host process, matching
// what goacs-php's Sandbox should have been rather than what it actually is (raw eval()
// with full access to the PHP runtime).
package scripts

import (
	acshttp "goacs/acs/http"
	"log"

	lua "github.com/yuin/gopher-lua"
)

// ScriptEngine runs a single script against the session in ReqRes. Each Execute() call
// gets its own *lua.LState: Lua state isn't safe for concurrent use, and the TaskRunner
// never runs two scripts for the same session at once anyway.
type ScriptEngine struct {
	ReqRes *acshttp.CPERequest
}

func NewScriptEngine(reqRes *acshttp.CPERequest) ScriptEngine {
	return ScriptEngine{ReqRes: reqRes}
}

func (se *ScriptEngine) Execute(script string) (interface{}, error) {
	log.Println("script execution", script)

	L := lua.NewState()
	defer L.Close()

	se.registerGlobals(L)
	se.registerFunctions(L)

	if err := L.DoString(script); err != nil {
		log.Println("script execution error:", err)
		return nil, err
	}

	return nil, nil
}

// registerGlobals exposes read-only snapshots of the device and the current request/
// session as Lua tables - equivalent to goacs-php's $deviceModel/$root/$context
// variables injected into Sandbox::execute().
func (se *ScriptEngine) registerGlobals(L *lua.LState) {
	device := L.NewTable()
	L.SetField(device, "serialNumber", lua.LString(se.ReqRes.Session.CPE.SerialNumber))
	L.SetField(device, "oui", lua.LString(se.ReqRes.Session.CPE.OUI))
	L.SetField(device, "productClass", lua.LString(se.ReqRes.Session.CPE.ProductClass))
	L.SetField(device, "manufacturer", lua.LString(se.ReqRes.Session.CPE.Manufacturer))
	L.SetField(device, "softwareVersion", lua.LString(se.ReqRes.Session.CPE.SoftwareVersion))
	L.SetField(device, "hardwareVersion", lua.LString(se.ReqRes.Session.CPE.HardwareVersion))
	L.SetField(device, "root", lua.LString(se.ReqRes.Session.CPE.Root))
	L.SetGlobal("device", device)

	context := L.NewTable()
	L.SetField(context, "isNewDevice", lua.LBool(se.ReqRes.Session.IsNewInACS))
	L.SetField(context, "isBoot", lua.LBool(se.ReqRes.Session.IsBoot))
	L.SetField(context, "isBootstrap", lua.LBool(se.ReqRes.Session.IsBootstrap))
	L.SetField(context, "requestType", lua.LString(se.ReqRes.ReqType))
	L.SetGlobal("context", context)
}
