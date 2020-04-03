# syntax = docker/dockerfile:experimental
FROM golang:1.13 as builder
WORKDIR /workspace
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod make build

FROM debian:latest AS base
RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt \
    apt-get update \
 && apt-get dist-upgrade -y \
 && apt-get install -y --no-install-recommends ca-certificates
RUN rm -rf /tmp/* /var/tmp/* \
 && rm -rf /var/lib/apt/lists/*
RUN update-ca-certificates
#FROM alpine:latest AS base
#RUN --mount=type=cache,target=/var/cache/apt \
#    --mount=type=cache,target=/var/lib/apt \
#    apk update \
# && apk add --no-cache openssh-client ca-certificates git

FROM base AS app
COPY --from=builder /workspace/bin/app /app
ENTRYPOINT ["/app"]
