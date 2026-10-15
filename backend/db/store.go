package db

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
}

type Event struct {
	ID                   uuid.UUID
	Title                string
	Description          *string
	URL                  *string
	StartsAt             time.Time
	EndsAt               time.Time
	Kind                 string
	Location             *string
	CommuteBeforeSeconds int
	CommuteAfterSeconds  int
	MainTagID            uuid.UUID
}

type Task struct {
	ID          uuid.UUID
	Title       string
	Description *string
	URL         *string
	DeadlineAt  time.Time
	IsCompleted bool
}

type Tag struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	IsDefault bool      `json:"is_default"`
}

// SharedEvent combines the owner's current event fields with the recipient's
// local overrides. Original* values come from the joined source event; they
// are not duplicated columns in the sharing tables.
type SharedEvent struct {
	ShareID             uuid.UUID
	EventID             uuid.UUID
	RecipientUserID     uuid.UUID
	OriginalTitle       string
	OriginalDescription *string
	OriginalURL         *string
	OriginalLocation    *string
	StartsAt            time.Time
	EndsAt              time.Time
	Kind                string
	LocalTitle          *string
	LocalDescription    *string
	LocalURL            *string
	LocalLocation       *string
	PrivateCustomInfo   *string
}

type TimelineEntry struct {
	ID                   uuid.UUID  `json:"id"`
	Type                 string     `json:"type"`
	Title                string     `json:"title"`
	Description          *string    `json:"description,omitempty"`
	URL                  *string    `json:"url,omitempty"`
	StartsAt             *time.Time `json:"starts_at,omitempty"`
	EndsAt               *time.Time `json:"ends_at,omitempty"`
	DeadlineAt           *time.Time `json:"deadline_at,omitempty"`
	Kind                 *string    `json:"kind,omitempty"`
	Location             *string    `json:"location,omitempty"`
	CommuteBeforeSeconds int        `json:"commute_before_seconds,omitempty"`
	CommuteAfterSeconds  int        `json:"commute_after_seconds,omitempty"`
	IsCompleted          bool       `json:"is_completed"`
	IsShared             bool       `json:"is_shared,omitempty"`
	SourceTitle          *string    `json:"source_title,omitempty"`
	CustomInfo           *string    `json:"custom_info,omitempty"`
	Tags                 []Tag      `json:"tags,omitempty"`
	MainTagID            *uuid.UUID `json:"main_tag_id,omitempty"`
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

var (
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrSessionNotFound     = errors.New("session not found")
	ErrSharedEventNotFound = errors.New("shared event not found")
	ErrTagAlreadyExists    = errors.New("tag already exists")
	ErrTagNotFound         = errors.New("tag not found")
	ErrLastTag             = errors.New("at least one tag is required")
	ErrDefaultTagNotFound  = errors.New("default tag not found")
)

func NewDBPool(ctx context.Context, connURL string) (*pgxpool.Pool, error) {

	pool, err := pgxpool.New(ctx, connURL)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func (s *Store) Register(ctx context.Context, email, passwordHash string) (uuid.UUID, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	userID := uuid.New()
	tagID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, default_tag_id)
		VALUES ($1, $2, $3, $4)
	`, userID, email, passwordHash, tagID)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return uuid.Nil, ErrEmailAlreadyExists
		}
		return uuid.Nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO tags (id, user_id, name) VALUES ($1, $2, 'general')
	`, tagID, userID); err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := s.db.QueryRow(
		ctx,
		`
		SELECT id, email, password_hash
		FROM users
		WHERE email = $1
		`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash)
	return user, err
}

func (s *Store) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time) error {
	_, err := s.db.Exec(
		ctx,
		`
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		`,
		userID,
		tokenHash,
		expiresAt,
	)
	return err
}

