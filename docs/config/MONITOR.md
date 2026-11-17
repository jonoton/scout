---
layout: default
title: Monitor
parent: Configuration
nav_order: 3
description: "Camera config: video source, frame rates, and which tuning files to use."
---

# Monitor (camera config)

Each camera has its own YAML file, named anything you like (for example `front-door.yaml`) and listed in [`manage.yaml`]({{ site.baseurl }}{% link config/MANAGE.md %}). It sets the video source and points to the optional files that tune detection, recording, and alerts.

{: .important }
Every camera needs a source (`url` or `filename`) and both `maxSourceFps` and `maxOutputFps` set above 0. Without them the camera doesn't start, and the log shows `No video source` or `Could not create video reader`.

## Source

| Field | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `url` | string | One of `url` / `filename` | none | RTSP or HTTP stream URL, for example `rtsp://user:pass@192.168.1.20:554/stream1`. |
| `filename` | string | One of `url` / `filename` | none | Path to a video file, useful for testing. Used instead of `url` when both are set. |
| `maxSourceFps` | int | **Yes** | none | Frames per second to read from the camera. |
| `maxOutputFps` | int | **Yes** | none | Frames per second sent through detection, recording, and live view. Lower values use less CPU; `2` to `5` is typical. |
| `quality` | int | No | `100` | JPEG quality (1 to 99) for frames Scout produces. Values outside that range keep `100`. |

{: .note }
Scout handles video only. Audio from the camera stream is not played in live view or saved in recordings.

## Stream connection (URL sources only)

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `captureTimeoutMilliSeconds` | int | `3000` | How long to wait for a frame before treating the read as failed. |
| `captureOptions` | string | none | FFmpeg capture options for OpenCV, as `key;value` pairs separated by `\|`. For example, `rtsp_transport;tcp\|err_detect;explode` forces RTSP over TCP. |
| `captureMaxFailureTolerance` | int | `10` | Consecutive failed reads before the stream is treated as disconnected. |

## Health and buffering

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `staleTimeout` | int | `20` | Seconds without new frames before the camera is considered stale and restarted. |
| `staleMaxRetry` | int | `10` | How many times to restart a stale camera before giving up. |
| `bufferSeconds` | int | `0` | Seconds of frames to buffer between processing stages, to absorb short CPU spikes. `0` turns buffering off. |
| `delayBufferMilliSeconds` | int | `0` | Hold detection results this many milliseconds before they reach recording, alerts, and live view. |

## Tuning files

All of these are optional. Paths are relative to `.config` or absolute, and several cameras can share one file.

| Field | Points to | If omitted |
| :--- | :--- | :--- |
| `motion` | [Motion detection settings]({{ site.baseurl }}{% link config/DETECTION.md %}#motion-detection) | Motion detection runs with built-in defaults. |
| `tensor` | [Object detection settings]({{ site.baseurl }}{% link config/DETECTION.md %}#object-detection) | Object detection runs with built-in defaults. |
| `face` | [Face detection settings]({{ site.baseurl }}{% link config/DETECTION.md %}#face-detection) | Face detection runs with built-in defaults. |
| `record` | [Event recording]({{ site.baseurl }}{% link config/RECORDING_ALERTS.md %}#event-recording) | No clips are recorded. |
| `continuous` | [Continuous recording]({{ site.baseurl }}{% link config/RECORDING_ALERTS.md %}#continuous-recording) | No continuous recording. |
| `alert` | [Alerts]({{ site.baseurl }}{% link config/RECORDING_ALERTS.md %}#alerts) | No alert images or notifications. |
| `notifyRx` | [Alert recipients]({{ site.baseurl }}{% link config/NOTIFICATIONS.md %}#recipients-notify-rxyaml) | Alerts are saved but not emailed or texted. |

## Example

```yaml
url: "rtsp://user:pass@192.168.1.20:554/stream1"
maxSourceFps: 8
maxOutputFps: 2
captureOptions: "rtsp_transport;tcp"
staleTimeout: 20
staleMaxRetry: 10
bufferSeconds: 4
tensor: tensor.yaml
record: record.yaml
alert: alert.yaml
notifyRx: notify-rx.yaml
```

Saving this file (or any file it points to) restarts the camera with the new settings.
