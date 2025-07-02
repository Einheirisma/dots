package repository

import (
	"database/sql"
	"log"
	"time"
)

type Notification struct {
	ID        string       `json:"id"`
	UserID    int          `json:"user_id"`
	Channel   string       `json:"channel"`
	Recipient string       `json:"recipient"`
	Subject   string       `json:"subject"`
	Body      string       `json:"body"`
	Status    string       `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	SentAt    sql.NullTime `json:"sent_at,omitempty"`
}

type NotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(n *Notification) error {
	_, err := r.db.Exec(
		"INSERT INTO notifications (id, user_id, channel, recipient, subject, body, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		n.ID,
		n.UserID,
		n.Channel,
		n.Recipient,
		n.Subject,
		n.Body,
		n.Status,
		time.Now(),
	)
	return err
}

func (r *NotificationRepository) UpdateStatus(id, status string) error {
	_, err := r.db.Exec(
		"UPDATE notifications SET status = ?, sent_at = ? WHERE id = ?",
		status,
		time.Now(),
		id,
	)
	return err
}

func (r *NotificationRepository) FindByUser(userID int, limit int, channel string) ([]Notification, error) {
	query := "SELECT id, user_id, channel, recipient, subject, body, status, created_at, sent_at FROM notifications WHERE user_id = ?"
	args := []interface{}{userID}

	if channel != "" && channel != "all" {
		query += " AND channel = ?"
		args = append(args, channel)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		log.Printf("Database query error: %v", err)
		return nil, err
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(
			&n.ID,
			&n.UserID,
			&n.Channel,
			&n.Recipient,
			&n.Subject,
			&n.Body,
			&n.Status,
			&n.CreatedAt,
			&n.SentAt,
		); err != nil {
			log.Printf("Scan error: %v", err)
			return nil, err
		}
		notifications = append(notifications, n)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Rows error: %v", err)
		return nil, err
	}

	return notifications, nil
}

func (r *NotificationRepository) GetStats(userID int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var total int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM notifications WHERE user_id = ?",
		userID,
	).Scan(&total)
	if err != nil {
		return nil, err
	}
	stats["total"] = total

	channels := []string{"email", "telegram", "whatsapp"}
	for _, channel := range channels {
		var count int
		err := r.db.QueryRow(
			"SELECT COUNT(*) FROM notifications WHERE user_id = ? AND channel = ?",
			userID, channel,
		).Scan(&count)
		if err != nil {
			stats[channel+"_count"] = 0
		} else {
			stats[channel+"_count"] = count
		}
	}

	statuses := []string{"queued", "sent", "failed"}
	for _, status := range statuses {
		var count int
		err := r.db.QueryRow(
			"SELECT COUNT(*) FROM notifications WHERE user_id = ? AND status = ?",
			userID, status,
		).Scan(&count)
		if err != nil {
			stats[status] = 0
		} else {
			stats[status] = count
		}
	}

	return stats, nil
}
