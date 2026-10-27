package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateUser(ctx context.Context, u *User, passwordHash string) error {
	email := strings.ToLower(strings.TrimSpace(u.Email))
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (id, email, password_hash, name)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`,
		u.ID, email, passwordHash, strings.TrimSpace(u.Name),
	).Scan(&u.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "users_email_lower") {
			return ErrConflict
		}
		return err
	}
	u.Email = email
	return nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (User, string, error) {
	var u User
	var hash string
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, name, created_at, password_hash
		FROM users WHERE email = $1`,
		strings.ToLower(strings.TrimSpace(email)),
	).Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, "", ErrInvalidCreds
	}
	return u, hash, err
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, name, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrUnauthorized
	}
	return u, err
}

func (r *Repository) CreateSession(ctx context.Context, id, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`,
		id, userID, tokenHash, expiresAt,
	)
	return err
}

func (r *Repository) UserByTokenHash(ctx context.Context, tokenHash string) (User, uuid.UUID, error) {
	var u User
	var sessionID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT s.id, u.id, u.email, u.name, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`,
		tokenHash,
	).Scan(&sessionID, &u.ID, &u.Email, &u.Name, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, uuid.Nil, ErrUnauthorized
	}
	return u, sessionID, err
}

func (r *Repository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (r *Repository) DeleteExpired(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= now()`)
	return err
}
