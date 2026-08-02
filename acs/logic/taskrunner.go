package logic

import (
	acshttp "goacs/acs/http"
	"goacs/acs/methods"
	"goacs/acs/queue"
	"goacs/acs/scripts"
	acsxml "goacs/acs/types"
	"goacs/models/tasks"
	"goacs/repository/mysql"
	"log"
	"strconv"

	"github.com/jmoiron/sqlx"
)

// DefaultRunScriptMaxCount is the fallback used when the "run_script_max_count" config
// key (Settings screen) is unset or invalid - a safety cap against a provisioning rule
// that would otherwise re-queue itself forever within a single HTTP round-trip.
const DefaultRunScriptMaxCount = 30

const runScriptMaxCountConfigKey = "run_script_max_count"

// runScriptMaxCount reads a positive integer from the "run_script_max_count" config key,
// falling back to DefaultRunScriptMaxCount if it's missing, non-numeric, or non-positive
// - same live-lookup-with-fallback pattern as scripts.configuredTimeoutSeconds. Config is
// re-read on every call (not cached), so changing it from the admin panel takes effect
// on the next round-trip without a restart.
func runScriptMaxCount(db *sqlx.DB) int {
	configRepository := mysql.NewConfigRepository(db)
	value, err := configRepository.GetValue(runScriptMaxCountConfigKey)
	if err != nil {
		return DefaultRunScriptMaxCount
	}

	count, err := strconv.Atoi(value)
	if err != nil || count <= 0 {
		return DefaultRunScriptMaxCount
	}

	return count
}

// TaskRunner drains the session's task queue for the current HTTP round-trip, mirroring
// goacs-php's App\ACS\Logic\TaskRunner::run(). CWMP is one SOAP body per HTTP response,
// so it recurses through "quiet" tasks (currently only RunScript) that don't write to
// the response, and stops as soon as one task actually sends a body.
type TaskRunner struct {
	reqRes *acshttp.CPERequest
	event  string
}

func NewTaskRunner(reqRes *acshttp.CPERequest, event string) *TaskRunner {
	return &TaskRunner{reqRes: reqRes, event: event}
}

func (tr *TaskRunner) Run() {
	tr.loadDeviceTasks()

	if len(tr.reqRes.Session.Tasks) == 0 {
		tr.runSetParamsProvisioningOnce()
		return
	}

	dbTask := tr.reqRes.Session.Tasks[0]
	tr.reqRes.Session.Tasks = tr.reqRes.Session.Tasks[1:]

	task := queue.Wrap(dbTask)
	ctx := &queue.RunContext{ReqRes: tr.reqRes}

	log.Println("Processing task:", task.Name())

	switch t := task.(type) {
	case queue.ScriptTask:
		tr.runScriptTask(t)

	case queue.WithResponse:
		body, err := t.ToResponse(ctx)
		if err != nil {
			log.Println("task response error:", err)
		}
		tr.reqRes.SendResponse(body)
		tr.markDone(dbTask)

	case queue.WithRequest:
		body, err := t.ToRequest(ctx)
		if err != nil {
			log.Println("task request error:", err)
			tr.markDone(dbTask)
			return
		}
		tr.reqRes.SendResponse(body)
		tr.markDone(dbTask)

	default:
		// Unrecognised task type: nothing to send, stop here without recursing -
		// matches the legacy dispatcher's behavior of silently falling through an
		// unmatched if/else chain (no response written, HTTP round-trip just ends).
		tr.markDone(dbTask)
	}
}

func (tr *TaskRunner) runScriptTask(t queue.ScriptTask) {
	if tr.reqRes.Session.RunnedScripts >= runScriptMaxCount(tr.reqRes.DBConnection) {
		// Script budget for this session exhausted - stop without writing a response,
		// same as the legacy dispatcher (RunScript matched no other if/else branch
		// once the count guard failed).
		return
	}

	tr.reqRes.Session.RunnedScripts++

	finished, err := scripts.Start(tr.reqRes, t.ScriptSource())
	if err != nil {
		log.Println("script execution error:", err)
		scripts.LogScriptError(tr.reqRes, err)
	}

	if !finished {
		// The script issued a blocking RPC (addObject, reboot, ...): its request has
		// already been written as this round-trip's response by scripts.Start(), and
		// the script goroutine is parked waiting for the CPE's reply on a future
		// request. Nothing more to do for this HTTP round-trip.
		return
	}

	parameterDecisions := methods.ParameterDecisions{ReqRes: tr.reqRes}
	parameterDecisions.PrepareParametersToSend()

	log.Println("RunnedScripts", tr.reqRes.Session.RunnedScripts)

	tr.Run() // RunScript is "quiet" - keep draining the queue within this round-trip.
}

// loadDeviceTasks pulls due, not-yet-done tasks for this CPE (plus the one-shot global
// "new device" task) from the DB into the session queue. Called again on every
// recursive Run(), so tasks already queued earlier in THIS request (but not yet marked
// done, because they haven't been reached yet) must not be re-added - otherwise a
// RunScript task still sitting in the queue would be fetched, queued and executed a
// second time before its own DoneTask() commits.
func (tr *TaskRunner) loadDeviceTasks() {
	tasksRepository := mysql.NewTasksRepository(tr.reqRes.DBConnection)
	cpeDatabaseTasks := tasksRepository.GetTasksForCPE(tr.reqRes.Session.CPE.UUID)

	if tr.reqRes.Session.IsNewInACS && tr.event == acsxml.GPVResp {
		newDeviceTask := tasksRepository.GetGlobalTask("new")
		if !tr.reqRes.Session.TaskExist(newDeviceTask) {
			tr.reqRes.Session.AddTask(newDeviceTask)
		}
	}

	filteredTasks := tasks.FilterTasksByEvent(tr.event, cpeDatabaseTasks)
	for _, cpeTask := range filteredTasks {
		if !tr.reqRes.Session.TaskExist(cpeTask) {
			tr.reqRes.Session.AddTask(cpeTask)
		}
	}
}

// runSetParamsProvisioningOnce fires exactly once per session, at the point where the
// task queue has drained naturally (GPN/GPV cascades finished, no operator tasks left)
// and freshly-read parameter values are available. Mirrors the fallback branch in
// goacs-php's TaskRunner::run() that queues provisions filtered on
// Types::SetParameterValuesProcessor once nothing else is pending.
func (tr *TaskRunner) runSetParamsProvisioningOnce() {
	if tr.reqRes.Session.ProvisionedSetParams {
		return
	}
	tr.reqRes.Session.ProvisionedSetParams = true

	matcher := NewProvisionMatcher(tr.reqRes)
	if err := matcher.QueueTasks(tr.reqRes.Session.CurrentEventCodes, acsxml.SetParameterValuesProcessor); err != nil {
		log.Println("provision matcher error (SetParameterValuesProcessor):", err)
		return
	}

	if len(tr.reqRes.Session.Tasks) > 0 {
		tr.Run()
	}
}

func (tr *TaskRunner) markDone(dbTask tasks.Task) {
	if dbTask.Id != 0 && !dbTask.Infinite {
		tasksRepository := mysql.NewTasksRepository(tr.reqRes.DBConnection)
		tasksRepository.DoneTask(dbTask.Id)
	}
}
