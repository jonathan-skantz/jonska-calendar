package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Feed is a secret URL that publishes the user's own events, or only those
// with TagID.
type Feed struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Name          string
	Token         string
	TagID         *uuid.UUID
	IncludeAlarms bool
	TimeZone      string
	CreatedAt     time.Time
}

// FeedFields are what the owner sets on a feed.
type FeedFields struct {
	Name          string
	TagID         *uuid.UUID
	IncludeAlarms bool
}

const feedColumns = `id, user_id, name, token, tag_id, include_alarms, time_zone, created_at`

func scanFeed(row pgx.CollectableRow) (Feed, error) {
	var feed Feed
	err := row.Scan(&feed.ID, &feed.UserID, &feed.Name, &feed.Token, &feed.TagID,
		&feed.IncludeAlarms, &feed.TimeZone, &feed.CreatedAt)
	return feed, err
}

func collectFeed(rows pgx.Rows, err error, notFound error) (Feed, error) {
	if err != nil {
		return Feed{}, err
	}
	feed, err := pgx.CollectExactlyOneRow(rows, scanFeed)
	if isNoRows(err) {
		return Feed{}, notFound
	}
	return feed, err
}

// CreateFeed creates a feed. Its tag must be one of the user's own.
func (s *Store) CreateFeed(ctx context.Context, userID uuid.UUID, fields FeedFields, token, timeZone string) (Feed, error) {
	rows, err := s.db.Query(ctx, `
		INSERT INTO calendar_feeds (user_id, name, token, tag_id, include_alarms, time_zone)
		SELECT $1, $2, $3, $4, $5, $6
		WHERE $4::uuid IS NULL OR EXISTS (SELECT 1 FROM tags WHERE id = $4 AND user_id = $1)
		RETURNING `+feedColumns,
		userID, fields.Name, token, fields.TagID, fields.IncludeAlarms, timeZone)
	return collectFeed(rows, err, ErrTagNotFound)
}

func (s *Store) ListFeeds(ctx context.Context, userID uuid.UUID) ([]Feed, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+feedColumns+` FROM calendar_feeds
		WHERE user_id = $1
		ORDER BY created_at, id
	`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanFeed)
}

// UpdateFeed replaces a feed's fields. It reports ErrFeedNotFound for a feed
// that is not the user's, and ErrTagNotFound for a tag that is not.
func (s *Store) UpdateFeed(ctx context.Context, userID, feedID uuid.UUID, fields FeedFields) (Feed, error) {
	var feed Feed
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		if fields.TagID != nil {
			owned, err := tagOwnedBy(ctx, tx, *fields.TagID, userID)
			if err != nil {
				return err
			}
			if !owned {
				return ErrTagNotFound
			}
		}
		rows, err := tx.Query(ctx, `
			UPDATE calendar_feeds SET name = $3, tag_id = $4, include_alarms = $5
			WHERE id = $1 AND user_id = $2
			RETURNING `+feedColumns,
			feedID, userID, fields.Name, fields.TagID, fields.IncludeAlarms)
		feed, err = collectFeed(rows, err, ErrFeedNotFound)
		return err
	})
	return feed, err
}

// ResetFeedToken gives a feed a new token, so the old URL stops working.
func (s *Store) ResetFeedToken(ctx context.Context, userID, feedID uuid.UUID, token string) (Feed, error) {
	rows, err := s.db.Query(ctx, `
		UPDATE calendar_feeds SET token = $3
		WHERE id = $1 AND user_id = $2
		RETURNING `+feedColumns,
		feedID, userID, token)
	return collectFeed(rows, err, ErrFeedNotFound)
}

func (s *Store) DeleteFeed(ctx context.Context, userID, feedID uuid.UUID) error {
	result, err := s.db.Exec(ctx, `DELETE FROM calendar_feeds WHERE id = $1 AND user_id = $2`, feedID, userID)
	return requireRowsAffected(result, err, ErrFeedNotFound)
}

// FeedByToken finds a feed by its secret token. The token is the only
// credential a feed request has.
func (s *Store) FeedByToken(ctx context.Context, token string) (Feed, error) {
	rows, err := s.db.Query(ctx, `SELECT `+feedColumns+` FROM calendar_feeds WHERE token = $1`, token)
	return collectFeed(rows, err, ErrFeedNotFound)
}

// ListEventsForFeed returns the events a feed publishes: its owner's own
// events, limited to its tag, with the owner's reminders only when the feed
// includes alarms.
func (s *Store) ListEventsForFeed(ctx context.Context, feed Feed) ([]ExportEvent, error) {
	return s.listEventsForExport(ctx, feed.UserID, feed.TagID, feed.IncludeAlarms)
}
