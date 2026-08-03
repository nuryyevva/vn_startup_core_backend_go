package user

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"vn_startup_core_backend_go/pkg/apperr"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (Profile, error) {
	profile, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return Profile{}, apperr.NotFound("profile_not_found", "Профиль пользователя не найден")
		}
		return Profile{}, err
	}
	return profile, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (Profile, error) {
	profile, err := s.repo.UpdateProfile(ctx, userID, UpdateProfileInput(req))
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return Profile{}, apperr.NotFound("profile_not_found", "Профиль пользователя не найден")
		}
		return Profile{}, err
	}
	return profile, nil
}
