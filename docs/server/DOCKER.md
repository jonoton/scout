---
layout: default
title: Docker
parent: Installation
nav_order: 1
---

# Setup with Docker (Recommended)

Using Docker is the recommended way to run Scout as it handles all dependencies for you.

## Prerequisites

1.  Install [Docker](https://www.docker.com/get-started/) on your server.

## Quick Start

1.  **Pull the image**:
    ```bash
    docker pull jonotoninnovation/scout
    ```
2.  **Run Scout**:
    ```bash
    docker run -it --rm \
      -p 8080:8080 \
      -v $(pwd)/.config:/scout/.config \
      -v $(pwd)/.logs:/scout/.logs \
      -v $(pwd)/data:/scout/data \
      jonotoninnovation/scout
    ```

## Custom Build

If you prefer to build the image yourself:

1.  Clone the repository:
    ```bash
    git clone https://github.com/jonoton/scout.git
    cd scout
    ```
2.  Build the image:
    ```bash
    docker build -t scout .
    ```

## Next Steps

1.  **[Configure Scout](../CONFIGURE)**: Edit the files in your mapping `.config` directory.
2.  **[Verify Setup](../CONFIGURE#verify-via-web-client)**: Open your browser to `http://localhost:8080`.
