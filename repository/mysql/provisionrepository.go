package mysql

import (
	"github.com/jmoiron/sqlx"
	"goacs/models/provisions"
	"log"
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
