package scripts

import (
	"bytes"
	"goacs/acs"
	acshttp "goacs/acs/http"
	acsxml "goacs/acs/types"
	"log"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureLog redirects the standard logger for the duration of fn and returns
// everything written to it - used to observe what a running script logged, since
// Start()/Resume() don't otherwise expose the script's internal Lua state.
func captureLog(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	original := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(original)

	fn()

	return buf.String()
}

func newBridgeTestRequest(session *acs.ACSSession, reqType string, body string) *acshttp.CPERequest {
	envelope := acsxml.NewEnvelope()
	return &acshttp.CPERequest{
		Response: httptest.NewRecorder(),
		Session:  session,
		Envelope: &envelope,
		Body:     []byte(body),
		ReqType:  reqType,
	}
}

// TestBridge_AddObject_BlocksUntilCPEResponds is the core scenario the whole bridge
// exists for: a script calls addObject() on one line and, once the (simulated) CPE
// answers on a LATER, separate HTTP round-trip, gets the real instance number back
// and keeps executing using it - all without the script author writing any
// callback/continuation code.
func TestBridge_AddObject_BlocksUntilCPEResponds(t *testing.T) {
	session := &acs.ACSSession{}
	session.CPE.Root = "InternetGatewayDevice"

	reqRes1 := newBridgeTestRequest(session, acsxml.InformReq, "")

	// log() is the only non-blocking function used here (deliberately) - setParameter/
	// saveDevice/etc. all hit the DB unconditionally, which this test has none of; see
	// functions_test.go for coverage of those against an in-memory session cache only.
	script := `
		local obj = addObject("InternetGatewayDevice.LANDevice.1.WLANConfiguration.")
		log("addObject result", tostring(obj.instance) .. " " .. obj.path)
	`

	finished, err := Start(reqRes1, script)
	require.NoError(t, err)
	assert.False(t, finished, "script should be suspended waiting for the AddObjectResponse")
	assert.NotNil(t, session.Script, "session should record the suspended script")

	// The AddObject request must have been written as THIS round-trip's response.
	written := reqRes1.Response.(*httptest.ResponseRecorder).Body.String()
	assert.Contains(t, written, "<cwmp:AddObject>")
	assert.Contains(t, written, "<ObjectName>InternetGatewayDevice.LANDevice.1.WLANConfiguration.</ObjectName>")

	// Simulate the CPE's reply arriving on a SEPARATE HTTP request.
	responseBody := `<?xml version="1.0"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <cwmp:AddObjectResponse xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
      <InstanceNumber>7</InstanceNumber>
      <Status>0</Status>
    </cwmp:AddObjectResponse>
  </soapenv:Body>
</soapenv:Envelope>`

	reqRes2 := newBridgeTestRequest(session, acsxml.AddObjResp, responseBody)

	var finished2 bool
	var err2 error
	logged := captureLog(t, func() {
		finished2, err2 = Resume(reqRes2)
	})

	require.NoError(t, err2)
	assert.True(t, finished2, "script should finish naturally after addObject() returns")
	assert.Nil(t, session.Script, "session should have no suspended script once it finishes")

	// Proves the real CPE-assigned instance number (7, not e.g. a placeholder) made it
	// all the way from the simulated response back into the still-running script.
	assert.Contains(t, logged, "7 InternetGatewayDevice.LANDevice.1.WLANConfiguration.7.")
}

func TestBridge_AddObject_CPEFault_RaisesLuaError(t *testing.T) {
	session := &acs.ACSSession{}
	session.CPE.Root = "InternetGatewayDevice"

	reqRes1 := newBridgeTestRequest(session, acsxml.InformReq, "")

	finished, err := Start(reqRes1, `addObject("InternetGatewayDevice.LANDevice.1.WLANConfiguration.")`)
	require.NoError(t, err)
	assert.False(t, finished)

	faultBody := `<?xml version="1.0"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <soapenv:Fault>
      <faultcode>Client</faultcode>
      <faultstring>CWMP fault</faultstring>
      <detail>
        <Fault>
          <FaultCode>9005</FaultCode>
          <FaultString>Invalid parameter name</FaultString>
        </Fault>
      </detail>
    </soapenv:Fault>
  </soapenv:Body>
</soapenv:Envelope>`

	reqRes2 := newBridgeTestRequest(session, acsxml.FaultResp, faultBody)

	finished2, err2 := Resume(reqRes2)
	assert.True(t, finished2)
	require.Error(t, err2)
	assert.Contains(t, err2.Error(), "9005")
}

func TestBridge_ProtocolViolation_UnrelatedResponse(t *testing.T) {
	session := &acs.ACSSession{}
	session.CPE.Root = "InternetGatewayDevice"

	reqRes1 := newBridgeTestRequest(session, acsxml.InformReq, "")

	finished, err := Start(reqRes1, `addObject("InternetGatewayDevice.LANDevice.1.WLANConfiguration.")`)
	require.NoError(t, err)
	assert.False(t, finished)

	// CPE sends something totally unrelated instead of the expected AddObjectResponse.
	reqRes2 := newBridgeTestRequest(session, acsxml.GPNResp, "")

	finished2, err2 := Resume(reqRes2)
	assert.True(t, finished2)
	require.Error(t, err2)
	assert.Contains(t, err2.Error(), "protocol violation")
	assert.Nil(t, session.Script)
}

func TestBridge_ScriptWithNoBlockingCall_FinishesImmediately(t *testing.T) {
	session := &acs.ACSSession{}
	session.CPE.Root = "InternetGatewayDevice"

	reqRes := newBridgeTestRequest(session, acsxml.InformReq, "")

	finished, err := Start(reqRes, `log("no blocking call", "just finishes")`)

	require.NoError(t, err)
	assert.True(t, finished)
	assert.Nil(t, session.Script)
}

func TestBridge_Resume_WithoutSuspendedScript_ReturnsError(t *testing.T) {
	session := &acs.ACSSession{}
	reqRes := newBridgeTestRequest(session, acsxml.AddObjResp, "")

	finished, err := Resume(reqRes)

	assert.True(t, finished)
	assert.Error(t, err)
}

// TestBridge_GetParameterValues_BlocksUntilCPEResponds mirrors the AddObject test:
// getParameterValues() must suspend the script, send a real GetParameterValues RPC,
// and hand the CPE's actual values back once they arrive on a later round-trip.
func TestBridge_GetParameterValues_BlocksUntilCPEResponds(t *testing.T) {
	session := &acs.ACSSession{}
	session.CPE.Root = "InternetGatewayDevice"

	reqRes1 := newBridgeTestRequest(session, acsxml.InformReq, "")

	script := `
		local values = getParameterValues("InternetGatewayDevice.DeviceInfo.SoftwareVersion")
		log("got value", values["InternetGatewayDevice.DeviceInfo.SoftwareVersion"])
	`

	finished, err := Start(reqRes1, script)
	require.NoError(t, err)
	assert.False(t, finished, "script should be suspended waiting for the GetParameterValuesResponse")

	written := reqRes1.Response.(*httptest.ResponseRecorder).Body.String()
	assert.Contains(t, written, "<cwmp:GetParameterValues>")
	assert.Contains(t, written, "<string>InternetGatewayDevice.DeviceInfo.SoftwareVersion</string>")

	responseBody := `<?xml version="1.0"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <cwmp:GetParameterValuesResponse xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
      <ParameterList>
        <ParameterValueStruct>
          <Name>InternetGatewayDevice.DeviceInfo.SoftwareVersion</Name>
          <Value xsi:type="xsd:string">1.2.3</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:GetParameterValuesResponse>
  </soapenv:Body>
</soapenv:Envelope>`

	reqRes2 := newBridgeTestRequest(session, acsxml.GPVResp, responseBody)

	var finished2 bool
	var err2 error
	logged := captureLog(t, func() {
		finished2, err2 = Resume(reqRes2)
	})

	require.NoError(t, err2)
	assert.True(t, finished2)
	assert.Contains(t, logged, "got value: 1.2.3")

	// The returned value must also have landed in the session's local parameter cache.
	value, err := session.CPE.GetParameterValue("InternetGatewayDevice.DeviceInfo.SoftwareVersion")
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", value)
}

// TestBridge_SetParameterValues_BlocksUntilCPEResponds mirrors the AddObject test for
// the SetParameterValues RPC: it must suspend the script, send the real request, and
// return the CPE-confirmed Status once the response arrives.
func TestBridge_SetParameterValues_BlocksUntilCPEResponds(t *testing.T) {
	session := &acs.ACSSession{}
	session.CPE.Root = "InternetGatewayDevice"

	reqRes1 := newBridgeTestRequest(session, acsxml.InformReq, "")

	script := `
		local status = setParameterValues({
			["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = "MyWifi",
		})
		log("setParameterValues result", tostring(status))
	`

	finished, err := Start(reqRes1, script)
	require.NoError(t, err)
	assert.False(t, finished, "script should be suspended waiting for the SetParameterValuesResponse")

	written := reqRes1.Response.(*httptest.ResponseRecorder).Body.String()
	assert.Contains(t, written, "<cwmp:SetParameterValues>")
	assert.Contains(t, written, "<Name>InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID</Name>")
	assert.Contains(t, written, "MyWifi")

	responseBody := `<?xml version="1.0"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <cwmp:SetParameterValuesResponse xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
      <Status>0</Status>
    </cwmp:SetParameterValuesResponse>
  </soapenv:Body>
</soapenv:Envelope>`

	reqRes2 := newBridgeTestRequest(session, acsxml.SPVResp, responseBody)

	var finished2 bool
	var err2 error
	logged := captureLog(t, func() {
		finished2, err2 = Resume(reqRes2)
	})

	require.NoError(t, err2)
	assert.True(t, finished2)
	assert.Contains(t, logged, "setParameterValues result: 0")

	// The confirmed value must also have landed in the session's local parameter cache
	// (no live DB in this test, so this only exercises the in-memory side of it).
	value, err := session.CPE.GetParameterValue("InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID")
	require.NoError(t, err)
	assert.Equal(t, "MyWifi", value)
}

// TestBridge_Safe_CPEFault_DoesNotAbortScript proves safe() lets a script survive a real
// CPE fault from a blocking bridge function: unlike a bare addObject() call (see
// TestBridge_AddObject_CPEFault_RaisesLuaError), the script keeps running past the
// faulting call, and the failure still gets logged automatically.
func TestBridge_Safe_CPEFault_DoesNotAbortScript(t *testing.T) {
	session := &acs.ACSSession{}
	session.CPE.Root = "InternetGatewayDevice"

	reqRes1 := newBridgeTestRequest(session, acsxml.InformReq, "")

	script := `
		local ok, err = safe(addObject, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.")
		log("safe() returned", tostring(ok) .. " " .. tostring(err))
		log("script kept running", "after the faulting call")
	`

	finished, err := Start(reqRes1, script)
	require.NoError(t, err)
	assert.False(t, finished, "script should be suspended waiting for the AddObjectResponse")

	faultBody := `<?xml version="1.0"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <soapenv:Fault>
      <faultcode>Client</faultcode>
      <faultstring>CWMP fault</faultstring>
      <detail>
        <Fault>
          <FaultCode>9005</FaultCode>
          <FaultString>Invalid parameter name</FaultString>
        </Fault>
      </detail>
    </soapenv:Fault>
  </soapenv:Body>
</soapenv:Envelope>`

	reqRes2 := newBridgeTestRequest(session, acsxml.FaultResp, faultBody)

	var finished2 bool
	var err2 error
	logged := captureLog(t, func() {
		finished2, err2 = Resume(reqRes2)
	})

	require.NoError(t, err2, "safe() must swallow the fault instead of letting it abort the script")
	assert.True(t, finished2, "script should finish naturally, having run past the faulting call")
	assert.Contains(t, logged, "safe() call failed:", "safe() must log the failure automatically")
	assert.Contains(t, logged, "9005")
	assert.Contains(t, logged, "safe() returned: false")
	assert.Contains(t, logged, "script kept running: after the faulting call")
}

// Note: pump()'s local-step-timeout branch (a script that never finishes and never
// calls a blocking function, e.g. an infinite loop) is not covered by a test here -
// exercising it for real would mean waiting out the actual LocalStepTimeout (5s), and
// it's a plain three-way select with no logic beyond what the other tests in this file
// already exercise for the other two branches.
