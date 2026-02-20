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

func (a App) botRun() {
	bot, err := tgbotapi.NewBotAPI(a.Bot.Token)
	if err != nil {
		a.Logger.Info().Err(err).Msg("Failed to create Telegram bot")
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		a.Logger.Info().Err(err).Msg("Failed to get Telegram updates")
	}

	for update := range updates {
		for _, adminID := range a.Config.Telegram.Admins {
			if update.Message.Chat.ID == adminID {

				if update.Message != nil {
					a.Logger.Info().Msg(update.Message.Text)
					parts := strings.SplitN(update.Message.Text, "@", 2)

					if parts[0] == "/start" {
						a.sendMainMenu(bot, update.Message.Chat.ID)
					}
					if parts[0] == "/containers" {
						err := a.HandleListContainers(bot, update.Message.Chat.ID)
						if err != nil {
							a.Logger.Info().Err(err).Msg("Failed to handle list containers")
						}
					}
				}

				if update.CallbackQuery != nil {
					a.Logger.Info().Msg(update.CallbackQuery.Data)
					data := update.CallbackQuery.Data

					switch data {

					case "/containers":
						err := a.HandleListContainers(bot, update.CallbackQuery.Message.Chat.ID)
						if err != nil {
							a.Logger.Info().Err(err).Msg("Failed to handle list containers")
						}

					default:
						a.Logger.Info().Msg(update.CallbackQuery.Data)
						parts := strings.SplitN(data, ":", 2)

						if len(parts) == 2 {
							action := parts[0]
							value := parts[1]

							if action == "start" || action == "stop" || action == "restart" {
								a.allowDoRun(data, bot, update.CallbackQuery.Message.Chat.ID)
							}

							if action == "allowstart" || action == "allowstop" || action == "allowrestart" {
								a.HandleDoContainerAction(value)

								parts := strings.SplitN(value, ":", 2)

								msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("*%s %s*", parts[0], escapeMDV2(parts[1])))
								msg.ParseMode = "MarkdownV2"

								_, err := bot.Send(msg)
								if err != nil {
									a.Logger.Error().Err(err).Msg("Error sending message")
								}

								a.sendMainMenu(bot, update.CallbackQuery.Message.Chat.ID)
							}

							if action == "cancelstop" || action == "cancelstart" {
								parts := strings.SplitN(value, ":", 2)

								msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("*cancel action for %s*", escapeMDV2(parts[1])))
								msg.ParseMode = "MarkdownV2"

								_, err := bot.Send(msg)
								if err != nil {
									a.Logger.Error().Err(err).Msg("Error sending message")
								}

								a.sendMainMenu(bot, update.CallbackQuery.Message.Chat.ID)
							}
						}
					}

					callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
					if _, err := bot.AnswerCallbackQuery(callback); err != nil {
						a.Logger.Error().Err(err).Msg("callback error")
					}
				}
			}
		}
		a.Logger.Info().Msg("Bad ID tried to access the bot")
		continue
	}
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

	if len(config.Telegram.Admins) > 0 {
		logger.Debug().Msg("Using admins whitelist")

		// bot.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		// 	return func(c tele.Context) error {
		// 		for _, chat := range config.Telegram.Admins {
		// 			if chat == c.Sender().ID {
		// 				return next(c)
		// 			}
		// 		}
		// 		return app.HandleUnauthorized(c)
		// 	}
		// })
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

func (a *App) sendMainMenu(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "👉 *Select an action:*")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👀 Show Containers", "/containers"),
		),
	)
	// msg.MessageThreadID = threadID
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "MarkdownV2"

	_, err := bot.Send(msg)
	if err != nil {
		a.Logger.Error().Err(err).Msg("Error sending message")
	}
}

func (a *App) allowDoRun(data string, bot *tgbotapi.BotAPI, chatID int64) {
	parts := strings.SplitN(data, ":", 2)
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🫵 *Are you sure?*\n*%s* *%s*", parts[0], parts[1]))

	if parts[0] == "stop" {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Yes", "allow"+parts[0]+":"+data),
				tgbotapi.NewInlineKeyboardButtonData("☑️ No", "cancel"+parts[0]+":"+data),
				tgbotapi.NewInlineKeyboardButtonData("🔄 Restart", "allowrestart"+":"+"restart:"+parts[1]),
			),
		)
		msg.ReplyMarkup = keyboard
		// msg.ParseMode = "MarkdownV2"
	} else {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Yes", "allow"+parts[0]+":"+data),
				tgbotapi.NewInlineKeyboardButtonData("☑️ No", "cancel"+parts[0]+":"+data),
			),
		)
		msg.ReplyMarkup = keyboard
		// msg.ParseMode = "MarkdownV2"
	}
	// msg.MessageThreadID = threadID
	msg.ParseMode = "MarkdownV2"

	_, err := bot.Send(msg)
	if err != nil {
		a.Logger.Error().Err(err).Msg("Error sending message")
	}
}
