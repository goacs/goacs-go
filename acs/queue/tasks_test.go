package queue

import (
	acsxml "goacs/acs/types"
	"goacs/models/tasks"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrap_DispatchesByTaskType(t *testing.T) {
	cases := []struct {
		name     string
		taskType string
		want     interface{}
	}{
		{"inform response", acsxml.InformResp, InformResponseTask{}},
		{"get parameter names", acsxml.GPNReq, GetParameterNamesTask{}},
		{"get parameter values", acsxml.GPVReq, GetParameterValuesTask{}},
		{"set parameter values", acsxml.SPVReq, SetParameterValuesTask{}},
		{"add object (operator)", tasks.AddObject, AddObjectTask{}},
		{"add object (auto-diff)", acsxml.AddObjReq, AddObjectTask{}},
		{"delete object (operator)", tasks.DeleteObject, DeleteObjectTask{}},
		{"delete object (auto-diff)", acsxml.DelObjReq, DeleteObjectTask{}},
		{"download", acsxml.Download, DownloadTask{}},
		{"upload firmware", tasks.UploadFirmware, DownloadTask{}},
		{"reboot", acsxml.Reboot, RebootTask{}},
		{"run script", tasks.RunScript, RunScriptTask{}},
		{"run diagnostics", tasks.RunDiagnostics, SetParameterValuesTask{}},
		{"unknown", "SomethingElse", UnknownTask{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := Wrap(tasks.Task{Task: tc.taskType})
			assert.IsType(t, tc.want, wrapped)
		})
	}
}

func TestAddObjectPath_PrefersPayloadOverParameterValues(t *testing.T) {
	// Operator-initiated AddObject task: path comes from Payload (set by AsAddObject()).
	opTask := tasks.Task{Task: tasks.AddObject, Payload: tasks.TaskPayload{"path": "InternetGatewayDevice.LANDevice."}}
	assert.Equal(t, "InternetGatewayDevice.LANDevice.", addObjectPath(opTask))

	// Auto-diff AddObject task (queued by GetParameterValuesResponseParser): path
	// comes from ParameterValues, since that code path never sets Payload["path"].
	autoTask := tasks.Task{
		Task:            acsxml.AddObjReq,
		ParameterValues: []acsxml.ParameterValueStruct{{Name: "InternetGatewayDevice.WLANConfiguration."}},
	}
	assert.Equal(t, "InternetGatewayDevice.WLANConfiguration.", addObjectPath(autoTask))
}

func TestGetParameterNamesTask_PathComesFromPayloadOnly(t *testing.T) {
	// GetParameterNamesTask.ToRequest reads its path exclusively from
	// Payload["path"] - unlike AddObjectTask/DeleteObjectTask it has no
	// ParameterInfo/ParameterValues fallback. A task built by only setting
	// ParameterInfo/NextLevel directly (as informmethods.go used to do for the
	// initial post-Inform walk) leaves Payload nil and silently resolves to an
	// empty path on the wire. Always build these via AsGetParameterNames so
	// Payload["path"] is populated.
	viaFieldsOnly := tasks.Task{Task: acsxml.GPNReq}
	viaFieldsOnly.ParameterInfo = append(viaFieldsOnly.ParameterInfo, acsxml.ParameterInfo{Name: "Device.", Done: false})
	viaFieldsOnly.NextLevel = true
	path, ok := viaFieldsOnly.Payload["path"].(string)
	assert.False(t, ok, "Payload path assertion should fail when only ParameterInfo/NextLevel are set")
	assert.Equal(t, "", path)

	viaHelper := tasks.NewCPETask("cpe-uuid")
	viaHelper.AsGetParameterNames("Device.")
	viaHelper.NextLevel = true
	path, ok = viaHelper.Payload["path"].(string)
	assert.True(t, ok, "Payload path assertion should succeed when built via AsGetParameterNames")
	assert.Equal(t, "Device.", path)
}

func TestExplicitParameterValues_ReadsAsDiagnosticsPayload(t *testing.T) {
	// Shape produced by tasks.Task.AsDiagnostics after a round-trip through the JSON
	// `payload` DB column: TaskPayload.Scan JSON-decodes into generic
	// []interface{}/map[string]interface{}, not the original []map[string]string.
	payload := tasks.TaskPayload{
		"parameters": []interface{}{
			map[string]interface{}{"name": "Device.IP.Diagnostics.DownloadDiagnostics.DiagnosticsState", "value": "Requested", "type": "xsd:string"},
			map[string]interface{}{"name": "Device.IP.Diagnostics.DownloadDiagnostics.DownloadURL", "value": "http://acs.example/speedtest/download", "type": "xsd:string"},
		},
	}

	params := explicitParameterValues(payload)

	if assert.Len(t, params, 2) {
		assert.Equal(t, "Device.IP.Diagnostics.DownloadDiagnostics.DiagnosticsState", params[0].Name)
		assert.Equal(t, "Requested", params[0].ValueStruct.Value)
		assert.Equal(t, "xsd:string", params[0].ValueStruct.Type)
		assert.Equal(t, "http://acs.example/speedtest/download", params[1].ValueStruct.Value)
	}
}

func TestExplicitParameterValues_NilWhenPayloadHasNoParameters(t *testing.T) {
	// The auto-diff SetParameterValues task queued by PrepareParametersToSend always has
	// an empty Payload (tasks.NewCPETask sets no fields) - explicitParameterValues must
	// return nil so SetParameterValuesTask.ToRequest falls back to the session's
	// PopParametersToAdd() diff queue, unchanged from before this feature existed.
	assert.Nil(t, explicitParameterValues(tasks.TaskPayload{}))
	assert.Nil(t, explicitParameterValues(nil))
}

func TestWrap_RunScriptTask_ReturnsScriptSource(t *testing.T) {
	dbTask := tasks.Task{Task: tasks.RunScript, Payload: tasks.TaskPayload{"script": "println(1)"}}

	wrapped := Wrap(dbTask)
	scriptTask, ok := wrapped.(ScriptTask)

	if assert.True(t, ok, "expected a ScriptTask") {
		assert.Equal(t, "println(1)", scriptTask.ScriptSource())
	}
}
