package scripts

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	acshttp "goacs/acs/http"
	"goacs/acs/methods"
	acsxml "goacs/acs/types"
	"goacs/repository"
	"goacs/repository/mysql"
	"log"
	"strconv"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// DefaultScriptTotalTimeout is the fallback used when the "script_total_timeout_seconds"
// config key (Settings screen) is unset or invalid. Bounds the entire lifetime of one
// script execution, including every blocking RPC round-trip it makes - not just CPU
// time. If the CPE never answers a blocking call within this budget, the script is
// aborted. Raise this via config for devices that are known to take a long time to
// reply mid-script (e.g. a firmware upgrade whose reboot/TransferComplete round-trip is
// slow) rather than editing the default.
const DefaultScriptTotalTimeout = 5 * time.Minute

const scriptTotalTimeoutConfigKey = "script_total_timeout_seconds"

// DefaultLocalStepTimeout is the fallback used when the "script_local_step_timeout_seconds"
// config key is unset or invalid. Guards against a script that neither finishes nor
// calls a blocking function (e.g. an accidental infinite loop) - this is a local
// CPU-bound wait, so it can usually stay short even when the total timeout above is
// raised for slow-updating devices.
const DefaultLocalStepTimeout = 5 * time.Second

const localStepTimeoutConfigKey = "script_local_step_timeout_seconds"

// configuredTimeoutSeconds reads a positive integer number of seconds from the given
// config key, falling back to def if there's no DB connection or the value is missing,
// non-numeric, or non-positive - same live-lookup-with-fallback pattern as piiValue
// (functions.go). Config is re-read on every call (not cached), so changing it from the
// admin panel takes effect on the next script/round-trip without a restart.
func configuredTimeoutSeconds(key string, def time.Duration) time.Duration {
	if !repository.HasConnection() {
		return def
	}

	configRepository := mysql.NewConfigRepository(repository.GetConnection())
	value, err := configRepository.GetValue(key)
	if err != nil {
		return def
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return def
	}

	return time.Duration(seconds) * time.Second
}

type pendingCall struct {
	kind       string
	requestXML string
	resultCh   chan rpcResult
	issuedAt   time.Time
}

type rpcResult struct {
	addObject          *acsxml.AddObjectResponseStruct
	deleteObject       *acsxml.DeleteObjectResponseStruct
	getParameterValues *acsxml.GetParameterValuesResponse
	setParameterValues *acsxml.SetParameterValuesResponseStruct
	fault              *acsxml.Fault
	err                error
}

type outcome struct{ err error }

// Session is the suspended state of a script execution that has issued a blocking RPC
// and is waiting for the CPE's response, which arrives as a separate HTTP request.
// It implements acs.ScriptSession so acs.ACSSession.Script can reference it without an
// import cycle.
type Session struct {
	cancel  context.CancelFunc
	fromLua chan *pendingCall // script goroutine -> pump: "send this, then wait for a reply"
	done    chan outcome      // script goroutine -> pump: "I'm finished" (buffered, never blocks the goroutine)
	pending *pendingCall      // the call currently awaiting a CPE response, if any
	bc      *bridgeContext
}

func (s *Session) Cancel() { s.cancel() }

// bridgeContext is what the blocking Lua-facing functions close over. reqRes is
// mutated in place by Resume() on every round-trip so a script that issues several
// blocking calls always builds its next request against the CURRENT round-trip's
// envelope - the script goroutine only ever touches it while it holds the baton
// (i.e. after being unblocked via resultCh and before blocking again), so this is
// data-race free without extra locking.
type bridgeContext struct {
	reqRes *acshttp.CPERequest
	ctx    context.Context
	bs     *Session

	// lastFault is the CPE Fault (if any) returned by the most recent blocking call,
	// set by call() right when logDeviceFault persists it. runScript consults this
	// after the script ends to tell whether the error that ended it already has a
	// device-log entry (see scriptFaultLoggedError).
	lastFault *acsxml.Fault
}

