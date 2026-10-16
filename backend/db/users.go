package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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
		SELECT u.id, u.email, u.password_hash, u.display_name
		FROM sessions AS s
		JOIN users AS u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()
		`,
		tokenHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName)
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

// SetDisplayName stores a trimmed display name. A nil name clears it so that
// readers fall back to the email local-part.
func (s *Store) SetDisplayName(ctx context.Context, userID uuid.UUID, displayName *string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE users SET display_name = $2 WHERE id = $1
	`, userID, displayName)
	return err
}
