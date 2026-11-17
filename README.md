<p align="center">
  <img src="http/public/images/hawk-wing.png" alt="Scout hawk logo" width="120">
</p>

<h1 align="center">Scout</h1>

<p align="center">
  <strong>Your cameras. Your network. Your storage.</strong><br>
  Self-hosted video security with motion, object, and face detection. No cloud, no monthly fees.
</p>

<p align="center">
  <a href="https://hub.docker.com/r/jonotoninnovation/scout"><img alt="Docker pulls" src="https://img.shields.io/docker/pulls/jonotoninnovation/scout?color=20a0ff&label=docker%20pulls"></a>
  <a href="https://pkg.go.dev/github.com/jonoton/scout"><img alt="Go reference" src="https://img.shields.io/badge/go-reference-20a0ff?logo=go&logoColor=white"></a>
  <a href="LICENSE"><img alt="MIT license" src="https://img.shields.io/badge/license-MIT-20a0ff"></a>
</p>

<p align="center">
  <a href="https://jonoton.github.io/scout/"><strong>Documentation</strong></a> ·
  <a href="https://jonoton.github.io/scout/server/DOCKER.html">Quick start</a> ·
  <a href="https://www.jonoton-innovation.com/scout">Website</a> ·
  <a href="#apps">Apps</a>
</p>

---

Scout connects to your IP cameras, watches for people, cars, and other objects, and records clips and sends alerts when it finds them. It runs on a computer you own and saves everything to your own disk.

## Features

- **Fewer false alarms.** Motion, then object, then face detection. Objects only count when they're moving, so spider webs, shadows, and parked cars don't trigger alerts.
- **Recording.** Clips when objects are detected (including a few seconds from before), plus optional 24/7 recording. Old files are deleted automatically by age or disk use.
- **Alerts.** The best snapshots by email or text message, with rate limits.
- **No database.** Settings are YAML files and recordings are plain video files. Edit a camera's file and Scout reloads it.
- **Any camera.** Works with any RTSP or HTTP stream.
- **Secure access.** Optional logins with hashed passwords and two-factor passcodes.
- **Multiple servers.** Link Scout servers together and view every camera from one place.
- **GPU support.** Object and face detection use an NVIDIA GPU when available.

## Quick start

You need Docker and one camera's stream URL.

```bash
mkdir -p ~/scout/.config ~/scout/.logs ~/scout/data && cd ~/scout
```

The container runs as user ID 1000. If `id -u` shows a different number, run `sudo chown -R 1000:1000 ~/scout`.

Add `~/scout/.config/manage.yaml`:

```yaml
data: /scout/data
monitors:
  - name: front-door
    config: front-door.yaml
```

And `~/scout/.config/front-door.yaml`:

```yaml
url: "rtsp://user:pass@192.168.1.20:554/stream1"
maxSourceFps: 8
maxOutputFps: 2
```

Start Scout and open `http://<server-ip>:8080`:

```bash
docker run -d --name scout --restart unless-stopped \
  -p 8080:8080 -e TZ=America/Chicago \
  -v ~/scout/.config:/scout/.config \
  -v ~/scout/.logs:/scout/.logs \
  -v ~/scout/data:/scout/data \
  jonotoninnovation/scout
```

This gives you live view with detection. To record clips and save alerts, follow the full **[Docker guide](https://jonoton.github.io/scout/server/DOCKER.html)**.

## Apps

| | Live view | Alerts, recordings, notifications |
| :--- | :---: | :---: |
| **Web client**, built in at `http://<server-ip>:8080` | ✓ | |
| **Scout** for Android: [Google Play](https://play.google.com/store/apps/details?id=com.jonoton.scout) · [Amazon Appstore](https://www.amazon.com/gp/mas/dl/android?p=com.jonoton.scout) | ✓ | ✓ |
| **Scout** for iOS: [App Store](https://apps.apple.com/us/app/scout-full/id1658406270) | ✓ | ✓ |
| **Scout Lite** for Android: [Google Play](https://play.google.com/store/apps/details?id=com.jonoton.scout.lite) · [Amazon Appstore](https://www.amazon.com/gp/mas/dl/android?p=com.jonoton.scout.lite) | ✓ | |

**Web client**
- Grid of 1 to 4 columns
- Resolution (320p to 1080p) and quality for each camera
- Pause and fullscreen
- Live frame rate and server RAM use

**Scout** (Android, Amazon, and iOS)
- Home and away connection settings
- Live view of all cameras, with fullscreen pan and zoom
- Alerts, recordings, and continuous recordings to browse, share, or download
- Notifications for object, person, and face detection

**Scout Lite** (Android and Amazon)
- Home and away connection settings
- Live view of all cameras, with fullscreen pan and zoom

## Documentation

| | |
| :--- | :--- |
| [Installation](https://jonoton.github.io/scout/INSTALL.html) | Docker, building from source, and the mobile apps |
| [Configuration](https://jonoton.github.io/scout/CONFIGURE.html) | How the YAML files fit together, with every option |
| [Usage](https://jonoton.github.io/scout/USAGE.html) | The web client, mobile apps, and command-line options |
| [Troubleshooting](https://jonoton.github.io/scout/TROUBLESHOOTING.html) | Cameras that don't appear, missing recordings, and more |
| [Recipes](https://jonoton.github.io/scout/RECIPES.html) | Remote access through a reverse proxy, GPU acceleration |
| [Development](https://jonoton.github.io/scout/DEVELOPMENT.html) | Build from source, profiling, and the docs site |

Example configs are in [`example/config`](example/config), and the docs source is in [`docs`](docs).

## License

[MIT](LICENSE) © Jonoton Innovation LLC
