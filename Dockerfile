FROM golang:1.24-alpine AS builder

RUN apk update && apk add --no-cache make gcc libc-dev tzdata git bash
RUN adduser -D -g '' letarette

WORKDIR /go/src/app
COPY . .

ENV GOSUMDB=off
ENV STATIC=1
RUN go generate

FROM scratch

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /bin/sh /bin/sh
COPY --from=builder /bin/mkdir /bin/mkdir
COPY --from=builder /bin/chown /bin/chown
COPY --from=builder /lib/ld-musl* /lib/

COPY --from=builder /go/src/app/letarette /letarette
COPY --from=builder /go/src/app/lrcli /lrcli
COPY --from=builder /go/src/app/lrload /lrload
COPY --from=builder /go/src/app/lrmon /lrmon
COPY --from=builder /go/src/app/tinysrv /tinysrv

RUN mkdir /db && chown letarette /db

USER letarette
ENV LETARETTE_DB_PATH=/db/letarette.db
ENV LETARETTE_NATS_URLS=nats://natsserver:4222
VOLUME [ "/db" ]

CMD [ "/letarette" ]
