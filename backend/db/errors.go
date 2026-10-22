package db

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrSessionNotFound      = errors.New("session not found")
	ErrSharedEventNotFound  = errors.New("shared event not found")
	ErrEventNotFound        = errors.New("event not found")
	ErrTaskNotFound         = errors.New("task not found")
	ErrTimelineItemNotFound = errors.New("timeline item not found")
	ErrUserNotFound         = errors.New("user not found")
	ErrAlreadyShared        = errors.New("event is already shared with this user")
	ErrTagAlreadyExists     = errors.New("tag already exists")
	ErrTagNotFound          = errors.New("tag not found")
	ErrLastTag              = errors.New("at least one tag is required")
	ErrMainTag              = errors.New("the main tag cannot be removed")
	ErrDefaultTagNotFound   = errors.New("default tag not found")
	ErrHouseholdNotFound    = errors.New("household not found")
	ErrAlreadyInHousehold   = errors.New("user already belongs to a household")
	ErrReminderExists       = errors.New("a reminder with that offset already exists")
	ErrReminderNotFound     = errors.New("reminder not found")
	ErrWeatherNotCached     = errors.New("weather forecast not cached")
	ErrFeedNotFound         = errors.New("feed not found")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrSubscriptionExists   = errors.New("already subscribed to this calendar")
)

func isUniqueViolation(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == "23505"
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// requireRowsAffected reports notFound when a write matched no row. Writes
// filter by owner, so another user's row is also "not found".
func requireRowsAffected(result pgconn.CommandTag, err error, notFound error) error {
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return notFound
	}
	return nil
}
