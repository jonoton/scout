---
layout: default
title: Docker
parent: Installation
nav_order: 1
description: "Run Scout with Docker: folders, a minimal camera config, and the run command."
---

# Run Scout with Docker
{: .no_toc }

Docker is the recommended way to run Scout. The image includes OpenCV and the
object and face detection models, so there is nothing else to install.
{: .fs-5 .fw-300 }

1. TOC
{:toc}

## Before you start

- A Linux machine that stays on (a mini PC, NAS, or Raspberry Pi-class board) with [Docker](https://docs.docker.com/get-started/get-docker/) installed.
- The RTSP or HTTP stream URL for at least one camera, for example `rtsp://user:pass@192.168.1.20:554/stream1`. Your camera's manual or app lists it.

## Quick start

<ol class="steps">
<li markdown="1">
**Create the folders Scout uses.** They hold your config, logs, and saved recordings.

```bash
mkdir -p ~/scout/.config ~/scout/.logs ~/scout/data
cd ~/scout
```

{: .important }
The container runs as user ID `1000`. If your own user ID is not 1000 (check with `id -u`), give that user ownership: `sudo chown -R 1000:1000 ~/scout`. Otherwise Scout can't write logs or recordings.
</li>

<li markdown="1">
**Add a minimal config.** Create these four files in `~/scout/.config`. Replace the camera URL with your own.

`manage.yaml` lists your cameras and where to save data:

```yaml
data: /scout/data
monitors:
  - name: front-door
    config: front-door.yaml
```

`front-door.yaml` is the camera:

```yaml
url: "rtsp://user:pass@192.168.1.20:554/stream1"
maxSourceFps: 8   # frames/sec read
maxOutputFps: 2   # frames/sec analyzed
record: record.yaml
alert: alert.yaml
```

`record.yaml` records a clip when an object (like a person or car) is detected:

```yaml
recordObjects: true
maxPreSec: 4
timeoutSec: 10
maxSec: 120
deleteAfterHours: 24
deleteAfterGB: 2
```

`alert.yaml` saves alert snapshots, which the mobile apps show:

```yaml
intervalMinutes: 5
maxImagesPerInterval: 1
saveQuality: 75
saveHighlighted: true
saveObjectsCount: 4
saveFacesCount: 4
deleteAfterHours: 24
deleteAfterGB: 2
```

{: .warning }
Always set `deleteAfterHours` and `deleteAfterGB` to values above 0. At `0`, Scout deletes every saved file on its hourly cleanup.
</li>

<li markdown="1">
**Start Scout.**

```bash
docker run -d --name scout --restart unless-stopped \
  -p 8080:8080 \
  -e TZ=America/Chicago \
  -v ~/scout/.config:/scout/.config \
  -v ~/scout/.logs:/scout/.logs \
  -v ~/scout/data:/scout/data \
  jonotoninnovation/scout
```

Set `TZ` to your [time zone](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones). Without it, timestamps use `America/New_York`.
</li>

<li markdown="1">
**Check it's working.** Open `http://<server-ip>:8080` in a browser. You should see your camera's live view with its frame rate. If not, check the log:

```bash
docker logs scout
```

See [Troubleshooting]({{ site.baseurl }}{% link TROUBLESHOOTING.md %}) for common startup errors.
</li>
</ol>

Motion, object, and face detection are on by default, so this setup already records clips and saves alerts when it sees a person, car, or other object. Clips are saved as `.mp4` files in `~/scout/data/recordings/`, which you can play with any video player, or browse in the mobile apps. To get email or text alerts, add [notification settings]({{ site.baseurl }}{% link config/NOTIFICATIONS.md %}).

## Docker Compose

The same setup as a `docker-compose.yml`:

```yaml
services:
  scout:
    image: jonotoninnovation/scout
    container_name: scout
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - TZ=America/Chicago
    volumes:
      - ./.config:/scout/.config
      - ./.logs:/scout/.logs
      - ./data:/scout/data
```

Start it with `docker compose up -d` from `~/scout`.

## Container reference

| Path in container | Purpose |
| :--- | :--- |
| `/scout/.config` | Your YAML config files. Scout reloads a camera when its config file changes. |
| `/scout/.logs` | Web request (`access`) and sign-in (`logins`) logs, written when logins are set up. |
| `/scout/data` | Recordings, continuous recordings, and alert images (set by `data` in `manage.yaml`). |

| Setting | Value |
| :--- | :--- |
| Port | `8080` (change with `port` in [`http.yaml`]({{ site.baseurl }}{% link config/HTTP.md %})) |
| User | UID/GID `1000` |
| Time zone | `TZ` environment variable (default `America/New_York`) |
| GPU | A CUDA image and Compose file are in [`example/recipes`](https://github.com/jonoton/scout/tree/master/example/recipes). |

## Update Scout

```bash
docker pull jonotoninnovation/scout
docker rm -f scout
# then run the same docker run command again
```

With Compose: `docker compose pull && docker compose up -d`.

## Build the image yourself

```bash
git clone https://github.com/jonoton/scout.git
cd scout
docker build -t scout .
```

The [`Dockerfile`](https://github.com/jonoton/scout/blob/master/Dockerfile) builds for `linux/amd64` and `linux/arm64` (for example, `docker buildx build --platform linux/amd64,linux/arm64 -t scout .`).

## Next steps

- [Configure Scout]({{ site.baseurl }}{% link CONFIGURE.md %}): add cameras, tune detection, and set up logins.
- [Set up the mobile apps]({{ site.baseurl }}{% link INSTALL.md %}#mobile-apps).
