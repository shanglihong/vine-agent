package profile

import (
	"context"
)

type profileService struct {
	profileRepo ProfileRepository
}

func NewProfileService(profileRepo ProfileRepository) ProfileService {
	return &profileService{
		profileRepo: profileRepo,
	}
}

func (p *profileService) GetByUserID(ctx context.Context, userID string) (*Profile, error) {
	return p.profileRepo.GetByUserID(ctx, userID)
}

func (p *profileService) Save(ctx context.Context, prof *Profile) error {
	return p.profileRepo.Save(ctx, prof)
}
