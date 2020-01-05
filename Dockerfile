FROM golang:1.13-alpine as builder

RUN apk update && apk add --no-cache sqlite-dev make gcc libc-dev tzdata git perl bash
RUN adduser -D -g '' letarette

WORKDIR /go/src/app
COPY . .
RUN make

FROM scratch

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /bin/sh /bin/sh
COPY --from=builder /bin/mkdir /bin/mkdir
COPY --from=builder /bin/chown /bin/chown
COPY --from=builder /lib/ld-musl-x86_64.so.1 /lib/ld-musl-x86_64.so.1

COPY --from=builder /go/src/app/letarette /letarette
COPY --from=builder /go/src/app/lrcli /lrcli
COPY --from=builder /go/src/app/lrload /lrload

RUN mkdir /db && chown letarette /db

USER letarette
ENV LETARETTE_DB_PATH=/db/letarette.db
ENV LETARETTE_NATS_URLS=nats://natsserver:4222
VOLUME [ "/db" ]

CMD [ "/letarette" ]
