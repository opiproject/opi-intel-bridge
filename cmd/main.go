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
	"github.com/opiproject/opi-spdk-bridge/pkg/server"
	"github.com/opiproject/opi-strongswan-bridge/pkg/ipsec"

	pc "github.com/opiproject/opi-api/common/v1/gen/go"
	ps "github.com/opiproject/opi-api/security/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 50051, "The Server port")
	var spdkSocket string
	flag.StringVar(&spdkSocket, "rpc_sock", "/var/tmp/spdk.sock", "Path to SPDK JSON RPC socket")
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	jsonRPC := server.NewUnixSocketJSONRPC(spdkSocket)
	frontendServer := frontend.NewServer(jsonRPC)
	backendServer := backend.NewServer(jsonRPC)
	middleendServer := middleend.NewServer(jsonRPC)

	pb.RegisterFrontendNvmeServiceServer(s, fe.NewServer(jsonRPC))
	pb.RegisterFrontendVirtioBlkServiceServer(s, frontendServer)
	pb.RegisterFrontendVirtioScsiServiceServer(s, frontendServer)
	pb.RegisterNVMfRemoteControllerServiceServer(s, backendServer)
	pb.RegisterNullDebugServiceServer(s, backendServer)
	pb.RegisterAioControllerServiceServer(s, backendServer)
	pb.RegisterMiddleendServiceServer(s, middleendServer)
	pc.RegisterInventorySvcServer(s, &inventory.Server{})
	ps.RegisterIPsecServer(s, &ipsec.Server{})

	reflection.Register(s)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
