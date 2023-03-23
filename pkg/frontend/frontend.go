// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (C) 2023 Intel Corporation

// Package frontend implements the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/frontend"
	"github.com/opiproject/opi-spdk-bridge/pkg/server"
)

// Server contains frontend related OPI services
type Server struct {
	pb.FrontendNvmeServiceServer
}

// NewServer creates initialized instance of NVMe server
func NewServer(jsonRPC server.JSONRPC) *Server {
	opiSpdkServer := frontend.NewServerWithSubsystemListener(jsonRPC, NewSubsystemListener())
	return &Server{
		opiSpdkServer,
	}
}
