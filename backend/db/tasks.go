package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateTask(ctx context.Context, userID uuid.UUID, task Task) (uuid.UUID, error) {
	var itemID uuid.UUID
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		itemID, err = insertTimelineItem(ctx, tx, userID, task.Title, task.Description, task.URL)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO tasks (timeline_item_id, deadline_at, is_completed)
			VALUES ($1, $2, $3)
		`, itemID, task.DeadlineAt, task.IsCompleted)
		return err
	})
	if err != nil {
		return uuid.Nil, err
	}
	return itemID, nil
}

func (s *Store) UpdateTask(ctx context.Context, userID, taskID uuid.UUID, task Task) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		// The owner check is performed in the same update as the mutation so a
		// caller cannot edit another user's task by guessing its UUID.
		result, err := tx.Exec(ctx, `
			UPDATE timeline_items AS item
			SET title = $3, description = $4, url = $5, updated_at = now()
			FROM tasks
			WHERE tasks.timeline_item_id = item.id
			  AND item.id = $1 AND item.user_id = $2
		`, taskID, userID, task.Title, task.Description, task.URL)
		if err := requireRowsAffected(result, err, ErrTaskNotFound); err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			UPDATE tasks
			SET deadline_at = $2
			WHERE timeline_item_id = $1
		`, taskID, task.DeadlineAt)
		if err != nil {
			return err
		}
		return nil
	})
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
	return requireRowsAffected(result, err, ErrTaskNotFound)
}
