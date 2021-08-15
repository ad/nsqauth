FROM golang:1.16.6-alpine as builder

ENV USER=appuser
ENV UID=10001

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

WORKDIR $GOPATH/src/app/
COPY ./go.mod ./go.sum ./
COPY ./vendor ./vendor
RUN go install ./vendor/...

COPY ./clickhouse ./clickhouse
COPY ./main.go ./main.go

ARG VER

#RUN go build -mod=vendor -ldflags="-X main.version=$VER -w -s" -a -o /go/bin/nsqauth .
RUN go build -ldflags="-X main.version=$VER" -o /go/bin/nsqauth .

FROM scratch
EXPOSE 7755
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /go/bin/nsqauth /go/bin/nsqauth

USER appuser:appuser

ENTRYPOINT ["/go/bin/nsqauth"]
