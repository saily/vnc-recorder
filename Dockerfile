FROM golang:alpine as build-env
LABEL maintainer="daniel@widerin.net"

ENV GO111MODULE=on
RUN apk --no-cache add git

COPY . /app
WORKDIR /app

RUN ls -lahR && go mod download && go build -o /vnc-recorder

FROM jrottenberg/ffmpeg:4.1-alpine
COPY --from=build-env /vnc-recorder /
ENTRYPOINT ["/vnc-recorder"]
CMD [""]
