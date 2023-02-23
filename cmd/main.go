// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Dell Inc, or its subsidiaries.

// main is the main package of the application
package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	fe "github.com/opiproject/opi-intel-bridge/pkg/frontend"
	"github.com/opiproject/opi-smbios-bridge/pkg/inventory"
	"github.com/opiproject/opi-spdk-bridge/pkg/backend"
	"github.com/opiproject/opi-spdk-bridge/pkg/frontend"
	"github.com/opiproject/opi-spdk-bridge/pkg/middleend"
	"github.com/opiproject/opi-strongswan-bridge/pkg/ipsec"

	pc "github.com/opiproject/opi-api/common/v1/gen/go"
	ps "github.com/opiproject/opi-api/security/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	pb.RegisterFrontendNvmeServiceServer(s, &fe.PluginFrontendNvme)
	pb.RegisterFrontendVirtioBlkServiceServer(s, &frontend.Server{})
	pb.RegisterFrontendVirtioScsiServiceServer(s, &frontend.Server{})
	pb.RegisterNVMfRemoteControllerServiceServer(s, &backend.Server{})
	pb.RegisterNullDebugServiceServer(s, &backend.Server{})
	pb.RegisterAioControllerServiceServer(s, &backend.Server{})
	pb.RegisterMiddleendServiceServer(s, &middleend.Server{})
	pc.RegisterInventorySvcServer(s, &inventory.Server{})
	ps.RegisterIPsecServer(s, &ipsec.Server{})
	reflection.Register(s)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