// Start begins executing a script in its own goroutine. It blocks the calling
// goroutine (the HTTP handler, via TaskRunner) until the script either finishes within
// this same round-trip (finished=true, no blocking call was made) or issues its first
// blocking RPC, in which case that RPC's request XML has already been written as this
// round-trip's HTTP response (finished=false) and the script goroutine is parked,
// waiting for a future round-trip to deliver the CPE's reply via Resume().
func Start(reqRes *acshttp.CPERequest, script string) (finished bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), configuredTimeoutSeconds(scriptTotalTimeoutConfigKey, DefaultScriptTotalTimeout))

	bc := &bridgeContext{reqRes: reqRes, ctx: ctx}
	bs := &Session{
		cancel:  cancel,
		fromLua: make(chan *pendingCall), // unbuffered: forces the script goroutine to
		// wait until the pump has taken the call before it can block on the reply,
		// so exactly one goroutine touches session/bc state at a time.
		done: make(chan outcome, 1),
		bc:   bc,
	}
	bc.bs = bs

	go runScript(bc, script)

	return pump(bs, reqRes)
}

// Resume is called by the dispatcher when session.Script is already set, meaning a
// script is suspended waiting for exactly this round-trip's response.
func Resume(reqRes *acshttp.CPERequest) (finished bool, err error) {
	bs, ok := reqRes.Session.Script.(*Session)
	if !ok || bs == nil {
		return true, errors.New("bridge: Resume called without a suspended script")
	}

	call := bs.pending
	if call == nil {
		return true, errors.New("bridge: Resume called but no pending call was recorded")
	}

	result := matchResponse(call.kind, reqRes)
	bs.pending = nil
	bs.bc.reqRes = reqRes // safe: the script goroutine is parked on call.resultCh below

	select {
	case call.resultCh <- result:
	default:
		// Should never happen (resultCh has capacity 1 and is only written once),
		// but never block the dispatcher goroutine on it regardless.
	}

	return pump(bs, reqRes)
}

func pump(bs *Session, reqRes *acshttp.CPERequest) (finished bool, err error) {
	localStepTimeout := configuredTimeoutSeconds(localStepTimeoutConfigKey, DefaultLocalStepTimeout)

	select {
	case call := <-bs.fromLua:
		bs.pending = call
		reqRes.Session.Script = bs
		reqRes.SendResponse(call.requestXML)
		return false, nil

	case o := <-bs.done:
		reqRes.Session.Script = nil
		return true, o.err

	case <-time.After(localStepTimeout):
		bs.cancel()
		reqRes.Session.Script = nil
		return true, fmt.Errorf("script did not call a blocking function or finish within %s", localStepTimeout)
	}
}

func runScript(bc *bridgeContext, script string) {
	defer func() {
		if r := recover(); r != nil {
			bc.bs.done <- outcome{err: fmt.Errorf("script panic: %v", r)}
		}
	}()

	se := NewScriptEngine(bc.reqRes)

	L := lua.NewState()
	defer L.Close()

	se.registerGlobals(L)
	se.registerFunctions(L)
	registerBlockingFunctions(L, bc)

	err := L.DoString(script)
	if err != nil && bc.lastFault != nil {
		err = scriptFaultLoggedError{err}
	}
	bc.bs.done <- outcome{err: err}
}

// call is used by every blocking Lua-facing function: it builds the request XML
// against the CURRENT round-trip's envelope, hands it to the pump, and blocks this
// goroutine until either a reply arrives or the script's overall timeout expires.
func (bc *bridgeContext) call(kind string, buildXML func(*acshttp.CPERequest) string) (rpcResult, error) {
	bc.lastFault = nil

	call := &pendingCall{
		kind:       kind,
		requestXML: buildXML(bc.reqRes),
		resultCh:   make(chan rpcResult, 1),
		issuedAt:   time.Now(),
	}

	select {
	case bc.bs.fromLua <- call:
	case <-bc.ctx.Done():
		return rpcResult{}, bc.ctx.Err()
	}

	select {
	case res := <-call.resultCh:
		if res.fault != nil {
			bc.lastFault = res.fault
			logDeviceFault(bc.reqRes, res.fault)
		}
		return res, res.err
	case <-bc.ctx.Done():
		return rpcResult{}, bc.ctx.Err()
	}
}

