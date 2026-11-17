---
layout: default
title: Troubleshooting
nav_order: 5
description: "Fixes for common problems: startup errors, cameras that don't appear, missing recordings or alerts."
---

# Troubleshooting
{: .no_toc }

Start with the server log: `docker logs scout` (or the terminal output when running from source). Most problems print a clear message there.
{: .fs-5 .fw-300 }

1. TOC
{:toc}

## Scout exits right away with "Required config file … manage.yaml not found"

Scout couldn't find `manage.yaml`. In Docker, check that your config folder is mounted at `/scout/.config` and that the file is named exactly `manage.yaml`. From source, see [where Scout looks for `.config`]({{ site.baseurl }}{% link CONFIGURE.md %}#where-scout-looks-for-config).

## A camera doesn't appear

Look for the camera's name in the log:

| Log message | Fix |
| :--- | :--- |
| `No video source for <name>` | Add a `url` (or `filename`) to the camera file. |
| `Could not create video reader for <name>` | Set both `maxSourceFps` and `maxOutputFps` above 0. |
| `monitor.NewConfig returned nil` | The camera file is missing, has a typo in its path in `manage.yaml`, or isn't valid YAML. |
| `Stale monitor <name>` | Scout connected but stopped getting frames. Check the URL in VLC, try `captureOptions: "rtsp_transport;tcp"`, or raise `captureTimeoutMilliSeconds`. |
| `Stale monitor DONE retrying` | Scout restarted the camera `staleMaxRetry` times and gave up. Fix the stream, then restart Scout. |

## Nothing is recorded, or no alerts

Recordings and alerts are triggered by **object detection**, which only runs on frames with motion. Check that:

1. `manage.yaml` has a `data` folder. Without it, recording and alerts are turned off.
2. The camera file has `record:` (and/or `alert:`) pointing at a file that exists.
3. `record.yaml` has `recordObjects: true`.
4. Neither `motion.yaml` nor `tensor.yaml` has `skip: true`.
5. If `tensor.yaml` has an `allowedList`, the objects you expect are in it (for example `person`).
6. In Docker, the `data` folder is writable by user ID 1000.

## Recordings or alerts disappear

Saved files are cleaned up every hour. If `deleteAfterHours` or `deleteAfterGB` is missing or `0`, everything is deleted at the next cleanup. Set both above 0 in `record.yaml`, `continuous.yaml`, and `alert.yaml`. Also note that `deleteAfterGB` is per camera.

## Check required recording and alert settings

These settings must be set, and above 0, whenever the file is used:

- `alert.yaml`: `intervalMinutes`
- `record.yaml` and `continuous.yaml`: `timeoutSec` and `maxSec`

## "Permission denied" in Docker

The container runs as user ID 1000. Give that user your Scout folders:

```bash
sudo chown -R 1000:1000 ~/scout
```

## Timestamps are in the wrong time zone

The image defaults to `America/New_York`. Pass your zone with `-e TZ=Europe/London` (or similar). Mounting `/etc/localtime` alone doesn't change it, because the image sets `TZ`.

## No email or text alerts

1. `notify-sender.yaml` is in `.config` with the right host, port, and an app password. Restart Scout after editing it.
2. The camera has both `alert:` and `notifyRx:` set.
3. Images are only attached while `maxSendAttachmentsPerHour` hasn't been reached; set it above 0.
4. Text messages depend on your carrier's email-to-text gateway, which some carriers have shut down. Try email.

## Signing in to the web client doesn't stick

The web client's sign-in cookie only works over HTTPS (or on `localhost`). Over plain `http://<server-ip>:8080` the browser drops it, so you're asked to sign in again. Access Scout through a [reverse proxy with HTTPS]({{ site.baseurl }}{% link RECIPES.md %}#remote-access-with-a-reverse-proxy).

## Everyone is signed out after a restart

Set `loginSigningKey` in [`http.yaml`]({{ site.baseurl }}{% link config/HTTP.md %}) to a fixed random string. Without it, Scout creates a new key on every start.

## Live view doesn't load through a reverse proxy

Turn on WebSocket support in the proxy. See [Remote access]({{ site.baseurl }}{% link RECIPES.md %}#remote-access-with-a-reverse-proxy).

## Still stuck?

[Open an issue on GitHub](https://github.com/jonoton/scout/issues).
