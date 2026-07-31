package scripts

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	acshttp "goacs/acs/http"
	acsxml "goacs/acs/types"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// ScriptTotalTimeout bounds the entire lifetime of one script execution, including
// every blocking RPC round-trip it makes - not just CPU time. If the CPE never answers
// a blocking call within this budget, the script is aborted.
const ScriptTotalTimeout = 5 * time.Minute

// LocalStepTimeout guards against a script that neither finishes nor calls a blocking
// function (e.g. an accidental infinite loop) - this is a local CPU-bound wait, so it
// can be short.
const LocalStepTimeout = 5 * time.Second

type pendingCall struct {
	kind       string
	requestXML string
	resultCh   chan rpcResult
	issuedAt   time.Time
}

type rpcResult struct {
	addObject    *acsxml.AddObjectResponseStruct
	deleteObject *acsxml.DeleteObjectResponseStruct
	fault        *acsxml.Fault
	err          error
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
}

// Start begins executing a script in its own goroutine. It blocks the calling
// goroutine (the HTTP handler, via TaskRunner) until the script either finishes within
// this same round-trip (finished=true, no blocking call was made) or issues its first
// blocking RPC, in which case that RPC's request XML has already been written as this
// round-trip's HTTP response (finished=false) and the script goroutine is parked,
// waiting for a future round-trip to deliver the CPE's reply via Resume().
func Start(reqRes *acshttp.CPERequest, script string) (finished bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), ScriptTotalTimeout)

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
	select {
	case call := <-bs.fromLua:
		bs.pending = call
		reqRes.Session.Script = bs
		reqRes.SendResponse(call.requestXML)
		return false, nil

	case o := <-bs.done:
		reqRes.Session.Script = nil
		return true, o.err

	case <-time.After(LocalStepTimeout):
		bs.cancel()
		reqRes.Session.Script = nil
		return true, fmt.Errorf("script did not call a blocking function or finish within %s", LocalStepTimeout)
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
	bc.bs.done <- outcome{err: err}
}

// call is used by every blocking Lua-facing function: it builds the request XML
// against the CURRENT round-trip's envelope, hands it to the pump, and blocks this
// goroutine until either a reply arrives or the script's overall timeout expires.
func (bc *bridgeContext) call(kind string, buildXML func(*acshttp.CPERequest) string) (rpcResult, error) {
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

		_, err := bc.call("Reboot", func(reqRes *acshttp.CPERequest) string {
			return reqRes.Envelope.RebootRequest(commandKey)
		})
		if err != nil {
			L.RaiseError("reboot(): %v", err)
			return 0
		}

		return 0
	}))
}
