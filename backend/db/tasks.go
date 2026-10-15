package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateTask(ctx context.Context, userID uuid.UUID, task Task) (uuid.UUID, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	var itemID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO timeline_items (user_id, title, description, url)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, userID, task.Title, task.Description, task.URL).Scan(&itemID)
	if err != nil {
		return uuid.Nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO tasks (timeline_item_id, deadline_at, is_completed)
		VALUES ($1, $2, $3)
	`, itemID, task.DeadlineAt, task.IsCompleted)
	if err != nil {
		return uuid.Nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return itemID, nil
}

func (s *Store) UpdateTask(ctx context.Context, userID, taskID uuid.UUID, task Task) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// The owner check is performed in the same update as the mutation so a
	// caller cannot edit another user's task by guessing its UUID.
	result, err := tx.Exec(ctx, `
		UPDATE timeline_items AS item
		SET title = $3, description = $4, url = $5, updated_at = now()
		FROM tasks
		WHERE tasks.timeline_item_id = item.id
		  AND item.id = $1 AND item.user_id = $2
	`, taskID, userID, task.Title, task.Description, task.URL)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	_, err = tx.Exec(ctx, `
		UPDATE tasks
		SET deadline_at = $2
		WHERE timeline_item_id = $1
	`, taskID, task.DeadlineAt)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) SetTaskCompleted(ctx context.Context, userID, taskID uuid.UUID, completed bool) error {
	result, err := s.db.Exec(ctx, `
		UPDATE tasks AS task
		SET is_completed = $3
		FROM timeline_items AS item
		WHERE task.timeline_item_id = item.id
		  AND task.timeline_item_id = $1
		  AND item.user_id = $2
	`, taskID, userID, completed)
	if err == nil && result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
