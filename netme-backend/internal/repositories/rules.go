package repositories

import (
	"database/sql"
	"fmt"

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

// ApplyCategoryRules runs two bulk SQL updates for a user after a transaction sync:
//  1. Apply merchant rules (highest priority — user-defined).
//  2. Apply Plaid primary category → user category mapping for any still-uncategorized transactions.
//
// Only non-pending transactions without an existing category_id override are touched.
func (r *RulesRepository) ApplyCategoryRules(userID string) error {
	// Pass 1: merchant rules.
	_, err := r.db.Exec(
		`UPDATE transactions t
		 SET category_id = cr.category_id, updated_at = now()
		 FROM category_rules cr
		 WHERE t.user_id = $1
		   AND cr.user_id = $1
		   AND t.category_id IS NULL
		   AND t.pending = false
		   AND LOWER(TRIM(COALESCE(t.merchant_name, t.name))) = cr.normalized_merchant`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("apply merchant rules: %w", err)
	}

	// Pass 2: Plaid category → user category mapping.
	_, err = r.db.Exec(
		`UPDATE transactions t
		 SET category_id = c.id, updated_at = now()
		 FROM categories c
		 WHERE t.user_id = $1
		   AND c.user_id = $1
		   AND t.category_id IS NULL
		   AND t.pending = false
		   AND t.category IS NOT NULL
		   AND t.category = ANY(c.plaid_primary_categories)`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("apply plaid category mapping: %w", err)
	}

	return nil
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
