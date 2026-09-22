package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/artem-cherevko/a-scam-bot-v2/internal/database"
	"github.com/artem-cherevko/a-scam-bot-v2/internal/repository"
)

var ErrUserNotFound = fmt.Errorf("user not found")

type UserService struct {
	uRepo repository.UserRepository
	gRepo repository.GuarantorRepository
}

func NewUserService(
	uRepo repository.UserRepository,
	gRepo repository.GuarantorRepository,
) *UserService {
	return &UserService{
		uRepo: uRepo,
		gRepo: gRepo,
	}
}

func (s *UserService) AddUser(user *database.User) error {
	return s.uRepo.CreateUser(user)
}

func (s *UserService) GetUserByID(id int64) (*database.User, error) {
	user, err := s.uRepo.GetUserByID(id)
	if errors.Is(err, ErrUserNotFound) {
		user, err = s.uRepo.GetUserByTelegramID(id)
		if errors.Is(err, ErrUserNotFound) {
			return nil, fmt.Errorf("user not found: %w", err)
		} else if err != nil {
			return nil, fmt.Errorf("failed to get user by Telegram ID: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return user, nil
}

func (s *UserService) GetUserByTelegramID(telegramID int64) (*database.User, error) {
	user, err := s.uRepo.GetUserByTelegramID(telegramID)
	if errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("user not found: %w", err)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user by Telegram ID: %w", err)
	}
	return user, nil
}

func (s *UserService) IncrementAddedScammers(tgID int64) error {
	return s.uRepo.IncrementAddedScammers(tgID)
}

func (s *UserService) GetUserByName(username string) (*database.User, error) {
	user, err := s.uRepo.GetUserByName(username)
	if errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("user not found: %w", err)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return user, nil
}

func (s *UserService) UpdateUser(user *database.User) error {
	return s.uRepo.UpdateUser(user)
}

func (s *UserService) DeleteUser(id int64) error {
	return s.uRepo.DeleteUser(id)
}

func (s *UserService) AddGuarantor(guarantor *database.Guarantors) error {
	user, err := s.uRepo.GetUserByTelegramID(guarantor.TgUserID)
	if err != nil {
		return err
	}

	if err := s.gRepo.CreateGuarantor(guarantor); err != nil {
		return err
	}

	switch guarantor.Rank {
	case database.Basic:
		user.Role = database.Guarantor

	case database.Top:
		user.Role = database.TopGuarantor

	default:
		return fmt.Errorf("unknown guarantor rank: %s", guarantor.Rank)
	}

	if err := s.uRepo.UpdateUser(user); err != nil {
		return err
	}

	return nil
}

func (s *UserService) GetAllGuarantors() ([]database.Guarantors, error) {
	return s.gRepo.GetAllGuarantors()
}
func (s *UserService) UpdateGuarantor(guarantor *database.Guarantors) error {
	if guarantor == nil {
		return errors.New("guarantor is nil")
	}

	if guarantor.Rank == database.Ranks("basic") {
		user, err := s.uRepo.GetUserByTelegramID(guarantor.TgUserID)
		if err != nil {
			return err
		}

		user.Role = database.Roles("guarantor")

		if err := s.uRepo.UpdateUser(user); err != nil {
			return err
		}

	} else if guarantor.Rank == database.Ranks("top") {
		user, err := s.uRepo.GetUserByTelegramID(guarantor.TgUserID)
		if err != nil {
			return err
		}

		user.Role = database.Roles("top-guarantor")

		if err := s.uRepo.UpdateUser(user); err != nil {
			return err
		}
	}

	return s.gRepo.UpdateGuarantor(guarantor)
}

func (s *UserService) ResetGuarantor(
	guarantor *database.Guarantors,
) error {
	if guarantor == nil {
		return errors.New("guarantor is nil")
	}

	guarantor.ChanelUrl = nil
	guarantor.ProofsCount = nil
	guarantor.Region = nil
	guarantor.TraineeIDs = nil

	return s.gRepo.UpdateGuarantor(guarantor)
}

func (s *UserService) CompleteUser(user *database.User, username string) error {
	if user.UserName != nil && *user.UserName != "" {
		return fmt.Errorf("username already exists")
	}

	username = strings.TrimPrefix(strings.TrimSpace(username), "@")

	if username == "" {
		return fmt.Errorf("username is empty")
	}

	user.UserName = &username

	return s.uRepo.UpdateUser(user)
}
func (s *UserService) DeleteGuarantor(id int64) error {
	guarantor, err := s.gRepo.GetGuarantorByID(id)
	if err != nil {
		return err
	}

	user, err := s.uRepo.GetUserByTelegramID(guarantor.TgUserID)
	if err != nil {
		return err
	}

	user.Role = database.RegularUser

	if err := s.uRepo.UpdateUser(user); err != nil {
		return err
	}

	if err := s.gRepo.DeleteGuarantor(id); err != nil {
		return err
	}

	return nil
}
func (s *UserService) WarnUser(actor *database.User, target *database.User) (bool, error) {
	if actor == nil || target == nil {
		return false, errors.New("user is nil")
	}

	if !canWarnRole(actor.Role) {
		return false, errors.New("you cannot warn users")
	}

	if !isRoleLower(target.Role, actor.Role) {
		return false, errors.New("you cannot warn this user")
	}

	if target.Warns >= 3 {
		return false, errors.New("user already has maximum warns")
	}

	newWarns := target.Warns + 1

	if newWarns == 3 {
		target.Warns = 0
		target.Role = database.RegularUser

		guarantor, err := s.gRepo.GetGuarantorByID(target.TgID)
		if err == nil && guarantor != nil {
			if err := s.gRepo.DeleteGuarantor(guarantor.TgUserID); err != nil {
				return false, err
			}
		}

		if err := s.uRepo.UpdateWarnsAndRole(
			target.TgID,
			0,
			database.RegularUser,
		); err != nil {
			return false, err
		}

		return true, nil
	}

	target.Warns = newWarns

	if err := s.uRepo.UpdateWarns(
		target.TgID,
		newWarns,
	); err != nil {
		return false, err
	}

	return false, nil
}
func (s *UserService) UnwarnUser(actor *database.User, target *database.User) error {
	if actor == nil || target == nil {
		return errors.New("user is nil")
	}

	if !canWarnRole(actor.Role) {
		return errors.New("you cannot unwarn users")
	}

	if !isRoleLower(target.Role, actor.Role) {
		return errors.New("you cannot unwarn this user")
	}

	if target.Warns == 0 {
		return errors.New("user has no warns")
	}

	newWarns := target.Warns - 1

	if err := s.uRepo.UpdateWarns(target.TgID, newWarns); err != nil {
		return err
	}

	target.Warns = newWarns

	return nil
}
func canWarnRole(role database.Roles) bool {
	switch role {
	case database.Tech,
		database.Owner,
		database.CoOwner,
		database.SeniorAdmin:
		return true

	default:
		return false
	}
}

func roleLevel(role database.Roles) int {
	switch role {
	case database.Tech:
		return 100

	case database.Owner:
		return 90

	case database.CoOwner:
		return 80

	case database.SeniorAdmin:
		return 70

	case database.Admin:
		return 60

	case database.JuniorAdmin:
		return 50

	case database.Intern:
		return 40

	case database.TopGuarantor:
		return 30

	case database.Guarantor:
		return 20

	case database.Trainee:
		return 10

	case database.RegularUser,
		database.Scammer,
		database.DodgyCharacter:
		return 0

	default:
		return -1
	}
}

func isRoleLower(target, actor database.Roles) bool {
	return roleLevel(target) < roleLevel(actor)
}
