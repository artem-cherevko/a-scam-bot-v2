package bot

import (
	"fmt"

	"github.com/artem-cherevko/a-scam-bot-v2/internal/database"
	"github.com/go-telegram/bot/models"
)

func SetRoleKb(tgID int64, role string) *models.InlineKeyboardMarkup {
	roles := []database.Roles{
		database.Tech,
		database.Owner,
		database.CoOwner,
		database.SeniorAdmin,
		database.Admin,
		database.JuniorAdmin,
		database.Intern,
		database.TopGuarantor,
		database.Guarantor,
		database.Trainee,
		database.Scammer,
		database.DodgyCharacter,
		database.RegularUser,
	}

	currentRole := database.Roles(role)

	var keyboard [][]models.InlineKeyboardButton

	for i, r := range roles {
		if r == currentRole {
			for _, lowerRole := range roles[i+1:] {
				keyboard = append(keyboard, []models.InlineKeyboardButton{
					{
						Text:         string(formatRole(string(lowerRole))),
						CallbackData: fmt.Sprintf("set_role:%d:%s", tgID, lowerRole),
					},
				})
			}
			break
		}
	}

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}
