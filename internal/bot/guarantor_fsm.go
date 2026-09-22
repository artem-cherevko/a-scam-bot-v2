package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/go-telegram/fsm"
)

// FSM states for guarantor editing.
const (
	StateDefault              fsm.StateID = "default"
	StateEditGuarantorRank    fsm.StateID = "edit_guarantor_rank"
	StateEditGuarantorChannel fsm.StateID = "edit_guarantor_channel"
	StateEditGuarantorProofs  fsm.StateID = "edit_guarantor_proofs"
	StateEditGuarantorRegion  fsm.StateID = "edit_guarantor_region"
)

type BotApp struct {
	b         *bot.Bot
	fsm       *fsm.FSM
	apiBaseURL string
}

func NewBotApp(b *bot.Bot, apiBaseURL string) *BotApp {
	return &BotApp{
		b:         b,
		fsm:       fsm.New(StateDefault, nil),
		apiBaseURL: apiBaseURL,
	}
}

// EditGuarantorKb creates the inline keyboard for guarantor editing.
func EditGuarantorKb(tgID int64) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         "🏆 Ранг",
					CallbackData: fmt.Sprintf("edit_g:%d:rank", tgID),
				},
			},
			{
				{
					Text:         "🔗 Канал",
					CallbackData: fmt.Sprintf("edit_g:%d:channel", tgID),
				},
			},
			{
				{
					Text:         "📊 Proofs",
					CallbackData: fmt.Sprintf("edit_g:%d:proofs", tgID),
				},
			},
			{
				{
					Text:         "🌍 Регион",
					CallbackData: fmt.Sprintf("edit_g:%d:region", tgID),
				},
			},
			{
				{
					Text:         "♻️ Сбросить",
					CallbackData: fmt.Sprintf("edit_g:%d:reset", tgID),
				},
			},
		},
	}
}

// RegisterGuarantorFSM registers the /edit_g, edit_g callback and /cancel flow.
//
// Expected API:
//
//	PUT /api/admin/guarantor/edit/:telegram
//	body: {"field":"rank|channel|proofs|region","value":"..."}
//
//	PUT /api/admin/guarantor/reset/:telegram
func (app *BotApp) RegisterGuarantorFSM() {
	// /edit_g <telegram ID | username>
	app.b.RegisterHandler(
		bot.HandlerTypeMessageText,
		"/edit_g",
		bot.MatchTypePrefix,
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			me, err := getUser(
				ctx,
				app.apiBaseURL,
				strconv.FormatInt(update.Message.From.ID, 10),
				update.Message.From.ID,
				false,
			)
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Вас нет в базе данных. Напишите /start для регистрации.",
				})
				return
			}

			if !canSetRole(me.Role) {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Вам недоступна эта команда.",
				})
				return
			}

			identifier := strings.TrimSpace(
				strings.TrimPrefix(update.Message.Text, "/edit_g"),
			)

			if identifier == "" {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Пример: /edit_g 6161935822",
				})
				return
			}

			target, err := getUser(
				ctx,
				app.apiBaseURL,
				identifier,
				update.Message.From.ID,
				false,
			)
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Пользователь не найден.",
				})
				return
			}

			app.fsm.Set(
				update.Message.From.ID,
				"guarantor_id",
				strconv.FormatInt(target.TgID, 10),
			)

			app.fsm.Transition(
				update.Message.From.ID,
				StateDefault,
			)

			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text: fmt.Sprintf(
					"🛡️ <b>Редактирование гаранта</b>\n\n"+
						"👤 %s\n"+
						"🪪 ID: <code>%d</code>\n"+
						"⚙️ Роль: %s\n\n"+
						"Выберите параметр:",
					formatUserName(target),
					target.TgID,
					formatRole(target.Role),
				),
				ParseMode:   "HTML",
				ReplyMarkup: EditGuarantorKb(target.TgID),
			})
		},
	)

	// Inline buttons.
	app.b.RegisterHandler(
		bot.HandlerTypeCallbackQueryData,
		"edit_g:",
		bot.MatchTypePrefix,
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.CallbackQuery == nil {
				return
			}

			callback := update.CallbackQuery
			if callback.Message.Message == nil {
				return
			}

			parts := strings.Split(callback.Data, ":")

			if len(parts) != 3 {
				return
			}

			targetTGID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return
			}

			field := parts[2]
			actorID := callback.From.ID

			app.fsm.Set(
				actorID,
				"guarantor_id",
				strconv.FormatInt(targetTGID, 10),
			)

			if field == "reset" {
				resp, err := apiRequest(
					ctx,
					http.MethodPut,
					app.apiBaseURL+"/api/admin/guarantor/reset/"+strconv.FormatInt(targetTGID, 10),
					actorID,
					nil,
				)
				if err != nil {
					_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
						CallbackQueryID: callback.ID,
						Text:            "❌ Ошибка API",
						ShowAlert:       true,
					})
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
						CallbackQueryID: callback.ID,
						Text:            "❌ Не удалось сбросить настройки",
						ShowAlert:       true,
					})
					return
				}

				_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
					CallbackQueryID: callback.ID,
					Text:            "Настройки сброшены",
				})

				_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
					ChatID:    callback.Message.Message.Chat.ID,
					MessageID: callback.Message.Message.ID,
					Text: fmt.Sprintf(
						"♻️ <b>Настройки гаранта сброшены</b>\n\n"+
							"🪪 ID: <code>%d</code>\n"+
							"🏆 Ранг: 🛡️ Гарант\n"+
							"📊 Proofs: 0\n"+
							"🔗 Канал: —\n"+
							"🌍 Регион: —",
						targetTGID,
					),
					ParseMode: "HTML",
				})
				if err != nil {
					log.Printf("failed to edit reset message: %v", err)
				}

				app.fsm.Reset(actorID)
				return
			}

			var state fsm.StateID

			switch field {
			case "rank":
				state = StateEditGuarantorRank
			case "channel":
				state = StateEditGuarantorChannel
			case "proofs":
				state = StateEditGuarantorProofs
			case "region":
				state = StateEditGuarantorRegion
			default:
				return
			}

			app.fsm.Transition(actorID, state)

			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: callback.ID,
			})

			text := map[string]string{
				"rank":    "🏆 Введите новый ранг:\n\n<code>basic</code> или <code>top</code>",
				"channel": "🔗 Введите ссылку на канал:",
				"proofs":  "📊 Введите количество proofs:",
				"region":  "🌍 Введите регион:",
			}[field]

			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    callback.Message.Message.Chat.ID,
				Text:      text,
				ParseMode: "HTML",
			})
		},
	)

	// /cancel
	app.b.RegisterHandler(
		bot.HandlerTypeMessageText,
		"/cancel",
		bot.MatchTypeExact,
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			userID := update.Message.From.ID

			state := app.fsm.Current(userID)

			if state == StateDefault {
				return
			}

			app.fsm.Reset(userID)

			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "❌ Редактирование отменено.",
			})
		},
	)
}

