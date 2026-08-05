package fault

import "time"

type Fault struct {
	UUID         string    `json:"uuid" db:"uuid"`
	CPEUUID      string    `json:"cpe_uuid" db:"cpe_uuid"`
	SerialNumber string    `json:"serial_number" db:"serial_number"`
	Code         string    `json:"code" db:"code"`
	Message      string    `json:"message" db:"message"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
