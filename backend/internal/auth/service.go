package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, email, password, name string) (User, string, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return User{}, "", err
	}
	if len(password) < MinPasswordLen {
		return User{}, "", fmt.Errorf("%w: senha com no mínimo %d caracteres", ErrInvalidInput, MinPasswordLen)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, "", err
	}
	u := User{ID: uuid.New(), Email: email, Name: strings.TrimSpace(name)}
	if err := s.repo.CreateUser(ctx, &u, string(hash)); err != nil {
		return User{}, "", err
	}
	token, err := s.createSession(ctx, u.ID)
	if err != nil {
		return User{}, "", err
	}
	return u, token, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (User, string, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return User{}, "", ErrInvalidCreds
	}
	u, hash, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return User{}, "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return User{}, "", ErrInvalidCreds
	}
	token, err := s.createSession(ctx, u.ID)
	if err != nil {
		return User{}, "", err
	}
	return u, token, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.repo.DeleteSessionByTokenHash(ctx, hashToken(rawToken))
}

func (s *Service) UserFromToken(ctx context.Context, rawToken string) (User, error) {
	if rawToken == "" {
		return User{}, ErrUnauthorized
	}
	u, _, err := s.repo.UserByTokenHash(ctx, hashToken(rawToken))
	return u, err
}

func (s *Service) createSession(ctx context.Context, userID uuid.UUID) (string, error) {
	raw, err := randomToken()
	if err != nil {
		return "", err
	}
	id := uuid.New()
	expires := time.Now().UTC().Add(SessionTTL)
	if err := s.repo.CreateSession(ctx, id, userID, hashToken(raw), expires); err != nil {
		return "", err
	}
	return raw, nil
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", fmt.Errorf("%w: e-mail obrigatório", ErrInvalidInput)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", fmt.Errorf("%w: e-mail inválido", ErrInvalidInput)
	}
	return email, nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
