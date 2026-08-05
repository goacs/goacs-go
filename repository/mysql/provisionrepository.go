package mysql

import (
	"database/sql"
	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
	"goacs/models/provisions"
	"goacs/repository"
	"log"
	"time"
)

type ProvisionRepository struct {
	db *sqlx.DB
}

func NewProvisionRepository(connection *sqlx.DB) ProvisionRepository {
	return ProvisionRepository{db: connection}
}

// GetAllWithRules returns every non-deleted provision together with its rules,
// ordered by priority (execution order), then id as a stable tiebreak.
func (r *ProvisionRepository) GetAllWithRules() ([]provisions.Provision, error) {
	var rows []provisions.Provision
	if err := r.db.Select(&rows, "SELECT * FROM provisions WHERE deleted_at IS NULL ORDER BY priority ASC, id ASC"); err != nil {
		log.Println("GetAllWithRules: provisions query error:", err)
		return nil, err
	}

	if err := r.attachRules(rows); err != nil {
		return rows, err
	}

	return rows, nil
}

// attachRules batch-loads and attaches provision_rules for the given provisions
// (in place, via their Id), avoiding one query per row.
func (r *ProvisionRepository) attachRules(rows []provisions.Provision) error {
	if len(rows) == 0 {
		return nil
	}

	ids := make([]int64, len(rows))
	byId := make(map[int64]*provisions.Provision, len(rows))
	for i := range rows {
		ids[i] = rows[i].Id
		byId[rows[i].Id] = &rows[i]
	}

	query, args, err := sqlx.In("SELECT * FROM provision_rules WHERE provision_id IN (?) ORDER BY id", ids)
	if err != nil {
		log.Println("attachRules: building rules query error:", err)
		return err
	}
	query = r.db.Rebind(query)

	var rules []provisions.ProvisionRule
	if err := r.db.Select(&rules, query, args...); err != nil {
		log.Println("attachRules: provision_rules query error:", err)
		return err
	}

	for _, rule := range rules {
		if p, ok := byId[rule.ProvisionId]; ok {
			p.Rules = append(p.Rules, rule)
		}
	}

	return nil
}

var provisionSortableColumns = map[string]bool{
	"name":       true,
	"created_at": true,
	"updated_at": true,
	"priority":   true,
}

var provisionFilterableColumns = map[string]bool{
	"name":       true,
	"created_at": true,
	"updated_at": true,
	"events":     true,
	"requests":   true,
}

// Find returns a single non-deleted provision together with its rules.
func (r *ProvisionRepository) Find(id int64) (*provisions.Provision, error) {
	var provision provisions.Provision
	err := r.db.Get(&provision, "SELECT * FROM provisions WHERE id=? AND deleted_at IS NULL LIMIT 1", id)
	if err != nil {
		return nil, repository.ErrNotFound
	}

	var rules []provisions.ProvisionRule
	_ = r.db.Select(&rules, "SELECT * FROM provision_rules WHERE provision_id=? ORDER BY id", id)
	provision.Rules = rules

	return &provision, nil
}

// List returns provisions (with their rules attached) for the list view - the
// configuration screen needs conditions visible per-row and for the simulator,
// so unlike a typical paginated list this also batch-loads rules.
func (r *ProvisionRepository) List(request repository.PaginatorRequest) ([]provisions.Provision, int) {
	dialect := goqu.Dialect("mysql")
	baseBuilder := dialect.From("provisions").Where(goqu.C("deleted_at").IsNull())

	for key, value := range request.Filter {
		if !provisionFilterableColumns[key] {
			continue
		}
		baseBuilder = baseBuilder.Where(goqu.Ex{key: goqu.Op{"ilike": "%" + value + "%"}})
	}

	var total int
	totalSql, _, _ := baseBuilder.Select(goqu.COUNT("*")).ToSQL()
	_ = r.db.Get(&total, totalSql)

	listBuilder := baseBuilder.Select("*").
		Offset(uint(request.CalcOffset())).
		Limit(uint(request.PerPage))

	sortColumns := request.SortColumns()
	if len(sortColumns) == 0 {
		listBuilder = listBuilder.OrderAppend(goqu.C("priority").Asc())
	}
	for _, sortColumn := range sortColumns {
		if !provisionSortableColumns[sortColumn.Column] {
			continue
		}
		if sortColumn.Descending {
			listBuilder = listBuilder.OrderAppend(goqu.C(sortColumn.Column).Desc())
		} else {
			listBuilder = listBuilder.OrderAppend(goqu.C(sortColumn.Column).Asc())
		}
	}

	listSql, _, _ := listBuilder.ToSQL()

	var rows []provisions.Provision
	_ = r.db.Select(&rows, listSql)
	_ = r.attachRules(rows)

	return rows, total
}

