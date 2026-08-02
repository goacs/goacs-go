package methods

import (
	"encoding/xml"
	acscontext "goacs/acs/context"
	"goacs/acs/http"
	acsxml "goacs/acs/types"
	"goacs/lib/cache"
	"goacs/models/cpe"
	"goacs/models/tasks"
	"goacs/repository"
	"goacs/repository/mysql"
	"log"
	"strings"
	"time"
)

const MAX_GPN_REQUESTS = 50
const MAX_CHUNK_SIZE = 1

type ParameterDecisions struct {
	ReqRes *http.CPERequest
}

func (pd *ParameterDecisions) ParameterNamesRequest(path string, nextlevel bool) string {
	pd.ReqRes.Session.CurrentState = acsxml.GPNReq
	return pd.ReqRes.Envelope.GPNRequest(path, nextlevel)
}

func (pd *ParameterDecisions) CpeParameterNamesResponseParser() {
	//acsxml.PrintParamsInfo(pd.ReqRes.Session.CPE.ParametersInfo, pd.ReqRes.Session.CPE.SerialNumber)

	var gpnr acsxml.GetParameterNamesResponse
	log.Println("CpeParameterNamesResponseParser")

	_ = xml.Unmarshal(pd.ReqRes.Body, &gpnr)
	pd.ReqRes.Session.CPE.AddParametersInfo(gpnr.ParameterList)
	pd.ReqRes.Session.GPNCount--

	cpeRepository := mysql.NewCPERepository(repository.GetConnection())
	if !pd.ReqRes.Session.LookupOnly {
		_ = cpeRepository.BulkInsertOrUpdateParameters(&pd.ReqRes.Session.CPE, pd.ReqRes.Session.CPE.GetObjectNamesToParameters())
	}

	nextLevelParams := pd.GetNextLevelParams(pd.ReqRes.Session.CPE.ParametersInfo)

	log.Println("Nextlevel", nextLevelParams)
	log.Println("GPNCount", pd.ReqRes.Session.GPNCount)

	if pd.ReqRes.Session.GPNCount < MAX_GPN_REQUESTS && len(nextLevelParams) > 0 {
		for _, nextLevelParam := range nextLevelParams {
			if pd.ReqRes.Session.GPNCount > MAX_GPN_REQUESTS {
				continue
			}

			parameterInfo := acsxml.ParameterInfo{
				Name:     nextLevelParam.Name,
				Writable: nextLevelParam.Writable,
				Done:     false,
			}

			task := tasks.NewCPETask(pd.ReqRes.Session.CPE.UUID)
			task.AsGetParameterNames(parameterInfo.Name)
			pd.ReqRes.Session.AddTask(task)
			pd.ReqRes.Session.AddParameterNamesToQueryValues(parameterInfo)
			pd.ReqRes.Session.GPNCount++
			log.Println("CURRENT GPN COUNT", pd.ReqRes.Session.GPNCount)
			log.Println("added task", task)
		}

		return //if we have nextLevelParams, then prevent GPVReq add task
	}

	if pd.ReqRes.Session.PrevState == acsxml.AddObjReq {
		pd.ReqRes.Session.ParameterNamesToQueryValues = gpnr.ParameterList
	}

	log.Println("Current GPN tasks", pd.ReqRes.Session.Tasks)
	if len(pd.ReqRes.Session.Tasks) == 0 {
		log.Println("ADDING GPVREQ for these params", pd.ReqRes.Session.ParameterNamesToQueryValues)
		for _, parameterNames := range acsxml.ChunkParameterInfo(pd.ReqRes.Session.ParameterNamesToQueryValues, MAX_CHUNK_SIZE) {
			if len(parameterNames) > 0 {
				log.Println("adding gpvreq for new device", parameterNames)
				task := tasks.NewCPETask(pd.ReqRes.Session.CPE.UUID)
				task.Task = acsxml.GPVReq
				task.ParameterInfo = parameterNames
				pd.ReqRes.Session.AddTask(task)
				pd.ReqRes.Session.GPVCount++
			}
		}
	}

}

