package mysql

import (
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

// GetAllWithRules returns every non-deleted provision together with its rules, in a
// stable order (rule bundles have no priority in the PHP model beyond insertion order).
func (r *ProvisionRepository) GetAllWithRules() ([]provisions.Provision, error) {
	var rows []provisions.Provision
	if err := r.db.Select(&rows, "SELECT * FROM provisions WHERE deleted_at IS NULL ORDER BY id"); err != nil {
		log.Println("GetAllWithRules: provisions query error:", err)
		return nil, err
	}

	if len(rows) == 0 {
		return rows, nil
	}

	ids := make([]int64, len(rows))
	byId := make(map[int64]*provisions.Provision, len(rows))
	for i := range rows {
		ids[i] = rows[i].Id
		byId[rows[i].Id] = &rows[i]
	}

	query, args, err := sqlx.In("SELECT * FROM provision_rules WHERE provision_id IN (?) ORDER BY id", ids)
	if err != nil {
		log.Println("GetAllWithRules: building rules query error:", err)
		return rows, err
	}
	query = r.db.Rebind(query)

	var rules []provisions.ProvisionRule
	if err := r.db.Select(&rules, query, args...); err != nil {
		log.Println("GetAllWithRules: provision_rules query error:", err)
		return rows, err
	}

	for _, rule := range rules {
		if p, ok := byId[rule.ProvisionId]; ok {
			p.Rules = append(p.Rules, rule)
		}
	}

	return rows, nil
}

var provisionSortableColumns = map[string]bool{
	"name":       true,
	"created_at": true,
	"updated_at": true,
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

// List returns provisions without their rules, for the paginated list view.
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
		listBuilder = listBuilder.OrderAppend(goqu.C("id").Asc())
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

	return rows, total
}

func (r *ProvisionRepository) Create(p *provisions.Provision) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}

	now := time.Now()
	result, err := tx.Exec(
		"INSERT INTO provisions (name, events, requests, script, created_at, updated_at) VALUES (?,?,?,?,?,?)",
		p.Name, p.Events, p.Requests, &p.Script, now, now,
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
		"UPDATE provisions SET name=?, events=?, requests=?, script=?, updated_at=? WHERE id=?",
		p.Name, p.Events, p.Requests, &p.Script, p.UpdatedAt, p.Id,
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

// Clone duplicates a provision (and its rules) under a new name, for the
// "clone" action in the configuration UI.
func (r *ProvisionRepository) Clone(id int64) (*provisions.Provision, error) {
	original, err := r.Find(id)
	if err != nil {
		return nil, err
	}

	clone := &provisions.Provision{
		Name:     original.Name + " (clone)",
		Events:   original.Events,
		Requests: original.Requests,
		Script:   original.Script,
	}
	for _, rule := range original.Rules {
		clone.Rules = append(clone.Rules, provisions.ProvisionRule{
			Parameter: rule.Parameter,
			Operator:  rule.Operator,
			Value:     rule.Value,
		})
	}

	if err := r.Create(clone); err != nil {
		return nil, err
	}

	return clone, nil
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
