---
layout: default
title: Recipes
nav_order: 4
---

# Recipes

Recipes are suggested configurations to coordinate Scout with other services. They are optional and can be adapted to your specific needs.

All recipe files can be found in the **[example/recipes directory](https://github.com/jonoton/scout/tree/master/example/recipes)**.

## External Route to Scout

If you want to access your Scout server from outside your network, you can use a reverse proxy.

### Setup Steps

1.  **Port Forwarding**: Configure your router to forward HTTP (80) and HTTPS (443) traffic to your Proxy Server.
2.  **Reverse Proxy**: Configure a proxy server (like Nginx or Traefik) to route traffic for a particular domain to your Scout server.
3.  **Dynamic DNS**: (Optional) Use a dynamic DNS client to update your domain's IP address if it changes.

See the **[docker-compose.rpi.reverse-proxy.yml](https://github.com/jonoton/scout/blob/master/example/recipes/docker-compose.rpi.reverse-proxy.yml)** example for a Raspberry Pi implementation.