func (pd *ParameterDecisions) GetNextLevelParams(params []acsxml.ParameterInfo) []acsxml.ParameterInfo {
	var newParams []acsxml.ParameterInfo
	for idx, param := range params {
		if needsToQueryParam(param) {
			log.Println("GetNExtLevelParamName", pd.ReqRes.Session.CPE.ParametersInfo[idx].Name)
			log.Println("DONE", pd.ReqRes.Session.CPE.ParametersInfo[idx].Done)
			pd.ReqRes.Session.CPE.ParametersInfo[idx].Done = true
			newParams = append(newParams, param)
		}
	}
	return newParams
}

func needsToQueryParam(param acsxml.ParameterInfo) bool {
	if param.Name[len(param.Name)-1:] != "." {
		return false
	}
	chunks := strings.Split(param.Name, ".")
	return len(chunks) > 2 && len(chunks) <= 3 && param.Done == false
}

/*func (pd *ParameterDecisions) getParamsBeginningWith(path string) []acsxml.ParameterInfo {
	var params []acsxml.ParameterInfo
	for idx, param := range pd.ReqRes.Session.CPE.ParametersInfo {
		if param.Done == false && param.Writable == "0" && param.Name[len(param.Name)-1:] == "." {
			pd.ReqRes.Session.CPE.ParametersInfo[idx].Done = true
			params = append(params, param)
		}
	}
}*/

func (pd *ParameterDecisions) GetParameterValuesRequest(parameters []acsxml.ParameterInfo) string {
	var request = pd.ReqRes.Envelope.GPVRequest(parameters)
	pd.ReqRes.Session.PrevState = acsxml.GPVReq
	return request
}

func (pd *ParameterDecisions) GetParameterValuesResponseParser() {
	var gpvr acsxml.GetParameterValuesResponse
	_ = xml.Unmarshal(pd.ReqRes.Body, &gpvr)
	log.Println("GetParameterValuesResponseParser")
	pd.ReqRes.Session.CPE.AddParameterValues(gpvr.ParameterList)
	pd.ReqRes.Session.FillCPESessionBaseInfo(gpvr.ParameterList)
	cpeRepository := mysql.NewCPERepository(repository.GetConnection())
	_, _, _ = cpeRepository.UpdateOrCreate(&pd.ReqRes.Session.CPE)

	pd.ReqRes.Session.GPVCount--

	// On-demand lookup: if GET /api/device/:uuid/lookup armed this flag before
	// kicking the device, snapshot the values it just returned into the cache
	// so GET /api/device/:uuid/parameters/cached can serve them. One-shot via
	// TTL expiry, same as goacs-php's Context (no explicit consume/forget).
	if _, enabled := cache.Global.Get(acscontext.KeyFor(acscontext.LookupParamsEnabledPrefix, pd.ReqRes.Session.CPE.SerialNumber)); enabled {
		cache.Global.Put(acscontext.KeyFor(acscontext.LookupParamsPrefix, pd.ReqRes.Session.CPE.SerialNumber), pd.ReqRes.Session.CPE.ParameterValues, 30*time.Minute)
	}

	//log.Println(pd.CPERequest.Session.CPE.ParameterValues)
	if pd.ReqRes.Session.LookupOnly {
		// Read-only: the snapshot above already covers what "lookup" is for -
		// no cpe_parameters write, no AddObj/DelObj diffing/mutation of the
		// device below.
	} else if pd.ReqRes.Session.IsNewInACS || pd.ReqRes.Session.PrevState == acsxml.AddObjReq {
		_ = cpeRepository.BulkInsertOrUpdateParameters(&pd.ReqRes.Session.CPE, pd.ReqRes.Session.CPE.ParameterValues)
	} else if pd.ReqRes.Session.IsBoot {
		if pd.ReqRes.Session.HasTaskOfType(acsxml.GPVReq) == false {
			cpeObjectParameters := pd.ReqRes.Session.CPE.GetObjectNamesToParameters()
			dbObjectParameters := cpeRepository.GetCPEParametersWithFlag(&pd.ReqRes.Session.CPE, "A")

			addparams, delparams := cpe.CompareObjectParameters(cpeObjectParameters, dbObjectParameters)

			if len(delparams) > 0 {
				for _, param := range delparams {
					task := tasks.NewCPETask(pd.ReqRes.Session.CPE.UUID)
					task.Task = acsxml.DelObjReq
					task.ParameterValues = []acsxml.ParameterValueStruct{param}
					log.Println("ADDING DELOBJECT TASK", task)
					pd.ReqRes.Session.AddTask(task)
				}

			}

			if len(addparams) > 0 {
				for _, param := range addparams {
					newName, err := acsxml.ObjectParamToInstance(param.Name)

					if err != nil {
						log.Println("addparams error", err)
						continue
					}
					param.Name = newName
					task := tasks.NewCPETask(pd.ReqRes.Session.CPE.UUID)
					task.Task = acsxml.AddObjReq
					task.ParameterValues = []acsxml.ParameterValueStruct{param}
					log.Println("ADDING ADDOBJECT TASK", task)
					pd.ReqRes.Session.AddTask(task)
				}

			}

		}
	}

}

