package repositories

import (
	"database/sql"

	"github.com/lib/pq"
	"github.com/vladyslavivchenko/netme/internal/models"
)

type RulesRepository struct {
	db *sql.DB
}

func NewRulesRepository(db *sql.DB) *RulesRepository {
	return &RulesRepository{db: db}
}

func (r *RulesRepository) Upsert(userID, normalizedMerchant, categoryID string) (*models.CategoryRule, error) {
	cat, err := r.getCategoryByID(categoryID, userID)
	if err != nil {
		return nil, err // category not found or not owned by user
	}

	rule := &models.CategoryRule{}
	err = r.db.QueryRow(
		`INSERT INTO category_rules (user_id, normalized_merchant, category_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, normalized_merchant)
		 DO UPDATE SET category_id = EXCLUDED.category_id, updated_at = now()
		 RETURNING id, user_id, normalized_merchant, category_id, created_at, updated_at`,
		userID, normalizedMerchant, categoryID,
	).Scan(&rule.ID, &rule.UserID, &rule.NormalizedMerchant, &rule.CategoryID, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return nil, err
	}
	rule.Category = cat
	return rule, nil
}

func (r *RulesRepository) ApplyToPast(userID, normalizedMerchant, categoryID string) (int64, error) {
	res, err := r.db.Exec(
		`UPDATE transactions SET category_id = $1, updated_at = now()
		 WHERE id IN (
		   SELECT id FROM transactions
		   WHERE user_id = $2
		     AND LOWER(TRIM(COALESCE(merchant_name, name))) = $3
		     AND pending = false
		   ORDER BY date DESC
		   LIMIT 500
		 )`,
		categoryID, userID, normalizedMerchant,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *RulesRepository) List(userID string) ([]*models.CategoryRule, error) {
	rows, err := r.db.Query(
		`SELECT cr.id, cr.user_id, cr.normalized_merchant, cr.category_id, cr.created_at, cr.updated_at,
		        c.id, c.user_id, c.name, c.icon, c.color, c.is_income, c.sort_order, c.plaid_primary_categories, c.created_at, c.updated_at
		 FROM category_rules cr
		 JOIN categories c ON c.id = cr.category_id
		 WHERE cr.user_id = $1
		 ORDER BY cr.normalized_merchant`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.CategoryRule
	for rows.Next() {
		rule := &models.CategoryRule{}
		cat := &models.Category{}
		if err := rows.Scan(
			&rule.ID, &rule.UserID, &rule.NormalizedMerchant, &rule.CategoryID, &rule.CreatedAt, &rule.UpdatedAt,
			&cat.ID, &cat.UserID, &cat.Name, &cat.Icon, &cat.Color, &cat.IsIncome, &cat.SortOrder,
			pq.Array(&cat.PlaidPrimaryCategories), &cat.CreatedAt, &cat.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rule.Category = cat
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *RulesRepository) Delete(userID, id string) error {
	res, err := r.db.Exec(`DELETE FROM category_rules WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *RulesRepository) getCategoryByID(categoryID, userID string) (*models.Category, error) {
	cat := &models.Category{}
	err := r.db.QueryRow(
		`SELECT id, user_id, name, icon, color, is_income, sort_order, plaid_primary_categories, created_at, updated_at
		 FROM categories WHERE id = $1 AND user_id = $2`, categoryID, userID,
	).Scan(&cat.ID, &cat.UserID, &cat.Name, &cat.Icon, &cat.Color, &cat.IsIncome, &cat.SortOrder,
		pq.Array(&cat.PlaidPrimaryCategories), &cat.CreatedAt, &cat.UpdatedAt)
	return cat, err
}
