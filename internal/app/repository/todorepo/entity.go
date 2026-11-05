package todorepo

import (
	"github.com/guregu/null/v6"

	"stg/internal/app/domain"
)

type todoEntity struct {
	ID        int64       `db:"id"`
	UserID    null.Int64  `db:"user_id"`
	Message   null.String `db:"message"`
	Done      null.Bool   `db:"done"`
	CreatedAt null.Time   `db:"created_at"`
}

func (t todoEntity) toDomain() domain.Todo {
	return domain.Todo{
		ID:      t.ID,
		UserID:  t.UserID.Int64,
		Message: t.Message.String,
		Done:    t.Done.Bool,
		Created: t.CreatedAt.Time,
	}
}
