package logic

import (
	"encoding/xml"
	"fmt"
	"goacs/acs"
	acshttp "goacs/acs/http"
	"goacs/acs/methods"
	acsxml "goacs/acs/types"
	"goacs/repository"
	"io"
	"io/ioutil"
	"log"
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
		log.Println("INVALID SESSION")
		reqRes.SendResponse("")
	}

	acs.AddCookieToResponseWriter(reqRes.Session, reqRes.Response)

	switch reqType {
	case acsxml.InformReq:
		informDecision := methods.InformDecision{ReqRes: &reqRes}
		informDecision.CpeInformRequestParser()

		if err := NewProvisionMatcher(&reqRes).QueueTasks(reqRes.Session.CurrentEventCodes, acsxml.InformReq); err != nil {
			log.Println("provision matcher error (Inform):", err)
		}

	case acsxml.EMPTY:
		log.Println("EMPTY RESPONSE")
		if len(session.Tasks) == 0 {
			acs.DeleteSession(session.Id)
		}

	case acsxml.GPNResp:
		parameterDecisions := methods.ParameterDecisions{ReqRes: &reqRes}
		parameterDecisions.CpeParameterNamesResponseParser()

	case acsxml.GPVResp:
		parameterDecisions := methods.ParameterDecisions{ReqRes: &reqRes}
		parameterDecisions.GetParameterValuesResponseParser()

	case acsxml.AddObjResp:
		log.Println("AddObjResp")
		log.Println(string(reqRes.Body))
		parameterDecisions := methods.ParameterDecisions{ReqRes: &reqRes}
		parameterDecisions.AddObjectResponseParser()

	case acsxml.DownloadResp:
		log.Println("DownloadResponse")
		log.Println(string(reqRes.Body))

	case acsxml.RebootResp:
		log.Println("RebootResponse")

	case acsxml.TransferComplete:
		log.Println("TransferComplete")
		log.Println(string(reqRes.Body))
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

func parseBody(buffer []byte) (string, acsxml.Envelope) {
	log.Println("Parsing body")
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
	log.Println("body parsed")
	return requestType, envelope
}
