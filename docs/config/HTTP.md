---
layout: default
title: HTTP
parent: Configuration
nav_order: 2
description: "http.yaml: port, logins, two-factor passcodes, and linked servers."
---

# HTTP (`http.yaml`)
{: .no_toc }

`http.yaml` configures the web server that the web client and mobile apps connect to. It's optional: without it, Scout serves on port `8080` with no login. The filename is fixed and the file must be in the `.config` folder. Restart Scout after changing it.

1. TOC
{:toc}

## Server

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `port` | int | `8080` | Port to listen on. |
| `limitPerSecond` | int | `100` | Most requests accepted per second. |
| `users` | list | none | Accounts allowed to sign in. **If empty, no login is required.** See [Logins](#logins). |
| `signInExpireDays` | int | `7` | Days a sign-in stays valid. |
| `loginLimitPerSecond` | int | `10` | Most login attempts accepted per second. |
| `twoFactorTimeoutSec` | int | `60` | Seconds a two-factor passcode stays valid. |
| `loginSigningKey` | string | random | Secret used to sign login tokens. If blank, a new one is generated on every start, which signs everyone out when Scout restarts. Set it to a long random string to keep people signed in. |
| `links` | list | none | Other Scout servers to show cameras from. See [Linked servers](#linked-servers). |
| `linkRetry` | int | `2` | Retries when a linked server doesn't respond. |
| `enableSwagger` | bool | `false` | Serve interactive API docs at `/swagger/`. |

## Logins

{: .warning }
Without `users`, anyone who can reach the server can watch your cameras. Add at least one user before making Scout reachable from outside your network.

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `user` | string | **Yes** | Username. |
| `password` | string | **Yes** | Password. Can be stored as a hash (see below). |
| `twoFactor` | object | No | Where to send a passcode after the password is accepted: `email` (list) and/or `sms` (carrier lists, as in [Notifications]({{ site.baseurl }}{% link config/NOTIFICATIONS.md %}#recipients-notify-rxyaml)). |

{: .important }
The web client keeps you signed in with a browser cookie that only works over HTTPS, or on `localhost`. Over plain HTTP on your network (for example `http://192.168.1.5:8080`), signing in to the web client won't stick. Put Scout behind a [reverse proxy with HTTPS]({{ site.baseurl }}{% link RECIPES.md %}#remote-access-with-a-reverse-proxy) if you use logins with the web client.

With `twoFactor` set, Scout sends a 6-character passcode (letters and numbers) that expires after `twoFactorTimeoutSec`. Passcodes are sent with the account in [`notify-sender.yaml`]({{ site.baseurl }}{% link config/NOTIFICATIONS.md %}#sender-notify-senderyaml), so that must be set up too.

```yaml
port: 8080
signInExpireDays: 7
loginSigningKey: "replace-with-a-long-random-string"
users:
  - user: admin
    password: "a-strong-password"
    twoFactor:
      email:
        - admin@example.com
  - user: family
    password: "another-strong-password"
    twoFactor:
      sms:
        tmobile:
          - "5551234567"
```

### Hashing passwords

Passwords start out as plain text. To replace them with hashes, run Scout once with `--secure-http-passwords`:

```bash
# Docker (uses the same .config folder as your running container)
docker run --rm -v ~/scout/.config:/scout/.config jonotoninnovation/scout \
  /scout/scout --secure-http-passwords

# From source
./scout --secure-http-passwords
```

User passwords become bcrypt hashes and link passwords become SHA-256 hashes. Passwords that are already hashed are left alone. Then restart Scout.

{: .note }
This rewrites `http.yaml`, so comments and formatting in the file are lost. Keep a copy if you want them.

### Login status endpoint

Apps and integrations can check whether a login is required with `GET /auth/status` (or `GET /login`):

```json
{"required": true}
```

## Linked servers

Linking lets one Scout server show the cameras of other Scout servers, so the apps only need to connect to one.

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `url` | string | **Yes** | Base URL of the other server, for example `http://192.168.1.10:8080`. |
| `name` | string | No | Prefix for the linked server's cameras. With `name: garage`, a camera called `door` there appears as `garage-door`. |
| `user` | string | No | Username on the other server, if it requires a login. |
| `password` | string | No | That user's password, in plain text or as the SHA-256 hash written by `--secure-http-passwords`. |

```yaml
links:
  - name: garage
    url: http://192.168.1.10:8080
    user: admin
    password: "garage-server-password"
```

{: .note }
Linked servers can't use two-factor logins. Use an account without `twoFactor` on the other server for linking.
