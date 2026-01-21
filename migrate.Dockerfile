FROM alpine:3.19

RUN apk add --no-cache curl \
    && curl -fsSL -o /usr/local/bin/goose https://github.com/pressly/goose/releases/download/v3.24.1/goose_linux_x86_64 \
    && chmod +x /usr/local/bin/goose

ENTRYPOINT ["goose"]
