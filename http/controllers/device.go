package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	acscontext "goacs/acs/context"
	acshttp "goacs/acs/http"
	"goacs/acs/types"
	"goacs/http/request"
	"goacs/http/response"
	"goacs/lib/cache"
	"goacs/models/cpe"
	"goacs/models/tasks"
	"goacs/repository"
	"goacs/repository/mysql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type UpdateDeviceTaskRequest struct {
	Event   string `json:"event" validate:"required"`
	Task    string `json:"task" validate:"required"`
	Payload string `json:"payload"`
}

type ParameterRequest struct {
	Name  string     `json:"name" validate:"required"`
	Value string     `json:"value"`
	Flag  types.Flag `json:"flag" validate:"required"`
}

type DeleteParameterRequest struct {
	Name string `json:"name" validate:"required"`
}

type AddObjectRequest struct {
	Name string `json:"name" binding:"required"`
	Key  string `json:"key"`
}

type AssignTemplateToDeviceRequest struct {
	TemplateId int64 `json:"template_id" validate:"required"`
	Priority   int64 `json:"priority" validate:"required"`
}

type AddTaskForCPERequest struct {
	Event   string `json:"event" validate:"required"`
	Task    string `json:"task" validate:"required"`
	Payload string `json:"payload"`
}

func GetDevice(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		response.ResponseError(ctx, 404, "Not found", "")
		return
	}

	response.ResponseData(ctx, cpeModel)

}

func DeleteDevice(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)

	if err != nil {
		response.ResponseError(ctx, 404, "Not found", "")
		return
	}

	cperepository.DeleteDevice(cpeModel)
	cperepository.DeleteAllParameters(cpeModel)

}

func GetDeviceParameters(ctx *gin.Context) {
	paginatorRequest := repository.PaginatorRequestFromContext(ctx)
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err == nil {
		parameters, total := cperepository.ListCPEParameters(cpeModel, paginatorRequest)
		responseData := repository.NewPaginatorResponse(paginatorRequest, total, parameters)
		response.ResponsePaginatior(ctx, responseData)
	}
}

func GetDeviceTemplates(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, _ := getCPEFromContext(ctx, cperepository)
	templaterepository := mysql.NewTemplateRepository(repository.GetConnection())
	templates := templaterepository.GetTemplatesForCPE(cpeModel)
	response.ResponseData(ctx, templates)
}

