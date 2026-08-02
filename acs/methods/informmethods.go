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
	}

	// GET /api/device/:uuid/provision armed this flag before kicking the device:
	// treat this Inform like a boot, forcing the full GetParameterNames/
	// GetParameterValues walk below instead of waiting for the next periodic one.
	if _, forced := cache.Global.Get(acscontext.KeyFor(acscontext.ProvisionPrefix, InformDecision.ReqRes.Session.CPE.SerialNumber)); forced {
		InformDecision.ReqRes.Session.IsBoot = true
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

	// Brand-new devices (IsNewInACS) deliberately do NOT trigger this walk anymore -
	// their parameters are read via the curated "init" Provision instead (see
	// contrib/database/06_init_provision.sql and acs/scripts/README.md), which uses
	// the Lua sandbox's blocking getParameterValues() to fetch just DeviceInfo. and
	// ManagementServer. in one round-trip instead of walking the entire device
	// model one leaf at a time. Already-known devices rebooting (a real IsBoot from
	// "0 BOOTSTRAP"/"1 BOOT"), forced via the admin "provision now" action
	// (GET /api/device/:uuid/provision), or forced via the admin "lookup now"
	// action (GET /api/device/:uuid/lookup, Session.LookupOnly), keep this full
	// walk unchanged - LookupOnly's walk just runs in a read-only mode, see the
	// guards in acs/methods/parametermethods.go.
	if (InformDecision.ReqRes.Session.IsBoot && !InformDecision.ReqRes.Session.IsNewInACS) || InformDecision.ReqRes.Session.LookupOnly {
		//InformDecision.ReqRes.Session.RunGPV = true
		InformDecision.ReqRes.Session.CurrentState = acsxml.GPNReq
		task = tasks.NewCPETask(InformDecision.ReqRes.Session.CPE.UUID)
		task.AsGetParameterNames(InformDecision.ReqRes.Session.CPE.Root + ".")
		task.ParameterInfo = append(task.ParameterInfo, acsxml.ParameterInfo{
			Name: InformDecision.ReqRes.Session.CPE.Root + ".",
			Done: false,
		})
		task.NextLevel = true
		InformDecision.ReqRes.Session.AddTask(task)
	}

}