func (s *Store) FindUserBySession(ctx context.Context, tokenHash []byte) (User, error) {
	var user User
	err := s.db.QueryRow(
		ctx,
		`
		SELECT u.id, u.email, u.password_hash
		FROM sessions AS s
		JOIN users AS u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()
		`,
		tokenHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrSessionNotFound
	}
	return user, err
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM sessions
		WHERE token_hash = $1
	`, tokenHash)
	return err
}

func (s *Store) CreateEvent(ctx context.Context, userID uuid.UUID, event Event) (uuid.UUID, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	if event.MainTagID != uuid.Nil {
		var validMainTag bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tags WHERE id = $1 AND user_id = $2)`, event.MainTagID, userID).Scan(&validMainTag); err != nil {
			return uuid.Nil, err
		}
		if !validMainTag {
			return uuid.Nil, ErrTagNotFound
		}
	}

	var itemID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO timeline_items (user_id, title, description, url)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, userID, event.Title, event.Description, event.URL).Scan(&itemID)
	if err != nil {
		return uuid.Nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO events (
			timeline_item_id, starts_at, ends_at, kind, location,
			commute_before_seconds, commute_after_seconds, main_tag_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE(NULLIF($8, '00000000-0000-0000-0000-000000000000'::uuid), (SELECT default_tag_id FROM users WHERE id = $9)))
	`, itemID, event.StartsAt, event.EndsAt, event.Kind, event.Location,
		event.CommuteBeforeSeconds, event.CommuteAfterSeconds, event.MainTagID, userID)
	if err != nil {
		return uuid.Nil, err
	}

	result, err := tx.Exec(ctx, `
		INSERT INTO timeline_item_tags (timeline_item_id, tag_id)
		SELECT $1, COALESCE(NULLIF($2, '00000000-0000-0000-0000-000000000000'::uuid), default_tag_id)
		FROM users WHERE id = $3
	`, itemID, event.MainTagID, userID)
	if err != nil {
		return uuid.Nil, err
	}
	if result.RowsAffected() != 1 {
		return uuid.Nil, ErrDefaultTagNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return itemID, nil
}

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

func (s *Store) UpdateEvent(ctx context.Context, userID, eventID uuid.UUID, event Event) error {
	// An event spans the generic timeline row and the event subtype row, so
	// both updates must commit together.
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if event.MainTagID == uuid.Nil {
		if err := tx.QueryRow(ctx, `SELECT main_tag_id FROM events WHERE timeline_item_id = $1`, eventID).Scan(&event.MainTagID); err != nil {
			return err
		}
	}
	var validMainTag bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tags WHERE id = $1 AND user_id = $2)`, event.MainTagID, userID).Scan(&validMainTag); err != nil {
		return err
	}
	if !validMainTag {
		return ErrTagNotFound
	}
	var attachedMainTag bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM timeline_item_tags AS item_tag
			JOIN timeline_items AS item ON item.id = item_tag.timeline_item_id
			WHERE item_tag.timeline_item_id = $1
			  AND item_tag.tag_id = $2
			  AND item.user_id = $3
		)
	`, eventID, event.MainTagID, userID).Scan(&attachedMainTag); err != nil {
		return err
	}
	if !attachedMainTag {
		return ErrTagNotFound
	}

	result, err := tx.Exec(ctx, `
		UPDATE timeline_items
		SET title = $3, description = $4, url = $5, updated_at = now()
		WHERE id = $1 AND user_id = $2
	`, eventID, userID, event.Title, event.Description, event.URL)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	result, err = tx.Exec(ctx, `
		UPDATE events
		SET starts_at = $2, ends_at = $3, kind = $4, location = $5,
			commute_before_seconds = $6, commute_after_seconds = $7, main_tag_id = $8
		WHERE timeline_item_id = $1
	`, eventID, event.StartsAt, event.EndsAt, event.Kind, event.Location,
		event.CommuteBeforeSeconds, event.CommuteAfterSeconds, event.MainTagID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return tx.Commit(ctx)
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

func (s *Store) ShareEvent(ctx context.Context, ownerID, eventID, recipientID uuid.UUID) (uuid.UUID, error) {
	var shareID uuid.UUID
	err := s.db.QueryRow(ctx, `
		INSERT INTO event_shares (event_id, recipient_user_id)
		SELECT e.timeline_item_id, $3
		FROM events AS e
		JOIN timeline_items AS item ON item.id = e.timeline_item_id
		WHERE e.timeline_item_id = $2
		  AND item.user_id = $1
		  AND item.user_id <> $3
		RETURNING id
	`, ownerID, eventID, recipientID).Scan(&shareID)
	return shareID, err
}

func (s *Store) FindSharedEvent(ctx context.Context, recipientID, shareID uuid.UUID) (SharedEvent, error) {
	var event SharedEvent
	err := s.db.QueryRow(ctx, `
		-- Source fields come from the owner's event. Override fields below are
		-- recipient-local and may be absent until the recipient changes them.
		SELECT
			share.id, share.event_id, share.recipient_user_id,
			item.title, item.description, item.url, event.location,
			event.starts_at, event.ends_at, event.kind,
			override.title, override.description, override.url,
			override.location, override.custom_info
		FROM event_shares AS share
		JOIN timeline_items AS item ON item.id = share.event_id
		JOIN events AS event ON event.timeline_item_id = share.event_id
		LEFT JOIN event_share_overrides AS override ON override.event_share_id = share.id
		WHERE share.id = $1 AND share.recipient_user_id = $2
	`, shareID, recipientID).Scan(
		&event.ShareID, &event.EventID, &event.RecipientUserID,
		&event.OriginalTitle, &event.OriginalDescription, &event.OriginalURL,
		&event.OriginalLocation, &event.StartsAt, &event.EndsAt, &event.Kind,
		&event.LocalTitle, &event.LocalDescription, &event.LocalURL,
		&event.LocalLocation, &event.PrivateCustomInfo,
	)
	return event, err
}

func (s *Store) UpdateSharedEventOverride(ctx context.Context, recipientID, shareID uuid.UUID, event SharedEvent) error {
	result, err := s.db.Exec(ctx, `
		INSERT INTO event_share_overrides (
			event_share_id, title, description, url, location, custom_info
		)
		SELECT share.id, $3, $4, $5, $6, $7
		FROM event_shares AS share
		WHERE share.id = $1 AND share.recipient_user_id = $2
		ON CONFLICT (event_share_id) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			url = EXCLUDED.url,
			location = EXCLUDED.location,
			custom_info = EXCLUDED.custom_info,
			updated_at = now()
	`, shareID, recipientID, event.LocalTitle, event.LocalDescription,
		event.LocalURL, event.LocalLocation, event.PrivateCustomInfo)
	if err == nil && result.RowsAffected() == 0 {
		return ErrSharedEventNotFound
	}
	return err
}

func (s *Store) ListTimeline(ctx context.Context, userID uuid.UUID) ([]TimelineEntry, error) {
	// Shared events are projected into the recipient's timeline; they are not
	// copied into timeline_items owned by the recipient.
	entries, err := s.listOwnedTimeline(ctx, userID)
	if err != nil {
		return nil, err
	}

	sharedEntries, err := s.listSharedTimeline(ctx, userID)
	if err != nil {
		return nil, err
	}
	entries = append(entries, sharedEntries...)
	if err := s.populateTimelineTags(ctx, userID, entries); err != nil {
		return nil, err
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return timelineEntryTime(entries[i]).Before(timelineEntryTime(entries[j]))
	})
	return entries, nil
}

func (s *Store) populateTimelineTags(ctx context.Context, userID uuid.UUID, entries []TimelineEntry) error {
	// Tags are personal organization, so only tags and items owned by this
	// user may appear in the timeline response.
	rows, err := s.db.Query(ctx, `
		SELECT item_tag.timeline_item_id, tag.id, tag.name, tag.color
		FROM timeline_item_tags AS item_tag
		JOIN timeline_items AS item ON item.id = item_tag.timeline_item_id
		JOIN tags AS tag ON tag.id = item_tag.tag_id
		WHERE item.user_id = $1
	`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	tagsByItem := make(map[uuid.UUID][]Tag)
	for rows.Next() {
		var itemID uuid.UUID
		var tag Tag
		if err := rows.Scan(&itemID, &tag.ID, &tag.Name, &tag.Color); err != nil {
			return err
		}
		tagsByItem[itemID] = append(tagsByItem[itemID], tag)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for index := range entries {
		entries[index].Tags = tagsByItem[entries[index].ID]
	}
	return nil
}

func (s *Store) listOwnedTimeline(ctx context.Context, userID uuid.UUID) ([]TimelineEntry, error) {
	entries := make([]TimelineEntry, 0)
	eventRows, err := s.db.Query(ctx, `
		SELECT item.id, item.title, item.description, item.url,
			event.starts_at, event.ends_at, event.kind, event.location,
			event.commute_before_seconds, event.commute_after_seconds, event.main_tag_id
		FROM timeline_items AS item
		JOIN events AS event ON event.timeline_item_id = item.id
		WHERE item.user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	for eventRows.Next() {
		var entry TimelineEntry
		entry.Type = "event"
		if err := eventRows.Scan(
			&entry.ID, &entry.Title, &entry.Description, &entry.URL,
			&entry.StartsAt, &entry.EndsAt, &entry.Kind, &entry.Location,
			&entry.CommuteBeforeSeconds, &entry.CommuteAfterSeconds,
			&entry.MainTagID,
		); err != nil {
			eventRows.Close()
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := eventRows.Err(); err != nil {
		eventRows.Close()
		return nil, err
	}
	eventRows.Close()

	taskRows, err := s.db.Query(ctx, `
		SELECT item.id, item.title, item.description, item.url,
			task.deadline_at, task.is_completed
		FROM timeline_items AS item
		JOIN tasks AS task ON task.timeline_item_id = item.id
		WHERE item.user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	for taskRows.Next() {
		var entry TimelineEntry
		entry.Type = "task"
		if err := taskRows.Scan(
			&entry.ID, &entry.Title, &entry.Description, &entry.URL,
			&entry.DeadlineAt, &entry.IsCompleted,
		); err != nil {
			taskRows.Close()
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := taskRows.Err(); err != nil {
		taskRows.Close()
		return nil, err
	}
	taskRows.Close()
	return entries, nil
}

func (s *Store) listSharedTimeline(ctx context.Context, userID uuid.UUID) ([]TimelineEntry, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			share.event_id,
			COALESCE(override.title, item.title),
			COALESCE(override.description, item.description),
			COALESCE(override.url, item.url),
			event.starts_at, event.ends_at, event.kind,
			COALESCE(override.location, event.location),
			event.commute_before_seconds, event.commute_after_seconds,
			item.title, override.custom_info, event.main_tag_id
		FROM event_shares AS share
		JOIN timeline_items AS item ON item.id = share.event_id
		JOIN events AS event ON event.timeline_item_id = share.event_id
		LEFT JOIN event_share_overrides AS override ON override.event_share_id = share.id
		WHERE share.recipient_user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]TimelineEntry, 0)
	for rows.Next() {
		var entry TimelineEntry
		entry.Type = "event"
		entry.IsShared = true
		if err := rows.Scan(
			&entry.ID, &entry.Title, &entry.Description, &entry.URL,
			&entry.StartsAt, &entry.EndsAt, &entry.Kind, &entry.Location,
			&entry.CommuteBeforeSeconds, &entry.CommuteAfterSeconds,
			&entry.SourceTitle, &entry.CustomInfo, &entry.MainTagID,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Store) CreateTag(ctx context.Context, userID uuid.UUID, name string) (Tag, error) {
	var tag Tag
	err := s.db.QueryRow(ctx, `
		INSERT INTO tags (user_id, name)
		VALUES ($1, $2)
		RETURNING id, name, color
	`, userID, name).Scan(&tag.ID, &tag.Name, &tag.Color)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
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

func (s *Store) CountItemTags(ctx context.Context, itemID uuid.UUID) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM timeline_item_tags WHERE timeline_item_id = $1
	`, itemID).Scan(&count)
	return count, err
}

func (s *Store) SetDefaultTag(ctx context.Context, userID, tagID uuid.UUID) error {
	result, err := s.db.Exec(ctx, `
		UPDATE users
		SET default_tag_id = tags.id
		FROM tags
		WHERE users.id = $1 AND tags.id = $2 AND tags.user_id = $1
	`, userID, tagID)
	if err == nil && result.RowsAffected() == 0 {
		return ErrTagNotFound
	}
	return err
}

func (s *Store) RenameTag(ctx context.Context, userID, tagID uuid.UUID, name, color string) error {
	result, err := s.db.Exec(ctx, `
		UPDATE tags
		SET name = $3, color = $4
		WHERE id = $1 AND user_id = $2
	`, tagID, userID, name, color)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return ErrTagAlreadyExists
	}
	if err == nil && result.RowsAffected() == 0 {
		return ErrTagNotFound
	}
	return err
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

	var isDefault bool
	if err := tx.QueryRow(ctx, `SELECT default_tag_id = $2 FROM users WHERE id = $1`, userID, tagID).Scan(&isDefault); err != nil {
		return err
	}
	if isDefault {
		if _, err := tx.Exec(ctx, `
			UPDATE users SET default_tag_id = (
				SELECT id FROM tags WHERE user_id = $1 AND id <> $2 ORDER BY name LIMIT 1
			) WHERE id = $1
		`, userID, tagID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO timeline_item_tags (timeline_item_id, tag_id)
		SELECT event.timeline_item_id, replacement.id
		FROM events AS event
		JOIN timeline_items AS item ON item.id = event.timeline_item_id
		CROSS JOIN LATERAL (
			SELECT id FROM tags
			WHERE user_id = $1 AND id <> $2
			ORDER BY name LIMIT 1
		) AS replacement
		WHERE item.user_id = $1
		  AND NOT EXISTS (
			SELECT 1 FROM timeline_item_tags AS existing
			WHERE existing.timeline_item_id = event.timeline_item_id
		  )
		ON CONFLICT DO NOTHING
	`, userID, tagID); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `DELETE FROM tags WHERE id = $1 AND user_id = $2`, tagID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrTagNotFound
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
	if err == nil && result.RowsAffected() == 0 {
		return ErrTagNotFound
	}
	return err
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
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTagNotFound
		}
		return err
	}
	var tagCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM timeline_item_tags WHERE timeline_item_id = $1`, itemID).Scan(&tagCount); err != nil {
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
	if err == nil && result.RowsAffected() == 0 {
		return ErrTagNotFound
	}
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

func (s *Store) DeleteTimelineItem(ctx context.Context, userID, itemID uuid.UUID) error {
	result, err := s.db.Exec(ctx, `
		DELETE FROM timeline_items
		WHERE id = $1 AND user_id = $2
	`, itemID, userID)
	if err == nil && result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

func timelineEntryTime(entry TimelineEntry) time.Time {
	if entry.StartsAt != nil {
		return *entry.StartsAt
	}
	if entry.DeadlineAt != nil {
		return *entry.DeadlineAt
	}
	return time.Time{}
}
