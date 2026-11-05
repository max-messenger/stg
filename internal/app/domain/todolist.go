package domain

import "time"

type Todo struct {
	ID      int64
	UserID  int64
	Message string
	Done    bool
	Created time.Time
}
