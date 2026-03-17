---
layout: default
title: Recording & Alerts Config
parent: Configuration
nav_order: 5
---

# Recording & Alerts Configuration

> 💡 **Tip**
> Filenames for recording and alerts (e.g., `record.yaml`, `alert.yaml`) can be customized in the [Monitor Config](MONITOR).

## Event Recording (Optional, `record.yaml`)

Triggered by object or person detection.

> 💡 **Note on Requirements**
> While recording and alert modules are optional, once enabled in the [Monitor Config](MONITOR), all fields marked as **Yes** (if any) become required.

| Field | Type | Req. | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `recordObjects` | bool | No | `false` | If true, records when any allowed object is seen. |
| `maxPreSec` | int | No | `0` | Seconds of video to include *before* the trigger. |
| `timeoutSec` | int | No | `0` | Seconds to wait after motion stops. |
| `maxSec` | int | No | `0` | Maximum duration for a single recording file. |
| `deleteAfterHours` | int | No | `0` | Auto-prune files older than this. |
| `deleteAfterGB` | int | No | `0` | Auto-prune if directory exceeds this (GB). |
| `codec` | string | No | `mp4v` | Video codec (4 characters). |
| `fileType` | string | No | `mp4` | Video file extension. |
| `bufferSeconds` | int | No | `0` | Number of seconds of pre-trigger video to buffer. |
| `portableOnly` | bool | No | `false` | If true, only saves a lightweight version. |

## Continuous Recording (Optional, `continuous.yaml`)

| Field | Type | Req. | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `timeoutSec` | int | No | `0` | Seconds to wait after segment ends. |
| `maxSec` | int | No | `0` | Duration of each video segment. |
| `deleteAfterHours` | int | No | `0` | Auto-prune files older than this. |
| `deleteAfterGB` | int | No | `0` | Disk usage limit for continuous recordings. |
| `codec` | string | No | `mp4v` | Video codec (4 characters). |
| `fileType` | string | No | `mp4` | Video file extension. |
| `bufferSeconds` | int | No | `0` | Number of seconds to buffer for continuous segments. |
| `portableOnly` | bool | No | `false` | If true, only saves a lightweight version for continuous. |

## Alert Rules (Optional, `alert.yaml`)

Defines how and when notifications are sent. Notification recipients are configured separately in [Notification Settings](NOTIFICATIONS).

| Field | Type | Req. | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `intervalMinutes` | int | No | `0` | Don't send more than one alert every X minutes. |
| `maxImagesPerInterval` | int | No | `0` | Limit snapshots sent per alert window. |
| `maxSendAttachmentsPerHour` | int | No | `0` | Limit total attachments sent per hour. |
| `saveQuality` | int | No | `0` | Image quality for alert snapshots (1-100). |
| `saveOriginal` | bool | No | `false` | Save the un-annotated original image. |
| `saveHighlighted` | bool | No | `false` | If true, saves the image with bounding boxes. |
| `saveObjectsCount` | int | No | `0` | Max objects to save snapshots for per image. |
| `saveFacesCount` | int | No | `0` | Max faces to save snapshots for per image. |
| `textAttachments` | bool | No | `false` | Send images as attachments in text messages. |
| `deleteAfterHours` | int | No | `0` | Auto-prune alerts older than this. |
| `deleteAfterGB` | int | No | `0` | Disk usage limit for alerts. |
