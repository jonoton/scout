---
layout: default
title: Configuration
nav_order: 3
has_children: true
description: "How Scout's YAML config files fit together."
---

# Configuration
{: .no_toc }

Scout is configured with YAML files in a `.config` folder. There is no database and no setup wizard: edit a file, and Scout picks it up.
{: .fs-5 .fw-300 }

1. TOC
{:toc}

## How the files fit together

`manage.yaml` is the starting point. It lists your cameras (Scout calls them *monitors*), and each camera's file points to the optional files that tune it.

```text
.config/
├── manage.yaml            required: data folder + list of cameras
├── http.yaml              optional: port, logins, linked servers
├── notify-sender.yaml     optional: SMTP account for email/text alerts
│
├── front-door.yaml        one file per camera (any name)
│   ├── motion: motion.yaml        tune motion detection
│   ├── tensor: tensor.yaml        tune object detection
│   ├── face: face.yaml            tune face detection
│   ├── record: record.yaml        record clips when objects are seen
│   ├── continuous: continuous.yaml  record around the clock
│   ├── alert: alert.yaml          save alert images / send alerts
│   └── notifyRx: notify-rx.yaml   who receives alerts
└── garage.yaml            cameras can share the same tuning files
```

| File | Name fixed? | Reference |
| :--- | :--- | :--- |
| `manage.yaml` | Yes | [Manage]({{ site.baseurl }}{% link config/MANAGE.md %}) |
| `http.yaml` | Yes | [HTTP]({{ site.baseurl }}{% link config/HTTP.md %}) |
| `notify-sender.yaml` | Yes | [Notifications]({{ site.baseurl }}{% link config/NOTIFICATIONS.md %}) |
| Camera files | No | [Monitor]({{ site.baseurl }}{% link config/MONITOR.md %}) |
| `motion.yaml`, `tensor.yaml`, `face.yaml` | No | [Detection]({{ site.baseurl }}{% link config/DETECTION.md %}) |
| `record.yaml`, `continuous.yaml`, `alert.yaml` | No | [Recording & Alerts]({{ site.baseurl }}{% link config/RECORDING_ALERTS.md %}) |
| `notify-rx.yaml` | No | [Notifications]({{ site.baseurl }}{% link config/NOTIFICATIONS.md %}) |

Paths inside camera files are relative to the `.config` folder, or absolute.

## Minimal setup

One camera that records clips when it sees an object:

```yaml
# manage.yaml
data: /scout/data
monitors:
  - name: front-door
    config: front-door.yaml
```

```yaml
# front-door.yaml
url: "rtsp://user:pass@192.168.1.20:554/stream1"
maxSourceFps: 8
maxOutputFps: 2
record: record.yaml
```

```yaml
# record.yaml
recordObjects: true
maxPreSec: 4
timeoutSec: 10
maxSec: 120
deleteAfterHours: 24
deleteAfterGB: 2
```

Complete examples are in the repository:

- [Minimal example](https://github.com/jonoton/scout/tree/master/example/config/min)
- [Full example](https://github.com/jonoton/scout/tree/master/example/config/full) (every option)

{: .note }
The examples use `data: /scout/data`, the path inside the Docker container. When running from source, change it to a folder on your machine.

## Where Scout looks for `.config`

1. Next to the Scout executable (in Docker: `/scout/.config`).
2. In the source checkout at `$GOPATH/src/github.com/jonoton/scout/.config`, when running from source with Go installed.
3. For everything except `http.yaml`: `./.config` or `../.config` relative to the folder you start Scout from.

Scout stops at startup with `Required config file ... not found` if it can't find `manage.yaml`.

## Applying changes

- **Camera files and the files they reference:** saved changes are picked up automatically. Scout restarts that camera within a second or so.
- **`manage.yaml`, `http.yaml`, `notify-sender.yaml`:** restart Scout (`docker restart scout`).

## Check your setup

Open `http://<server-ip>:8080` (or the `port` from `http.yaml`). The web client shows each camera's live view, its input and output frame rate, and the server's RAM use. If a camera is missing, see [Troubleshooting]({{ site.baseurl }}{% link TROUBLESHOOTING.md %}).
