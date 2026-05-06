package service

import (
	"context"

	"github.com/ZRishu/smart-portfolio/internal/modules/content/model"
	"github.com/ZRishu/smart-portfolio/internal/modules/content/repository"
)

type ProfileService interface {
	GetProfile(ctx context.Context) (*model.Profile, error)
	GetEducation(ctx context.Context) ([]model.Education, error)
	GetExperience(ctx context.Context) ([]model.Experience, error)
	GetCertifications(ctx context.Context) ([]model.Certification, error)
	GetAchievements(ctx context.Context) ([]model.Achievement, error)
	GetSkills(ctx context.Context) ([]model.Skill, error)
}

type profileService struct {
	repo *repository.ProfileRepository
}

func NewProfileService(repo *repository.ProfileRepository) ProfileService {
	return &profileService{repo: repo}
}

func (s *profileService) GetProfile(ctx context.Context) (*model.Profile, error) {
	return s.repo.GetProfile(ctx)
}

func (s *profileService) GetEducation(ctx context.Context) ([]model.Education, error) {
	return s.repo.GetEducation(ctx)
}

func (s *profileService) GetExperience(ctx context.Context) ([]model.Experience, error) {
	return s.repo.GetExperience(ctx)
}

func (s *profileService) GetCertifications(ctx context.Context) ([]model.Certification, error) {
	return s.repo.GetCertifications(ctx)
}

func (s *profileService) GetAchievements(ctx context.Context) ([]model.Achievement, error) {
	return s.repo.GetAchievements(ctx)
}

func (s *profileService) GetSkills(ctx context.Context) ([]model.Skill, error) {
	return s.repo.GetSkills(ctx)
}
