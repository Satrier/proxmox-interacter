package app

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// func (a *App) HandleHelp(c tele.Context) error {
// 	a.Logger.Info().
// 		Int64("sender_id", c.Sender().ID).
// 		Str("sender", c.Sender().Username).
// 		Str("text", c.Text()).
// 		Msg("Got help query")

// 	template, err := a.TemplateManager.Render("help", a.Version)
// 	if err != nil {
// 		a.Logger.Error().Err(err).Msg("Error rendering help template")
// 		return c.Reply(fmt.Sprintf("Error rendering template: %s", err))
// 	}

// 	return a.BotReply(c, template)
// }

func escapeMDV2(s string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(s)
}

func (a *App) checkIdAdmin(update tgbotapi.Update) bool {
	for _, adminID := range a.Config.Telegram.Admins {
		switch {
		case update.Message != nil && update.Message.Chat.ID == adminID:
			return true
		case update.CallbackQuery != nil && update.CallbackQuery.Message.Chat.ID == adminID:
			return true
		}
	}
	return false
}
