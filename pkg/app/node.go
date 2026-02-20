package app

// func (a *App) HandleNodeInfo(c tele.Context) error {
// 	a.Logger.Info().
// 		Int64("sender_id", c.Sender().ID).
// 		Str("sender", c.Sender().Username).
// 		Str("text", c.Text()).
// 		Msg("Got node info query")

// 	args := strings.SplitN(c.Text(), " ", 2)
// 	command, args := args[0], args[1:]

// 	if len(args) != 1 {
// 		return c.Reply(fmt.Sprintf("Usage: %s <node name>", command))
// 	}

// 	clusters, err := a.ProxmoxManager.GetNodes()
// 	if err != nil {
// 		return a.BotReply(c, fmt.Sprintf("Error fetching nodes: %s", err))
// 	}

// 	node, err := clusters.FindNode(args[0])
// 	if err != nil {
// 		return a.BotReply(c, fmt.Sprintf("Error finding node: %s\n", err))
// 	}

// 	template, err := a.TemplateManager.Render("node", node)
// 	if err != nil {
// 		a.Logger.Error().Err(err).Msg("Error rendering node template")
// 		return c.Reply(fmt.Sprintf("Error rendering template: %s", err))
// 	}

// 	return a.BotReply(c, template)
// }
