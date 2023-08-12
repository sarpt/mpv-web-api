FROM golang:1.21.0-alpine3.18 AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY cmd cmd
COPY internal internal
COPY pkg pkg

WORKDIR /app/cmd/mpv-web-api
RUN go build -o /build/mpv-web-api

FROM alpine:latest

RUN apk update
RUN apk upgrade
RUN apk add --no-cache ffmpeg

COPY --from=build /build/mpv-web-api /usr/local/bin

EXPOSE 3001
ENTRYPOINT [ "mpv-web-api", "--start-mpv-instance=false" ]