package types

const (
	InformReq        string = "inform"
	InformResp       string = "InformResponse"
	EMPTY            string = "empty"
	GPNResp          string = "GetParameterNamesResponse"
	GPNReq           string = "GetParameterNames"
	GPVResp          string = "GetParameterValuesResponse"
	GPVReq           string = "GetParameterValues"
	GRPCMReq         string = "GetRPCMethods"
	GRPCMResp        string = "GetRPCMethodsResponse"
	SPVReq           string = "SetParameterValues"
	SPVResp          string = "SetParameterValuesResponse"
	FaultResp        string = "FaultResp"
	AddObjReq        string = "AddObjectRequest"
	AddObjResp       string = "AddObjectResponse"
	DelObjReq        string = "DeleteObjectRequest"
	DelObjResp       string = "DeleteObjectResponse"
	UNKNOWN          string = "unknown"
	Download         string = "Download"
	DownloadResp     string = "DownloadResponse"
	Reboot           string = "Reboot"
	RebootResp       string = "RebootResponse"
	TransferComplete string = "TransferComplete"
	TCResp           string = "TransferCompleteResponse"

	// SetParameterValuesProcessor is not a real CWMP wire type - it is a synthetic
	// request-type marker used only in the "requests" column of a provisioning rule,
	// letting a rule target the point where the queue has drained and freshly-read
	// parameter values are available, right before the session would otherwise end.
	// Mirrors goacs-php's Types::SetParameterValuesProcessor.
	SetParameterValuesProcessor string = "SetParameterValuesProcessor"
)
