package app

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func (a *App) HandleListContainers(bot *tgbotapi.BotAPI, chatID int64) error {
	clusters, err := a.ProxmoxManager.GetNodes()
	if err != nil {
		return fmt.Errorf("Error fetching nodes: %s", err)
	}

	for _, cluster := range clusters {
		rows := [][]tgbotapi.InlineKeyboardButton{}

		for _, node := range cluster.Nodes {
			for _, container := range node.Containers {
				if container.Status == "running" {
					btn := tgbotapi.NewInlineKeyboardButtonData("🟢 "+container.Name, "stop"+":"+container.Name)
					rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
				}
				if container.Status == "stopped" {
					btn := tgbotapi.NewInlineKeyboardButtonData("⚪ "+container.Name, "start"+":"+container.Name)
					rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
				}
			}
		}
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🖥️ *%s*", escapeMDV2(cluster.Name)))
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
		msg.ParseMode = "MarkdownV2"

		_, err = bot.Send(msg)
		if err != nil {
			a.Logger.Error().Err(err).Msg("Error sending message")
		}
	}

	return nil
}
