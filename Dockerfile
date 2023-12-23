# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.

FROM docker.io/library/golang:1.21.5@sha256:672a2286da3ee7a854c3e0a56e0838918d0dbb1c18652992930293312de898a6 as builder

WORKDIR /app

# Download necessary Go modules
COPY go.mod ./
COPY go.sum ./
RUN go mod download

ENV CGO_ENABLED=0

# build an app
COPY cmd/ cmd/
COPY pkg/ pkg/
RUN go build -v -o /opi-intel-bridge ./cmd/...

# second stage to reduce image size
FROM alpine:3.19@sha256:51b67269f354137895d43f3b3d810bfacd3945438e94dc5ac55fdac340352f48
RUN apk add --no-cache --no-check-certificate hwdata && rm -rf /var/cache/apk/*
COPY --from=builder /opi-intel-bridge /
COPY --from=docker.io/fullstorydev/grpcurl:v1.8.9-alpine /bin/grpcurl /usr/local/bin/
EXPOSE 50051
CMD [ "/opi-intel-bridge", "-grpc_port=50051", "-http_port=8082" ]
HEALTHCHECK CMD grpcurl -plaintext localhost:50051 list || exit 1
