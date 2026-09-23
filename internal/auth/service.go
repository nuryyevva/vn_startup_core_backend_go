package auth

import (
	"context"
	"errors"
	"net/mail"

	"github.com/google/uuid"

	"vn_startup_core_backend_go/pkg/apperr"
)

const minPasswordLength = 8

type Service struct {
	repo   Repository
	issuer *TokenIssuer
}

func NewService(repo Repository, issuer *TokenIssuer) *Service {
	return &Service{repo: repo, issuer: issuer}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (User, TokenPair, error) {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return User{}, TokenPair{}, apperr.BadRequest("invalid_email", "Некорректный email")
	}
	if len(req.Password) < minPasswordLength {
		return User{}, TokenPair{}, apperr.BadRequest("weak_password", "Пароль должен содержать не менее 8 символов")
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		return User{}, TokenPair{}, err
	}

	user, err := s.repo.CreateUser(ctx, req.Email, hash)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return User{}, TokenPair{}, apperr.Conflict("email_taken", "Этот email уже зарегистрирован")
		}
		return User{}, TokenPair{}, err
	}

	tokens, err := s.issueTokens(user)
	if err != nil {
		return User{}, TokenPair{}, err
	}

	return user, tokens, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (User, TokenPair, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, TokenPair{}, apperr.Unauthorized("invalid_credentials", "Неверный email или пароль")
		}
		return User{}, TokenPair{}, err
	}

	ok, err := VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !ok {
		return User{}, TokenPair{}, apperr.Unauthorized("invalid_credentials", "Неверный email или пароль")
	}

	tokens, err := s.issueTokens(user)
	if err != nil {
		return User{}, TokenPair{}, err
	}

	return user, tokens, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	claims, err := s.issuer.ParseRefreshToken(refreshToken)
	if err != nil {
		return TokenPair{}, apperr.Unauthorized("invalid_refresh_token", "Недействительный refresh-токен")
	}

	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return TokenPair{}, apperr.Unauthorized("invalid_refresh_token", "Недействительный refresh-токен")
		}
		return TokenPair{}, err
	}

	return s.issueTokens(user)
}

// ChangePassword verifies currentPassword against userID's stored hash and,
// if it matches, replaces it with a hash of newPassword. Existing tokens
// issued before the change remain valid until they expire — there is no
// server-side token revocation in this service (see Refresh).
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return apperr.Unauthorized("invalid_credentials", "Неверный текущий пароль")
		}
		return err
	}

	ok, err := VerifyPassword(currentPassword, user.PasswordHash)
	if err != nil || !ok {
		return apperr.Unauthorized("invalid_credentials", "Неверный текущий пароль")
	}

	if len(newPassword) < minPasswordLength {
		return apperr.BadRequest("weak_password", "Пароль должен содержать не менее 8 символов")
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.repo.UpdatePasswordHash(ctx, userID, hash)
}

func (s *Service) issueTokens(user User) (TokenPair, error) {
	access, err := s.issuer.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := s.issuer.GenerateRefreshToken(user.ID, user.Role)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer"}, nil
}
