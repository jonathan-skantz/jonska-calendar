package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ListTransitCardDays returns the reported days in the inclusive range for the
// user and, when they belong to a household, every other member of it.
func (s *Store) ListTransitCardDays(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]TransitCardDay, error) {
	rows, err := s.db.Query(ctx, `
		SELECT d.day, d.user_id, user_display_name(u.display_name, u.email), d.need
		FROM transit_card_days AS d
		JOIN users AS u ON u.id = d.user_id
		WHERE d.day BETWEEN $2 AND $3
		  AND d.user_id IN (
			SELECT $1::uuid
			UNION
			SELECT member.user_id
			FROM household_members AS member
			JOIN household_members AS me ON me.household_id = member.household_id
			WHERE me.user_id = $1
		  )
		ORDER BY d.day, d.user_id
	`, userID, from, to)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (TransitCardDay, error) {
		var day TransitCardDay
		err := row.Scan(&day.Day, &day.UserID, &day.DisplayName, &day.Need)
		return day, err
	})
}

// SetTransitCardDays records an explicit answer for every day in the inclusive
// range: days in needed are 'yes', days in maybe are 'maybe', and all others
// 'no', so an empty selection is still a submission.
func (s *Store) SetTransitCardDays(ctx context.Context, userID uuid.UUID, from, to time.Time, needed, maybe []time.Time) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO transit_card_days (user_id, day, need)
		SELECT $1, series.day,
			CASE
				WHEN series.day = ANY($4::date[]) THEN 'yes'
				WHEN series.day = ANY($5::date[]) THEN 'maybe'
				ELSE 'no'
			END
		FROM (
			SELECT generate_series($2::date, $3::date, interval '1 day')::date AS day
		) AS series
		ON CONFLICT (user_id, day) DO UPDATE
		SET need = EXCLUDED.need, updated_at = now()
	`, userID, from, to, needed, maybe)
	return err
}
