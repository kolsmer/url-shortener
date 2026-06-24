package models

import "time"

type URL struct {
	OriginalURL string    `json:"original_url"`
	ShortCode   string    `json:"short_code"`
	CreatedAt   time.Time `json:"created_at"`
	UserID      int64     `json:"user_id"`
}
