---
layout: default
title: Notification Settings
parent: Configuration
nav_order: 6
---

# Notification Settings

Scout uses two configuration files for notifications.

1.  **`notify-sender.yaml`**: This filename is fixed and **cannot** be customized.
2.  **`notify-rx.yaml`**: This is the recommended filename for recipient configuration, but it can be customized per monitor using the `notifyRx` field in the [Monitor Config](MONITOR).

## Outgoing Mail (`notify-sender.yaml`)

This file configures the SMTP server used to send emails and SMS alerts. The filename `notify-sender.yaml` is required and must reside in the same directory as your other config files.

| Field | Type | Req. | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `host` | string | **Yes** | - | SMTP server hostname (e.g., `smtp.gmail.com`). |
| `port` | int | **Yes** | - | SMTP server port (e.g., `587` for TLS). |
| `user` | string | **Yes** | - | SMTP account username. |
| `password` | string | **Yes** | - | SMTP account password or app password. |

## Recipient List (`notify-rx.yaml`)

This file defines who receives alerts. It supports both direct email and carrier-based SMS (via email-to-text gateways). While `notify-rx.yaml` is the standard name, you can specify a different filename for each monitor in [Monitor Config](MONITOR).

### Email Recipients

Add a list of email addresses under the `email` key.

```yaml
email:
  - "user@example.com"
```

### SMS Recipients

SMS is handled via email-to-text gateways provided by carriers. List phone numbers under their respective carrier key.

| Carrier | Key |
| :--- | :--- |
| Verizon | `verizon` |
| AT&T | `att` |
| T-Mobile | `tmobile` |
| Sprint | `sprint` |

**Example:**

```yaml
sms:
  verizon:
    - "1234567890"
  att:
    - "0987654321"
```
