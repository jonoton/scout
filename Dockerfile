# Scout Docker Image
# Build:
#   docker build --build-arg UID=$(id -u) --build-arg GID=$(id -g) -t scout:latest .
# Run in foreground and remove after:
#   docker run -it --rm -p 8080:8080 -v HOST_DATA_PATH:/scout/data -v HOST_CONFIG_PATH:/scout/.config scout:latest
# Run in background and keep:
#   docker run -d -p 8080:8080 -v HOST_DATA_PATH:/scout/data -v HOST_CONFIG_PATH:/scout/.config scout:latest
# Keep Host Date Time Localization
#   Add to the lines above: -v /etc/localtime:/etc/localtime:ro


# GoCV
FROM golang:latest AS gocv
LABEL maintainer="jonotoninnovation"
WORKDIR /go/src
RUN apt update && apt install -y sudo
RUN go get -u -d gocv.io/x/gocv && cd gocv.io/x/gocv && make install && go run ./cmd/version/main.go && go install gocv.io/x/gocv


# Scout
FROM gocv AS scout
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
RUN mkdir -p /scout/data && mkdir -p /scout/.config && ln -s /scout/.config /go/src/github.com/jonoton/scout/.config
RUN go get -d -v ./... && go install -v ./...
CMD /go/bin/scout
