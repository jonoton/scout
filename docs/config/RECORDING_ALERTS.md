---
layout: default
title: Recording & Alerts
parent: Configuration
nav_order: 5
description: "Event recording, continuous recording, alerts, and automatic cleanup."
---

# Recording & Alerts
{: .no_toc }

1. TOC
{:toc}

Recordings and alerts are saved under the `data` folder from [`manage.yaml`]({{ site.baseurl }}{% link config/MANAGE.md %}). Each feature is turned on by pointing a [camera config]({{ site.baseurl }}{% link config/MONITOR.md %}) at its file. File names are up to you.

| Feature | Camera field | Saved to | Triggered by |
| :--- | :--- | :--- | :--- |
| Event recording | `record` | `data/recordings/` | A detected object |
| Continuous recording | `continuous` | `data/continuous/` | Every frame |
| Alerts | `alert` | `data/alerts/` | A detected object |

{: .warning }
In all three files, set `deleteAfterHours` and `deleteAfterGB` above `0`. Scout cleans up every hour, and a value of `0` means "delete everything": all files older than 0 hours, or everything once the folder is over 0 GB.

## Event recording

Records a clip whenever [object detection]({{ site.baseurl }}{% link config/DETECTION.md %}#object-detection) finds an object (for example a person or car).

| Field | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `recordObjects` | bool | **Yes** | `false` | Set to `true` to record when objects are detected. |
| `timeoutSec` | int | **Yes** | none | Keep recording until no object has been seen for this many seconds. Must be above 0. |
| `maxSec` | int | **Yes** | none | Longest clip, in seconds. A longer event continues in a new clip. Must be above 0. |
| `maxPreSec` | int | No | `0` | Seconds of video from before the detection to include at the start of the clip. |
| `deleteAfterHours` | int | **Yes** | none | Delete clips older than this many hours. |
| `deleteAfterGB` | int | **Yes** | none | When this camera's clips use more than this many GB, delete the oldest. |
| `portableOnly` | bool | No | `false` | Scout saves a full-size clip and a smaller *portable* copy for mobile viewing. Set to `true` to save only the portable copy. |
| `codec` | string | No | `mp4v` | Video codec as a 4-character [FourCC](https://fourcc.org/codecs.php) code. Must be supported by your OpenCV/FFmpeg build. |
| `fileType` | string | No | `mp4` | File extension. |
| `bufferSeconds` | int | No | `0` | Seconds of frames to buffer before the recorder, to absorb short CPU spikes. |

```yaml
recordObjects: true
maxPreSec: 4
timeoutSec: 10
maxSec: 120
deleteAfterHours: 72
deleteAfterGB: 20
```

## Continuous recording

Records around the clock in back-to-back segments.

| Field | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `maxSec` | int | **Yes** | none | Length of each segment, in seconds. Must be above 0. |
| `timeoutSec` | int | **Yes** | none | If no frames arrive for this many seconds (for example the camera dropped out), close the current segment. Must be above 0. |
| `deleteAfterHours` | int | **Yes** | none | Delete segments older than this many hours. |
| `deleteAfterGB` | int | **Yes** | none | When this camera's segments use more than this many GB, delete the oldest. |
| `portableOnly` | bool | No | `false` | Save only the smaller portable copy. |
| `codec` | string | No | `mp4v` | Video codec (4-character FourCC). |
| `fileType` | string | No | `mp4` | File extension. |
| `bufferSeconds` | int | No | `0` | Seconds of frames to buffer before the recorder. |

```yaml
maxSec: 300
timeoutSec: 10
deleteAfterHours: 36
deleteAfterGB: 50
portableOnly: true
```

{: .tip }
Continuous recording uses far more disk than event recording. Size `deleteAfterGB` to your disk, and remember the limit applies per camera.

## Alerts

While objects are being detected, Scout collects the best frames (those with the most faces and objects first). Once per interval it saves them as images and, if the camera has [recipients]({{ site.baseurl }}{% link config/NOTIFICATIONS.md %}), sends an email or text. Saved alerts are listed in the mobile apps even without notifications set up.

| Field | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `intervalMinutes` | int | **Yes** | none | Send at most one alert per this many minutes. Must be above 0. |
| `maxImagesPerInterval` | int | No | `1` | Best frames kept per alert. Values below 1 are treated as 1. |
| `saveHighlighted` | bool | No | `false` | Save the frame with detection boxes drawn on it. |
| `saveOriginal` | bool | No | `false` | Save the frame without boxes. |
| `saveObjectsCount` | int | No | `0` | Also save a cropped image of up to this many objects per frame. |
| `saveFacesCount` | int | No | `0` | Also save a cropped image of up to this many faces per frame. |
| `saveQuality` | int | No | `0` | JPEG quality (1 to 100) of saved full-frame images. At `0` they are saved at the lowest quality, so set it, for example to `75`. Cropped object and face images are always saved at `100`. |
| `maxSendAttachmentsPerHour` | int | No | `0` | How many alerts per hour include images. Later alerts that hour are sent as text only. At `0`, alerts never include images. |
| `textAttachments` | bool | No | `false` | Also send images by text message (MMS), for images up to 300 KB. |
| `deleteAfterHours` | int | **Yes** | none | Delete alert images older than this many hours. |
| `deleteAfterGB` | int | **Yes** | none | When this camera's alert images use more than this many GB, delete the oldest. |

```yaml
intervalMinutes: 5
maxImagesPerInterval: 1
maxSendAttachmentsPerHour: 2
saveQuality: 75
saveHighlighted: true
saveObjectsCount: 4
saveFacesCount: 4
deleteAfterHours: 24
deleteAfterGB: 2
```

Who receives alerts is set separately, in [Notifications]({{ site.baseurl }}{% link config/NOTIFICATIONS.md %}).
