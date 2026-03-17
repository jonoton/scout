---
layout: default
title: Manage Config
parent: Configuration
nav_order: 1
---

# Manage Configuration (`manage.yaml`)

> ❗ **Important**
> `manage.yaml` is a **required** filename and must exist in the `.config` directory.

The `manage.yaml` file is the master configuration that tells Scout where to store data and which monitors to load.

| Field | Type | Req. | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `data` | string | No | `./data` | The root directory where all alerts, recordings, and logs will be saved. (Relative to the Scout executable by default). |
| `monitors` | list | **Yes** | - | A list of monitor configurations. |

### Monitor Entry (Required)

| Field | Type | Req. | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `name` | string | **Yes** | - | A unique name for the monitor (e.g., `front_door`). |
| `config` | string | **Yes** | - | Path to the specific [Monitor Configuration](MONITOR) file (e.g., `cam1.yaml`) relative to `.config/`. |
