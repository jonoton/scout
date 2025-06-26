# Dockerfile
# Build:
#   docker buildx build --platform linux/amd64,linux/arm64 -t scout:latest .

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
WORKDIR /go/src/github.com/jonoton/scout
COPY . .
RUN go get -d -v ./... && GOOS=${TARGETOS} GOARCH=${TARGETARCH} go install -v ./...

# scout
FROM scout-builder-stage AS scout
ARG UNAME=user
ARG UID=1000
ARG GID=1000
RUN groupadd -g $GID -o $UNAME
RUN useradd -m -u $UID -g $GID -o -s /bin/bash $UNAME
RUN mkdir /scout && mkdir -p /scout/data && mkdir -p /scout/.config && mkdir -p /scout/.logs
RUN chown -R $UID:$GID /scout
RUN chown -R $UID:$GID /go
RUN ln -s /scout/.config /go/src/github.com/jonoton/scout/.config && ln -s /scout/.logs /go/src/github.com/jonoton/scout/.logs
USER $UNAME
ENV OPENCV_FFMPEG_LOGLEVEL=fatal
CMD ["/go/bin/scout"]