func (r *ProvisionRepository) Create(p *provisions.Provision) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}

	var maxPriority sql.NullInt64
	if err := tx.Get(&maxPriority, "SELECT MAX(priority) FROM provisions WHERE deleted_at IS NULL"); err != nil {
		_ = tx.Rollback()
		return err
	}
	p.Priority = int(maxPriority.Int64) + 1

	now := time.Now()
	result, err := tx.Exec(
		"INSERT INTO provisions (name, events, requests, script, priority, enabled, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?)",
		p.Name, p.Events, p.Requests, &p.Script, p.Priority, p.Enabled, now, now,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	p.Id, _ = result.LastInsertId()
	p.CreatedAt = now
	p.UpdatedAt = now

	if err := insertProvisionRules(tx, p.Id, p.Rules); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (r *ProvisionRepository) Update(p *provisions.Provision) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}

	p.UpdatedAt = time.Now()
	_, err = tx.Exec(
		"UPDATE provisions SET name=?, events=?, requests=?, script=?, priority=?, enabled=?, updated_at=? WHERE id=?",
		p.Name, p.Events, p.Requests, &p.Script, p.Priority, p.Enabled, p.UpdatedAt, p.Id,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err := tx.Exec("DELETE FROM provision_rules WHERE provision_id=?", p.Id); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := insertProvisionRules(tx, p.Id, p.Rules); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// Delete soft-deletes the provision; its rules are left in place (harmless,
// since GetAllWithRules/Find only ever join through a non-deleted provision).
func (r *ProvisionRepository) Delete(id int64) error {
	_, err := r.db.Exec("UPDATE provisions SET deleted_at=? WHERE id=?", time.Now(), id)
	return err
}

// UpdateEnabled flips the enabled flag for a single provision, for the
// list view's inline enable/disable toggle.
func (r *ProvisionRepository) UpdateEnabled(id int64, enabled bool) error {
	_, err := r.db.Exec("UPDATE provisions SET enabled=?, updated_at=? WHERE id=? AND deleted_at IS NULL", enabled, time.Now(), id)
	return err
}

// Reorder bulk-assigns priority = index+1 for each id in the given order, for
// the list view's drag-and-drop reorder and editable-priority input.
func (r *ProvisionRepository) Reorder(ids []int64) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}

	now := time.Now()
	for i, id := range ids {
		if _, err := tx.Exec("UPDATE provisions SET priority=?, updated_at=? WHERE id=? AND deleted_at IS NULL", i+1, now, id); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// Clone duplicates a provision (and its rules) directly below the original in
// priority order, shifting every lower-priority provision down by one, for the
// "clone" action in the configuration UI.
func (r *ProvisionRepository) Clone(id int64) (*provisions.Provision, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}

	var original provisions.Provision
	if err := tx.Get(&original, "SELECT * FROM provisions WHERE id=? AND deleted_at IS NULL LIMIT 1", id); err != nil {
		_ = tx.Rollback()
		return nil, repository.ErrNotFound
	}

	var rules []provisions.ProvisionRule
	if err := tx.Select(&rules, "SELECT * FROM provision_rules WHERE provision_id=? ORDER BY id", id); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	insertAt := original.Priority + 1
	if _, err := tx.Exec("UPDATE provisions SET priority = priority + 1 WHERE priority >= ? AND deleted_at IS NULL", insertAt); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	now := time.Now()
	result, err := tx.Exec(
		"INSERT INTO provisions (name, events, requests, script, priority, enabled, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?)",
		original.Name+" (copy)", original.Events, original.Requests, &original.Script, insertAt, original.Enabled, now, now,
	)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	cloneId, _ := result.LastInsertId()
	if err := insertProvisionRules(tx, cloneId, rules); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.Find(cloneId)
}

func insertProvisionRules(tx *sqlx.Tx, provisionId int64, rules []provisions.ProvisionRule) error {
	for _, rule := range rules {
		if _, err := tx.Exec(
			"INSERT INTO provision_rules (provision_id, parameter, operator, value, created_at, updated_at) VALUES (?,?,?,?,?,?)",
			provisionId, rule.Parameter, rule.Operator, rule.Value, time.Now(), time.Now(),
		); err != nil {
			log.Println("insertProvisionRules error:", err)
			return err
		}
	}

	return nil
}
