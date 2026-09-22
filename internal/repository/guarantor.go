package repository

import (
	"errors"
	"fmt"

	"github.com/artem-cherevko/a-scam-bot-v2/internal/database"
	"gorm.io/gorm"
)

type GuarantorRepository interface {
	// CreateGuarantor creates a new guarantor in the database.
	CreateGuarantor(guarantor *database.Guarantors) error

	// GetAllGuarantors retrieves all guarantors from the database.
	GetAllGuarantors() ([]database.Guarantors, error)

	// GetGuarantorByID retrieves a guarantor from the database by their ID.
	GetGuarantorByID(id int64) (*database.Guarantors, error)

	// UpdateGuarantor updates an existing guarantor's information in the database.
	UpdateGuarantor(guarantor *database.Guarantors) error

	// DeleteGuarantor removes a guarantor from the database by their ID.
	DeleteGuarantor(id int64) error
}

type GuarantorRepositoryImpl struct {
	db *gorm.DB
}

func NewGuarantorRepository(db *gorm.DB) *GuarantorRepositoryImpl {
	return &GuarantorRepositoryImpl{
		db: db,
	}
}

func (r *GuarantorRepositoryImpl) CreateGuarantor(guarantor *database.Guarantors) error {
	return r.db.Create(guarantor).Error
}

func (r *GuarantorRepositoryImpl) GetAllGuarantors() ([]database.Guarantors, error) {
	var guarantors []database.Guarantors
	if err := r.db.Select("tg_user_id", "rank", "chanel_url", "proofs_count", "region", "trainee_ids").Find(&guarantors).Error; err != nil {
		return nil, err
	}
	return guarantors, nil
}

func (r *GuarantorRepositoryImpl) GetGuarantorByID(id int64) (*database.Guarantors, error) {
	var guarantor database.Guarantors
	if err := r.db.Select(
		"id",
		"tg_user_id",
		"rank",
		"chanel_url",
		"proofs_count",
		"region",
		"trainee_ids",
	).Model(database.Guarantors{}).Where("tg_user_id = ?", id).First(&guarantor).Error; err != nil {
		return nil, err
	}
	return &guarantor, nil
}
func (r *GuarantorRepositoryImpl) UpdateGuarantor(guarantor *database.Guarantors) error {
	if guarantor == nil {
		return errors.New("guarantor is nil")
	}

	result := r.db.
		Model(&database.Guarantors{}).
		Where("id = ?", guarantor.ID).
		Updates(map[string]any{
			"tg_user_id":   guarantor.TgUserID,
			"rank":         guarantor.Rank,
			"chanel_url":   guarantor.ChanelUrl,
			"proofs_count": guarantor.ProofsCount,
			"region":       guarantor.Region,
			"trainee_ids":  guarantor.TraineeIDs,
		})

	if result.Error != nil {
		return fmt.Errorf(
			"can't update guarantor with ID %d: %w",
			guarantor.ID,
			result.Error,
		)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("guarantor with ID %d not found", guarantor.ID)
	}

	return nil
}
func (r *GuarantorRepositoryImpl) DeleteGuarantor(id int64) error {
	return r.db.Where("tg_user_id = ?", id).Delete(&database.Guarantors{}).Error
}
