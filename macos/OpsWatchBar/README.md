# OpsWatchBar

OpsWatchBar is the macOS menu bar companion for OpsWatch.

It lists visible windows, lets you pick one, and starts the Go watcher in the background with `--window-id`.

## Run

From the OpsWatch repo root:

```bash
cd macos/OpsWatchBar
OPSWATCH_ROOT=/Users/vishal/go/src/github.com/vdplabs/opswatch swift run
```

The menu bar item appears as `OpsWatch`.

## Configure

The app reads configuration from environment variables when it starts:

```bash
export OPSWATCH_VISION_PROVIDER=ollama
export OPSWATCH_MODEL=llama3.2-vision
export OPSWATCH_INTERVAL=10s
export OPSWATCH_MAX_IMAGE_DIMENSION=1000
export OPSWATCH_OLLAMA_NUM_PREDICT=128
export OPSWATCH_ALERT_COOLDOWN=2m
export OPSWATCH_MIN_ANALYSIS_INTERVAL=30s
export OPSWATCH_ENVIRONMENT=prod
```

Optional context can make alerts more specific:

```bash
export OPSWATCH_INTENT="Add a CNAME record for api.example.com"
export OPSWATCH_EXPECTED_ACTION="add CNAME record in existing hosted zone"
export OPSWATCH_PROTECTED_DOMAIN=example.com
```

If these are omitted, OpsWatch still watches for high-risk actions such as DNS zone creation and destructive terminal commands.

Logs are written to:

```text
/tmp/opswatch-menubar.log
```

The log opens automatically when you click `Start Watching`. The watcher also sends macOS notifications for emitted alerts.

## Permissions

macOS may ask for Screen Recording permission for Terminal, Swift, or the built app. If the window list is incomplete or captures fail, grant permission in:

System Settings -> Privacy & Security -> Screen Recording
