package repositories

import (
	"database/sql"
	"encoding/json"
)

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

// LogRawEvent stores any Plaid payload for debugging. userID may be empty for webhook events.
func (r *EventRepository) LogRawEvent(userID, eventType string, payload any) {
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	var uid *string
	if userID != "" {
		uid = &userID
	}
	_, _ = r.db.Exec(
		`INSERT INTO plaid_raw_events (user_id, event_type, payload) VALUES ($1, $2, $3)`,
		uid, eventType, string(b),
	)
}

// PurgeOldRawEvents deletes raw event rows older than the given number of days.
// Call weekly to cap table growth (GDPR Article 5(1)(e)).
func (r *EventRepository) PurgeOldRawEvents(olderThanDays int) (int64, error) {
	res, err := r.db.Exec(
		`DELETE FROM plaid_raw_events WHERE created_at < now() - ($1::int * interval '1 day')`,
		olderThanDays,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
