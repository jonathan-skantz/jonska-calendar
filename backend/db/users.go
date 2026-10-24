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
	return s.createUser(ctx, func(tx pgx.Tx, userID, tagID uuid.UUID) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, default_tag_id)
			VALUES ($1, $2, $3, $4)
		`, userID, email, passwordHash, tagID)
		return err
	})
}

// CreateDemoUser creates a demo user that is deleted after expiresAt. Its
// password hash is not a valid hash, so no one can log in as it.
func (s *Store) CreateDemoUser(ctx context.Context, email, displayName string, expiresAt time.Time) (uuid.UUID, error) {
	return s.createUser(ctx, func(tx pgx.Tx, userID, tagID uuid.UUID) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, default_tag_id, display_name, demo_expires_at)
			VALUES ($1, $2, 'demo', $3, $4, $5)
		`, userID, email, tagID, displayName, expiresAt)
		return err
	})
}

// createUser runs insertUser and creates the user's first tag in one
// transaction. Register keeps its own insert, so the migration tests can run
// it against older schemas.
func (s *Store) createUser(ctx context.Context, insertUser func(tx pgx.Tx, userID, tagID uuid.UUID) error) (uuid.UUID, error) {
	userID := uuid.New()
	tagID := uuid.New()
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		if err := insertUser(tx, userID, tagID); err != nil {
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
		SELECT id, email, password_hash, demo_expires_at
		FROM users
		WHERE email = $1
		`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DemoExpiresAt)
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
		SELECT u.id, u.email, u.password_hash, u.display_name, u.demo_expires_at
		FROM sessions AS s
		JOIN users AS u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()
		`,
		tokenHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &user.DemoExpiresAt)
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

// IsDemoUser reports whether the user is a demo user.
func (s *Store) IsDemoUser(ctx context.Context, userID uuid.UUID) (bool, error) {
	var isDemo bool
	err := s.db.QueryRow(ctx, `
		SELECT demo_expires_at IS NOT NULL FROM users WHERE id = $1
	`, userID).Scan(&isDemo)
	if isNoRows(err) {
		return false, ErrUserNotFound
	}
	return isDemo, err
}

// CountLiveDemoUsers counts the demo users that have not expired yet.
func (s *Store) CountLiveDemoUsers(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM users WHERE demo_expires_at > now()
	`).Scan(&count)
	return count, err
}

// DeleteExpiredDemoUsers removes demo users past their expiry. Everything
// they own is removed with them by the users foreign keys' cascades.
func (s *Store) DeleteExpiredDemoUsers(ctx context.Context) (int64, error) {
	tag, err := s.db.Exec(ctx, `DELETE FROM users WHERE demo_expires_at <= now()`)
	return tag.RowsAffected(), err
}

// DeleteExpiredSessions removes sessions past their expiry, which lookups
// already ignore, and returns how many it removed.
func (s *Store) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	tag, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= now()`)
	return tag.RowsAffected(), err
}

// SetDisplayName stores a trimmed display name. A nil name clears it so that
// readers fall back to the email local-part.
func (s *Store) SetDisplayName(ctx context.Context, userID uuid.UUID, displayName *string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE users SET display_name = $2 WHERE id = $1
	`, userID, displayName)
	return err
}