func AssignTemplateToDevice(ctx *gin.Context) {
	var assignDeviceRequest AssignTemplateToDeviceRequest
	_ = ctx.BindJSON(&assignDeviceRequest)
	log.Println(assignDeviceRequest)
	validator := request.NewApiValidator(ctx, assignDeviceRequest)
	verr := validator.Validate()

	if verr != nil {
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, _ := getCPEFromContext(ctx, cperepository)
	templaterepository := mysql.NewTemplateRepository(repository.GetConnection())
	err := templaterepository.AssignTemplateToDevice(cpeModel, assignDeviceRequest.TemplateId, assignDeviceRequest.Priority)

	if err != nil {
		response.Response500(ctx, "", err)
		return
	}

	response.ResponseData(ctx, "")
}

func UpdateDeviceTemplatePriority(ctx *gin.Context) {
	var assignDeviceRequest AssignTemplateToDeviceRequest
	_ = ctx.BindJSON(&assignDeviceRequest)

	validator := request.NewApiValidator(ctx, assignDeviceRequest)
	if verr := validator.Validate(); verr != nil {
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	templaterepository := mysql.NewTemplateRepository(repository.GetConnection())
	if err := templaterepository.UpdatePriority(cpeModel, assignDeviceRequest.TemplateId, assignDeviceRequest.Priority); err != nil {
		response.Response500(ctx, "", err)
		return
	}

	response.ResponseData(ctx, "")
}

func UnassignTemplateFromDevice(ctx *gin.Context) {
	templateId, _ := strconv.Atoi(ctx.Param("template_id"))
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, _ := getCPEFromContext(ctx, cperepository)
	templaterepository := mysql.NewTemplateRepository(repository.GetConnection())

	err := templaterepository.UnassignTemplateFromDevice(cpeModel, int64(templateId))

	if err != nil {
		response.Response500(ctx, "", err)
		return
	}

	response.ResponseData(ctx, "")

}

func GetDevicesList(ctx *gin.Context) {
	paginatorRequest := repository.PaginatorRequestFromContext(ctx)
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpes, total := cperepository.List(paginatorRequest)
	responseData := repository.NewPaginatorResponse(paginatorRequest, total, cpes)
	response.ResponsePaginatior(ctx, responseData)
}

func CreateParameter(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	parameter := parameterBaseRequest(ctx)

	_, err = cperepository.CreateParameter(cpeModel, parameter)

	if err != nil {
		response.Response500(ctx, "Cannot save parameter", "")
	}

	response.ResponseData(ctx, "")

}

func UpdateParameter(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	parameter := parameterBaseRequest(ctx)
	_, err = cperepository.UpdateParameter(cpeModel, parameter)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	ctx.JSON(204, "")
}

func DeleteParameter(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, _ := getCPEFromContext(ctx, cperepository)

	var parameterRequest DeleteParameterRequest
	_ = ctx.ShouldBind(&parameterRequest)

	validator := request.NewApiValidator(ctx, parameterRequest)

	err := validator.Validate()

	if err != nil {
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	_, err = cperepository.DeleteParameter(cpeModel, parameterRequest.Name)

	if err != nil {
		response.Response500(ctx, err.Error(), "")
	}

	response.ResponseData(ctx, "")
}

func parameterBaseRequest(ctx *gin.Context) types.ParameterValueStruct {
	var parameterRequest ParameterRequest
	_ = ctx.ShouldBind(&parameterRequest)

	validator := request.NewApiValidator(ctx, parameterRequest)

	err := validator.Validate()

	if err != nil {
		response.ResponseValidationErrors(ctx, validator)
		return types.ParameterValueStruct{}
	}

	//set Send as default if no system flag
	if parameterRequest.Flag.System == false {
		parameterRequest.Flag.Send = true //if user changed value, then need to send it, true?
	}

	return types.ParameterValueStruct{
		Name: parameterRequest.Name,
		ValueStruct: types.ValueStruct{
			Value: parameterRequest.Value,
			Type:  "",
		},
		Flag: parameterRequest.Flag,
	}
}

func getCPEFromContext(ctx *gin.Context, cpeRepository mysql.CPERepository) (*cpe.CPE, error) {
	cpeModel, err := cpeRepository.Find(ctx.Param("uuid"))

	if err != nil {
		response.ResponseError(ctx, 404, err.Error(), "")
		return nil, err
	}

	return cpeModel, nil
}

func GetParameterValues(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)

	if err != nil {
		response.ResponseError(ctx, http.StatusBadRequest, err.Error(), "")
		return
	}

	parameters, err := cperepository.GetCPEParameters(cpeModel)

	if err != nil {
		response.ResponseError(ctx, http.StatusBadRequest, err.Error(), "")
		return
	}

	acsRequest := acshttp.NewACSRequest(cpeModel)
	acsRequest.GetParameterValues(cpe.DetermineDeviceTreeRootPath(parameters))

}

func SetParameterValues(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)

	if err != nil {
		response.ResponseError(ctx, http.StatusBadRequest, err.Error(), "")
		return
	}

	acsRequest := acshttp.NewACSRequest(cpeModel)
	acsRequest.SetParameterValues()

}

func GetDeviceQueuedTasks(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)

	if err != nil {
		response.ResponseError(ctx, http.StatusBadRequest, err.Error(), "")
		return
	}

	taskRepository := mysql.NewTasksRepository(repository.GetConnection())
	deviceTasks := taskRepository.GetTasksForCPE(cpeModel.UUID)

	response.ResponseData(ctx, deviceTasks)
}

func AddDeviceTask(ctx *gin.Context) {
	var addTaskRequest AddTaskForCPERequest
	_ = ctx.ShouldBindJSON(&addTaskRequest)

	validator := request.NewApiValidator(ctx, addTaskRequest)
	verr := validator.Validate()

	if verr != nil {
		log.Println(validator.Errors)
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	cperepository := mysql.NewCPERepository(repository.GetConnection())
	taskrepository := mysql.NewTasksRepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)

	if err != nil {
		return
	}

	task := tasks.NewCPETask(cpeModel.UUID)
	task.Task = addTaskRequest.Task
	task.Event = addTaskRequest.Event
	_ = json.Unmarshal([]byte(addTaskRequest.Payload), &task.Payload)

	taskrepository.AddTask(task)
	response.ResponseData(ctx, "")
}

func GetDeviceTask(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	task, err := deviceTaskFromContext(ctx, cpeModel)
	if err != nil {
		return
	}

	response.ResponseData(ctx, task)
}