// matchResponse interprets the current round-trip's parsed body as the CPE's reply to
// the given pending call kind. A CPE Fault is delivered to the script as a value
// (result.fault), not a Go error, so provisioning scripts can react to it; anything
// else unexpected (a protocol violation - e.g. the CPE sent something unrelated) is an
// error. Known, narrow limitation: on a protocol violation this round-trip is treated
// as belonging to the suspended script and is not re-routed to the normal dispatcher
// path, even if it was e.g. a genuine new Inform - the CPE gets an empty response and
// is expected to retry.
func matchResponse(kind string, reqRes *acshttp.CPERequest) rpcResult {
	if reqRes.ReqType == acsxml.FaultResp {
		var fault acsxml.Fault
		if err := xml.Unmarshal(reqRes.Body, &fault); err != nil {
			return rpcResult{err: err}
		}
		return rpcResult{fault: &fault}
	}

	switch kind {
	case "AddObject":
		if reqRes.ReqType != acsxml.AddObjResp {
			return rpcResult{err: protocolViolation(kind, reqRes.ReqType)}
		}
		var v acsxml.AddObjectResponseStruct
		if err := xml.Unmarshal(reqRes.Body, &v); err != nil {
			return rpcResult{err: err}
		}
		return rpcResult{addObject: &v}

	case "DeleteObject":
		if reqRes.ReqType != acsxml.DelObjResp {
			return rpcResult{err: protocolViolation(kind, reqRes.ReqType)}
		}
		var v acsxml.DeleteObjectResponseStruct
		if err := xml.Unmarshal(reqRes.Body, &v); err != nil {
			return rpcResult{err: err}
		}
		return rpcResult{deleteObject: &v}

	case "Reboot":
		if reqRes.ReqType != acsxml.RebootResp {
			return rpcResult{err: protocolViolation(kind, reqRes.ReqType)}
		}
		return rpcResult{}

	case "GetParameterValues":
		if reqRes.ReqType != acsxml.GPVResp {
			return rpcResult{err: protocolViolation(kind, reqRes.ReqType)}
		}
		var v acsxml.GetParameterValuesResponse
		if err := xml.Unmarshal(reqRes.Body, &v); err != nil {
			return rpcResult{err: err}
		}
		return rpcResult{getParameterValues: &v}

	case "SetParameterValues":
		if reqRes.ReqType != acsxml.SPVResp {
			return rpcResult{err: protocolViolation(kind, reqRes.ReqType)}
		}
		var v acsxml.SetParameterValuesResponseStruct
		if err := xml.Unmarshal(reqRes.Body, &v); err != nil {
			return rpcResult{err: err}
		}
		return rpcResult{setParameterValues: &v}

	default:
		return rpcResult{err: fmt.Errorf("bridge: unknown pending call kind %q", kind)}
	}
}

func protocolViolation(expectedKind, gotReqType string) error {
	return fmt.Errorf("protocol violation: expected the CPE's response to %s, got %s", expectedKind, gotReqType)
}

