package app

import (
	"fmt"
	loggerPkg "main/pkg/logger"
	"main/pkg/proxmox"
	"main/pkg/types"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/rs/zerolog"
)

// Telegram hard limit is 4096 chars. Keep some headroom.
const MaxMessageSize = 4000

type App struct {
	Config         types.Config
	ProxmoxManager *proxmox.Manager
	// TemplateManager *templates.TemplateManager
	Cluster types.ClusterInfos
	// Client  *proxmox.Client
	Logger  *zerolog.Logger
	Bot     *tgbotapi.BotAPI
	Version string
}

func NewApp(config *types.Config, version string) *App {
	logger := loggerPkg.GetLogger(config.Log)
	// templateManager := templates.NewTemplateManager()

	bot, err := tgbotapi.NewBotAPI(config.Telegram.Token)
	if err != nil {
		logger.Fatal().Err(err).Msg("Could not start Telegram bot")
	}

	proxmoxManager := proxmox.NewManager(config, logger)
	app := &App{
		Logger:         logger,
		ProxmoxManager: proxmoxManager,
		// TemplateManager: templateManager,
		Bot:     bot,
		Version: version,
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
				// a.HandleListContainers(a.Bot, chatID)
				a.sendMainMenu(chatID)
			}
			if update.Message.IsCommand() && update.Message.Command() == "containers" {
				chatID := update.Message.Chat.ID
				a.Logger.Info().Msgf("Run containers for chat ID: %d", chatID)
				a.HandleListContainers(chatID)
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

			dispatch := strings.SplitN(q.Data, ":", 2)
			if len(dispatch) < 1 {
				continue
			}

			switch dispatch[0] {
			case "containers":
				a.HandleListContainers(chatID)

			case "proxmoxList":
				a.HandleListProxmox(chatID, msgID)

			case "showVm":
				if len(dispatch) != 2 {
					continue
				}

				a.ShowContainer(dispatch[1], chatID, msgID)

			case "clusterdown":
				if len(dispatch) != 2 {
					continue
				}

				msg := tgbotapi.NewEditMessageText(chatID, msgID, fmt.Sprintf("❌ *%s is offline*", escapeMDV2(dispatch[1])))
				msg.ParseMode = "MarkdownV2"

				_, err = a.Bot.Send(msg)
				if err != nil {
					a.Logger.Error().Err(err).Msg("Error sending message")
				}

				a.sendMainMenu(chatID)

			case "vm":
				a.allowDoRun(chatID, msgID, q.Data)

			case "back":
				a.sendMainMenu(chatID)

			case "stop", "start":
				// q.Data = "action:clusterIndex:vmid:name"
				if len(strings.SplitN(q.Data, ":", 4)) != 4 {
					continue
				}

				a.allowDoRun(chatID, msgID, q.Data)

			case "allow":
				// q.Data = "allow:action:clusterIndex:vmid:name"
				fields := strings.SplitN(q.Data, ":", 5)
				if len(fields) != 5 {
					continue
				}

				action, clusterIndex, vmid, name := fields[1], fields[2], fields[3], fields[4]

				msg := tgbotapi.NewEditMessageText(chatID, msgID, fmt.Sprintf("✅ *%s %s*", action, escapeMDV2(name)))
				msg.ParseMode = "MarkdownV2"

				_, err := a.Bot.Send(msg)
				if err != nil {
					a.Logger.Error().Err(err).Msg("Error sending message")
				}

				a.HandleDoContainerAction(chatID, action, clusterIndex, vmid, name)
				a.sendMainMenu(chatID)

			case "cancel":
				// q.Data = "cancel:action:clusterIndex:vmid:name"
				fields := strings.SplitN(q.Data, ":", 5)
				if len(fields) != 5 {
					continue
				}

				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ *cancel action for %s*", escapeMDV2(fields[4])))
				msg.ParseMode = "MarkdownV2"

				_, err := a.Bot.Send(msg)
				if err != nil {
					a.Logger.Error().Err(err).Msg("Error sending message")
				}

				a.sendMainMenu(chatID)
			}
		}
	}
}

func (a *App) sendMainMenu(chatID int64) (int, error) {
	msg := tgbotapi.NewMessage(chatID, "👉 *Select an action:*")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👀 Show Containers", "containers"),
			tgbotapi.NewInlineKeyboardButtonData("👀 Show Proxmox", "proxmoxList"),
		),
	)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "MarkdownV2"

	sent, err := a.Bot.Send(msg)
	if err != nil {
		a.Logger.Error().Err(err).Msg("Error sending message")
		return 0, err
	}
	return sent.MessageID, nil
}

// allowDoRun renders the "Are you sure?" confirmation step for a container action.
// data is "action:clusterIndex:vmid:name" as built by containerCallbackData; the name is kept
// raw (unescaped) in callback data and only escaped when rendered into Markdown message text,
// so it can never corrupt the identifier used for lookup.
func (a *App) allowDoRun(chatID int64, msgID int, data string) {
	fields := strings.SplitN(data, ":", 4)
	if len(fields) != 4 {
		return
	}

	action, clusterIndex, vmid, name := fields[0], fields[1], fields[2], fields[3]
	rest := clusterIndex + ":" + vmid + ":" + name

	msg := tgbotapi.NewEditMessageText(chatID, msgID, fmt.Sprintf("🫵 *Are you sure?*\n*%s* *%s*", action, escapeMDV2(name)))

	if action == "stop" {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Yes", "allow:"+action+":"+rest),
				tgbotapi.NewInlineKeyboardButtonData("☑️ No", "cancel:"+action+":"+rest),
				tgbotapi.NewInlineKeyboardButtonData("🔄 Restart", "allow:restart:"+rest),
			),
		)
		msg.ReplyMarkup = &keyboard
	} else {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Yes", "allow:"+action+":"+rest),
				tgbotapi.NewInlineKeyboardButtonData("☑️ No", "cancel:"+action+":"+rest),
			),
		)
		msg.ReplyMarkup = &keyboard
	}
	msg.ParseMode = "MarkdownV2"

	_, err := a.Bot.Send(msg)
	if err != nil {
		a.Logger.Error().Err(err).Msg("Error sending message")
	}
}
