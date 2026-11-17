---
layout: default
title: Android
parent: Installation
nav_order: 2
description: "Connect the Scout mobile app to your server."
---

# Android setup

1.  **Download and Install**: Get Scout from [Google Play](https://play.google.com/store/apps/details?id=com.jonoton.scout) or the [Amazon Appstore](https://www.amazon.com/gp/mas/dl/android?p=com.jonoton.scout). Scout Lite, a live-view-only version, is also available ([Google Play](https://play.google.com/store/apps/details?id=com.jonoton.scout.lite), [Amazon](https://www.amazon.com/gp/mas/dl/android?p=com.jonoton.scout.lite)).
2.  **Open Scout**: Launch the application on your device.
3.  **Accept EULA**: Read the [End User License Agreement](https://www.jonoton-innovation.com/scout/setup/mobile/eula) and accept to proceed.
4.  **Configure Server**:
    *   Navigate to **Settings** in the Menu.
    *   Fill in the **Home** and **Away** server fields (see below). Enter the hostname or IP address, plus `:port` if needed (for example `192.168.1.5:8080`), without `http://` or `https://`.
    *   Turn on **Secure** for an address that uses HTTPS.
    *   Select **Save**.
5.  **Login (If required)**:
    *   Navigate to **Login** in the Menu.
    *   Enter your Username and Password.
    *   Select **Login**.
6.  **Verify**: Check for the "Connected" status in the Menu header.
7.  **Enjoy!**: You can now view your monitors and manage alerts.

## Home and Away

The app keeps two server addresses, each with its own **Secure** (HTTPS) checkbox:

| Setting | Use | Example | Secure |
| :--- | :--- | :--- | :--- |
| **Home** | On your home Wi-Fi | The server's local IP and port, `192.168.1.5:8080` | Off |
| **Away** | Anywhere else | Your domain, `scout.example.com` | On |

Add the port unless it's the standard one (80 for HTTP, 443 for HTTPS). Scout listens on `8080` by default, so a local address usually needs `:8080`. The Away address needs remote access set up first, ideally a reverse proxy with HTTPS. See [Remote access]({{ site.baseurl }}{% link RECIPES.md %}#remote-access-with-a-reverse-proxy).

The **Login** step is only needed if [logins]({{ site.baseurl }}{% link config/HTTP.md %}#logins) are set up on the server.
