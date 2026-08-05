package methods

import acsxml "goacs/acs/types"

// diagnosticsCompleteEventCode is the CWMP Inform event code TR-143 uses to signal a
// DownloadDiagnostics/UploadDiagnostics run has finished.
const diagnosticsCompleteEventCode = "8 DIAGNOSTICS COMPLETE"

// diagnosticsRootPrefix mirrors http/controllers.diagnosticsPrefix - TR-143 nests under
// Device.IP.Diagnostics. for TR-181 (Device:2), directly under InternetGatewayDevice. for
// legacy TR-098.
func diagnosticsRootPrefix(root string) string {
	if root == "Device" {
		return root + ".IP.Diagnostics."
	}
	return root + "."
}

// diagnosticsResultParameterInfo lists the TR-143 result leaves for both
// Download/UploadDiagnostics, fully qualified for the given data-model root - used to
// queue a targeted GetParameterValues when a CPE reports "8 DIAGNOSTICS COMPLETE" (see
// InformDecision.CpeInformRequestParser), instead of waiting for an operator to poll.
func diagnosticsResultParameterInfo(root string) []acsxml.ParameterInfo {
	prefix := diagnosticsRootPrefix(root)
	commonLeaves := []string{"DiagnosticsState", "BOMTime", "EOMTime", "TCPOpenRequestTime", "TCPOpenResponseTime"}

	var info []acsxml.ParameterInfo
	for _, leaf := range commonLeaves {
		info = append(info, acsxml.ParameterInfo{Name: prefix + "DownloadDiagnostics." + leaf})
	}
	info = append(info, acsxml.ParameterInfo{Name: prefix + "DownloadDiagnostics.TestBytesReceived"})

	for _, leaf := range commonLeaves {
		info = append(info, acsxml.ParameterInfo{Name: prefix + "UploadDiagnostics." + leaf})
	}
	info = append(info, acsxml.ParameterInfo{Name: prefix + "UploadDiagnostics.TestBytesSent"})

	return info
}

func hasEventCode(codes []string, want string) bool {
	for _, code := range codes {
		if code == want {
			return true
		}
	}
	return false
}
