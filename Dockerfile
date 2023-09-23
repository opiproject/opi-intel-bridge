# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.

FROM docker.io/library/golang:1.21.1@sha256:c416ceeec1cdf037b80baef1ccb402c230ab83a9134b34c0902c542eb4539c82 as builder

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
FROM alpine:3.18@sha256:7144f7bab3d4c2648d7e59409f15ec52a18006a128c733fcff20d3a4a54ba44a
RUN apk add --no-cache --no-check-certificate hwdata && rm -rf /var/cache/apk/*
COPY --from=builder /opi-intel-bridge /
COPY --from=docker.io/fullstorydev/grpcurl:v1.8.7-alpine /bin/grpcurl /usr/local/bin/
EXPOSE 50051
CMD [ "/opi-intel-bridge", "-grpc_port=50051", "-http_port=8082" ]
HEALTHCHECK CMD grpcurl -plaintext localhost:50051 list || exit 1
