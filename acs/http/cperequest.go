package http

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"goacs/acs"
	acsxml "goacs/acs/types"
	"goacs/models/log"
	"goacs/repository"
	"goacs/repository/mysql"
	"net/http"
)

type CPERequest struct {
	Request      *http.Request
	Response     http.ResponseWriter
	DBConnection *sqlx.DB
	Session      *acs.ACSSession
	Envelope     *acsxml.Envelope
	Body         []byte
	ReqType      string
}

func (r *CPERequest) SendResponse(body string) {
	//log.Println(body)
	r.logResponseConversation(body)
	_, _ = fmt.Fprint(r.Response, body)
}

// logResponseConversation mirrors the request-side hook in
// acs/logic.logConversation for the outbound leg of the exchange.
func (r *CPERequest) logResponseConversation(body string) {
	if !repository.HasConnection() {
		return
	}

	logRepository := mysql.NewLogRepository(repository.GetConnection())
	if !logRepository.ConversationLoggingEnabled(&r.Session.CPE) {
		return
	}

	_ = logRepository.Save(&log.Log{
		CPEUUID:   r.Session.CPE.UUID,
		FullXML:   body,
		Message:   acsxml.ParseRPCName([]byte(body)),
		Type:      log.TypeInfo,
		From:      log.FromACS,
		SessionId: r.Session.Id,
	})
}
