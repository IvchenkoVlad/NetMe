package models

type NetWorth struct {
	Assets      float64 `json:"assets"`
	Liabilities float64 `json:"liabilities"`
	NetWorth    float64 `json:"net_worth"`
}

type TopCategory struct {
	CategoryID string  `json:"category_id"`
	Name       string  `json:"name"`
	Icon       string  `json:"icon"`
	Color      string  `json:"color"`
	Spent      float64 `json:"spent"`
	Pct        float64 `json:"pct"`
}

type AnalyticsOverview struct {
	NetWorth      NetWorth       `json:"net_worth"`
	MonthlyTotals []MonthlyTotal `json:"monthly_totals"`
	TopCategories []TopCategory  `json:"top_categories"`
}
