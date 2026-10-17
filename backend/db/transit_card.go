package db

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListTransitCardDays returns the reported days in the inclusive range for the
// user and, when they belong to a household, every other member of it.
// The display name falls back to the email local-part when it is not set.
func (s *Store) ListTransitCardDays(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]TransitCardDay, error) {
	rows, err := s.db.Query(ctx, `
		SELECT d.day, d.user_id, COALESCE(NULLIF(u.display_name, ''), split_part(u.email, '@', 1)), d.need
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
	defer rows.Close()

	days := []TransitCardDay{}
	for rows.Next() {
		var day TransitCardDay
		if err := rows.Scan(&day.Day, &day.UserID, &day.DisplayName, &day.Need); err != nil {
			return nil, err
		}
		days = append(days, day)
	}
	return days, rows.Err()
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
