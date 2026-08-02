// Package queue provides the polymorphic Task Queue abstraction driving a CWMP
// session, mirroring goacs-php's App\ACS\Entities\Tasks (Task/WithRequest/WithResponse)
// and App\ACS\Logic\TaskRunner. It wraps the existing DB-backed models/tasks.Task
// (which stays exactly as-is for repository/sqlx compatibility) with behavior, so the
// TaskRunner can dispatch on interface rather than a growing if/else chain on strings.
package queue

import (
	acshttp "goacs/acs/http"
)

// RunContext carries everything a queued task needs to execute against the current
// HTTP round-trip.
type RunContext struct {
	ReqRes *acshttp.CPERequest
}

// Task is the common contract for anything the TaskRunner can pull off the session
// queue.
type Task interface {
	Name() string
}

// WithResponse produces the SOAP body answering the CPE request that is currently
// being handled (e.g. InformResponse answers an incoming Inform).
type WithResponse interface {
	Task
	ToResponse(ctx *RunContext) (string, error)
}

// WithRequest produces a new ACS -> CPE request sent as the body of the current HTTP
// response (e.g. GetParameterNames, AddObject, Download, Reboot).
type WithRequest interface {
	Task
	ToRequest(ctx *RunContext) (string, error)
}

// ScriptTask marks a task that runs a provisioning script. Recognised specially by the
// TaskRunner ahead of the goroutine+channel bridge that will land in a later phase.
type ScriptTask interface {
	Task
	ScriptSource() string
}
