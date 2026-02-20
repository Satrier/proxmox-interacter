package app

// func (a *App) HandleAbout(c tele.Context) error {
// 	a.Logger.Info().
// 		Int64("sender_id", c.Sender().ID).
// 		Str("sender", c.Sender().Username).
// 		Str("text", c.Text()).
// 		Msg("Got about query")

// 	template, err := a.TemplateManager.Render("about", a.Version)
// 	if err != nil {
// 		a.Logger.Error().Err(err).Msg("Error rendering about template")
// 		return c.Reply(fmt.Sprintf("Error rendering template: %s", err))
// 	}

// 	return a.BotReply(c, template)
// }
