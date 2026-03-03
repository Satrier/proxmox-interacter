package app

import (
	"fmt"
	"main/pkg/proxmox"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func (a *App) HandleListContainers(chatID int64) error {
	clusters, err := a.ProxmoxManager.GetNodes()
	if err != nil {
		a.Logger.Error().Err(err).Msg("Error fetching nodes")
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

		btn := tgbotapi.NewInlineKeyboardButtonData("⬅︎ Go back ", "back:home")
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🖥️ *%s*", escapeMDV2(cluster.Name)))
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
		msg.ParseMode = "MarkdownV2"

		_, err = a.Bot.Send(msg)
		if err != nil {
			a.Logger.Error().Err(err).Msg("Error sending message")
		}
	}

	return nil
}

func (a *App) HandleListProxmox(chatID int64, msgID int) error {
	clusters, err := a.ProxmoxManager.GetNodes()
	if err != nil {
		a.Logger.Error().Err(err).Msg("Error fetching nodes")
	}

	rows := [][]tgbotapi.InlineKeyboardButton{}

	for _, cluster := range clusters {
		a.Logger.Info().Msgf("Cluster: %s", cluster.Name)
		if cluster.Error != nil {
			btn := tgbotapi.NewInlineKeyboardButtonData("🟥 "+cluster.Name, "clusterdown:"+cluster.Name)
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
		} else {
			btn := tgbotapi.NewInlineKeyboardButtonData("🟩 "+cluster.Name, "showVm"+":"+cluster.Name)
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
		}
	}

	btn := tgbotapi.NewInlineKeyboardButtonData("⬅︎ Go back ", "back:home")
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))

	msg := tgbotapi.NewEditMessageText(chatID, msgID, "✅ All Proxmox Clusters:")

	button := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg.ReplyMarkup = &button
	msg.ParseMode = "MarkdownV2"

	_, err = a.Bot.Send(msg)
	if err != nil {
		a.Logger.Error().Err(err).Msg("Error sending message")
	}

	return nil
}

func (a *App) HandleListByProxmoxName(clusterName string, chatID int64, msgID int) error {
	clusters, err := a.ProxmoxManager.GetNodes()
	if err != nil {
		a.Logger.Error().Err(err).Msg("Error fetching nodes")
	}

	rows := [][]tgbotapi.InlineKeyboardButton{}

	for _, cluster := range clusters {

		if cluster.Name == clusterName {

			for _, node := range cluster.Nodes {
				for _, container := range node.Containers {
					a.Logger.Info().Msgf("Cluster: %s", cluster.Name)
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
		}
	}

	btn := tgbotapi.NewInlineKeyboardButtonData("⬅︎ Go back ", "back:home")
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))

	msg := tgbotapi.NewEditMessageText(chatID, msgID, "✅ All Proxmox Nodes for cluster "+escapeMDV2(clusterName)+":")

	button := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg.ReplyMarkup = &button
	msg.ParseMode = "MarkdownV2"

	_, err = a.Bot.Send(msg)
	if err != nil {
		a.Logger.Error().Err(err).Msg("Error sending message")
	}

	return nil
}

func (a *App) ShowContainer(cluster string, chatID int64, msgID int) error {
	for _, client := range a.ProxmoxManager.Clients {
		if client.Config.Name == cluster {
			c := proxmox.Manager{
				Clients: []*proxmox.Client{client},
			}

			clusters, err := c.GetNodes()
			if err != nil {
				a.Logger.Error().Err(err).Msg("Error fetching nodes")
			}

			rows := [][]tgbotapi.InlineKeyboardButton{}

			for _, cluster := range clusters {
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
			}

			btn := tgbotapi.NewInlineKeyboardButtonData("⬅︎ Go back ", "back:home")
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))

			msg := tgbotapi.NewEditMessageText(chatID, msgID, "✅ All  Nodes for cluster "+escapeMDV2(cluster)+":")

			button := tgbotapi.NewInlineKeyboardMarkup(rows...)
			msg.ReplyMarkup = &button
			msg.ParseMode = "MarkdownV2"

			_, err = a.Bot.Send(msg)
			if err != nil {
				a.Logger.Error().Err(err).Msg("Error sending message")
			}
		}
	}

	return nil
}
