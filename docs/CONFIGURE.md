---
layout: default
title: Configuration
nav_order: 3
---

# Configure Scout

Scout is configured using YAML files located in the `.config` directory. 

## Config Directory Location

Scout looks for the `.config` directory in the following order:
1.  The same directory as the Scout executable.
2.  `$GOPATH/src/jonoton/scout` directory.

## Detailed Configuration Guides

For a complete reference of all available options, please see the following guides:

*   **[Manage Config (`manage.yaml`)](config/MANAGE.md)**: Data storage and monitor lists.
*   **[Web Server Config (`http.yaml`)](config/HTTP.md)**: Ports, security, and remote server links.
*   **[Monitor Config (`cam.yaml`)](config/MONITOR.md)**: Specific camera settings and source URLs.
*   **[Detection Pipeline](config/DETECTION.md)**: Motion, Object, and Face detection settings.
*   **[Recording & Alerts](config/RECORDING_ALERTS.md)**: Recording rules and notification settings.

## Configuration Examples

While the guides above list every option, you can also look at complete examples:
*   **[Full Configuration Example](https://github.com/jonoton/scout/tree/master/example/config/full)**
*   **[Minimum Configuration Example](https://github.com/jonoton/scout/tree/master/example/config/min)**

## Verify via Web Client

Once configured and running, you can verify your setup using the built-in web client:

1.  Open your browser to the Scout server address (e.g., `http://localhost:8080`).
2.  **Hostname Options**:
    *   `localhost` (if running on the same machine).
    *   Router assigned IP (e.g., `192.168.1.5`).
    *   External domain name (if configured).
3.  **Port**: The port is configured in `http.yaml` (default is `8080`).

The web client provides quick visibility into:
*   RAM usage
*   Frame Rate for all monitors
*   Live view of all monitors
