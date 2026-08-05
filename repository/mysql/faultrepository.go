package mysql

import (
	"fmt"
	"goacs/models/cpe"
	"goacs/models/fault"
	"goacs/repository"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type FaultRepository struct {
	db *sqlx.DB
}

func NewFaultRepository() *FaultRepository {
	return &FaultRepository{
		db: repository.GetConnection(),
	}
}

func SaveFault(cpe *cpe.CPE, code string, message string) {
	repository := NewFaultRepository()
	repository.SaveFault(cpe, code, message)
}

func (r *FaultRepository) SaveFault(cpe *cpe.CPE, code string, message string) {
	uuidInstance, _ := uuid.NewRandom()
	uuidString := uuidInstance.String()

	r.db.Exec("INSERT INTO faults VALUES (?,?,?,?,?)",
		uuidString,
		cpe.UUID,
		code,
		message,
		time.Now())

}

func (r *FaultRepository) GetLastDay(limit int) []fault.Fault {
	faults := []fault.Fault{}
	err := r.db.Select(&faults, "SELECT faults.*, COALESCE(cpe.serial_number, '') as serial_number FROM faults LEFT JOIN cpe ON cpe.uuid = faults.cpe_uuid WHERE faults.created_at >= NOW() - INTERVAL 1 DAY LIMIT ?", strconv.Itoa(limit))
	if err != nil {
		fmt.Println(err)
		return faults
	}
	return faults
}

func (r *FaultRepository) CountLastDay() (fault_count int64) {
	err := r.db.Get(&fault_count, "SELECT count(uuid) FROM faults WHERE created_at >= NOW() - INTERVAL 1 DAY")

	if err != nil {
		return 0
	}

	return fault_count
}
