package repository

import (
	"database/sql"
	"log"
)

type RateLimit struct {
	UserID      sql.NullInt64 `json:"user_id"`
	Channel     string        `json:"channel"`
	MaxRequests int           `json:"max_requests"`
	Interval    int           `json:"interval"`
}

type RateLimitRepository struct {
	db *sql.DB
}

func NewRateLimitRepository(db *sql.DB) *RateLimitRepository {
	return &RateLimitRepository{db: db}
}

func (r *RateLimitRepository) FindByUserAndChannel(userID int, channel string) (RateLimit, error) {
	var limit RateLimit

	query := `
		SELECT user_id, channel, max_requests, interval_seconds 
		FROM rate_limits 
		WHERE user_id = ? AND channel = ?
		ORDER BY user_id DESC 
		LIMIT 1`

	err := r.db.QueryRow(query, userID, channel).Scan(
		&limit.UserID,
		&limit.Channel,
		&limit.MaxRequests,
		&limit.Interval,
	)

	if err == nil {
		return limit, nil
	} else if err != sql.ErrNoRows {
		log.Printf("Error querying rate limits: %v", err)
	}

	query = `
		SELECT user_id, channel, max_requests, interval_seconds 
		FROM rate_limits 
		WHERE user_id IS NULL AND channel = ? 
		LIMIT 1`

	err = r.db.QueryRow(query, channel).Scan(
		&limit.UserID,
		&limit.Channel,
		&limit.MaxRequests,
		&limit.Interval,
	)

	if err == nil {
		return limit, nil
	} else if err != sql.ErrNoRows {
		log.Printf("Error querying default rate limits: %v", err)
	}

	return RateLimit{
		Channel:     channel,
		MaxRequests: 10,
		Interval:    60,
	}, nil
}
