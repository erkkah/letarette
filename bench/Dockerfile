FROM golang:1.14-alpine as builder

RUN apk update && apk add --no-cache libc-dev tzdata git bash

WORKDIR /go/src/app
COPY . .

ENV GOSUMDB=off
RUN go build ./cmd/tinysrv

