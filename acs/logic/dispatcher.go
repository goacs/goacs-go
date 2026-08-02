package logic

import (
	"encoding/xml"
	"fmt"
	"goacs/acs"
	acshttp "goacs/acs/http"
	"goacs/acs/methods"
	"goacs/acs/scripts"
	acsxml "goacs/acs/types"
	"goacs/models/log"
	"goacs/repository"
	"goacs/repository/mysql"
	"io"
	"io/ioutil"
	stdlog "log"
	"net/http"
)

// HandleCPERequest is the single entry point for the /acs endpoint: it parses the
// incoming SOAP envelope, runs the processor matching its type (updating session/DB
// state), then hands off to the TaskRunner to drain the session's task queue for this
// HTTP round-trip. Mirrors goacs-php's App\ACS\Logic\ControllerLogic::process().
func HandleCPERequest(request *http.Request, w http.ResponseWriter) {
	buffer, err := ioutil.ReadAll(request.Body)

	session := acs.GetSessionFromRequest(request)
	if session == nil {
		session = acs.CreateEmptySession(acs.GenerateSessionId())
	}

	if err != io.EOF && err != nil {
		return
	}

	reqType, envelope := parseBody(buffer)

	reqRes := acshttp.CPERequest{
		Request:      request,
		Response:     w,
		DBConnection: repository.GetConnection(),
		Session:      session,
		Envelope:     &envelope,
		Body:         buffer,
		ReqType:      reqType,
	}

	if session.IsNew && reqType != acsxml.InformReq {
		stdlog.Println("INVALID SESSION")
		reqRes.SendResponse("")
	}

	acs.AddCookieToResponseWriter(reqRes.Session, reqRes.Response)

	logConversation(&reqRes, log.FromDevice, buffer)

	if session.Script != nil {
		// A script is suspended waiting for the CPE's reply to a blocking RPC
		// (addObject, reboot, ...) - this round-trip belongs to it, regardless of
		// what envelope type was parsed above. Takes priority over the normal
		// per-type switch below.
		finished, err := scripts.Resume(&reqRes)
		if err != nil {
			stdlog.Println("script resume error:", err)
			scripts.LogScriptError(&reqRes, err)
		}
		if !finished {
			return
		}
		// Mirrors taskrunner.go's runScriptTask: a script that finishes here
		// (having used at least one blocking call) must still get the same
		// template/stored-parameter diff-and-push pass as one that never
		// blocked at all - otherwise a script whose only job is fetching a
		// value via a blocking call could never trigger it.
		parameterDecisions := methods.ParameterDecisions{ReqRes: &reqRes}
		parameterDecisions.PrepareParametersToSend()
		NewTaskRunner(&reqRes, reqType).Run()
		return
	}

	switch reqType {
	case acsxml.InformReq:
		informDecision := methods.InformDecision{ReqRes: &reqRes}
		informDecision.CpeInformRequestParser()

		if err := NewProvisionMatcher(&reqRes).QueueTasks(reqRes.Session.CurrentEventCodes, acsxml.InformReq); err != nil {
			stdlog.Println("provision matcher error (Inform):", err)
		}

	case acsxml.EMPTY:
		stdlog.Println("EMPTY RESPONSE")
		if len(session.Tasks) == 0 {
			acs.DeleteSession(session.Id)
		}

	case acsxml.GPNResp:
		parameterDecisions := methods.ParameterDecisions{ReqRes: &reqRes}
		parameterDecisions.CpeParameterNamesResponseParser()

	case acsxml.GPVResp:
		parameterDecisions := methods.ParameterDecisions{ReqRes: &reqRes}
		parameterDecisions.GetParameterValuesResponseParser()

	case acsxml.SPVResp:
		parameterDecisions := methods.ParameterDecisions{ReqRes: &reqRes}
		parameterDecisions.SetParameterValuesResponseParser()

	case acsxml.AddObjResp:
		stdlog.Println("AddObjResp")
		stdlog.Println(string(reqRes.Body))
		parameterDecisions := methods.ParameterDecisions{ReqRes: &reqRes}
		parameterDecisions.AddObjectResponseParser()

	case acsxml.DownloadResp:
		stdlog.Println("DownloadResponse")
		stdlog.Println(string(reqRes.Body))

	case acsxml.RebootResp:
		stdlog.Println("RebootResponse")

	case acsxml.TransferComplete:
		stdlog.Println("TransferComplete")
		stdlog.Println(string(reqRes.Body))
		reqRes.SendResponse(reqRes.Envelope.TransferCompleteResponse())
		return

	case acsxml.FaultResp:
		var faultresponse acsxml.Fault
		_ = xml.Unmarshal(buffer, &faultresponse)
		reqRes.Session.CPE.Fault = faultresponse
		faultDecision := methods.FaultDecision{ReqRes: &reqRes}
		faultDecision.ResponseDecision()
		if len(session.Tasks) == 0 {
			acs.DeleteSession(session.Id)
		}

	default:
		fmt.Println("UNSUPPORTED REQTYPE ", reqType)
	}

	NewTaskRunner(&reqRes, reqType).Run()
}

// logConversation persists the raw CWMP XML exchange when enabled (globally via
// the "conversation_log" setting, or per-device via cpe.debug) - port of
// goacs-php's Log::logConversation gate. Silently returns on any repository
// error since this is a debugging aid, never allowed to break the CWMP flow.
func logConversation(reqRes *acshttp.CPERequest, from string, body []byte) {
	if !repository.HasConnection() {
		return
	}

	logRepository := mysql.NewLogRepository(repository.GetConnection())
	if !logRepository.ConversationLoggingEnabled(&reqRes.Session.CPE) {
		return
	}

	_ = logRepository.Save(&log.Log{
		CPEUUID:   reqRes.Session.CPE.UUID,
		FullXML:   string(body),
		Type:      log.TypeInfo,
		From:      from,
		SessionId: reqRes.Session.Id,
	})
}

func parseBody(buffer []byte) (string, acsxml.Envelope) {
	stdlog.Println("Parsing body")
	var envelope acsxml.Envelope
	err := xml.Unmarshal(buffer, &envelope)

	requestType := acsxml.EMPTY

	if err == nil {
		switch envelope.Type() {
		case "inform":
			requestType = acsxml.InformReq
		case "getparameternamesresponse":
			requestType = acsxml.GPNResp
		case "getparametervaluesresponse":
			requestType = acsxml.GPVResp
		case "setparametervaluesresponse":
			requestType = acsxml.SPVResp
		case "addobjectresponse":
			requestType = acsxml.AddObjResp
		case "deleteobjectresponse":
			requestType = acsxml.DelObjResp
		case "downloadresponse":
			requestType = acsxml.DownloadResp
		case "rebootresponse":
			requestType = acsxml.RebootResp
		case "transfercomplete":
			requestType = acsxml.TransferComplete
		case "fault":
			requestType = acsxml.FaultResp
		default:
			fmt.Println("UNSUPPORTED envelope type " + envelope.Type())
			requestType = acsxml.UNKNOWN
		}
	}
	stdlog.Println("body parsed")
	return requestType, envelope
}
