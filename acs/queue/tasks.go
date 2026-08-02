package queue

import (
	"goacs/acs/methods"
	acsxml "goacs/acs/types"
	"goacs/lib"
	"goacs/models/tasks"
)

// dbTaskAdapter is embedded by every wrapper below to expose the raw DB-backed task
// (used by the TaskRunner for id/infinite bookkeeping) without repeating Name().
type dbTaskAdapter struct{ db tasks.Task }

func (a dbTaskAdapter) Name() string { return a.db.Task }

type InformResponseTask struct{ dbTaskAdapter }

func (t InformResponseTask) ToResponse(ctx *RunContext) (string, error) {
	informDecision := methods.InformDecision{ReqRes: ctx.ReqRes}
	return informDecision.CpeInformResponse(), nil
}

type GetParameterNamesTask struct{ dbTaskAdapter }

func (t GetParameterNamesTask) ToRequest(ctx *RunContext) (string, error) {
	path, _ := t.db.Payload["path"].(string)
	pd := methods.ParameterDecisions{ReqRes: ctx.ReqRes}
	return pd.ParameterNamesRequest(path, t.db.NextLevel), nil
}

type GetParameterValuesTask struct{ dbTaskAdapter }

func (t GetParameterValuesTask) ToRequest(ctx *RunContext) (string, error) {
	pd := methods.ParameterDecisions{ReqRes: ctx.ReqRes}
	return pd.GetParameterValuesRequest(t.db.ParameterInfo), nil
}

type SetParameterValuesTask struct{ dbTaskAdapter }

func (t SetParameterValuesTask) ToRequest(ctx *RunContext) (string, error) {
	params := ctx.ReqRes.Session.PopParametersToAdd()
	// Remembered here since the response confirming these values arrives on
	// a later round-trip, after ParametersToAdd has already been popped -
	// see ParameterDecisions.SetParameterValuesResponseParser.
	ctx.ReqRes.Session.PendingSetParameterValues = params
	ctx.ReqRes.Session.PrevState = acsxml.SPVReq
	return ctx.ReqRes.Envelope.SetParameterValues(params), nil
}

// AddObjectTask handles both operator-initiated AddObject (task type "AddObject",
// created via tasks.NewCPETask().AsAddObject()) and the auto-diff AddObject tasks
// queued by GetParameterValuesResponseParser using the raw "AddObjectRequest" type
// string - previously these two spellings were never unified, so the auto-diff variant
// silently never ran under the old if/else dispatcher. Wrap() maps both here.
type AddObjectTask struct{ dbTaskAdapter }

func (t AddObjectTask) ToRequest(ctx *RunContext) (string, error) {
	path := addObjectPath(t.db)

	ctx.ReqRes.Session.PrevState = acsxml.AddObjReq
	body := ctx.ReqRes.Envelope.AddObjectRequest(path, "")

	gpnTask := tasks.NewCPETask(ctx.ReqRes.Session.CPE.UUID)
	gpnTask.AsGetParameterNames(path)
	ctx.ReqRes.Session.AddTask(gpnTask)

	return body, nil
}

func addObjectPath(db tasks.Task) string {
	if path, ok := db.Payload["path"].(string); ok {
		return path
	}
	if len(db.ParameterValues) > 0 {
		return db.ParameterValues[0].Name
	}
	return ""
}

// DeleteObjectTask, likewise, unifies "DeleteObject" (operator-initiated) and
// "DeleteObjectRequest" (auto-diff) task type spellings - see AddObjectTask.
type DeleteObjectTask struct{ dbTaskAdapter }

func (t DeleteObjectTask) ToRequest(ctx *RunContext) (string, error) {
	path := addObjectPath(t.db)

	ctx.ReqRes.Session.PrevState = acsxml.DelObjReq
	body := ctx.ReqRes.Envelope.DeleteObjectRequest(path, "")
	ctx.ReqRes.Session.AddParameterToDelete(acsxml.ParameterValueStruct{Name: path})

	gpnTask := tasks.NewCPETask(ctx.ReqRes.Session.CPE.UUID)
	gpnTask.AsGetParameterNames(path)
	ctx.ReqRes.Session.AddTask(gpnTask)

	return body, nil
}

// DownloadTask handles both firmware Download and UploadFirmware task type strings -
// upload is a Download RPC with a different filetype, exactly as PHP models it.
type DownloadTask struct{ dbTaskAdapter }

func (t DownloadTask) ToRequest(ctx *RunContext) (string, error) {
	ctx.ReqRes.Session.PrevState = acsxml.Download

	filename, _ := t.db.Payload["filename"].(string)
	filetype, _ := t.db.Payload["filetype"].(string)

	url, err := lib.GetFileUrl(filename, ctx.ReqRes.Request)
	if err != nil {
		return "", err
	}

	return ctx.ReqRes.Envelope.DownloadRequest(acsxml.DownloadRequestStruct{
		FileType: filetype,
		URL:      url,
	}), nil
}

// RebootTask was previously unreachable: the "Reboot" task type constant existed but
// had no XML builder and no case in the dispatcher's if/else chain.
type RebootTask struct{ dbTaskAdapter }

func (t RebootTask) ToRequest(ctx *RunContext) (string, error) {
	ctx.ReqRes.Session.PrevState = acsxml.Reboot
	commandKey, _ := t.db.Payload["command_key"].(string)
	return ctx.ReqRes.Envelope.RebootRequest(commandKey), nil
}

type RunScriptTask struct{ dbTaskAdapter }

func (t RunScriptTask) ScriptSource() string {
	script, _ := t.db.Payload["script"].(string)
	return script
}

// UnknownTask is returned for any task type string not recognised above. The
// TaskRunner treats it as a no-op it can safely mark done and skip, matching the old
// dispatcher's behavior of silently falling through unmatched if/else branches.
type UnknownTask struct{ dbTaskAdapter }

// Wrap adapts a DB-backed tasks.Task (whether loaded from the tasks table or
// constructed ad-hoc mid-session via tasks.NewCPETask().AsXxx()) into the polymorphic
// Task interface the TaskRunner dispatches on. Mirrors PHP's Task::toACSTask().
func Wrap(db tasks.Task) Task {
	adapter := dbTaskAdapter{db: db}

	switch db.Task {
	case acsxml.InformResp:
		return InformResponseTask{adapter}
	case acsxml.GPNReq:
		return GetParameterNamesTask{adapter}
	case acsxml.GPVReq:
		return GetParameterValuesTask{adapter}
	case acsxml.SPVReq:
		return SetParameterValuesTask{adapter}
	case tasks.AddObject, acsxml.AddObjReq:
		return AddObjectTask{adapter}
	case tasks.DeleteObject, acsxml.DelObjReq:
		return DeleteObjectTask{adapter}
	case acsxml.Download, tasks.UploadFirmware:
		return DownloadTask{adapter}
	case acsxml.Reboot:
		return RebootTask{adapter}
	case tasks.RunScript:
		return RunScriptTask{adapter}
	default:
		return UnknownTask{adapter}
	}
}
