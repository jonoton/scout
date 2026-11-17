---
layout: default
title: Installation
nav_order: 2
has_children: true
description: "Install the Scout server and the mobile apps."
---

# Installation

Scout has two parts: the **server**, which runs on a machine on your network and does the recording and detection, and the **apps** you watch from.

## Server

| Option | Best for |
| :--- | :--- |
| **[Docker]({{ site.baseurl }}{% link server/DOCKER.md %})** (recommended) | Most people. OpenCV and the detection models are included, so it's the quickest way to get running. |
| **[From source]({{ site.baseurl }}{% link DEVELOPMENT.md %})** | Developers, or platforms without Docker. You build OpenCV yourself. |

The server needs a machine that stays on, on the same network as your cameras. Object detection is the heaviest part. Lowering `maxOutputFps` and detection's `scaleWidth` reduces CPU use, and an NVIDIA GPU can take over detection (see [Recipes]({{ site.baseurl }}{% link RECIPES.md %}#nvidia-gpu-acceleration)).

## Web client

Built into the server. Open `http://<server-ip>:8080` in a browser; there's nothing to install. See [Usage]({{ site.baseurl }}{% link USAGE.md %}#web-client).

## Mobile apps

The Scout apps add alerts, recordings, continuous recordings, and notifications on top of live view. Scout Lite (Android and Amazon) is live view only.

| Platform | Download | Setup |
| :--- | :--- | :--- |
| Android | [Scout](https://play.google.com/store/apps/details?id=com.jonoton.scout) · [Scout Lite](https://play.google.com/store/apps/details?id=com.jonoton.scout.lite) | [Android setup]({{ site.baseurl }}{% link mobile/ANDROID.md %}) |
| Amazon (Fire) | [Scout](https://www.amazon.com/gp/mas/dl/android?p=com.jonoton.scout) · [Scout Lite](https://www.amazon.com/gp/mas/dl/android?p=com.jonoton.scout.lite) | [Android setup]({{ site.baseurl }}{% link mobile/ANDROID.md %}) |
| iOS | [Scout](https://apps.apple.com/us/app/scout-full/id1658406270) | [iOS setup]({{ site.baseurl }}{% link mobile/IOS.md %}) |

## After installing

1. [Configure Scout]({{ site.baseurl }}{% link CONFIGURE.md %}) with your cameras.
2. Set up [logins]({{ site.baseurl }}{% link config/HTTP.md %}#logins) before allowing access from outside your network.
3. Optional: [email and text alerts]({{ site.baseurl }}{% link config/NOTIFICATIONS.md %}).
