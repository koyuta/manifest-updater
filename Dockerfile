# syntax = docker/dockerfile:experimental
ARG base="alpine:3.11.5"

FROM $base AS base
RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt \
    apk update \
 && apk add --no-cache openssh-client ca-certificates nmap-ncat

FROM golang:1.13 as builder
WORKDIR /workspace
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod make build

FROM base AS app
COPY --from=builder /workspace/bin/app /app
ENTRYPOINT ["/app"]
