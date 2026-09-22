package api

import (
	"errors"
	"fmt"

	"github.com/artem-cherevko/a-scam-bot-v2/internal/database"
	"github.com/artem-cherevko/a-scam-bot-v2/internal/service"
)

type Handler struct {
	userService  *service.UserService
	adminService *service.AdminService
}

func NewHandler(userService *service.UserService, adminService *service.AdminService) *Handler {
	return &Handler{
		userService:  userService,
		adminService: adminService,
	}
}

func (h *Handler) AddUser(user *database.User) error {
	return h.userService.AddUser(user)
}

func (h *Handler) UpdateUser(user *database.User) error {
	return h.userService.UpdateUser(user)
}

func (h *Handler) DeleteUser(id int64) error {
	return h.userService.DeleteUser(id)
}

func (h *Handler) GetUser(id *int64, username *string) (*database.User, error) {
	if id != nil {
		user, err := h.userService.GetUserByID(*id)
		if errors.Is(err, service.ErrUserNotFound) {
			return nil, fmt.Errorf("user not found: %w", err)
		} else if err != nil {
			return nil, fmt.Errorf("failed to get user by ID: %w", err)
		}
		return user, nil
	}

	if username != nil {
		user, err := h.userService.GetUserByName(*username)
		if errors.Is(err, service.ErrUserNotFound) {
			return nil, fmt.Errorf("user not found: %w", err)
		} else if err != nil {
			return nil, fmt.Errorf("failed to get user by username: %w", err)
		}
		return user, nil
	}

	return nil, fmt.Errorf("either ID or username must be provided")
}

func (h *Handler) AddGuarantor(guarantor *database.Guarantors) error {
	return h.userService.AddGuarantor(guarantor)
}

func (h *Handler) UpdateGuarantor(guarantor *database.Guarantors) error {
	return h.userService.UpdateGuarantor(guarantor)
}

func (h *Handler) DeleteGuarantor(id int64) error {
	return h.userService.DeleteGuarantor(id)
}

func (h *Handler) GetAllAdmins() ([]*database.User, error) {
	admins, err := h.adminService.GetAllAdmins()
	if err != nil {
		return nil, fmt.Errorf("failed to get all admins: %w", err)
	}
	return admins, nil
}
