---
layout: default
title: Monitor Config
parent: Configuration
nav_order: 3
---

# Monitor Configuration (`cam.yaml`)

> 💡 **Tip**
> While `manage.yaml` and `http.yaml` are required filenames, monitor configuration files can be named anything (e.g., `front_door.yaml`, `garage.yaml`).

Each monitor (camera) has its own configuration file defining its source and specific detection behaviors.

> ❗ **Important: Source Required**
> You **must** provide either a `filename` or a `url` for each monitor. One of these two fields is required to provide a video source for Scout to process.

| Field | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `filename` | string | No* | - | Path to a local video file (for testing/simulated feeds). |
| `url` | string | No* | - | RTSP/HTTP URL for an IP camera stream. |
| `maxSourceFps` | int | No | `0` | Limit the frame rate coming from the source. |
| `maxOutputFps` | int | No | `0` | Limit the frame rate processed by detection. |
| `quality` | int | No | `0` | JPEG quality (1-100) for snapshots and live view. |
| `captureTimeoutMilliSeconds` | int | No | `0` | Timeout for frame capture. |
| `staleTimeout` | int | No | `20` | Seconds before camera is considered "stale". |
| `staleMaxRetry` | int | No | `10` | Max restart attempts for a stale camera. |
| `bufferSeconds` | int | No | `0` | Seconds to buffer for pre-alert recording. |
| `delayBufferMilliSeconds` | int | No | `0` | Delay processing by this amount. |
| `motion` | string | No | - | Path to [Motion Config](DETECTION#motion-detection-motionyaml) (Recommended: `motion.yaml`). |
| `tensor` | string | No | - | Path to [Object Detection Config](DETECTION#object-detection-tensoryaml) (Recommended: `tensor.yaml`). |
| `face` | string | No | - | Path to [Face Detection Config](DETECTION#face-detection-faceyaml) (Recommended: `face.yaml`). |
| `notifyRx` | string | No | - | Path to notification receiver config (Recommended: `notify-rx.yaml`). |
| `alert` | string | No | - | Path to [Alert Rules Config](RECORDING_ALERTS#alert-rules-alertyaml) (Recommended: `alert.yaml`). |
| `record` | string | No | - | Path to [Event Recording Config](RECORDING_ALERTS#event-recording-recordyaml) (Recommended: `record.yaml`). |
| `continuous` | string | No | - | Path to [Continuous Recording Config](RECORDING_ALERTS#continuous-recording-continuousyaml) (Recommended: `continuous.yaml`). |

