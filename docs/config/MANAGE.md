---
layout: default
title: Manage
parent: Configuration
nav_order: 1
description: "manage.yaml: the data folder and the list of cameras."
---

# Manage (`manage.yaml`)

`manage.yaml` is the only file Scout requires. It sets where saved files go and lists your cameras. The filename is fixed and the file must be in the `.config` folder.

| Field | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `data` | string | Yes, to save anything | none | Folder for recordings, continuous recordings, and alert images. Scout creates `recordings/`, `continuous/`, and `alerts/` inside it. In Docker use `/scout/data`. |
| `monitors` | list | Yes | none | Your cameras. See below. |

{: .warning }
If `data` is missing or empty, Scout still shows live video, but recording, continuous recording, and alerts are silently turned off.

## Monitor entries

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `name` | string | Yes | Unique camera name, shown in the apps and used as the prefix of saved file names (for example `front-door`). |
| `config` | string | Yes | The camera's [monitor config]({{ site.baseurl }}{% link config/MONITOR.md %}) file, relative to `.config` or an absolute path. |

## Example

```yaml
data: /scout/data
monitors:
  - name: front-door
    config: front-door.yaml
  - name: driveway
    config: driveway.yaml
```

Changes to `manage.yaml` take effect after restarting Scout.
