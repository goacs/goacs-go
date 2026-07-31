package scripts

import (
	"goacs/acs"
	acshttp "goacs/acs/http"
	acsxml "goacs/acs/types"
	"testing"

	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
)

// newTestEngine builds a ScriptEngine against an in-memory session/CPE only - no DB
// connection - so it can only safely exercise functions that read from the session's
// local parameter cache (getParameterValue on a cache hit, parameterExist/
// parameterNotExist, and the device/context globals). Functions that unconditionally
// hit the DB (setParameter, deleteParameter, saveDevice, template assignment) need a
// live MySQL instance and are exercised in integration testing instead.
func newTestEngine() (ScriptEngine, *acs.ACSSession) {
	session := &acs.ACSSession{
		IsNewInACS: true,
		IsBoot:     true,
	}
	session.CPE.Root = "InternetGatewayDevice"
	session.CPE.SerialNumber = "SN123456"
	session.CPE.AddParameterValues([]acsxml.ParameterValueStruct{
		{Name: "InternetGatewayDevice.DeviceInfo.SoftwareVersion", ValueStruct: acsxml.ValueStruct{Value: "1.2.3"}},
	})

	reqRes := &acshttp.CPERequest{Session: session, ReqType: acsxml.InformReq}
	return NewScriptEngine(reqRes), session
}

// runWithRecorder executes a script with an extra `record(value)` global bound so the
// test can observe values computed inside Lua without needing Execute() to return them.
func runWithRecorder(t *testing.T, se ScriptEngine, script string) []lua.LValue {
	t.Helper()

	L := lua.NewState()
	defer L.Close()

	se.registerGlobals(L)
	se.registerFunctions(L)

	var recorded []lua.LValue
	L.SetGlobal("record", L.NewFunction(func(L *lua.LState) int {
		recorded = append(recorded, L.CheckAny(1))
		return 0
	}))

	assert.NoError(t, L.DoString(script))
	return recorded
}

func TestGetParameterValue_CacheHit(t *testing.T) {
	se, _ := newTestEngine()

	recorded := runWithRecorder(t, se, `
		record(getParameterValue("InternetGatewayDevice.DeviceInfo.SoftwareVersion"))
	`)

	assert.Len(t, recorded, 1)
	assert.Equal(t, "1.2.3", recorded[0].String())
}

func TestParameterExistAndNotExist(t *testing.T) {
	se, _ := newTestEngine()

	recorded := runWithRecorder(t, se, `
		record(parameterExist("InternetGatewayDevice.DeviceInfo.SoftwareVersion"))
		record(parameterNotExist("InternetGatewayDevice.DeviceInfo.SoftwareVersion"))
		record(parameterExist("InternetGatewayDevice.Nope"))
	`)

	assert.Equal(t, []lua.LValue{lua.LTrue, lua.LFalse, lua.LFalse}, recorded)
}

func TestDeviceAndContextGlobals(t *testing.T) {
	se, _ := newTestEngine()

	recorded := runWithRecorder(t, se, `
		record(device.serialNumber)
		record(device.root)
		record(context.isNewDevice)
		record(context.isBoot)
		record(context.requestType)
	`)

	assert.Equal(t, "SN123456", recorded[0].String())
	assert.Equal(t, "InternetGatewayDevice", recorded[1].String())
	assert.Equal(t, lua.LTrue, recorded[2])
	assert.Equal(t, lua.LTrue, recorded[3])
	assert.Equal(t, acsxml.InformReq, recorded[4].String())
}

func TestScriptSyntaxError_ReturnsError(t *testing.T) {
	se, _ := newTestEngine()

	_, err := se.Execute(`this is not valid lua {{{`)
	assert.Error(t, err)
}

func TestSetParameter_InvalidFlags_RaisesLuaError(t *testing.T) {
	se, _ := newTestEngine()

	_, err := se.Execute(`setParameter("Some.Path", "value", "not-a-real-flag-set-Q")`)
	assert.Error(t, err)
}
