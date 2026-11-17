---
layout: default
title: Development
nav_order: 7
description: "Build Scout from source, profile it, and preview these docs."
---

# Development
{: .no_toc }

Build Scout from source, profile it, and work on these docs.
{: .fs-5 .fw-300 }

1. TOC
{:toc}

## Dev container (easiest)

The repository includes a [VS Code dev container](https://github.com/jonoton/scout/tree/master/.devcontainer) with Go, OpenCV/GoCV, and Ruby (for the docs) already installed. Open the folder in VS Code and choose **Reopen in Container**.

## Build on Linux

### Prerequisites

- Git
- Go **1.27.3** or newer (see [`go.mod`](https://github.com/jonoton/scout/blob/master/go.mod))
- OpenCV, built through GoCV **v0.43.0**. GoCV's `make install` downloads and compiles OpenCV, which needs build tools (`build-essential`, `cmake`, `pkg-config`) and FFmpeg/GTK development packages. The [`Dockerfile`](https://github.com/jonoton/scout/blob/master/Dockerfile) lists the exact Ubuntu packages.

### Steps

1. Clone the repository:

    ```bash
    git clone https://github.com/jonoton/scout.git
    cd scout
    ```

2. Install OpenCV with GoCV (this takes a while):

    ```bash
    go mod download
    cd "$(go env GOMODCACHE)/gocv.io/x/gocv@v0.43.0"
    make install        # or: make install_cuda   for NVIDIA GPU support
    cd -
    ```

3. Build Scout into the repository folder:

    ```bash
    go build -o scout .
    ```

    To set the version printed by `--version`, add `-ldflags "-X main.Version=v1.2.3"`.

### Run from source

Scout finds its web files (`http/`), detection models (`data/`), and config (`.config/`) next to the executable, so run the binary from the repository folder:

1. Create `.config/` in the repository folder and add a config. Start from [`example/config/min`](https://github.com/jonoton/scout/tree/master/example/config/min), and change `data: /scout/data` to a writable folder such as `data: ./storage`.
2. Run it:

    ```bash
    ./scout
    ```

3. Open `http://localhost:8080`.

## Profiling

Scout can serve Go's [`pprof`](https://pkg.go.dev/net/http/pprof) profiles. Viewing graphs needs [Graphviz](https://graphviz.org/) (`sudo apt install graphviz`).

1. Run Scout with the `profile` build tag. This starts a profiling server on port `6060`:

    ```bash
    go build -tags profile -o scout . && ./scout
    ```

    All available profiles are listed at `http://localhost:6060/debug/pprof/`.

2. In a second terminal, open a profile in the browser viewer:

    | Profile | Command |
    | :--- | :--- |
    | CPU (2-second sample) | `go tool pprof -http localhost:8081 "http://localhost:6060/debug/pprof/profile?seconds=2"` |
    | Memory | `go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/heap` |
    | Goroutines | `go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/goroutine` |
    | Shared frames | `go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/github.com/jonoton/go-sharedmat/sharedmat.counts` |

3. To also track OpenCV `Mat` allocations (useful for finding memory leaks), add GoCV's `matprofile` tag:

    ```bash
    go build -tags "profile matprofile" -o scout . && ./scout
    # in a second terminal:
    go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/gocv.io/x/gocv.Mat
    ```

In the viewer, **View → Flame Graph** is the quickest way to find hot spots.

## API docs

The HTTP API is documented with Swagger in [`http/docs`](https://github.com/jonoton/scout/tree/master/http/docs). Set `enableSwagger: true` in [`http.yaml`]({{ site.baseurl }}{% link config/HTTP.md %}) to browse it at `/swagger/`. To regenerate it after changing handlers:

```bash
$(go env GOPATH)/bin/swag init -o http/docs -g http/http.go --propertyStrategy pascalcase
```

## Working on these docs

The docs are a [Jekyll](https://jekyllrb.com/) site using the [Just the Docs](https://just-the-docs.com/) theme, in the [`docs`](https://github.com/jonoton/scout/tree/master/docs) folder.

```bash
cd docs
bundle install
bundle exec jekyll serve
```

Then open `http://localhost:4000`.

- Colors and fonts: `_sass/custom/setup.scss` and `_sass/color_schemes/scout.scss`
- Component styles: `_sass/custom/custom.scss`
- Callouts: put `{: .note }`, `{: .tip }`, `{: .important }`, or `{: .warning }` on the line above a paragraph.
