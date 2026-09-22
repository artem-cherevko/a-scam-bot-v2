package database

import (
	"time"

	"github.com/lib/pq"
)

type Ranks string

const (
	Top   Ranks = "top"
	Basic Ranks = "basic"
)

type Guarantors struct {
	ID       int   `gorm:"primaryKey;autoIncrement"`
	TgUserID int64 `gorm:"uniqueIndex;not null"`

	Rank        Ranks `gorm:"default:'basic'"`
	ChanelUrl   *string
	ProofsCount *string `gorm:"default:'0'"`
	Region      *string
	TraineeIDs  *pq.Int64Array `gorm:"type:bigint[]"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
