---
layout: default
title: Recipes
nav_order: 6
description: "Example setups: remote access through a reverse proxy, and GPU acceleration."
---

# Recipes
{: .no_toc }

Example setups for running Scout alongside other services. The files are in [`example/recipes`](https://github.com/jonoton/scout/tree/master/example/recipes); adjust paths and versions to your system.
{: .fs-5 .fw-300 }

1. TOC
{:toc}

## Remote access with a reverse proxy

To use the mobile apps away from home, put Scout behind a reverse proxy that adds HTTPS, rather than exposing port 8080 directly.

{: .warning }
Set up [logins]({{ site.baseurl }}{% link config/HTTP.md %}#logins), ideally with two-factor passcodes, before Scout is reachable from the internet.

1. **Forward ports.** On your router, forward ports 80 and 443 to the machine running the proxy.
2. **Run a reverse proxy** such as [Nginx Proxy Manager](https://nginxproxymanager.com/), Caddy, or Traefik. Route your domain to `http://<scout-ip>:8080` and request a Let's Encrypt certificate.
3. **Allow WebSockets.** Live video uses WebSocket connections on `/live/<camera>`. In Nginx Proxy Manager, turn on **Websockets Support** for the host; in plain Nginx, pass the `Upgrade` and `Connection` headers.
4. **Keep your domain pointed home (optional).** If your home IP address changes, run a dynamic DNS client such as [ddclient](https://ddclient.net/).

[`docker-compose.rpi.reverse-proxy.yml`](https://github.com/jonoton/scout/blob/master/example/recipes/docker-compose.rpi.reverse-proxy.yml) runs Nginx Proxy Manager (admin UI on port 81), its MariaDB database, and ddclient on a Raspberry Pi.

## NVIDIA GPU acceleration

Object and face detection use CUDA automatically when Scout is built with it and a GPU is available. [`docker-compose.cuda10_2-amd64.scout.yml`](https://github.com/jonoton/scout/blob/master/example/recipes/docker-compose.cuda10_2-amd64.scout.yml) shows the setup: a CUDA build of the image and `runtime: nvidia` (requires the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html)). Add `TZ` to it as in the [Docker guide]({{ site.baseurl }}{% link server/DOCKER.md %}#docker-compose) to get local timestamps.

{: .note }
The image tag in that file is an older CUDA build. Check [Docker Hub](https://hub.docker.com/r/jonotoninnovation/scout/tags) for current tags.

To force the CPU for one camera, set `forceCpu: true` in its [`tensor` or `face` file]({{ site.baseurl }}{% link config/DETECTION.md %}).
