package methods

import (
	"encoding/xml"
	acscontext "goacs/acs/context"
	"goacs/acs/http"
	acsxml "goacs/acs/types"
	"goacs/lib"
	"goacs/lib/cache"
	"goacs/models/tasks"
	"goacs/repository/mysql"
	"log"
)

type InformDecision struct {
	ReqRes *http.CPERequest
}

func (InformDecision *InformDecision) CpeInformResponse() string {
	InformDecision.ReqRes.Session.PrevState = acsxml.InformReq
	return InformDecision.ReqRes.Envelope.InformResponse()
}

func (InformDecision *InformDecision) CpeInformRequestParser() {
	env := new(lib.Env)
	var inform acsxml.Inform
	_ = xml.Unmarshal(InformDecision.ReqRes.Body, &inform)
	log.Println("SESSION FROM InformReq", InformDecision.ReqRes.Session.IsNew, InformDecision.ReqRes.Session.ReadAllParameters)

	InformDecision.ReqRes.Session.FillCPESessionFromInform(inform)
	cpeRepository := mysql.NewCPERepository(InformDecision.ReqRes.DBConnection)
	_, cpeExist, _ := cpeRepository.UpdateOrCreate(&InformDecision.ReqRes.Session.CPE)
	InformDecision.ReqRes.Session.ReadAllParameters = !cpeExist
	InformDecision.ReqRes.Session.IsNewInACS = !cpeExist
	InformDecision.ReqRes.Session.Provision = !cpeExist
	InformDecision.ReqRes.Session.IsNew = false

	if env.Get("DEBUG", "false") == "true" {
		InformDecision.ReqRes.Session.IsBoot = true
		InformDecision.ReqRes.Session.EnsureEventCode("1 BOOT")
	}

	// GET /api/device/:uuid/provision armed this flag before kicking the device:
	// treat this Inform like a boot so that a provision scoped to "1 BOOT" matches
	// and its own script does whatever discovery/setup it needs - there is no
	// separate ACS-side walk anymore, forcing IsBoot only affects provisioning
	// rule matching (see Session.EnsureEventCode).
	if _, forced := cache.Global.Get(acscontext.KeyFor(acscontext.ProvisionPrefix, InformDecision.ReqRes.Session.CPE.SerialNumber)); forced {
		InformDecision.ReqRes.Session.IsBoot = true
		InformDecision.ReqRes.Session.EnsureEventCode("1 BOOT")
	}

	// GET /api/device/:uuid/lookup armed this flag before kicking the device:
	// also force the full walk below, but in a read-only mode - see
	// Session.LookupOnly and the guards in acs/methods/parametermethods.go -
	// so the results only ever reach the LookupParamsPrefix cache, never
	// cpe_parameters.
	if _, lookup := cache.Global.Get(acscontext.KeyFor(acscontext.LookupParamsEnabledPrefix, InformDecision.ReqRes.Session.CPE.SerialNumber)); lookup {
		InformDecision.ReqRes.Session.LookupOnly = true
	}

	_, _ = cpeRepository.SaveParameters(&InformDecision.ReqRes.Session.CPE)
	task := tasks.NewCPETask(InformDecision.ReqRes.Session.CPE.UUID)
	task.Task = acsxml.InformResp
	InformDecision.ReqRes.Session.AddTask(task)

}
