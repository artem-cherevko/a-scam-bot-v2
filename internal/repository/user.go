package repository

import (
	"github.com/artem-cherevko/a-scam-bot-v2/internal/database"
	"gorm.io/gorm"
)

type UserRepository interface {
	// CreateUser creates a new user in the database.
	CreateUser(user *database.User) error

	IncrementAddedScammers(tgID int64) error

	// GetAllUsers retrieves all users from the database.
	GetAllUsers() ([]*database.User, error)

	// GetAllUserNamesByIds
	GetAllUserNamesByIDs(ids []int64) ([]*database.User, error)

	// GetAllAdmins retrieves all admins from the database.
	GetAllAdmins() ([]*database.User, error)

	// GetUserByID retrieves a user from the database by their ID.
	GetUserByID(id int64) (*database.User, error)

	// GetUserByTelegramID retrieves a user from the database by their Telegram ID.
	GetUserByTelegramID(telegramID int64) (*database.User, error)

	// GetUserByName retrieves a user from the database by their username.
	GetUserByName(username string) (*database.User, error)

	UpdateWarnsAndRole(tgID int64, warns uint16, role database.Roles) error

	UpdateWarns(tgID int64, warns uint16) error

	// UpdateUser updates an existing user's information in the database.
	UpdateUser(user *database.User) error

	// DeleteUser removes a user from the database by their ID.
	DeleteUser(id int64) error
}

type UserRepositoryImpl struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepositoryImpl {
	return &UserRepositoryImpl{
		db: db,
	}
}

func (r *UserRepositoryImpl) CreateUser(user *database.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepositoryImpl) GetAllUsers() ([]*database.User, error) {
	var users []*database.User
	if err := r.db.Select("id", "tg_id", "user_name", "role").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepositoryImpl) GetAllAdmins() ([]*database.User, error) {
	roles := []database.Roles{
		"owner",
		"co-owner",
		"senior-admin",
		"admin",
		"junior-admin",
		"intern",
	}
	users := []*database.User{}
	if err := r.db.Select("id", "tg_id", "user_name", "role", "warns", "added_scammers").Where("role IN ?", roles).Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepositoryImpl) UpdateWarnsAndRole(
	tgID int64,
	warns uint16,
	role database.Roles,
) error {
	return r.db.Model(&database.User{}).
		Where("tg_id = ?", tgID).
		Updates(map[string]interface{}{
			"warns": warns,
			"role":  role,
		}).Error
}

func (r *UserRepositoryImpl) IncrementAddedScammers(tgID int64) error {
	result := r.db.
		Model(&database.User{}).
		Where("tg_id = ?", tgID).
		UpdateColumn(
			"added_scammers",
			gorm.Expr("added_scammers + ?", 1),
		)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *UserRepositoryImpl) GetUserByID(id int64) (*database.User, error) {
	var user database.User
	if err := r.db.Select("id", "tg_id", "user_name", "role", "photo_id", "user_searched", "scammer_reason", "added_scammers").First(&user, id).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepositoryImpl) GetAllUserNamesByIDs(ids []int64) ([]*database.User, error) {
	var users []*database.User
	if err := r.db.Select("tg_id", "user_name").Where("tg_id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepositoryImpl) GetUserByTelegramID(telegramID int64) (*database.User, error) {
	var user database.User
	if err := r.db.Select("id", "tg_id", "user_name", "role", "photo_id", "user_searched", "scammer_reason", "added_scammers", "warns").Where("tg_id = ?", telegramID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) GetUserByName(username string) (*database.User, error) {
	var user database.User
	if err := r.db.Select("id", "tg_id", "user_name", "role", "photo_id", "user_searched", "scammer_reason", "added_scammers", "warns").Where("user_name = ?", username).First(&user).Error; err != nil {
		return nil, err
	}

	if err := r.UpdateUser(&database.User{ID: user.ID, UserSearched: user.UserSearched + 1}); err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *UserRepositoryImpl) UpdateUser(user *database.User) error {
	return r.db.
		Model(&database.User{}).
		Where("id = ?", user.ID).
		Updates(user).
		Error
}
func (r *UserRepositoryImpl) UpdateWarns(tgID int64, warns uint16) error {
	return r.db.Model(&database.User{}).
		Where("tg_id = ?", tgID).
		Update("warns", warns).Error
}
func (r *UserRepositoryImpl) DeleteUser(id int64) error {
	return r.db.Delete(&database.User{TgID: id}).Error
}
