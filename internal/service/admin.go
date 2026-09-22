package service

import (
	"github.com/artem-cherevko/a-scam-bot-v2/internal/database"
	"github.com/artem-cherevko/a-scam-bot-v2/internal/repository"
)

type AdminService struct {
	uRepo repository.UserRepository
	gRepo repository.GuarantorRepository
}

func NewAdminService(
	uRepo repository.UserRepository,
	gRepo repository.GuarantorRepository,
) *AdminService {
	return &AdminService{
		uRepo: uRepo,
		gRepo: gRepo,
	}
}

func (s *AdminService) GetAllUsers() ([]*database.User, error) {
	users, err := s.uRepo.GetAllUsers()
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s *AdminService) GetAllAdmins() ([]*database.User, error) {
	return s.uRepo.GetAllAdmins()
}
