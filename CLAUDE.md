# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`proxmox-interacter` is a Go CLI/daemon that runs a Telegram bot for interacting with one or more Proxmox
clusters (list nodes/containers/VMs, start/stop/restart, scale resources, list storages) without needing to
log into the Proxmox web UI. It's a single static binary, driven by a TOML config file.

## Build, run, lint

```sh
go build -o proxmox-interacter ./cmd/proxmox-interacter.go   # build
go run ./cmd/proxmox-interacter.go --config <path>            # run (config path is required)
go vet ./...
golangci-lint run                                              # see .golangci.yml for enabled/disabled linters
```

There are currently no `*_test.go` files in the repo, so there is no test suite to run yet.

Releases are built via GoReleaser (`.goreleaser.yml`): binary entrypoint is `./cmd/proxmox-interacter.go`,
targets `linux`/`darwin`, `CGO_ENABLED=0`, packaged as `deb`/`rpm`. CI is in `.github/workflows/actions.yaml`
(lint/build on push) and `.github/workflows/release.yaml` (release on tag).

Config: copy `config.example.toml` and fill in `[telegram]` (bot token, admin chat IDs) and one or more
`[[proxmox]]` blocks (name, url, user, API token). Proxmox API tokens must be created with "Privilege
Separation" unchecked, or the bot only gets partial data.

## Architecture

- `cmd/proxmox-interacter.go` — entrypoint. Uses `cobra` for the CLI, requires `--config`, loads config via
  `pkg.LoadConfig`, constructs `app.App`, calls `Start()`.
- `pkg/load_config.go` — reads/validates the TOML config into `pkg/types.Config`.
- `pkg/types/` — plain data types shared across the app: Proxmox config/API response shapes
  (`proxmox.go`, `container.go`, `container_config.go`, `node.go`, `storage.go`, `scale.go`,
  `cluster_info.go`), and app config (`config.go`). This is the vocabulary the rest of the code speaks.
- `pkg/proxmox/` — the Proxmox API layer:
  - `client.go`: one `Client` per configured Proxmox cluster; raw HTTP calls to the Proxmox API
    (`/api2/json/...`, `/api2/extjs/...`) for resources, container config, and start/stop/reboot/scale
    actions.
  - `manager.go`: `Manager` wraps multiple `Client`s (one per cluster) and fans out `GetNodes()` across all
    of them concurrently (goroutines + mutex). Higher-level operations (`StartContainer`,
    `StopContainer`, `ScaleContainer`, etc.) look up the right cluster/client by container or cluster name
    before delegating to the `Client`.
  - `parsers.go`: turns raw Proxmox API responses into the `pkg/types` structs.
- `pkg/app/` — the Telegram bot itself, built on `github.com/go-telegram-bot-api/telegram-bot-api`
  (note: `gopkg.in/telebot.v3` is also a dependency but the bot is currently wired up with `tgbotapi`, not
  telebot — see `App.Start`/`App.botRun` in `app.go`).
  - `app.go`: `App` struct holds the Telegram bot, `proxmox.Manager`, logger, config. `Start()` launches
    `botRun()`, a single loop over `Bot.GetUpdatesChan` that dispatches both plain commands
    (`update.Message`) and inline-keyboard button presses (`update.CallbackQuery`). Callback data is a
    colon-separated string (`action:arg1:arg2...`) switched on in `botRun`; e.g. `"stop:<name>"` triggers a
    confirmation prompt, `"allowstop:stop:<name>"` actually performs it. `CallbackPrefix*` constants for
    these action strings live in `constants.go`. Every inbound update is gated by `checkIdAdmin` against
    `config.Telegram.Admins` before being processed.
  - One file per command/flow: `containers.go` (list containers/clusters), `container.go` (container
    detail), `container_action.go` (start/stop/restart), `container_scale.go` (resize CPU/memory/swap),
    `node.go`, `disks.go`, `status.go`, `about.go`, `help.go`. `types.go` holds render-only structs used to
    pass data into templates.
- `templates/` + `pkg/templates/templates.go` — HTML templates (`go:embed`) for rendering rich
  messages/pages. Some of this rendering pipeline is currently disconnected: several handlers in
  `app.go`/`App.Start()` that would use `TemplateManager` are commented out while the bot is mid-refactor
  toward callback-query-driven navigation (see `sendMainMenu`, `allowDoRun` for the current pattern). Check
  what's actually wired into `botRun()` before assuming a handler is live.
- `pkg/logger/` — zerolog setup driven by `[log]` config (level, json output).
- `pkg/utils/utils.go` — small formatting helpers (byte sizes, durations, bool→Yes/No, generic `Filter`)
  used by templates and handlers.

## Conventions specific to this repo

- Module path is `main` (see `go.mod`), so internal imports look like `main/pkg/...`, not a GitHub path.
- Errors from Proxmox/business logic are logged with `a.Logger.Error()...` and generally *not* propagated to
  the user beyond a Telegram error message — follow that pattern rather than introducing panics.
- Telegram messages use `MarkdownV2`; any dynamic string interpolated into a message must go through
  `escapeMDV2` first.
- `golangci-lint` here runs `enable-all` with a specific disable list (see `.golangci.yml`) — it's stricter
  than default, so lean on it rather than guessing which linters matter.
