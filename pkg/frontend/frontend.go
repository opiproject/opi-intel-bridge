// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (C) 2023 Intel Corporation

// Package frontend implements the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"log"

	"github.com/philippgille/gokv"

	"github.com/opiproject/gospdk/spdk"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/frontend"
)

// Server contains frontend related OPI services
type Server struct {
	pb.FrontendNvmeServiceServer
	pb.FrontendVirtioBlkServiceServer

	nvme  *frontend.NvmeParameters
	rpc   spdk.JSONRPC
	store gokv.Store
}

// NewServer creates initialized instance of Nvme server
func NewServer(jsonRPC spdk.JSONRPC, store gokv.Store) *Server {
	if jsonRPC == nil {
		log.Panic("nil for JSONRPC is not allowed")
	}
	if store == nil {
		log.Panic("nil for Store is not allowed")
	}
	opiSpdkServer := frontend.NewCustomizedServer(
		jsonRPC, store,
		map[pb.NvmeTransportType]frontend.NvmeTransport{
			pb.NvmeTransportType_NVME_TRANSPORT_PCIE: NewNvmeNpiTransport(),
			pb.NvmeTransportType_NVME_TRANSPORT_TCP:  frontend.NewNvmeTCPTransport(),
		},
		NewMevBlkTransport())
	return &Server{
		opiSpdkServer,
		opiSpdkServer,
		&opiSpdkServer.Nvme,
		jsonRPC,
		store,
	}
}
