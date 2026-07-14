package repositories

import (
	"database/sql"
	"sort"

	"github.com/lib/pq"
	"github.com/vladyslavivchenko/netme/internal/models"
)

type BudgetRepository struct {
	db *sql.DB
}

func NewBudgetRepository(db *sql.DB) *BudgetRepository {
	return &BudgetRepository{db: db}
}

// ─── Categories ───────────────────────────────────────────────────────────────

func (r *BudgetRepository) HasCategories(userID string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM categories WHERE user_id = $1`, userID).Scan(&count)
	return count > 0, err
}

func (r *BudgetRepository) SeedDefaultCategories(userID string) error {
	for _, d := range models.DefaultCategories {
		_, err := r.db.Exec(
			`INSERT INTO categories (user_id, name, icon, color, is_income, sort_order, plaid_primary_categories)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			userID, d.Name, d.Icon, d.Color, d.Income, d.Order, pq.Array(d.Plaid),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *BudgetRepository) GetCategories(userID string) ([]*models.Category, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, name, icon, color, is_income, sort_order, plaid_primary_categories, created_at, updated_at
		 FROM categories WHERE user_id = $1 ORDER BY sort_order, name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []*models.Category
	for rows.Next() {
		c := &models.Category{}
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Icon, &c.Color, &c.IsIncome, &c.SortOrder,
			pq.Array(&c.PlaidPrimaryCategories), &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (r *BudgetRepository) CreateCategory(userID, name, icon, color string, isIncome bool, plaid []string) (*models.Category, error) {
	c := &models.Category{}
	err := r.db.QueryRow(
		`INSERT INTO categories (user_id, name, icon, color, is_income, sort_order, plaid_primary_categories)
		 VALUES ($1, $2, $3, $4, $5, (SELECT COALESCE(MAX(sort_order),0)+1 FROM categories WHERE user_id=$1), $6)
		 RETURNING id, user_id, name, icon, color, is_income, sort_order, plaid_primary_categories, created_at, updated_at`,
		userID, name, icon, color, isIncome, pq.Array(plaid),
	).Scan(&c.ID, &c.UserID, &c.Name, &c.Icon, &c.Color, &c.IsIncome, &c.SortOrder,
		pq.Array(&c.PlaidPrimaryCategories), &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *BudgetRepository) UpdateCategory(id, userID, name, icon, color string, plaid []string) (*models.Category, error) {
	c := &models.Category{}
	err := r.db.QueryRow(
		`UPDATE categories SET name=$3, icon=$4, color=$5, plaid_primary_categories=$6, updated_at=now()
		 WHERE id=$1 AND user_id=$2
		 RETURNING id, user_id, name, icon, color, is_income, sort_order, plaid_primary_categories, created_at, updated_at`,
		id, userID, name, icon, color, pq.Array(plaid),
	).Scan(&c.ID, &c.UserID, &c.Name, &c.Icon, &c.Color, &c.IsIncome, &c.SortOrder,
		pq.Array(&c.PlaidPrimaryCategories), &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *BudgetRepository) DeleteCategory(id, userID string) error {
	_, err := r.db.Exec(`DELETE FROM categories WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

// ─── Budgets ──────────────────────────────────────────────────────────────────

func (r *BudgetRepository) SetBudget(userID, categoryID, month string, amount float64) (*models.Budget, error) {
	b := &models.Budget{}
	err := r.db.QueryRow(
		`INSERT INTO budgets (user_id, category_id, month, amount)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, category_id, month) DO UPDATE SET amount=EXCLUDED.amount, updated_at=now()
		 RETURNING id, user_id, category_id, month, amount, created_at, updated_at`,
		userID, categoryID, month, amount,
	).Scan(&b.ID, &b.UserID, &b.CategoryID, &b.Month, &b.Amount, &b.CreatedAt, &b.UpdatedAt)
	return b, err
}

// ─── Summary ──────────────────────────────────────────────────────────────────

func (r *BudgetRepository) GetTransactionsForMonth(userID, month string) ([]*models.Transaction, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, account_id, plaid_transaction_id, amount, currency_code, name, merchant_name,
		        to_char(date, 'YYYY-MM-DD'), to_char(authorized_date, 'YYYY-MM-DD'),
		        category, category_detailed, payment_channel, pending, category_id, created_at, updated_at
		 FROM transactions
		 WHERE user_id = $1 AND to_char(date, 'YYYY-MM') = $2 AND pending = false
		 ORDER BY date DESC`, userID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []*models.Transaction
	for rows.Next() {
		t := &models.Transaction{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.AccountID, &t.PlaidTransactionID, &t.Amount, &t.CurrencyCode,
			&t.Name, &t.MerchantName, &t.Date, &t.AuthorizedDate,
			&t.Category, &t.CategoryDetailed, &t.PaymentChannel, &t.Pending,
			&t.CategoryID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}

func (r *BudgetRepository) GetBudgetsForMonth(userID, month string) (map[string]float64, error) {
	rows, err := r.db.Query(
		`SELECT category_id, amount FROM budgets WHERE user_id=$1 AND month=$2`, userID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]float64)
	for rows.Next() {
		var catID string
		var amount float64
		if err := rows.Scan(&catID, &amount); err != nil {
			return nil, err
		}
		m[catID] = amount
	}
	return m, rows.Err()
}

func (r *BudgetRepository) GetMonthlyHistory(userID string, months int) ([]models.MonthlyTotal, error) {
	rows, err := r.db.Query(
		`SELECT to_char(date, 'YYYY-MM') AS month,
		        SUM(CASE WHEN amount < 0 THEN ABS(amount) ELSE 0 END) AS income,
		        SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END) AS spending
		 FROM transactions
		 WHERE user_id = $1 AND pending = false
		   AND date >= date_trunc('month', now()) - ($2::int - 1) * interval '1 month'
		 GROUP BY 1 ORDER BY 1`, userID, months)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.MonthlyTotal
	for rows.Next() {
		var m models.MonthlyTotal
		if err := rows.Scan(&m.Month, &m.Income, &m.Spending); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// GetTopCategories returns the top N spending categories for a given month.
// It reuses BuildSummary's category-mapping logic via a dedicated SQL query for efficiency.
func (r *BudgetRepository) GetTopCategories(userID, month string, limit int) ([]models.TopCategory, error) {
	if err := r.EnsureCategories(userID); err != nil {
		return nil, err
	}

	summary, err := r.BuildSummary(userID, month)
	if err != nil {
		return nil, err
	}

	// Collect non-income categories with spending, sort descending, take top N.
	spending := make([]models.CategorySummary, 0, len(summary.Categories))
	var totalSpending float64
	for _, c := range summary.Categories {
		if !c.IsIncome && c.Spent > 0 {
			spending = append(spending, c)
			totalSpending += c.Spent
		}
	}
	sort.Slice(spending, func(i, j int) bool { return spending[i].Spent > spending[j].Spent })

	if limit > len(spending) {
		limit = len(spending)
	}
	top := make([]models.TopCategory, limit)
	for i, c := range spending[:limit] {
		pct := 0.0
		if totalSpending > 0 {
			pct = c.Spent / totalSpending * 100
		}
		top[i] = models.TopCategory{
			CategoryID: c.ID,
			Name:       c.Name,
			Icon:       c.Icon,
			Color:      c.Color,
			Spent:      c.Spent,
			Pct:        pct,
		}
	}
	return top, nil
}

// EnsureCategories seeds defaults if user has none yet.
func (r *BudgetRepository) EnsureCategories(userID string) error {
	has, err := r.HasCategories(userID)
	if err != nil || has {
		return err
	}
	return r.SeedDefaultCategories(userID)
}

// BuildSummary maps transactions → categories in Go and returns totals.
func (r *BudgetRepository) BuildSummary(userID, month string) (*models.BudgetSummary, error) {
	if err := r.EnsureCategories(userID); err != nil {
		return nil, err
	}

	cats, err := r.GetCategories(userID)
	if err != nil {
		return nil, err
	}
	txns, err := r.GetTransactionsForMonth(userID, month)
	if err != nil {
		return nil, err
	}
	budgetMap, err := r.GetBudgetsForMonth(userID, month)
	if err != nil {
		return nil, err
	}

	// Build lookup maps
	catByID := make(map[string]*models.Category, len(cats))
	plaidToCat := make(map[string]*models.Category)
	var catchAll *models.Category
	for _, c := range cats {
		catByID[c.ID] = c
		if len(c.PlaidPrimaryCategories) == 0 {
			catchAll = c
			continue
		}
		for _, p := range c.PlaidPrimaryCategories {
			plaidToCat[p] = c
		}
	}

	spent := make(map[string]float64)
	count := make(map[string]int)

	for _, t := range txns {
		var cat *models.Category
		// Prefer explicit user override; fall back to Plaid mapping
		if t.CategoryID != nil {
			cat = catByID[*t.CategoryID]
		}
		if cat == nil && t.Category != nil {
			cat = plaidToCat[*t.Category]
		}
		if cat == nil {
			cat = catchAll
		}
		if cat == nil {
			continue
		}
		spent[cat.ID] += t.Amount
		count[cat.ID]++
	}

	summary := &models.BudgetSummary{Month: month}
	for _, c := range cats {
		s := spent[c.ID]
		cs := models.CategorySummary{
			Category:         *c,
			Spent:            s,
			BudgetLimit:      budgetMap[c.ID],
			TransactionCount: count[c.ID],
		}
		if c.IsIncome {
			summary.Income += s
		} else {
			summary.Spending += s
		}
		summary.Categories = append(summary.Categories, cs)
	}
	if summary.Categories == nil {
		summary.Categories = []models.CategorySummary{}
	}
	return summary, nil
}
