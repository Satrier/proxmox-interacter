package app

import (
	"fmt"
	loggerPkg "main/pkg/logger"
	"main/pkg/proxmox"
	"main/pkg/templates"
	"main/pkg/types"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/rs/zerolog"
)

// Telegram hard limit is 4096 chars. Keep some headroom.
const MaxMessageSize = 4000

type App struct {
	Config          types.Config
	ProxmoxManager  *proxmox.Manager
	TemplateManager *templates.TemplateManager
	Logger          *zerolog.Logger
	Bot             *tgbotapi.BotAPI
	Version         string
}

func NewApp(config *types.Config, version string) *App {
	logger := loggerPkg.GetLogger(config.Log)
	templateManager := templates.NewTemplateManager()

	bot, err := tgbotapi.NewBotAPI(config.Telegram.Token)
	if err != nil {
		logger.Fatal().Err(err).Msg("Could not start Telegram bot")
	}

	proxmoxManager := proxmox.NewManager(config, logger)
	app := &App{
		Logger:          logger,
		ProxmoxManager:  proxmoxManager,
		TemplateManager: templateManager,
		Bot:             bot,
		Version:         version,
		Config: types.Config{
			Telegram: types.TelegramConfig{
				Admins: config.Telegram.Admins,
			},
		},
	}

	return app
}

func (a *App) Start() {
	// a.Bot.Handle("/status", a.HandleStatus)
	// a.Bot.Handle("/containers", a.HandleListContainers)
	// a.Bot.Handle("/container", a.HandleContainerInfo)
	// a.Bot.Handle("/node", a.HandleNodeInfo)
	// a.Bot.Handle("/start", a.HandleContainerAction("start"))
	// a.Bot.Handle("/stop", a.HandleContainerAction("stop"))
	// a.Bot.Handle("/restart", a.HandleContainerAction("restart"))
	// a.Bot.Handle("/scale", a.HandleContainerScale)
	// a.Bot.Handle("/disks", a.HandleListDisks)
	// a.Bot.Handle("/about", a.HandleAbout)
	// a.Bot.Handle("/help", a.HandleHelp)

	// a.Bot.Handle(tele.OnCallback, a.HandleCallback)

	a.Logger.Info().Msg("Telegram bot listening")
	a.botRun()

}

func (a App) botRun() {
	update := tgbotapi.NewUpdate(0)
	update.Timeout = 60

	updates, err := a.Bot.GetUpdatesChan(update)
	if err != nil {
		a.Logger.Info().Err(err).Msg("Failed to get Telegram updates")
	}

	for update := range updates {
		check := a.checkIdAdmin(update)
		if !check {
			a.Logger.Info().Msg("Unauthorized user tried to access the bot")
			continue
		}

		if update.Message != nil {
			if update.Message.IsCommand() && update.Message.Command() == "start" {
				chatID := update.Message.Chat.ID
				a.Logger.Info().Msgf("Run start menu for chat ID: %d", chatID)
				a.HandleListContainers(a.Bot, chatID)
			}
			if update.Message.IsCommand() && update.Message.Command() == "containers" {
				chatID := update.Message.Chat.ID
				a.Logger.Info().Msgf("Run containers for chat ID: %d", chatID)
				a.HandleListContainers(a.Bot, chatID)
			}
			continue
		}

		if update.CallbackQuery != nil {
			q := update.CallbackQuery
			chatID := q.Message.Chat.ID
			msgID := q.Message.MessageID

			a.Logger.Info().Msgf("Received callback query for chat ID: %d", chatID)

			_, err = a.Bot.AnswerCallbackQuery(tgbotapi.NewCallback(q.ID, ""))
			if err != nil {
				a.Logger.Info().Err(err).Msg("Failed to answer callback query")
			}

			parts := strings.Split(q.Data, ":")
			if len(parts) < 1 {
				continue
			}

			switch parts[0] {
			case "containers":
				a.HandleListContainers(a.Bot, chatID)

			case "stop", "start":
				if len(parts) != 2 {
					continue
				}

				a.allowDoRun(a.Bot, chatID, msgID, q.Data)

			case "allowstop", "allowstart", "allowrestart":
				if len(parts) != 3 {
					continue
				}

				value := strings.Join(parts[1:], ":")

				msg := tgbotapi.NewEditMessageText(chatID, msgID, fmt.Sprintf("✅ *%s %s*", parts[1], parts[2]))
				msg.ParseMode = "MarkdownV2"

				_, err := a.Bot.Send(msg)
				if err != nil {
					a.Logger.Error().Err(err).Msg("Error sending message")
				}

				a.HandleDoContainerAction(value)
				// a.sendMainMenu(a.Bot, chatID)

			case "cancelstop":
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ *cancel action for %s*", escapeMDV2(parts[2])))
				msg.ParseMode = "MarkdownV2"

				_, err := a.Bot.Send(msg)
				if err != nil {
					a.Logger.Error().Err(err).Msg("Error sending message")
				}

				a.HandleListContainers(a.Bot, chatID)

			case "cancelstart":
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ *cancel action for %s*", escapeMDV2(parts[2])))
				msg.ParseMode = "MarkdownV2"

				_, err := a.Bot.Send(msg)
				if err != nil {
					a.Logger.Error().Err(err).Msg("Error sending message")
				}

				a.HandleListContainers(a.Bot, chatID)
			}
		}
	}
}

func (a *App) sendMainMenu(bot *tgbotapi.BotAPI, chatID int64) (int, error) {
	msg := tgbotapi.NewMessage(chatID, "👉 *Select an action:*")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👀 Show Containers", "containers"),
		),
	)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "MarkdownV2"

	sent, err := bot.Send(msg)
	if err != nil {
		a.Logger.Error().Err(err).Msg("Error sending message")
		return 0, err
	}
	return sent.MessageID, nil
}

func (a *App) allowDoRun(bot *tgbotapi.BotAPI, chatID int64, msgID int, data string) {
	parts := strings.SplitN(data, ":", 2)
	msg := tgbotapi.NewEditMessageText(chatID, msgID, fmt.Sprintf("🫵 *Are you sure?*\n*%s* *%s*", parts[0], escapeMDV2(parts[1])))

	if parts[0] == "stop" {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Yes", "allow"+parts[0]+":"+escapeMDV2(data)),
				tgbotapi.NewInlineKeyboardButtonData("☑️ No", "cancel"+parts[0]+":"+escapeMDV2(data)),
				tgbotapi.NewInlineKeyboardButtonData("🔄 Restart", "allowrestart"+":"+"restart:"+escapeMDV2(parts[1])),
			),
		)
		msg.ReplyMarkup = &keyboard
	} else {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Yes", "allow"+parts[0]+":"+escapeMDV2(data)),
				tgbotapi.NewInlineKeyboardButtonData("☑️ No", "cancel"+parts[0]+":"+escapeMDV2(data)),
			),
		)
		msg.ReplyMarkup = &keyboard
	}
	msg.ParseMode = "MarkdownV2"

	_, err := bot.Send(msg)
	if err != nil {
		a.Logger.Error().Err(err).Msg("Error sending message")
	}
}
