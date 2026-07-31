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

func TestWrap_RunScriptTask_ReturnsScriptSource(t *testing.T) {
	dbTask := tasks.Task{Task: tasks.RunScript, Payload: tasks.TaskPayload{"script": "println(1)"}}

	wrapped := Wrap(dbTask)
	scriptTask, ok := wrapped.(ScriptTask)

	if assert.True(t, ok, "expected a ScriptTask") {
		assert.Equal(t, "println(1)", scriptTask.ScriptSource())
	}
}
