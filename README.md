# handy-router

A local routing service for [Handy](https://handy.computer) that intercepts speech-to-text transcriptions and routes them to different destinations based on a spoken wake word.

## What it does

Handy transcribes your voice and pastes it into the focused text field. **handy-router** sits in between — it receives the transcription, checks for a wake word, and routes accordingly:

| Say this | What happens |
|----------|-------------|
| `"transcribe hello world"` | Enhances the text via LLM, types it into the active window, and hands it back to Handy |
| `"gemini what is rust"` | Opens Gemini in your browser with "what is rust" |
| `"chat write a regex for emails"` | Starts an OpenCode session with the prompt, then opens a TUI attached to it |
| `"hello world"` | No wake word — **no-op**, nothing is pasted |

## How it works

handy-router exposes an OpenAI-compatible `/v1/chat/completions` endpoint. Handy's **Custom post-processing provider** sends transcriptions to this endpoint. The router:

1. Extracts the raw transcription from the request
2. Matches a wake word (`transcribe`, `gemini`, `chat`, or none)
3. Routes to the appropriate destination
4. Returns the response to Handy

For the `transcribe` route, the router enhances the transcription through a configurable LLM (DeepSeek V4 Flash via OpenCode Go by default), then injects the result **directly into the active window** using [`ydotool`](https://github.com/ReimuNotMoe/ydotool). This works in terminals, text boxes, and any other focused input — unlike a `Ctrl+V` paste, which terminals reject. The enhanced text is also returned to Handy so it can place it on the clipboard.

> **Important:** Set Handy's paste method to **None** so it does not paste a second time. The router does the injection; Handy only sees the returned text (and can copy it to the clipboard per its clipboard handling setting).

With no wake word, the router returns an empty response so Handy is a no-op.

## OpenCode integration

The `chat:` route uses a persistent headless OpenCode server instead of shelling out one-shot commands:

1. `opencode serve` runs as a systemd user service on port **11342**
2. On `chat:` the router calls the server API to create a new session
3. It sends the transcript as the first message (`prompt_async`)
4. It opens your terminal running `opencode attach <server> --session <id>`
5. The TUI opens attached to that session, so you can watch the reply stream in and continue interactively

This gives you a real, resumable OpenCode session for every voice command rather than a process that exits immediately.

```toml
[opencode]
server_url = "http://127.0.0.1:11342"   # opencode serve endpoint
serve_dir = "~/Projects/OpenCode"        # working directory for the server
dir = "~/Projects/OpenCode"              # directory for the attached TUI
```

## Quick start

### Prerequisites

- [Go](https://go.dev/doc/install) (on Arch: `sudo pacman -S go`)
- [Handy](https://handy.computer) with post-processing enabled
- [`ydotool`](https://github.com/ReimuNotMoe/ydotool) (on Arch: `sudo pacman -S ydotool`) — for typing the enhanced text into the active window. The `ydotoold` user service must be running.
- [opencode](https://opencode.ai) (for the `chat:` route)
- An API key for [OpenCode Go](https://opencode.ai/go) (for text enhancement)

### Install and run

```bash
git clone https://github.com/PercyJax/handy-router.git
cd handy-router

# Build and install to /usr/local/bin, copy config, install systemd service
make install

# Enable and start
make enable
```

### Configure Handy

1. Open Handy Settings
2. Go to **Advanced > Experimental Features** and enable **Post Processing**
3. Set:
   - **Provider**: Custom
   - **Base URL**: `http://localhost:11341/v1`
   - **Model**: `handy-router` (any string works)
   - **Prompt**: `${output}` (the router reads the transcription from the user message; it does not use the prompt)
4. Set Handy's **Paste Method** to **None**. The router types the enhanced text itself, so Handy must not paste again. Optionally set **Clipboard Handling** to "Copy to Clipboard" if you also want it on the clipboard.

### Set your API key

Edit `~/.config/handy-router/config.toml`:

```toml
[enhance]
api_key = "your-opencode-go-api-key"
```

Then restart the service: `make restart`

### After code changes

When you pull new code and want to update the running service:

```bash
# Rebuild, reinstall binary, and restart service
make reinstall
make restart
```

This rebuilds the binary, copies it to `/usr/local/bin`, and restarts the systemd service. Your config is preserved.

If the service won't restart, check logs: `make logs`

## Configuration

Config is loaded from (in order):
1. `./config.toml` (current directory)
2. `~/.config/handy-router/config.toml`

CLI flags override config:
```bash
./handy-router --port 9999 --host 0.0.0.0
```

### Full config reference

```toml
[server]
port = 11341
host = "127.0.0.1"

[routes.prefixes]
transcribe = "enhance"  # "transcribe ..." -> enhance via LLM, then paste
gemini = "gemini"       # "gemini ..." -> open Gemini
chat = "code"           # "chat ..." -> open OpenCode

[gemini]
url_template = "https://gemini.google.com/app?q={query}"
browser = "firefox"              # empty = system default (xdg-open)
browser_args = ["--new-window"]  # opens a focused window instead of a background tab

[opencode]
binary = "opencode"
terminal = "ghostty"       # change to your terminal (kitty, alacritty, konsole, etc.)
terminal_args = ["-e"]
extra_args = []            # e.g. ["--model", "anthropic/claude-sonnet-4-20250514"]
dir = "~/Projects/OpenCode"
server_url = "http://127.0.0.1:11342"
serve_dir = "~/Projects/OpenCode"

[logging]
level = "info"             # debug, info, warn, error

[enhance]
enabled = true
endpoint = "https://opencode.ai/zen/go/v1/chat/completions"
model = "deepseek-v4-flash"
api_key = ""               # your OpenCode Go API key
paste = true               # type the enhanced text into the active window
paste_binary = "ydotool"   # typing tool (ydotool)
```

### LLM Enhancement

The `[enhance]` section controls the `transcribe` route:

- **endpoint**: Any OpenAI-compatible chat completions API
- **model**: Model ID to use
- **api_key**: API key for the endpoint
- **enabled**: Set to `false` to skip enhancement (raw text is returned as-is)
- **paste**: When `true`, the enhanced text is typed into the active window via `paste_binary`
- **paste_binary**: Path/name of the typing tool (default `ydotool`)

If the LLM call fails (no API key, network error), the raw transcription is used instead.

## Makefile targets

```bash
make install    # build, install binary + config + both systemd services
make reinstall  # uninstall + install (for updating after code changes)
make uninstall  # remove binary + services (keeps config)
make enable     # enable and start handy-router
make disable    # stop and disable handy-router
make start      # start handy-router
make stop       # stop handy-router
make restart    # restart handy-router
make status     # show handy-router status
make logs       # follow handy-router logs (Ctrl+C to stop)

# opencode-serve service
make opencode-serve-enable   # enable and start opencode serve
make opencode-serve-disable  # stop and disable opencode serve
make opencode-serve-start    # start opencode serve
make opencode-serve-stop     # stop opencode serve
make opencode-serve-restart  # restart opencode serve
make opencode-serve-status   # show opencode serve status
make opencode-serve-logs     # follow opencode serve logs
```

## Services

Two systemd **user services**:

| Service | Port | Purpose |
|---------|------|---------|
| `handy-router.service` | 11341 | Receives Handy transcriptions and routes them |
| `opencode-serve.service` | 11342 | Headless OpenCode server for the `chat:` route |

Both start automatically after your graphical session. Install them with `make install`, then enable with `make enable` and `make opencode-serve-enable`.

The `transcribe` route also relies on the `ydotoold` user service (provided by the `ydotool` package). Enable it once with `systemctl --user enable --now ydotool.service`.

To keep them running when not logged in (optional):
```bash
loginctl enable-linger $USER
```

## Status page

Open `http://localhost:11341` in your browser to see:
- Server status and port
- Active route prefixes
- Recent transcription history

## Platform support

Primary target: **Linux / Arch / KDE Plasma (Wayland)**.

| Platform | Browser | Terminal | Text injection | Notes |
|----------|---------|----------|----------------|-------|
| Linux (KDE/Arch) | `firefox` / `xdg-open` | configurable | `ydotool` | Primary target |
| Linux (other) | `xdg-open` | configurable | `ydotool` | Should work |
| macOS | `open` | configurable | not implemented | Untested |

The `transcribe` route injects text via `ydotool`, which is Linux-only. On other platforms, set `paste = false` and let Handy paste instead.

## License

MIT