// SetParameterValuesResponseParser handles the CPE's reply to a
// SetParameterValues request built from a SetParameterValuesTask (queue.go) -
// whether that task came from PrepareParametersToSend's template/stored-
// parameter diff, or an operator-queued task. A Fault response never reaches
// here (acs/logic/dispatcher.go routes acsxml.FaultResp to FaultDecision
// instead), so arriving here means the CPE accepted the values: they're
// persisted into cpe_parameters, confirming what was only provisionally
// queued in Session.PendingSetParameterValues.
func (pd *ParameterDecisions) SetParameterValuesResponseParser() {
	var spvr acsxml.SetParameterValuesResponseStruct
	_ = xml.Unmarshal(pd.ReqRes.Body, &spvr)
	log.Println("SetParameterValuesResponseParser, status", spvr.Status)

	confirmed := pd.ReqRes.Session.PendingSetParameterValues
	pd.ReqRes.Session.PendingSetParameterValues = nil
	if len(confirmed) == 0 {
		return
	}

	pd.ReqRes.Session.CPE.AddParameterValues(confirmed)
	cpeRepository := mysql.NewCPERepository(repository.GetConnection())
	_ = cpeRepository.BulkInsertOrUpdateParameters(&pd.ReqRes.Session.CPE, confirmed)
}

func (pd *ParameterDecisions) PrepareParametersToSend() {
	pd.ReqRes.Session.PrevState = acsxml.SPVReq
	cpeRepository := mysql.NewCPERepository(repository.GetConnection())
	templateRepository := mysql.NewTemplateRepository(repository.GetConnection())

	cpeDBParameters, err := cpeRepository.GetCPEParameters(&pd.ReqRes.Session.CPE)
	if err != nil {
		log.Println("Error GetParameterValuesResponseParser ", err.Error())
	}

	templateParameters := templateRepository.GetPrioritizedParametersForCPE(&pd.ReqRes.Session.CPE)
	cpeDBParameters = cpe.CombineTemplateParameters(cpeDBParameters, templateParameters)

	if len(cpeDBParameters) > 0 {
		//Get modified parameters
		//Check for AddObject instances
		diffParameters := pd.ReqRes.Session.CPE.GetChangedParametersToWrite(&cpeDBParameters)
		log.Println("DIFF PARAMS", diffParameters)
		if len(diffParameters) > 0 {
			for _, diffParam := range diffParameters {
				pd.ReqRes.Session.AddParameterToAdd(diffParam)
			}
		}
	}

	//TODO: Check why some parameters are writeable, but cpe returns fault on it
	if len(pd.ReqRes.Session.ParametersToAdd) > 0 {
		log.Println("ADD SPVREQ TASK")
		task := tasks.NewCPETask(pd.ReqRes.Session.CPE.UUID)
		task.Task = acsxml.SPVReq
		pd.ReqRes.Session.AddTask(task)
		pd.ReqRes.Session.SPVCount++
	}
}

func (pd *ParameterDecisions) AddObjectResponseParser() acsxml.AddObjectResponseStruct {
	var addObject acsxml.AddObjectResponseStruct
	_ = xml.Unmarshal(pd.ReqRes.Body, &addObject)

	return addObject
}
