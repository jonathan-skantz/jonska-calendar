package db

import (
	"context"

	"github.com/google/uuid"
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
	defer rows.Close()
	household.Members = []HouseholdMember{}
	for rows.Next() {
		var member HouseholdMember
		if err := rows.Scan(&member.UserID, &member.DisplayName); err != nil {
			return nil, err
		}
		household.Members = append(household.Members, member)
	}
	return &household, rows.Err()
}

func (s *Store) CreateHousehold(ctx context.Context, userID uuid.UUID, name string, transitCardCount int) (uuid.UUID, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	var householdID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO households (name, transit_card_count) VALUES ($1, $2) RETURNING id
	`, name, transitCardCount).Scan(&householdID); err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO household_members (household_id, user_id) VALUES ($1, $2)
	`, householdID, userID); err != nil {
		if isUniqueViolation(err) {
			return uuid.Nil, ErrAlreadyInHousehold
		}
		return uuid.Nil, err
	}
	return householdID, tx.Commit(ctx)
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
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

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
	return tx.Commit(ctx)
}
