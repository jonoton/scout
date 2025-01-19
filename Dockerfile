# Dockerfile
# Build:
#   docker buildx build --platform linux/amd64,linux/arm64 -t scout:latest .

# gocv amd64
FROM ubuntu:20.04 AS gocv-amd64
LABEL maintainer="jonotoninnovation"
ENV DEBIAN_FRONTEND="noninteractive"
ENV TZ="America/New_York"
RUN apt update && apt install -y sudo git wget build-essential tzdata
RUN apt purge -y golang
RUN mkdir /Downloads
RUN wget -c https://go.dev/dl/go1.22.5.linux-amd64.tar.gz -O - | tar -xz -C /Downloads
ENV GOROOT="/Downloads/go"
ENV PATH=$PATH:$GOROOT/bin
RUN which go && go version
ENV GOPATH=/go
ENV GO111MODULE=on
RUN mkdir -p "$GOPATH/src"
WORKDIR /go/src
RUN mkdir -p $GOPATH/pkg/mod/gocv.io/x/gocv@v0.37.0
RUN git clone --depth 1 --branch v0.37.0 https://github.com/hybridgroup/gocv.git $GOPATH/pkg/mod/gocv.io/x/gocv@v0.37.0
RUN cd $GOPATH/pkg/mod/gocv.io/x/gocv@v0.37.0 && make install
RUN cd $GOPATH/pkg/mod/gocv.io/x/gocv@v0.37.0 && go install -v .

# gocv arm64
FROM ubuntu:20.04 AS gocv-arm64
LABEL maintainer="jonotoninnovation"
ENV DEBIAN_FRONTEND="noninteractive"
ENV TZ="America/New_York"
RUN apt update && apt install -y sudo git wget build-essential tzdata
RUN apt purge -y golang
RUN mkdir /Downloads
RUN wget -c https://go.dev/dl/go1.22.5.linux-arm64.tar.gz -O - | tar -xz -C /Downloads
ENV GOROOT="/Downloads/go"
ENV PATH=$PATH:$GOROOT/bin
RUN which go && go version
ENV GOPATH=/go
ENV GO111MODULE=on
RUN mkdir -p "$GOPATH/src"
WORKDIR /go/src
RUN mkdir -p $GOPATH/pkg/mod/gocv.io/x/gocv@v0.37.0
RUN git clone --depth 1 --branch v0.37.0 https://github.com/hybridgroup/gocv.git $GOPATH/pkg/mod/gocv.io/x/gocv@v0.37.0
RUN cd $GOPATH/pkg/mod/gocv.io/x/gocv@v0.37.0 && make install
RUN cd $GOPATH/pkg/mod/gocv.io/x/gocv@v0.37.0 && go install -v .

# scout
FROM gocv-${TARGETARCH}${TARGETVARIANT} AS scout
LABEL maintainer="jonotoninnovation"
ARG UNAME=user
ARG UID=1000
ARG GID=1000
RUN groupadd -g $GID -o $UNAME
RUN useradd -m -u $UID -g $GID -o -s /bin/bash $UNAME
RUN mkdir /scout && chown $UID:$GID /scout
WORKDIR /go/src/github.com/jonoton/scout
COPY . .
RUN chown -R $UID:$GID /go
USER $UNAME
RUN mkdir -p /scout/data && mkdir -p /scout/.config && mkdir -p /scout/.logs
RUN ln -s /scout/.config /go/src/github.com/jonoton/scout/.config && ln -s /scout/.logs /go/src/github.com/jonoton/scout/.logs
RUN go get -d -v ./... && go install -v ./...
CMD ["/go/bin/scout"]
