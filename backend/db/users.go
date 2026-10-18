package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Register creates the user together with their first tag, which becomes
// their default tag.
func (s *Store) Register(ctx context.Context, email, passwordHash string) (uuid.UUID, error) {
	userID := uuid.New()
	tagID := uuid.New()
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, default_tag_id)
			VALUES ($1, $2, $3, $4)
		`, userID, email, passwordHash, tagID); err != nil {
			if isUniqueViolation(err) {
				return ErrEmailAlreadyExists
			}
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO tags (id, user_id, name) VALUES ($1, $2, 'general')
		`, tagID, userID)
		return err
	})
	if err != nil {
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
	if isNoRows(err) {
		return User{}, ErrUserNotFound
	}
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
		SELECT u.id, u.email, u.password_hash, u.display_name
		FROM sessions AS s
		JOIN users AS u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()
		`,
		tokenHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName)
	if isNoRows(err) {
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

// SetDisplayName stores a trimmed display name. A nil name clears it so that
// readers fall back to the email local-part.
func (s *Store) SetDisplayName(ctx context.Context, userID uuid.UUID, displayName *string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE users SET display_name = $2 WHERE id = $1
	`, userID, displayName)
	return err
}
