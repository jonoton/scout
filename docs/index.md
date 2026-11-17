---
layout: default
title: Home
nav_order: 1
description: "Scout is self-hosted video security with motion, object, and face detection. No cloud, no monthly fees."
permalink: /
---

<div class="hero">
  <img class="hero-hawk" src="{{ '/assets/images/hawk-wing.png' | relative_url }}" alt="">
  <p class="hero-eyebrow">Open source · Self-hosted</p>
  <h1 class="hero-title">Scout</h1>
  <p class="hero-tagline">Your cameras. Your network. Your storage. Video security with motion, object, and face detection, and no cloud required.</p>
  <div class="hero-buttons">
    <a href="{{ site.baseurl }}{% link server/DOCKER.md %}" class="btn btn-primary">Get started</a>
    <a href="https://github.com/jonoton/scout" class="btn btn-ghost">GitHub</a>
  </div>
</div>

Scout connects to your IP cameras, watches for people, cars, and other objects, and records clips and sends alerts when it finds them. It runs on a computer you own, saves everything to your own disk, and has apps for Android, iOS, and the web.

## Why Scout

| | Typical cloud camera service | Scout |
| :--- | :--- | :--- |
| Monthly fee | Often required for recording history | None |
| Where video is stored | The provider's servers | Your own disk |
| Works without internet | Usually not | Yes, on your local network |
| Cameras | The provider's own | Any camera with an RTSP or HTTP stream |
| Source code | Closed | Open source ([MIT](https://github.com/jonoton/scout/blob/master/LICENSE)) |
{: .compare }

<div class="feature-grid">
  <div class="feature-card">
    <span class="feature-kicker">Detection</span>
    <p class="feature-title">Fewer false alarms</p>
    <p>Motion, then object, then face detection. An object only counts if it's moving, so spider webs, shadows, and parked cars don't set off alerts.</p>
  </div>
  <div class="feature-card">
    <span class="feature-kicker">Recording</span>
    <p class="feature-title">Clips and 24/7</p>
    <p>Record when an object is seen, including a few seconds from before it appeared, or record around the clock. Old files are deleted automatically.</p>
  </div>
  <div class="feature-card">
    <span class="feature-kicker">Alerts</span>
    <p class="feature-title">Email and text</p>
    <p>Get the best snapshots by email or text, with limits so a busy afternoon doesn't flood your inbox.</p>
  </div>
  <div class="feature-card">
    <span class="feature-kicker">Simple</span>
    <p class="feature-title">Files, not a database</p>
    <p>Settings are YAML files and recordings are plain video files. Edit a camera's file and Scout reloads it.</p>
  </div>
  <div class="feature-card">
    <span class="feature-kicker">Secure</span>
    <p class="feature-title">Logins and two-factor</p>
    <p>Optional user accounts with hashed passwords and passcodes sent by email or text.</p>
  </div>
  <div class="feature-card">
    <span class="feature-kicker">Scales</span>
    <p class="feature-title">Link servers</p>
    <p>Run Scout on several machines and see every camera from one server.</p>
  </div>
</div>

## Quick start

With Docker and one camera's stream URL, the [Docker guide]({{ site.baseurl }}{% link server/DOCKER.md %}) gets you from nothing to recording in four steps. The core of it:

```bash
docker run -d --name scout --restart unless-stopped \
  -p 8080:8080 -e TZ=America/Chicago \
  -v ~/scout/.config:/scout/.config \
  -v ~/scout/.logs:/scout/.logs \
  -v ~/scout/data:/scout/data \
  jonotoninnovation/scout
```

Then open `http://<server-ip>:8080`.

## Explore the docs

- **[Installation]({{ site.baseurl }}{% link INSTALL.md %})**: server options and the mobile apps.
- **[Configuration]({{ site.baseurl }}{% link CONFIGURE.md %})**: how the YAML files fit together, with a reference for every option.
- **[Usage]({{ site.baseurl }}{% link USAGE.md %})**: the web client, mobile apps, and command-line options.
- **[Troubleshooting]({{ site.baseurl }}{% link TROUBLESHOOTING.md %})**: fixes for cameras that don't appear, missing recordings, and more.
- **[Recipes]({{ site.baseurl }}{% link RECIPES.md %})**: remote access through a reverse proxy, and GPU acceleration.
- **[Development]({{ site.baseurl }}{% link DEVELOPMENT.md %})**: build from source, profile, and work on these docs.

## Get the apps

[Google Play](https://play.google.com/store/apps/details?id=com.jonoton.scout) · [App Store](https://apps.apple.com/us/app/scout-full/id1658406270) · [Amazon Appstore](https://www.amazon.com/gp/mas/dl/android?p=com.jonoton.scout) · [All apps and setup]({{ site.baseurl }}{% link INSTALL.md %}#mobile-apps)
