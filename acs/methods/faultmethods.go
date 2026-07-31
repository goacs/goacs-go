package methods

import (
	"goacs/acs/http"
	acsxml "goacs/acs/types"
	"goacs/models/log"
	"goacs/repository"
	"goacs/repository/mysql"
	stdlog "log"
)

type FaultDecision struct {
	ReqRes *http.CPERequest
}

func (FaultDecision *FaultDecision) ResponseDecision() {
	stdlog.Print(string(FaultDecision.ReqRes.Body))
	FaultDecision.ReqRes.Session.PrevState = acsxml.FaultResp
	faultRepository := mysql.NewFaultRepository()
	faultRepository.SaveFault(&FaultDecision.ReqRes.Session.CPE,
		FaultDecision.ReqRes.Session.CPE.Fault.DetailFaultCode,
		FaultDecision.ReqRes.Session.CPE.Fault.DetailFaultString,
	)

	if repository.HasConnection() {
		logRepository := mysql.NewLogRepository(repository.GetConnection())
		_ = logRepository.Save(&log.Log{
			CPEUUID:   FaultDecision.ReqRes.Session.CPE.UUID,
			FullXML:   string(FaultDecision.ReqRes.Body),
			Code:      FaultDecision.ReqRes.Session.CPE.Fault.DetailFaultCode,
			Message:   FaultDecision.ReqRes.Session.CPE.Fault.DetailFaultString,
			Type:      log.TypeFault,
			From:      log.FromDevice,
			SessionId: FaultDecision.ReqRes.Session.Id,
		})
	}
}
