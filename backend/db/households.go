package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Household queries never take a household ID from the client. The household
// is always derived from the caller's own membership row.

// GetHousehold returns the caller's household, or nil when they have none.
func (s *Store) GetHousehold(ctx context.Context, userID uuid.UUID) (*Household, error) {
	var household Household
	err := s.db.QueryRow(ctx, `
		SELECT h.id, h.name, h.transit_card_count
		FROM households AS h
		JOIN household_members AS me ON me.household_id = h.id
		WHERE me.user_id = $1
	`, userID).Scan(&household.ID, &household.Name, &household.TransitCardCount)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT u.id, user_display_name(u.display_name, u.email)
		FROM household_members AS m
		JOIN users AS u ON u.id = m.user_id
		WHERE m.household_id = $1
		ORDER BY m.joined_at, u.id
	`, household.ID)
	if err != nil {
		return nil, err
	}
	household.Members, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (HouseholdMember, error) {
		var member HouseholdMember
		err := row.Scan(&member.UserID, &member.DisplayName)
		return member, err
	})
	if err != nil {
		return nil, err
	}
	return &household, nil
}

func (s *Store) CreateHousehold(ctx context.Context, userID uuid.UUID, name string, transitCardCount int) (uuid.UUID, error) {
	var householdID uuid.UUID
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO households (name, transit_card_count) VALUES ($1, $2) RETURNING id
		`, name, transitCardCount).Scan(&householdID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO household_members (household_id, user_id) VALUES ($1, $2)
		`, householdID, userID)
		if isUniqueViolation(err) {
			return ErrAlreadyInHousehold
		}
		return err
	})
	if err != nil {
		return uuid.Nil, err
	}
	return householdID, nil
}

func (s *Store) UpdateHousehold(ctx context.Context, userID uuid.UUID, name string, transitCardCount int) error {
	result, err := s.db.Exec(ctx, `
		UPDATE households
		SET name = $2, transit_card_count = $3
		WHERE id = (SELECT household_id FROM household_members WHERE user_id = $1)
	`, userID, name, transitCardCount)
	return requireRowsAffected(result, err, ErrHouseholdNotFound)
}

// AddHouseholdMember adds memberID to the caller's household.
func (s *Store) AddHouseholdMember(ctx context.Context, userID, memberID uuid.UUID) error {
	result, err := s.db.Exec(ctx, `
		INSERT INTO household_members (household_id, user_id)
		SELECT household_id, $2 FROM household_members WHERE user_id = $1
	`, userID, memberID)
	if isUniqueViolation(err) {
		return ErrAlreadyInHousehold
	}
	return requireRowsAffected(result, err, ErrHouseholdNotFound)
}

// LeaveHousehold removes the caller and deletes the household once it is empty.
func (s *Store) LeaveHousehold(ctx context.Context, userID uuid.UUID) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		var householdID uuid.UUID
		if err := tx.QueryRow(ctx, `
			DELETE FROM household_members WHERE user_id = $1 RETURNING household_id
		`, userID).Scan(&householdID); err != nil {
			if isNoRows(err) {
				return ErrHouseholdNotFound
			}
			return err
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM households AS h
			WHERE h.id = $1
			  AND NOT EXISTS (SELECT 1 FROM household_members WHERE household_id = h.id)
		`, householdID); err != nil {
			return err
		}
		return nil
	})
}
