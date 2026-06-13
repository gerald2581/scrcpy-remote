# scrcpy-remote

A single Go binary that serves a **local web dashboard** to connect to and launch
[scrcpy](https://scrcpy.org/) against a **remote Android phone** reached over a
[Tailscale](https://tailscale.com/) mesh. scrcpy itself opens the phone's screen in its own
window — this tool just manages devices, connects, and launches with a chosen quality preset.

> **Not a latency reducer.** Mirroring speed is governed by the network (RTT, packet loss),
> scrcpy's pipeline, and the transport (TCP). Across countries it will lag; this tool makes
> setup/connection convenient and repeatable, it does not make the link faster. (A
> lower-latency UDP/WebRTC approach is a separate project.)

## Phone side (do once — by whoever has the phone)

1. Install **Tailscale** and sign in to the **same tailnet** as the controller PC.
2. Settings → About → tap **Build number** 7× to unlock **Developer Options**.
3. Developer Options → enable **Wireless Debugging**. Note its **IP & port**, and use
   **Pair device with pairing code** to get the **pairing port + code**.
4. Note the phone's **Tailscale IP** (`100.x.x.x`) — you'll enter it in the dashboard.

> ⚠️ Wireless Debugging may turn **off after a reboot**. Without physical access to the
> phone you cannot re-enable it remotely — keep that in mind for an unattended device.

## Controller side (the PC you watch from)

1. Install **Tailscale** (needs admin) and sign in to the same tailnet.
2. Put **`scrcpy`** and **`adb`** on your `PATH` (or in `~/.scrcpy-remote/bin/`). On Windows
   the scrcpy release zip already bundles `adb.exe`.
3. Run the binary:
   ```
   ./scrcpy-remote            # opens http://127.0.0.1:8787 in your browser
   ./scrcpy-remote --port 9000
   ```
4. In the dashboard: **Add device** (name + Tailscale IP + adb port) → **Connect** → **Launch**.
   (First time only: pair via `adb pair <ip>:<pairPort> <code>` from a terminal.)

## Quality presets (the smoothness knob)

| Preset | Trade-off |
|---|---|
| **Balanced** (default) | 1024px / 2 Mbps / 30 fps — good general default |
| **Fast** | 800px / 1.5 Mbps / 20 fps, no audio — smoother on slow/long links, blurrier |
| **HD** | 1600px / 8 Mbps / 60 fps — sharp, needs a good link |

## Troubleshooting

- **`adb connect` fails:** check Wireless Debugging is still on (it may have reset after a
  phone reboot); re-pair if needed.
- **Laggy / choppy:** the Tailscale peer may be using a **relay** instead of a **direct**
  connection (`tailscale status` shows which) — direct is faster. Also drop to the **Fast**
  preset, and prefer the phone on **Wi‑Fi** over mobile data.
- **Dashboard port in use:** start with `--port <n>`.

## Security

- The dashboard binds **`127.0.0.1` only** — it is never exposed to the network.
- adb traffic rides the **Tailscale encrypted mesh**; adb is **never** exposed publicly
  (adb = full device control with no auth, so this matters).

## Build

```
go build -o scrcpy-remote .
# cross-compile:
GOOS=windows GOARCH=amd64 go build -o dist/scrcpy-remote-win.exe .
GOOS=darwin  GOARCH=arm64 go build -o dist/scrcpy-remote-mac .
GOOS=linux   GOARCH=amd64 go build -o dist/scrcpy-remote-linux .
```
Web assets are embedded (`go:embed`), so each binary is standalone.
