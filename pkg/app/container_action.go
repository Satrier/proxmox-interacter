package app

import (
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type containerAction struct {
	action                   string
	doneAction               string
	resultIcon               string
	shouldContainerBeStarted bool
	actionPrefix             string
	cancelActionPrefix       string
	function                 func(clusterIndex int, vmid int64) error
}

func (a *App) getAction(action string) containerAction {
	actions := map[string]containerAction{
		"restart": {
			action:                   "restart",
			doneAction:               "restarted",
			resultIcon:               "🔄",
			shouldContainerBeStarted: true,
			actionPrefix:             CallbackPrefixRestart,
			cancelActionPrefix:       CallbackPrefixCancelRestart,
			function:                 a.ProxmoxManager.RestartContainerByVMID,
		},
		"stop": {
			action:                   "stop",
			doneAction:               "stopped",
			resultIcon:               "🔴",
			shouldContainerBeStarted: true,
			actionPrefix:             CallbackPrefixStop,
			cancelActionPrefix:       CallbackPrefixCancelStop,
			function:                 a.ProxmoxManager.StopContainerByVMID,
		},
		"start": {
			action:                   "start",
			doneAction:               "started",
			resultIcon:               "🟢",
			shouldContainerBeStarted: false,
			actionPrefix:             CallbackPrefixStart,
			cancelActionPrefix:       CallbackPrefixCancelStart,
			function:                 a.ProxmoxManager.StartContainerByVMID,
		},
	}

	return actions[action]
}

// HandleDoContainerAction executes a confirmed start/stop/restart action against a specific
// container, identified unambiguously by (clusterIndex, vmid) rather than by name, so that
// containers sharing a name across different Proxmox clusters can never be confused with
// each other.
func (a *App) HandleDoContainerAction(chatID int64, actionName, clusterIndexStr, vmidStr, name string) {
	action := a.getAction(actionName)

	clusterIndex, err := strconv.Atoi(clusterIndexStr)
	if err != nil {
		a.sendActionError(chatID, actionName, name, fmt.Errorf("invalid cluster reference"))
		return
	}

	vmid, err := strconv.ParseInt(vmidStr, 10, 64)
	if err != nil {
		a.sendActionError(chatID, actionName, name, fmt.Errorf("invalid container reference"))
		return
	}

	container, _, err := a.ProxmoxManager.FindContainerByVMID(clusterIndex, vmid)
	if err != nil {
		a.Logger.Error().Err(err).Msg("Error finding container")
		a.sendActionError(chatID, actionName, name, err)
		return
	}

	if container.IsRunning() && !action.shouldContainerBeStarted {
		a.Logger.Info().Msg("Container is already running!")
	} else if !container.IsRunning() && action.shouldContainerBeStarted {
		a.Logger.Info().Msg("Container is not running!")
	}

	if err := action.function(clusterIndex, vmid); err != nil {
		a.Logger.Error().Err(err).Msg(fmt.Sprintf("Error %s container", action.doneAction))
		a.sendActionError(chatID, actionName, name, err)
	}
}

func (a *App) sendActionError(chatID int64, actionName, name string, err error) {
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"❌ *Could not %s %s: %s*",
		escapeMDV2(actionName),
		escapeMDV2(name),
		escapeMDV2(err.Error()),
	))
	msg.ParseMode = "MarkdownV2"

	if _, sendErr := a.Bot.Send(msg); sendErr != nil {
		a.Logger.Error().Err(sendErr).Msg("Error sending message")
	}
}
