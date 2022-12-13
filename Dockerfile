FROM golang:1.20rc1-bullseye as build-env

ENV GO111MODULE=on
RUN apk --no-cache add git

COPY . /app
WORKDIR /app

RUN ls -lahR && go mod download && go build -o /vnc-recorder

FROM linuxserver/ffmpeg:version-4.4-cli
COPY --from=build-env /vnc-recorder /
ENTRYPOINT ["/vnc-recorder"]
CMD [""]
