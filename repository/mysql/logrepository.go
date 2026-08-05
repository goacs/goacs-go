package mysql

import (
	"goacs/models/cpe"
	"goacs/models/log"
	"goacs/repository"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
)

type LogRepository struct {
	db *sqlx.DB
}

func NewLogRepository(connection *sqlx.DB) LogRepository {
	return LogRepository{db: connection}
}

var logSortableColumns = map[string]bool{
	"code":       true,
	"message":    true,
	"type":       true,
	"from":       true,
	"created_at": true,
}

var logFilterableColumns = logSortableColumns

// ConversationLoggingEnabled mirrors goacs-php's gate on Log::logConversation:
// full request/response XML is only persisted when the global "conversation_log"
// setting is on, or debug mode is toggled for this specific device.
func (r *LogRepository) ConversationLoggingEnabled(cpeModel *cpe.CPE) bool {
	if cpeModel != nil && cpeModel.Debug {
		return true
	}

	configRepository := NewConfigRepository(r.db)
	value, err := configRepository.GetValue("conversation_log")
	if err != nil {
		return false
	}

	return value == "1" || value == "true"
}

func (r *LogRepository) Save(l *log.Log) error {
	dialect := goqu.Dialect("mysql")

	query, args, _ := dialect.Insert("logs").Prepared(true).Rows(goqu.Record{
		"cpe_uuid":   l.CPEUUID,
		"full_xml":   l.FullXML,
		"code":       l.Code,
		"message":    l.Message,
		"type":       l.Type,
		"from":       l.From,
		"session_id": l.SessionId,
		"detail":     l.Detail,
	}).ToSQL()

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return err
	}

	if id, idErr := result.LastInsertId(); idErr == nil {
		l.Id = id
	}

	if repository.OnLogSaved != nil {
		l.CreatedAt = time.Now()
		repository.OnLogSaved(l)
	}

	return nil
}

func (r *LogRepository) ListForCPE(cpeUUID string, request repository.PaginatorRequest) ([]log.Log, int) {
	dialect := goqu.Dialect("mysql")
	baseBuilder := dialect.From("logs").Where(goqu.C("cpe_uuid").Eq(cpeUUID))
	baseBuilder = applyLogFilters(baseBuilder, request)

	return r.paginate(baseBuilder, request, true)
}

func (r *LogRepository) ListFaultsToday(limit int) []log.Log {
	var logs []log.Log
	dialect := goqu.Dialect("mysql")

	query, args, _ := dialect.From("logs").Prepared(true).
		Where(goqu.C("type").Eq(log.TypeFault), goqu.L("created_at >= NOW() - INTERVAL 1 DAY")).
		Order(goqu.C("created_at").Desc()).
		Limit(uint(limit)).
		ToSQL()

	_ = r.db.Select(&logs, query, args...)
	return logs
}

func (r *LogRepository) ListFaults(request repository.PaginatorRequest) ([]log.Log, int) {
	dialect := goqu.Dialect("mysql")
	baseBuilder := dialect.From("logs").Where(goqu.C("type").Eq(log.TypeFault))
	baseBuilder = applyLogFilters(baseBuilder, request)

	return r.paginate(baseBuilder, request, true)
}

func (r *LogRepository) DeleteAllForCPE(cpeUUID string) error {
	dialect := goqu.Dialect("mysql")

	query, args, _ := dialect.Delete("logs").Prepared(true).
		Where(goqu.C("cpe_uuid").Eq(cpeUUID)).
		ToSQL()

	_, err := r.db.Exec(query, args...)
	return err
}

func (r *LogRepository) GetForSession(cpeUUID, sessionId string) []log.Log {
	var logs []log.Log
	dialect := goqu.Dialect("mysql")

	query, args, _ := dialect.From("logs").Prepared(true).
		Where(goqu.C("cpe_uuid").Eq(cpeUUID), goqu.C("session_id").Eq(sessionId)).
		Order(goqu.C("created_at").Asc()).
		ToSQL()

	_ = r.db.Select(&logs, query, args...)
	return logs
}

func applyLogFilters(builder *goqu.SelectDataset, request repository.PaginatorRequest) *goqu.SelectDataset {
	for key, value := range request.Filter {
		if !logFilterableColumns[key] {
			continue
		}
		builder = builder.Where(goqu.Ex{key: goqu.Op{"ilike": "%" + value + "%"}})
	}

	return builder
}

func (r *LogRepository) paginate(baseBuilder *goqu.SelectDataset, request repository.PaginatorRequest, defaultSortDesc bool) ([]log.Log, int) {
	var total int
	totalSql, _, _ := baseBuilder.Select(goqu.COUNT("*")).ToSQL()
	_ = r.db.Get(&total, totalSql)

	listBuilder := baseBuilder.Select("*").
		Offset(uint(request.CalcOffset())).
		Limit(uint(request.PerPage))

	sortColumns := request.SortColumns()
	if len(sortColumns) == 0 && defaultSortDesc {
		listBuilder = listBuilder.OrderAppend(goqu.C("created_at").Desc())
	}

	for _, sortColumn := range sortColumns {
		if !logSortableColumns[sortColumn.Column] {
			continue
		}
		if sortColumn.Descending {
			listBuilder = listBuilder.OrderAppend(goqu.C(sortColumn.Column).Desc())
		} else {
			listBuilder = listBuilder.OrderAppend(goqu.C(sortColumn.Column).Asc())
		}
	}

	listSql, _, _ := listBuilder.ToSQL()

	var logs []log.Log
	_ = r.db.Select(&logs, listSql)

	return logs, total
}
