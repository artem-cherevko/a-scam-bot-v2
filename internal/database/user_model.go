package database

import (
	"time"
)

type Roles string

const (
	Tech Roles = "tech"

	// Administration
	Owner       Roles = "owner"
	CoOwner     Roles = "co-owner"
	SeniorAdmin Roles = "senior-admin"
	Admin       Roles = "admin"
	JuniorAdmin Roles = "junior-admin"
	Intern      Roles = "intern"

	// Guarantors
	TopGuarantor Roles = "top-guarantor"
	Guarantor    Roles = "guarantor"
	Trainee      Roles = "trainee"

	// Scam
	Scammer        Roles = "scammer"
	DodgyCharacter Roles = "dodgy-character"

	RegularUser Roles = "user"
)

type User struct {
	ID            int64 `gorm:"autoIncrement;primaryKey;not null"`
	TgID          int64 `gorm:"uniqueIndex;not null"`
	UserName      *string
	Role          Roles `gorm:"type:varchar(32);not null;default:'user';check:role IN ('tech','owner','co-owner','senior-admin','admin','junior-admin','intern','top-guarantor','guarantor','trainee','scammer','dodgy-character','user')"`
	PhotoID       *string
	UserSearched  uint `gorm:"default:0"`
	ScammerReason *string

	// For admins
	Warns         uint16 `gorm:"default:0"`
	AddedScammers uint   `gorm:"default:0"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u *User) IsAdmin() bool {
	switch u.Role {
	case Tech,
		Owner,
		CoOwner,
		SeniorAdmin,
		Admin,
		JuniorAdmin,
		Intern:
		return true
	default:
		return false
	}
}
