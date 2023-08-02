FROM golang:1.20.7 AS build

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

WORKDIR /app
COPY --from=build /build/mpv-web-api /app

EXPOSE 3001
ENTRYPOINT [ "mpv-web-api", "--start-mpv-instance=false" ]