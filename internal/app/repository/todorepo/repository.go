package todorepo

import (
	"context"
	"fmt"

	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"

	"stg/internal/app/domain"
	"stg/pkg/connections/postgres"
)

const (
	tableName    = "todos"
	fieldID      = "id"
	fieldUserID  = "user_id"
	fieldMessage = "message"
	fieldDone    = "done"
	fieldCreated = "created_at"

	queryNameAddTodo  = "insert_todo"
	queryNameListTodo = "select_todo"
	queryNameEditTodo = "update_todo"

	ReadPGPool  = "stg-sel"
	WritePGPool = "stg-upd"
)

type Repository struct {
	read  *postgres.Postgres
	write *postgres.Postgres
}

func NewRepository(pool *postgres.Pool) (*Repository, error) {
	read, err := pool.GetPool(ReadPGPool)
	if err != nil {
		return nil, fmt.Errorf("get pool read: %w", err)
	}

	write, err := pool.GetPool(WritePGPool)
	if err != nil {
		return nil, fmt.Errorf("get pool write: %w", err)
	}

	return &Repository{
		read:  read,
		write: write,
	}, nil
}

func (r *Repository) Add(ctx context.Context, d domain.Todo) (int64, error) {
	sb := sqlbuilder.InsertInto(tableName)
	sb.Cols(fieldUserID, fieldMessage)
	sb.Values(d.UserID, d.Message)

	q, args := sb.Build()

	err := r.write.Call(ctx, queryNameAddTodo, func(ctx context.Context, db postgres.Queryable) error {
		row := db.QueryRow(ctx, fmt.Sprintf("%s returning %s", q, fieldID), args...)
		qErr := row.Scan(&d.ID)
		if qErr != nil {
			return qErr
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return d.ID, nil
}

func (r *Repository) Edit(ctx context.Context, d domain.Todo) (int64, error) {
	sb := sqlbuilder.Update(tableName)
	sb.Set(sb.Assign(fieldMessage, d.Message))

	sb.Where(sb.Equal(fieldUserID, d.UserID))
	sb.Where(sb.Equal(fieldID, d.ID))

	q, args := sb.Build()

	err := r.write.Call(ctx, queryNameEditTodo, func(ctx context.Context, db postgres.Queryable) error {
		row := db.QueryRow(ctx, fmt.Sprintf("%s returning %s", q, fieldID), args...)
		qErr := row.Scan(&d.ID)
		if qErr != nil {
			return qErr
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return d.ID, nil
}

func (r *Repository) List(ctx context.Context, userID int64) ([]domain.Todo, error) {
	sb := sqlbuilder.
		Select(fieldID, fieldUserID, fieldMessage, fieldDone, fieldCreated).
		From(tableName)
	sb.Where(sb.Equal(fieldUserID, userID))

	result := make([]domain.Todo, 0)
	q, args := sb.Build()

	err := r.read.Call(ctx, queryNameListTodo, func(ctx context.Context, db postgres.Queryable) error {
		rows, qErr := db.Query(ctx, q, args...)
		if qErr != nil {
			return qErr
		}

		defer rows.Close()

		items, err := pgx.CollectRows[todoEntity](rows, pgx.RowToStructByNameLax[todoEntity])
		if err != nil {
			return err
		}

		for i := range items {
			result = append(result, items[i].toDomain())
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) Delete(ctx context.Context, item domain.Todo) error {
	sb := sqlbuilder.DeleteFrom(tableName)
	sb.Where(sb.Equal(fieldUserID, item.UserID))
	sb.Where(sb.Equal(fieldID, item.ID))

	q, args := sb.Build()

	err := r.write.Call(ctx, queryNameAddTodo, func(ctx context.Context, db postgres.Queryable) error {
		_, qErr := db.Exec(ctx, q, args...)
		if qErr != nil {
			return qErr
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) Toggle(ctx context.Context, item domain.Todo) error {
	sb := sqlbuilder.Update(tableName)
	sb.Set(sb.Assign(fieldDone, item.Done))
	sb.Where(sb.Equal(fieldUserID, item.UserID))
	sb.Where(sb.Equal(fieldID, item.ID))

	q, args := sb.Build()

	err := r.write.Call(ctx, queryNameAddTodo, func(ctx context.Context, db postgres.Queryable) error {
		_, qErr := db.Exec(ctx, q, args...)
		if qErr != nil {
			return qErr
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
