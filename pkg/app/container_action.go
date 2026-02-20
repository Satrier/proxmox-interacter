package app

import (
	"fmt"
	"strings"
)

type containerAction struct {
	action                   string
	doneAction               string
	shouldContainerBeStarted bool
	actionPrefix             string
	cancelActionPrefix       string
	function                 func(string) error
}

func (a *App) getAction(action string) containerAction {
	actions := map[string]containerAction{
		"restart": {
			action:                   "restart",
			doneAction:               "restarted",
			shouldContainerBeStarted: true,
			actionPrefix:             CallbackPrefixRestart,
			cancelActionPrefix:       CallbackPrefixCancelRestart,
			function:                 a.ProxmoxManager.RestartContainer,
		},
		"stop": {
			action:                   "stop",
			doneAction:               "stopped",
			shouldContainerBeStarted: true,
			actionPrefix:             CallbackPrefixStop,
			cancelActionPrefix:       CallbackPrefixCancelStop,
			function:                 a.ProxmoxManager.StopContainer,
		},
		"start": {
			action:                   "start",
			doneAction:               "started",
			shouldContainerBeStarted: false,
			actionPrefix:             CallbackPrefixStart,
			cancelActionPrefix:       CallbackPrefixCancelStart,
			function:                 a.ProxmoxManager.StartContainer,
		},
	}

	return actions[action]
}

func (a *App) HandleDoContainerAction(actionName string) error {
	args := strings.SplitN(actionName, ":", 2)
	action := a.getAction(args[0])
	// data = args[1]

	clusters, err := a.ProxmoxManager.GetNodes()
	if err != nil {
		a.Logger.Error().Err(err).Msg("Error fetching nodes")
	}

	container, _, err := clusters.FindContainer(args[1])
	if err != nil {

		if err != nil {
			a.Logger.Error().Err(err).Msg("Error rendering container template")
		}

	}

	if container.IsRunning() && !action.shouldContainerBeStarted {
		a.Logger.Info().Msg("Container is already running!")
	} else if !container.IsRunning() && action.shouldContainerBeStarted {
		a.Logger.Info().Msg("Container is not running!")
	}

	if err := action.function(args[1]); err != nil {
		a.Logger.Error().Err(err).Msg(fmt.Sprintf("Error %s container", action.doneAction))
	}

	return nil
}
