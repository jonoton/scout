---
layout: default
title: Development
nav_order: 6
---

# Development Guide

Information for developers looking to build, test, and profile Scout.

## Setup from Source

Building from source allows you to run the latest development version and customize the build.

### Prerequisites

1.  Install **Git** and **Golang**.

### Step-by-Step Installation

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/jonoton/scout.git
    cd scout
    ```
2.  **Download Dependencies**:
    ```bash
    go mod download
    ```
3.  **Build GoCV**:
    Scout depends on GoCV. You may need to install the GoCV dependencies on your system:
    ```bash
    cd $(go env GOPATH)/pkg/mod/gocv.io/x/gocv@v0.37.0
    sudo make install # or install_cuda for GPU support
    cd -
    ```
4.  **Install Scout**:
    ```bash
    go install ./...
    ```

### Running Scout

1.  Ensure your `.config` directory is populated.
2.  Run the executable from your `$GOPATH/bin` or the project root:
    ```bash
    scout
    ```

## Profiling with GoLang

Scout supports profiling using the standard Go `pprof` tool.

### Prerequisites

1.  **Graphviz**: Required for generating visual profile graphs.
    ```bash
    sudo apt install graphviz
    ```

### How to Profile

1.  **Run Scout with profiling enabled**:
    ```bash
    go run -tags profile github.com/jonoton/scout
    ```
    This starts a pprof HTTP server on `localhost:6060`.

2.  **Capture and View Profiles**:
    *   **CPU Profile**:
        ```bash
        go tool pprof -http localhost:8081 http://localhost:6060
        ```
    *   **Memory Profile**:
        ```bash
        go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/heap
        ```
    *   **GoCV Mat Profile**:
        ```bash
        go run -tags matprofile github.com/jonoton/scout
        go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/gocv.io/x/gocv.Mat
        ```
    *   **SharedMat Profile**:
        ```bash
        go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/github.com/jonoton/go-sharedmat.counts
        ```

3.  **Navigate the Web UI**: The `View -> Flame Graph` is highly recommended for identifying bottlenecks.