func registerBlockingFunctions(L *lua.LState, bc *bridgeContext) {
	L.SetGlobal("addObject", L.NewFunction(func(L *lua.LState) int {
		path := L.CheckString(1)

		result, err := bc.call("AddObject", func(reqRes *acshttp.CPERequest) string {
			return reqRes.Envelope.AddObjectRequest(path, "")
		})
		if err != nil {
			L.RaiseError("addObject(%q): %v", path, err)
			return 0
		}
		if result.fault != nil {
			L.RaiseError("addObject(%q): CPE fault %s: %s", path, result.fault.DetailFaultCode, result.fault.DetailFaultString)
			return 0
		}

		instance := result.addObject.InstanceNumber
		tbl := L.NewTable()
		L.SetField(tbl, "instance", lua.LNumber(instance))
		L.SetField(tbl, "status", lua.LNumber(result.addObject.Status))
		L.SetField(tbl, "path", lua.LString(fmt.Sprintf("%s%d.", path, instance)))
		L.Push(tbl)
		return 1
	}))

	L.SetGlobal("deleteObject", L.NewFunction(func(L *lua.LState) int {
		path := L.CheckString(1)

		result, err := bc.call("DeleteObject", func(reqRes *acshttp.CPERequest) string {
			return reqRes.Envelope.DeleteObjectRequest(path, "")
		})
		if err != nil {
			L.RaiseError("deleteObject(%q): %v", path, err)
			return 0
		}
		if result.fault != nil {
			L.RaiseError("deleteObject(%q): CPE fault %s: %s", path, result.fault.DetailFaultCode, result.fault.DetailFaultString)
			return 0
		}

		L.Push(lua.LNumber(result.deleteObject.Status))
		return 1
	}))

	L.SetGlobal("reboot", L.NewFunction(func(L *lua.LState) int {
		commandKey := L.OptString(1, "")

		result, err := bc.call("Reboot", func(reqRes *acshttp.CPERequest) string {
			return reqRes.Envelope.RebootRequest(commandKey)
		})
		if err != nil {
			L.RaiseError("reboot(): %v", err)
			return 0
		}
		if result.fault != nil {
			L.RaiseError("reboot(): CPE fault %s: %s", result.fault.DetailFaultCode, result.fault.DetailFaultString)
			return 0
		}

		return 0
	}))

	// getParameterValues issues a real GetParameterValues RPC to the CPE and blocks
	// for its reply, unlike getParameterValue (functions.go) which only ever reads the
	// session's local cache/DB. Every value returned is merged into the session's local
	// cache and persisted the same way the ordinary GetParameterValues walk does
	// (methods.ParameterDecisions.PersistFetchedParameterValues, shared with
	// GetParameterValuesResponseParser) - this dispatch bypasses that normal
	// switch-based path entirely (acs/logic/dispatcher.go's session.Script != nil
	// branch), so without this call the values would only ever live in memory for the
	// rest of this session.
	L.SetGlobal("getParameterValues", L.NewFunction(func(L *lua.LState) int {
		n := L.GetTop()
		if n == 0 {
			L.RaiseError("getParameterValues(): at least one parameter path is required")
			return 0
		}

		infos := make([]acsxml.ParameterInfo, n)
		for i := 1; i <= n; i++ {
			infos[i-1] = acsxml.ParameterInfo{Name: L.CheckString(i)}
		}

		result, err := bc.call("GetParameterValues", func(reqRes *acshttp.CPERequest) string {
			return reqRes.Envelope.GPVRequest(infos)
		})
		if err != nil {
			L.RaiseError("getParameterValues(): %v", err)
			return 0
		}
		if result.fault != nil {
			L.RaiseError("getParameterValues(): CPE fault %s: %s", result.fault.DetailFaultCode, result.fault.DetailFaultString)
			return 0
		}

		(&methods.ParameterDecisions{ReqRes: bc.reqRes}).PersistFetchedParameterValues(result.getParameterValues.ParameterList)

		tbl := L.NewTable()
		for _, parameter := range result.getParameterValues.ParameterList {
			L.SetField(tbl, parameter.Name, lua.LString(parameter.ValueStruct.Value))
		}
		L.Push(tbl)
		return 1
	}))

	// setParameterValues issues a real SetParameterValues RPC to the CPE and blocks for
	// its reply, unlike setParameter (functions.go) which only updates the local
	// cache/DB and defers the actual CWMP write to a later round-trip via the task
	// queue. Takes a table of {[path] = value}. On a confirmed (non-fault) reply, the
	// local cache is updated - and persisted to the DB, same as setParameter - so the
	// rest of the script and the admin panel both see the value the CPE just confirmed.
	L.SetGlobal("setParameterValues", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)

		var parameters []acsxml.ParameterValueStruct
		var rangeErr error
		tbl.ForEach(func(k, v lua.LValue) {
			if rangeErr != nil {
				return
			}
			if k.Type() != lua.LTString {
				rangeErr = fmt.Errorf("setParameterValues(): table keys must be parameter path strings, got %s", k.Type())
				return
			}

			name := k.String()
			parameter := acsxml.ParameterValueStruct{
				Name:        name,
				ValueStruct: acsxml.ValueStruct{Value: v.String()},
			}
			if current := bc.reqRes.Session.CPE.GetParameter(name); current != nil {
				parameter.ValueStruct.Type = current.ValueStruct.Type
			}
			parameters = append(parameters, parameter)
		})
		if rangeErr != nil {
			L.RaiseError("%v", rangeErr)
			return 0
		}
		if len(parameters) == 0 {
			L.RaiseError("setParameterValues(): at least one parameter is required")
			return 0
		}

		result, err := bc.call("SetParameterValues", func(reqRes *acshttp.CPERequest) string {
			return reqRes.Envelope.SetParameterValues(parameters)
		})
		if err != nil {
			L.RaiseError("setParameterValues(): %v", err)
			return 0
		}
		if result.fault != nil {
			L.RaiseError("setParameterValues(): CPE fault %s: %s", result.fault.DetailFaultCode, result.fault.DetailFaultString)
			return 0
		}

		for _, parameter := range parameters {
			bc.reqRes.Session.CPE.AddParameter(parameter)
			if repository.HasConnection() {
				cpeRepository := mysql.NewCPERepository(repository.GetConnection())
				if _, err := cpeRepository.UpdateParameter(&bc.reqRes.Session.CPE, parameter); err != nil {
					log.Println("setParameterValues: update error:", err)
				}
			}
		}

		L.Push(lua.LNumber(result.setParameterValues.Status))
		return 1
	}))
}
