package db

import (
	"context"

	"github.com/google/uuid"
)

func (s *Store) CreateTag(ctx context.Context, userID uuid.UUID, name string) (Tag, error) {
	var tag Tag
	err := s.db.QueryRow(ctx, `
		INSERT INTO tags (user_id, name)
		VALUES ($1, $2)
		RETURNING id, name, color
	`, userID, name).Scan(&tag.ID, &tag.Name, &tag.Color)
	if isUniqueViolation(err) {
		return Tag{}, ErrTagAlreadyExists
	}
	return tag, err
}

func (s *Store) ListTags(ctx context.Context, userID uuid.UUID) ([]Tag, error) {
	rows, err := s.db.Query(ctx, `
		SELECT tag.id, tag.name, tag.color, tag.id = users.default_tag_id
		FROM tags AS tag
		JOIN users ON users.id = tag.user_id
		WHERE tag.user_id = $1
		ORDER BY tag.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := make([]Tag, 0)
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.IsDefault); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (s *Store) CountItemTags(ctx context.Context, userID, itemID uuid.UUID) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, `
		SELECT count(*)
		FROM timeline_item_tags AS item_tag
		JOIN timeline_items AS item ON item.id = item_tag.timeline_item_id
		WHERE item_tag.timeline_item_id = $1 AND item.user_id = $2
	`, itemID, userID).Scan(&count)
	return count, err
}

func (s *Store) SetDefaultTag(ctx context.Context, userID, tagID uuid.UUID) error {
	result, err := s.db.Exec(ctx, `
		UPDATE users
		SET default_tag_id = tags.id
		FROM tags
		WHERE users.id = $1 AND tags.id = $2 AND tags.user_id = $1
	`, userID, tagID)
	return requireRowsAffected(result, err, ErrTagNotFound)
}

func (s *Store) RenameTag(ctx context.Context, userID, tagID uuid.UUID, name, color string) error {
	result, err := s.db.Exec(ctx, `
		UPDATE tags
		SET name = $3, color = $4
		WHERE id = $1 AND user_id = $2
	`, tagID, userID, name, color)
	if isUniqueViolation(err) {
		return ErrTagAlreadyExists
	}
	return requireRowsAffected(result, err, ErrTagNotFound)
}

func (s *Store) DeleteTag(ctx context.Context, userID, tagID uuid.UUID) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var tagCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM tags WHERE user_id = $1`, userID).Scan(&tagCount); err != nil {
		return err
	}
	if tagCount <= 1 {
		return ErrLastTag
	}
	var replacementTagID uuid.UUID
	if err := tx.QueryRow(ctx, `
		SELECT id FROM tags WHERE user_id = $1 AND id <> $2 ORDER BY name LIMIT 1
	`, userID, tagID).Scan(&replacementTagID); err != nil {
		return err
	}

	var isDefault bool
	if err := tx.QueryRow(ctx, `SELECT default_tag_id = $2 FROM users WHERE id = $1`, userID, tagID).Scan(&isDefault); err != nil {
		return err
	}
	if isDefault {
		if _, err := tx.Exec(ctx, `
			UPDATE users SET default_tag_id = $2 WHERE id = $1
		`, userID, replacementTagID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO timeline_item_tags (timeline_item_id, tag_id)
		SELECT event.timeline_item_id, $3
		FROM events AS event
		JOIN timeline_items AS item ON item.id = event.timeline_item_id
		WHERE item.user_id = $1
		  AND (event.main_tag_id = $2 OR NOT EXISTS (
			SELECT 1 FROM timeline_item_tags AS existing
			WHERE existing.timeline_item_id = event.timeline_item_id
			  AND existing.tag_id <> $2
		  ))
		ON CONFLICT DO NOTHING
	`, userID, tagID, replacementTagID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE events AS event
		SET main_tag_id = $3
		FROM timeline_items AS item
		WHERE item.id = event.timeline_item_id
		  AND item.user_id = $1
		  AND event.main_tag_id = $2
	`, userID, tagID, replacementTagID); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `DELETE FROM tags WHERE id = $1 AND user_id = $2`, tagID, userID)
	if err := requireRowsAffected(result, err, ErrTagNotFound); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) AddTagToItem(ctx context.Context, userID, itemID, tagID uuid.UUID) error {
	result, err := s.db.Exec(ctx, `
		INSERT INTO timeline_item_tags (timeline_item_id, tag_id)
		SELECT item.id, tag.id
		FROM timeline_items AS item
		JOIN tags AS tag ON tag.id = $3
		WHERE item.id = $2 AND item.user_id = $1 AND tag.user_id = $1
		ON CONFLICT DO NOTHING
	`, userID, itemID, tagID)
	return requireRowsAffected(result, err, ErrTagNotFound)
}

func (s *Store) RemoveTagFromItem(ctx context.Context, userID, itemID, tagID uuid.UUID) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var isEvent bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM events WHERE timeline_item_id = $1)
		FROM timeline_items
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`, itemID, userID).Scan(&isEvent); err != nil {
		if isNoRows(err) {
			return ErrTagNotFound
		}
		return err
	}
	var tagCount int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM timeline_item_tags AS item_tag
		JOIN timeline_items AS item ON item.id = item_tag.timeline_item_id
		WHERE item_tag.timeline_item_id = $1 AND item.user_id = $2
	`, itemID, userID).Scan(&tagCount); err != nil {
		return err
	}
	if isEvent && tagCount <= 1 {
		return ErrLastTag
	}
	result, err := tx.Exec(ctx, `
		DELETE FROM timeline_item_tags AS item_tag
		USING timeline_items AS item, tags AS tag
		WHERE item_tag.timeline_item_id = item.id
		  AND item_tag.tag_id = tag.id
		  AND item.id = $1 AND tag.id = $2
		  AND item.user_id = $3 AND tag.user_id = $3
	`, itemID, tagID, userID)
	if err := requireRowsAffected(result, err, ErrTagNotFound); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
