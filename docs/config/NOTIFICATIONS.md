---
layout: default
title: Notifications
parent: Configuration
nav_order: 6
description: "Send alerts and login passcodes by email or text message."
---

# Notifications

Scout sends alerts, and two-factor login passcodes, through an email account you provide. Text messages are sent through your mobile carrier's email-to-text gateway. Two files are involved:

| File | Name fixed? | Purpose |
| :--- | :--- | :--- |
| `notify-sender.yaml` | Yes | The SMTP account Scout sends from. One per server. |
| `notify-rx.yaml` | No | Who receives a camera's alerts. Set with the camera's `notifyRx` field. |

Alerts are only sent for cameras that also have an [`alert` file]({{ site.baseurl }}{% link config/RECORDING_ALERTS.md %}#alerts).

## Sender (`notify-sender.yaml`)

Must be in the `.config` folder. Restart Scout after changing it.

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `host` | string | **Yes** | SMTP server, for example `smtp.gmail.com`. |
| `port` | int | **Yes** | `587` (STARTTLS) or `465` (SSL). |
| `user` | string | **Yes** | SMTP username, usually the email address. |
| `password` | string | **Yes** | SMTP password. For Gmail and most other providers, use an [app password](https://support.google.com/accounts/answer/185833), not your normal password. |

```yaml
host: smtp.gmail.com
port: 587
user: scout.alerts@example.com
password: "your-app-password"
```

{: .tip }
Use a separate email account for Scout. The password is stored in plain text in this file.

## Recipients (`notify-rx.yaml`)

| Field | Type | Description |
| :--- | :--- | :--- |
| `email` | list | Email addresses. |
| `sms` | object | Phone numbers grouped by carrier: `verizon`, `att`, or `tmobile`. |

```yaml
email:
  - you@example.com
sms:
  verizon:
    - "5551234567"
  tmobile:
    - "5559876543"
```

Phone numbers are 10 digits with no spaces or dashes, because they become email addresses such as `5551234567@vtext.com`.

| Key | Carrier | Gateway |
| :--- | :--- | :--- |
| `verizon` | Verizon | `@vtext.com` (images: `@vzwpix.com`) |
| `att` | AT&T | `@txt.att.net` (images: `@mms.att.net`) |
| `tmobile` | T-Mobile | `@tmomail.net` |

{: .note }
Carriers have been retiring their email-to-text gateways. If texts don't arrive, use email (most phones can notify you of new email from a specific sender) or the Scout mobile app.

Email alerts include a summary and, within the `maxSendAttachmentsPerHour` limit, the alert images. Text alerts are a short summary. Images are only sent by text if `textAttachments` is on.