// HandleGuarantorFSMMessage should be called from the bot's message/default handler.
//
// If your installed go-telegram/bot version does not expose a default-handler
// option, call this function from the common text-message handler after command
// handlers, or use it as the default handler in your bot setup.
func (app *BotApp) HandleGuarantorFSMMessage(
	ctx context.Context,
	b *bot.Bot,
	update *models.Update,
) {
	if update.Message == nil {
		return
	}

	userID := update.Message.From.ID

	state := app.fsm.Current(userID)
	if state == StateDefault {
		return
	}

	value := strings.TrimSpace(update.Message.Text)

	if value == "" {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Значение не может быть пустым.",
		})
		return
	}

	rawGuarantorID, ok := app.fsm.Get(userID, "guarantor_id")
	if !ok {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Сессия редактирования истекла. Начните заново через /edit_g.",
		})
		app.fsm.Reset(userID)
		return
	}

	guarantorID, ok := rawGuarantorID.(string)
	if !ok || guarantorID == "" {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Некорректная сессия редактирования. Начните заново через /edit_g.",
		})
		app.fsm.Reset(userID)
		return
	}

	field := ""
	switch state {
	case StateEditGuarantorRank:
		field = "rank"
	case StateEditGuarantorChannel:
		field = "channel"
	case StateEditGuarantorProofs:
		field = "proofs"
	case StateEditGuarantorRegion:
		field = "region"
	default:
		return
	}

	if err := validateGuarantorField(field, value); err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ " + err.Error(),
		})
		return
	}

	body := struct {
		Field string `json:"field"`
		Value string `json:"value"`
	}{
		Field: field,
		Value: value,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return
	}

resp, err := apiRequest(
				ctx,
				http.MethodPut,
				app.apiBaseURL+"/api/admin/guarantor/edit/"+guarantorID,
				userID,
				payload,
			)
	if err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка при обращении к API.",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var response struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&response)

		message := response.Error
		if message == "" {
			message = "Не удалось изменить значение."
		}

		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ " + message,
		})
		return
	}

	fieldName := map[string]string{
		"rank":    "🏆 Ранг",
		"channel": "🔗 Канал",
		"proofs":  "📊 Proofs",
		"region":  "🌍 Регион",
	}[field]

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: fmt.Sprintf(
			"✅ %s успешно изменён.\n\n"+
				"🪪 Гарант: <code>%s</code>\n"+
				"📝 Новое значение: <code>%s</code>",
			fieldName,
			html.EscapeString(guarantorID),
			html.EscapeString(value),
		),
		ParseMode: "HTML",
	})

	app.fsm.Transition(userID, StateDefault)
}

// Example initialization:
//
// b, err := bot.New(cfg.BOT_TOKEN, opts...)
// if err != nil {
//     return err
// }
//
// app := NewBotApp(b)
// app.RegisterGuarantorFSM()
//
// Register HandleGuarantorFSMMessage from your common text/default handler.

func validateGuarantorField(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("значение не может быть пустым")
	}

	switch field {
	case "rank":
		if value != "basic" && value != "top" {
			return fmt.Errorf("ранг должен быть basic или top")
		}
	case "channel":
		if !strings.HasPrefix(value, "https://t.me/") && !strings.HasPrefix(value, "http://t.me/") && !strings.HasPrefix(value, "@") {
			return fmt.Errorf("укажите Telegram-канал в формате https://t.me/... или @username")
		}
	case "proofs":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("proofs должны быть целым числом не меньше 0")
		}
	case "region":
		if len([]rune(value)) > 64 {
			return fmt.Errorf("регион слишком длинный (максимум 64 символа)")
		}
	default:
		return fmt.Errorf("неизвестное поле")
	}

	return nil
}
