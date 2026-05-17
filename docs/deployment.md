# Deployment

This guide covers running llamactl as a long-running service on supported platforms.
For interactive use, simply running `llamactl --config /path/to/config.yaml`
in a terminal is enough. For background operation that survives reboots and
auto-restarts on crash, use a platform-native service manager.

## macOS LaunchAgent

LaunchAgent is the native macOS mechanism for user-scoped background services.
It starts llamactl at login, restarts it on crash, and integrates with
the standard `launchctl` tooling.

### Prerequisites

- llamactl installed and on `PATH` (see [Installation](installation.md))
- Backend(s) installed and configured in your `config.yaml`
  (see [Configuration](configuration.md))
- Verify that `llamactl --config /path/to/config.yaml` starts the server
  successfully in foreground before installing the LaunchAgent — this isolates
  config issues from service issues.

### Plist template

Create the file `~/Library/LaunchAgents/com.example.llamactl.plist` with the
following content. Replace placeholders (`<...>`) with absolute paths
on your machine.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
    "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.example.llamactl</string>

    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/llamactl</string>
        <string>--config</string>
        <string>/Users/&lt;USERNAME&gt;/.config/llamactl/config.yaml</string>
    </array>

    <!-- PATH must include any backend binaries (e.g. mlx_lm.server when MLX
         is installed in a Python venv). Adjust to match your setup. -->
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/Users/&lt;USERNAME&gt;/.venvs/mlx-llm/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin</string>
    </dict>

    <key>WorkingDirectory</key>
    <string>/Users/&lt;USERNAME&gt;</string>

    <key>RunAtLoad</key>
    <true/>

    <!-- Always restart, regardless of exit reason. Combined with
         ThrottleInterval, this gives the daemon resilience without
         tight crash loops. -->
    <key>KeepAlive</key>
    <true/>

    <key>ThrottleInterval</key>
    <integer>10</integer>

    <key>StandardOutPath</key>
    <string>/Users/&lt;USERNAME&gt;/Library/Logs/llamactl/llamactl.log</string>

    <key>StandardErrorPath</key>
    <string>/Users/&lt;USERNAME&gt;/Library/Logs/llamactl/llamactl.err.log</string>

    <key>ProcessType</key>
    <string>Background</string>
</dict>
</plist>
```

!!! warning "Absolute paths required"
    LaunchAgent plists do not expand `~`, `$HOME`, or other shell variables.
    Every path in the plist must be absolute. Replace `<USERNAME>` with the
    output of `echo $USER`.

!!! tip "PATH for MLX users"
    If your MLX backend is installed in a Python venv (the common pattern on
    Apple Silicon), the venv `bin` directory must be in `EnvironmentVariables.PATH`
    so that llamactl can find `mlx_lm.server`. Alternatively, set the full path
    to the executable directly in `backends.mlx.command` in your `config.yaml`.

### Installation

```bash
# Ensure the log directory exists (StandardOutPath/StandardErrorPath will fail otherwise)
mkdir -p ~/Library/Logs/llamactl

# Load the LaunchAgent
launchctl bootstrap "gui/$(id -u)" ~/Library/LaunchAgents/com.example.llamactl.plist

# Verify it is running
launchctl print "gui/$(id -u)/com.example.llamactl" | head -20
```

The output of `launchctl print` should show `state = running` and a non-zero
`pid`. The HTTP endpoint should respond at the address configured in your
`config.yaml` (default: `http://127.0.0.1:8080`).

### Inspection and debugging

```bash
# Combined status (state, pid, last exit code, keepalive)
launchctl print "gui/$(id -u)/com.example.llamactl"

# Tail logs in real time
tail -F ~/Library/Logs/llamactl/llamactl.log ~/Library/Logs/llamactl/llamactl.err.log

# Check that the HTTP endpoint responds
curl -s http://127.0.0.1:8080/v1/models | python3 -m json.tool
```

### Updating the plist

If you edit the plist after installation, you must reload it for the changes
to take effect. launchd caches the plist contents in memory at `bootstrap`
time, so editing the file alone does not propagate.

```bash
launchctl bootout "gui/$(id -u)/com.example.llamactl"
sleep 2
launchctl bootstrap "gui/$(id -u)" ~/Library/LaunchAgents/com.example.llamactl.plist
```

!!! note "Why both bootout and bootstrap?"
    `launchctl reload` was deprecated in macOS Catalina. The modern equivalent
    is the explicit `bootout` then `bootstrap` sequence shown above. The brief
    `sleep 2` gives launchd time to fully release the previous instance.

### Stopping and uninstalling

```bash
# Stop the service without uninstalling (will not auto-restart)
launchctl bootout "gui/$(id -u)/com.example.llamactl"

# Full uninstall
launchctl bootout "gui/$(id -u)/com.example.llamactl"
rm ~/Library/LaunchAgents/com.example.llamactl.plist

# Optional: remove the log directory
rm -rf ~/Library/Logs/llamactl
```

!!! warning "Killing the process directly is not enough"
    With `KeepAlive=true`, killing the llamactl process via `kill <pid>` will
    cause launchd to immediately restart it. To stop the service, use
    `launchctl bootout`.

### Common issues

**`Could not find service in domain for user: 501`**

You queried the wrong service identifier. The domain is `gui/$(id -u)` (the
current user's GUI session). The service label must match the `Label` field
in the plist exactly.

**Port already in use on startup (`address already in use`)**

Another process is already bound to the port configured in `config.yaml`.
Find and stop the offending process:

```bash
lsof -nP -iTCP:8080 -sTCP:LISTEN
# Then kill the listed PID, or change `server.port` in config.yaml
```

**Crash loop visible in `proxy.err.log`**

A misconfiguration in either the plist or `config.yaml` is causing llamactl
to exit at startup. `ThrottleInterval=10` prevents this from saturating the
system, but it will keep retrying every 10 seconds. Tail the error log to
see the actual exit reason:

```bash
tail -50 ~/Library/Logs/llamactl/llamactl.err.log
```

Common culprits: wrong path to `--config`, missing backend binary on `PATH`,
permission issues on `data_dir`.

## Linux systemd

A user-level systemd unit is the equivalent of a LaunchAgent on Linux. The
basic shape is:

```ini
# ~/.config/systemd/user/llamactl.service
[Unit]
Description=llamactl - local LLM management
After=network.target

[Service]
ExecStart=/usr/local/bin/llamactl --config %h/.config/llamactl/config.yaml
Restart=on-failure
RestartSec=10s
Environment=PATH=/usr/local/bin:/usr/bin:/bin

[Install]
WantedBy=default.target
```

Install with:

```bash
systemctl --user daemon-reload
systemctl --user enable --now llamactl.service
systemctl --user status llamactl.service
journalctl --user -u llamactl -f
```

> A complete systemd guide (including caveats for backends like vLLM that
> need GPU access) is welcome via [pull request](https://github.com/lordmathis/llamactl/blob/main/CONTRIBUTING.md).

## Verifying the service is healthy

Regardless of platform, llamactl exposes the same OpenAI-compatible probe:

```bash
# Unauthenticated probe (works if require_inference_auth=false)
curl -s http://127.0.0.1:8080/v1/models | python3 -m json.tool

# With auth enabled
curl -s -H "Authorization: Bearer <your-inference-key>" \
    http://127.0.0.1:8080/v1/models | python3 -m json.tool
```

For instance-level diagnostics, see [Troubleshooting](troubleshooting.md).
