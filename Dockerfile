# Dockerfile
# Build:
#   docker buildx build --platform linux/amd64,linux/arm64 -t scout:latest .
# Run in foreground and remove after:
#   docker run -it --rm -p 8080:8080 -v HOST_CONFIG_PATH:/scout/.config -v HOST_LOGS_PATH:/scout/.logs -v HOST_DATA_PATH:/scout/data scout:latest

# golang amd64
FROM ubuntu:20.04 AS golang-amd64-stage
LABEL maintainer="jonotoninnovation"
ENV DEBIAN_FRONTEND="noninteractive"
ENV TZ="America/New_York"
RUN apt update && apt install -y sudo git wget curl build-essential cmake gcc g++ pkg-config unzip tzdata
RUN apt purge -y golang
RUN mkdir /Downloads
RUN wget -c https://go.dev/dl/go1.23.5.linux-amd64.tar.gz -O - | tar -xz -C /Downloads
ENV GOROOT="/Downloads/go"
ENV PATH=$PATH:$GOROOT/bin
RUN which go && go version
ENV GOPATH=/go
ENV GO111MODULE=on
RUN mkdir -p "$GOPATH/src"
WORKDIR /go/src

# golang arm64
FROM ubuntu:20.04 AS golang-arm64-stage
LABEL maintainer="jonotoninnovation"
ENV DEBIAN_FRONTEND="noninteractive"
ENV TZ="America/New_York"
RUN apt update && apt install -y sudo git wget curl build-essential cmake gcc g++ pkg-config unzip tzdata
RUN apt purge -y golang
RUN mkdir /Downloads
RUN wget -c https://go.dev/dl/go1.23.5.linux-arm64.tar.gz -O - | tar -xz -C /Downloads
ENV GOROOT="/Downloads/go"
ENV PATH=$PATH:$GOROOT/bin
RUN which go && go version
ENV GOPATH=/go
ENV GO111MODULE=on
RUN mkdir -p "$GOPATH/src"
WORKDIR /go/src

# gocv
FROM golang-${TARGETARCH}${TARGETVARIANT}-stage AS gocv-stage
RUN export WANT_GENERAL_DEBS="sudo git wget curl build-essential cmake gcc g++ pkg-config unzip tzdata" &&\
    export WANT_OPENCV_DEBS="unzip wget build-essential cmake curl git libgtk2.0-dev pkg-config libavcodec-dev libavformat-dev libswscale-dev libtbb-dev libjpeg-dev libpng-dev libtiff-dev libdc1394-dev libharfbuzz-dev libfreetype-dev" &&\
    export WANT_OPENCV_DEBS_EXTRA="libavutil-dev libv4l-dev libswresample-dev libgstreamer-plugins-base1.0-dev libgstreamer1.0-dev libxvidcore-dev libx264-dev libgtk-3-dev libopenexr-dev libwebp-dev libatlas-base-dev gfortran" && \
    apt update && apt install -y $WANT_GENERAL_DEBS $WANT_OPENCV_DEBS $WANT_OPENCV_DEBS_EXTRA
RUN mkdir -p $GOPATH/pkg/mod/gocv.io/x/gocv@v0.37.0
RUN git clone --depth 1 --branch v0.37.0 https://github.com/hybridgroup/gocv.git $GOPATH/pkg/mod/gocv.io/x/gocv@v0.37.0
RUN cd $GOPATH/pkg/mod/gocv.io/x/gocv@v0.37.0 && make install
RUN cd $GOPATH/pkg/mod/gocv.io/x/gocv@v0.37.0 && go install -v .

# scout builder
FROM --platform=$BUILDPLATFORM gocv-stage AS scout-builder-stage
ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION_ARG=""
WORKDIR /go/src/github.com/jonoton/scout
COPY . .
RUN go get -d -v ./... && GOOS=${TARGETOS} GOARCH=${TARGETARCH} go install -v -ldflags "-X main.Version=${VERSION_ARG}" ./...

# scout
FROM ubuntu:20.04 AS scout
LABEL maintainer="jonotoninnovation"
ENV DEBIAN_FRONTEND="noninteractive"
ENV TZ="America/New_York"

RUN apt update && apt install -y \
    tzdata \
    ca-certificates \
    libgtk2.0-0 \
    libavcodec58 \
    libavformat58 \
    libswscale5 \
    libtbb2 \
    libjpeg8 \
    libpng16-16 \
    libtiff5 \
    libdc1394-22 \
    libharfbuzz0b \
    libfreetype6 \
    libavutil56 \
    libv4l-0 \
    libswresample3 \
    libgstreamer-plugins-base1.0-0 \
    libgstreamer1.0-0 \
    libxvidcore4 \
    libx264-155 \
    libgtk-3-0 \
    libopenexr24 \
    libwebp6 \
    libatlas3-base \
    && rm -rf /var/lib/apt/lists/*

COPY --from=scout-builder-stage /go/bin/scout /scout/scout
COPY --from=scout-builder-stage /usr/local/lib/ /usr/local/lib/
COPY --from=scout-builder-stage /go/src/github.com/jonoton/scout/http/public /scout/http/public
COPY --from=scout-builder-stage /go/src/github.com/jonoton/scout/http/templates /scout/http/templates
COPY --from=scout-builder-stage /go/src/github.com/jonoton/scout/data /scout/.data
RUN ldconfig

ARG UNAME=user
ARG UID=1000
ARG GID=1000
RUN groupadd -g $GID -o $UNAME
RUN useradd -m -u $UID -g $GID -o -s /bin/bash $UNAME
RUN mkdir -p /scout/data && mkdir -p /scout/.config && mkdir -p /scout/.logs
RUN chown -R $UID:$GID /scout

USER $UNAME
ENV PATH=$PATH:/scout
ENV OPENCV_FFMPEG_LOGLEVEL=fatal
EXPOSE 8080

VOLUME ["/scout/.config", "/scout/.logs", "/scout/data"]

WORKDIR /scout
CMD ["/scout/scout"]
