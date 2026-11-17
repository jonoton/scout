---
layout: default
title: Usage
nav_order: 4
description: "Using the web client, the mobile apps, and Scout's command-line options."
---

# Usage
{: .no_toc }

1. TOC
{:toc}

## Web client

Open `http://<server-ip>:8080` in any browser. If [logins]({{ site.baseurl }}{% link config/HTTP.md %}#logins) are set up, sign in first (and enter the passcode if two-factor is on).

**Status bar:** server RAM use, connection status, and the **Grid** menu (Auto, or 1 to 4 columns) and the **Cameras** menu to show or hide cameras.

**Each camera card:**

- Live view, with input and output frame rates.
- **Pause / Resume** to stop streaming a camera and save bandwidth.
- **Fullscreen** to view one camera.
- **Settings** to pick the stream resolution (320p, 480p, 640p, 960p, 720p HD, or 1080p FHD; default 640p) and JPEG quality (25%, 50%, 75%, or 90%; default 50%). **Apply to All Cameras** copies the choice to every camera.

Layout, hidden cameras, and stream settings are saved in your browser.

The web client is for live viewing. Recordings and alerts are browsed in the mobile apps, or directly on disk (see below).

## Recordings on disk

Everything Scout saves is a plain file in the `data` folder from [`manage.yaml`]({{ site.baseurl }}{% link config/MANAGE.md %}), so you can play, copy, or back it up with any tool (VLC, a file manager, `rsync`):

| Folder | Contents |
| :--- | :--- |
| `data/recordings/` | Event clips (`.mp4` by default) |
| `data/continuous/` | Continuous recording segments |
| `data/alerts/` | Alert images (`.jpg`) |

File names start with the camera name. Each clip is saved as a full-size *Full* file and a smaller *Portable* copy, unless `portableOnly` is set. With the Docker setup, the folder is `~/scout/data` on the host.

## Mobile apps

The apps connect to the same server as the web client and add alerts, recordings, continuous recordings, and push notifications (Scout Lite is live view only). See [Installation]({{ site.baseurl }}{% link INSTALL.md %}#mobile-apps) for download links and setup.

## Command-line options

Run Scout with no options to start the server. Extra options run a single task and exit:

| Option | What it does |
| :--- | :--- |
| `--secure-http-passwords` | Replaces plain-text passwords in `http.yaml` with hashes. See [Hashing passwords]({{ site.baseurl }}{% link config/HTTP.md %}#hashing-passwords). |
| `--version` | Prints the version. Official builds set it; builds from source print nothing unless built with a version (see [Development]({{ site.baseurl }}{% link DEVELOPMENT.md %})). |

Any other option prints the usage summary.

## Logs

| Log | Where |
| :--- | :--- |
| Server log | Standard output (`docker logs scout`). |
| Web requests | `.logs/access` |
| Sign-in attempts | `.logs/logins` |

The `access` and `logins` logs are only written when [logins]({{ site.baseurl }}{% link config/HTTP.md %}#logins) are set up. They rotate at 1 MB and keep 5 old files for up to 28 days.