func UpdateDeviceTask(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	task, err := deviceTaskFromContext(ctx, cpeModel)
	if err != nil {
		return
	}

	var updateRequest UpdateDeviceTaskRequest
	_ = ctx.ShouldBindJSON(&updateRequest)

	validator := request.NewApiValidator(ctx, updateRequest)
	if verr := validator.Validate(); verr != nil {
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	task.Event = updateRequest.Event
	task.Task = updateRequest.Task
	_ = json.Unmarshal([]byte(updateRequest.Payload), &task.Payload)

	taskrepository := mysql.NewTasksRepository(repository.GetConnection())
	taskrepository.UpdateTask(task)
	response.ResponseData(ctx, "")
}

func DeleteDeviceTask(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	task, err := deviceTaskFromContext(ctx, cpeModel)
	if err != nil {
		return
	}

	taskrepository := mysql.NewTasksRepository(repository.GetConnection())
	if err := taskrepository.Delete(task.Id); err != nil {
		response.Response500(ctx, "Cannot delete task", err)
		return
	}

	response.ResponseData(ctx, "")
}

// deviceTaskFromContext loads the :taskid task and verifies it actually
// belongs to this device, so a valid task id for a different CPE can't be
// used to read/update/delete a task through the wrong device's URL.
func deviceTaskFromContext(ctx *gin.Context, cpeModel *cpe.CPE) (tasks.Task, error) {
	taskId, err := strconv.ParseInt(ctx.Param("taskid"), 10, 64)
	if err != nil {
		response.ResponseError(ctx, http.StatusBadRequest, "Invalid task id", "")
		return tasks.Task{}, err
	}

	taskrepository := mysql.NewTasksRepository(repository.GetConnection())
	task := taskrepository.GetTask(taskId)

	if task.ForName != tasks.TASK_CPE || task.ForID != cpeModel.UUID {
		response.ResponseError(ctx, http.StatusNotFound, "Not found", "")
		return tasks.Task{}, repository.ErrNotFound
	}

	return task, nil
}

func AddObject(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	addObjectRequest := AddObjectRequest{}
	err = ctx.ShouldBindJSON(&addObjectRequest)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	acsRequest := acshttp.NewACSRequest(cpeModel)
	acsRequest.AddObject(addObjectRequest.Name)
}

func Kick(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	kickCPE(cpeModel)
	response.ResponseData(ctx, "")
}

// kickCPE issues a TR-069 Connection Request so the CPE opens a new CWMP
// session against us on its own initiative - used directly by Kick, and as
// the delivery mechanism for the "provision now" / "lookup now" actions
// below (they just prime some state beforehand via the cache).
func kickCPE(cpeModel *cpe.CPE) {
	acsRequest := acshttp.NewACSRequest(cpeModel)
	acsRequest.Kick()
}

// GetDeviceProvision forces the device's next Inform to run a full
// GetParameterNames/GetParameterValues walk (as if it had just booted), then
// kicks it so that Inform happens now rather than at the next periodic
// interval. Port of goacs-php's DeviceController::provision.
func GetDeviceProvision(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	cache.Global.Put(acscontext.KeyFor(acscontext.ProvisionPrefix, cpeModel.SerialNumber), true, 5*time.Minute)
	kickCPE(cpeModel)
	response.ResponseData(ctx, "")
}

// GetDeviceLookup arms a one-shot cache flag consumed by
// ParameterDecisions.GetParameterValuesResponseParser (acs/methods/parametermethods.go),
// which snapshots the next GetParameterValues result into the lookup cache,
// then kicks the device so that lookup happens now. Port of goacs-php's
// DeviceController::lookup.
func GetDeviceLookup(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	cache.Global.Put(acscontext.KeyFor(acscontext.LookupParamsEnabledPrefix, cpeModel.SerialNumber), true, 5*time.Minute)
	kickCPE(cpeModel)
	response.ResponseData(ctx, "")
}

func ClearDeviceCache(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	cache.Global.Forget(acscontext.KeyFor(acscontext.LookupParamsPrefix, cpeModel.SerialNumber))
	cache.Global.Forget(acscontext.KeyFor(acscontext.LookupParamsEnabledPrefix, cpeModel.SerialNumber))
	cache.Global.Forget(acscontext.KeyFor(acscontext.ProvisionPrefix, cpeModel.SerialNumber))

	response.ResponseData(ctx, "")
}

func GetDeviceCachedParameters(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	parameters, ok := cachedParametersFor(cpeModel)
	if !ok {
		response.ResponseData(ctx, repository.NewPaginatorResponse(repository.PaginatorRequestFromContext(ctx), 0, []types.ParameterValueStruct{}))
		return
	}

	paginatorRequest := repository.PaginatorRequestFromContext(ctx)
	filtered := filterCachedParameters(parameters, paginatorRequest.Filter)

	start := paginatorRequest.CalcOffset()
	end := start + paginatorRequest.PerPage
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	response.ResponsePaginatior(ctx, repository.NewPaginatorResponse(paginatorRequest, len(filtered), filtered[start:end]))
}

func DownloadDeviceCachedParametersCSV(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	parameters, _ := cachedParametersFor(cpeModel)

	var buffer bytes.Buffer
	buffer.WriteString("name;type;value;flags\n")
	for _, parameter := range parameters {
		buffer.WriteString(fmt.Sprintf("%s;%s;%s;%s\n",
			parameter.Name, parameter.ValueStruct.Type, parameter.ValueStruct.Value, parameter.Flag.AsString()))
	}

	ctx.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s-cached-parameters.csv", cpeModel.SerialNumber))
	ctx.Data(http.StatusOK, "text/csv", buffer.Bytes())
}

func cachedParametersFor(cpeModel *cpe.CPE) ([]types.ParameterValueStruct, bool) {
	value, ok := cache.Global.Get(acscontext.KeyFor(acscontext.LookupParamsPrefix, cpeModel.SerialNumber))
	if !ok {
		return nil, false
	}

	parameters, ok := value.([]types.ParameterValueStruct)
	return parameters, ok
}

func filterCachedParameters(parameters []types.ParameterValueStruct, filter map[string]string) []types.ParameterValueStruct {
	if len(filter) == 0 {
		return parameters
	}

	filtered := make([]types.ParameterValueStruct, 0, len(parameters))
	for _, parameter := range parameters {
		if nameFilter, ok := filter["name"]; ok && !strings.Contains(parameter.Name, nameFilter) {
			continue
		}
		if valueFilter, ok := filter["value"]; ok && !strings.Contains(parameter.ValueStruct.Value, valueFilter) {
			continue
		}
		if typeFilter, ok := filter["type"]; ok && !strings.Contains(parameter.ValueStruct.Type, typeFilter) {
			continue
		}
		filtered = append(filtered, parameter)
	}

	return filtered
}

type PatchParametersRequest struct {
	Parameters []ParameterRequest `json:"parameters" validate:"required"`
}

func PatchDeviceParameters(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	var patchRequest PatchParametersRequest
	_ = ctx.ShouldBindJSON(&patchRequest)

	validator := request.NewApiValidator(ctx, patchRequest)
	if verr := validator.Validate(); verr != nil {
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	parameters := make([]types.ParameterValueStruct, 0, len(patchRequest.Parameters))
	for _, parameterRequest := range patchRequest.Parameters {
		parameters = append(parameters, types.ParameterValueStruct{
			Name:        parameterRequest.Name,
			ValueStruct: types.ValueStruct{Value: parameterRequest.Value},
			Flag:        parameterRequest.Flag,
		})
	}

	if ok := cperepository.BulkInsertOrUpdateParameters(cpeModel, parameters); !ok {
		response.Response500(ctx, "Cannot patch parameters", "")
		return
	}

	response.ResponseData(ctx, "")
}

func GetDeviceLogs(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	paginatorRequest := repository.PaginatorRequestFromContext(ctx)
	logRepository := mysql.NewLogRepository(repository.GetConnection())
	logs, total := logRepository.ListForCPE(cpeModel.UUID, paginatorRequest)

	response.ResponsePaginatior(ctx, repository.NewPaginatorResponse(paginatorRequest, total, logs))
}

func DownloadDeviceLogs(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	sessionId := ctx.Query("session_id")
	if sessionId == "" {
		response.ResponseError(ctx, http.StatusBadRequest, "session_id is required", "")
		return
	}

	logRepository := mysql.NewLogRepository(repository.GetConnection())
	logs := logRepository.GetForSession(cpeModel.UUID, sessionId)

	var buffer bytes.Buffer
	for _, entry := range logs {
		buffer.WriteString(fmt.Sprintf("[%s] %s (%s)\n%s\n\n", entry.CreatedAt.Format("2006-01-02 15:04:05"), entry.From, entry.Type, entry.FullXML))
	}

	ctx.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.log", cpeModel.SerialNumber, sessionId))
	ctx.Data(http.StatusOK, "text/plain", buffer.Bytes())
}
