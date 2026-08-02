// Package log holds the CWMP conversation/fault log entry, a superset of
// models/fault.Fault (device_id, full_xml, code, message, type, from,
// session_id, detail) mirroring goacs-php's App\Models\Log.
package log

import (
	"time"

	"gopkg.in/guregu/null.v4"
)

const (
	TypeFault    = "FAULT"
	TypeInfo     = "INFO"
	TypeError    = "ERROR"
	TypeRequest  = "REQUEST"
	TypeResponse = "RESPONSE"
)

const (
	FromDevice = "device"
	FromACS    = "acs"
)

type Log struct {
	Id        int64       `db:"id" json:"id"`
	CPEUUID   string      `db:"cpe_uuid" json:"cpe_uuid"`
	FullXML   string      `db:"full_xml" json:"full_xml"`
	Code      string      `db:"code" json:"code"`
	Message   string      `db:"message" json:"message"`
	Type      string      `db:"type" json:"type"`
	From      string      `db:"from" json:"from"`
	SessionId string      `db:"session_id" json:"session_id"`
	Detail    null.String `db:"detail" json:"detail"`
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
}
