---
layout: default
title: HTTP Config
parent: Configuration
nav_order: 2
---

# HTTP Configuration (`http.yaml`)

> ❗ **Important**
> `http.yaml` is a **required** filename if you wish to use the web server (recommended).

The `http.yaml` file configures the web server, security settings, and linking between multiple Scout servers.

| Field | Type | Req. | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `port` | int | No | `8080` | The port the web server will listen on. |
| `limitPerSecond` | int | No | `100` | Rate limit for general API requests. |
| `loginLimitPerSecond` | int | No | `10` | Rate limit specifically for login attempts. |
| `users` | list | No | - | A list of authorized users. If empty, login is disabled. |
| `signInExpireDays` | int | No | `7` | How many days a login session remains valid. |
| `links` | list | No | - | Links to other Scout servers to aggregate monitors. |
| `linkRetry` | int | No | `2` | Number of retries when connecting to linked servers. |
| `twoFactorTimeoutSec` | int | No | `60` | Timeout for receiving 2FA codes. |
| `loginSigningKey` | string | No | (Auto) | Random key used to sign tokens. Generated on every start if blank. |
| `enableSwagger` | bool | No | `false` | Whether to enable the Swagger UI at `/swagger`. |

### User Authentication (Optional)

> 💡 **Note on Requirements**
> If you add a user to the `users` list, the `user` and `password` fields below become **required**.

| Field | Type | Req. | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `user` | string | **Yes** | - | Username. |
| `password` | string | **Yes** | - | Password. |
| `twoFactor` | object | No | - | Configuration for receiving 2FA codes. |

> 🔒 **Security Tip: Securing Passwords**
> By default, passwords in `http.yaml` are stored in plaintext so they are easy to set up. You can secure them (hash them) by running Scout with the following argument:
> ```bash
> ./scout --secure-http-passwords
> ```
> This will replace all plaintext passwords in `http.yaml` with secure bcrypt hashes.

#### Two-Factor Configuration (`twoFactor`)

If `twoFactor` is configured, users will be prompted for a 6-digit passcode after providing their password.

| Field | Type | Req. | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `email` | list | No | - | List of email addresses to send codes to. |
| `sms` | object | No | - | SMS configuration grouped by carrier. |

**Supported SMS Carriers:**
*   `verizon`
*   `att`
*   `tmobile`
*   `sprint`

#### Authentication Example

```yaml
users:
  - user: "admin"
    password: "secure_password"
    twoFactor:
      email:
        - "admin@example.com"
      sms:
        att:
          - "555-0199"
```

### Server Links (Optional)

> 💡 **Note on Requirements**
> If you add a server link to the `links` list, the `url` field below becomes **required**.

| Field | Type | Req. | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `name` | string | No | - | Friendly name for the linked server. |
| `url` | string | **Yes** | - | The base URL of the remote Scout server (e.g., `http://192.168.1.10:8080`). |
| `user` | string | No | - | Username for the remote server. |
| `password` | string | No | - | Password for the remote server. |

> 🔒 **Security Tip: Securing Link Passwords**
> Similar to user passwords, link passwords can be secured using the `--secure-http-passwords` argument. When run, it will convert link passwords to secure hashes.
