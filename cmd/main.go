// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (C) 2023 Intel Corporation

// main is the main package of the application
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/opiproject/gospdk/spdk"
	fe "github.com/opiproject/opi-intel-bridge/pkg/frontend"
	me "github.com/opiproject/opi-intel-bridge/pkg/middleend"
	"github.com/opiproject/opi-smbios-bridge/pkg/inventory"
	"github.com/opiproject/opi-spdk-bridge/pkg/backend"
	"github.com/opiproject/opi-spdk-bridge/pkg/frontend"
	"github.com/opiproject/opi-spdk-bridge/pkg/utils"
	"github.com/opiproject/opi-strongswan-bridge/pkg/ipsec"

	pc "github.com/opiproject/opi-api/common/v1/gen/go"
	ps "github.com/opiproject/opi-api/security/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"

	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/gomap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

func main() {
	var grpcPort int
	flag.IntVar(&grpcPort, "grpc_port", 50051, "The gRPC server port")

	var httpPort int
	flag.IntVar(&httpPort, "http_port", 8082, "The HTTP server port")

	var spdkAddress string
	flag.StringVar(&spdkAddress, "spdk_addr", "/var/tmp/spdk.sock", "Points to SPDK unix socket/tcp socket to interact with")

	var tlsFiles string
	flag.StringVar(&tlsFiles, "tls", "", "TLS files in server_cert:server_key:ca_cert format.")

	flag.Parse()

	// Create KV store for persistence
	options := gomap.DefaultOptions
	options.Codec = utils.ProtoCodec{}
	// TODO: we can change to redis or badger at any given time
	store := gomap.NewStore(options)
	defer func(store gokv.Store) {
		err := store.Close()
		if err != nil {
			log.Panic(err)
		}
	}(store)

	go runGatewayServer(grpcPort, httpPort)
	runGrpcServer(grpcPort, spdkAddress, tlsFiles, store)
}

func runGrpcServer(grpcPort int, spdkAddress string, tlsFiles string, store gokv.Store) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var serverOptions []grpc.ServerOption
	if tlsFiles == "" {
		log.Println("TLS files are not specified. Use insecure connection.")
	} else {
		log.Println("Use TLS certificate files:", tlsFiles)
		config, err := utils.ParseTLSFiles(tlsFiles)
		if err != nil {
			log.Fatal("Failed to parse string with tls paths:", err)
		}
		log.Println("TLS config:", config)
		var option grpc.ServerOption
		if option, err = utils.SetupTLSCredentials(config); err != nil {
			log.Fatal("Failed to setup TLS:", err)
		}
		serverOptions = append(serverOptions, option)
	}
	s := grpc.NewServer(serverOptions...)

	jsonRPC := spdk.NewSpdkJSONRPC(spdkAddress)
	frontendOpiIntelServer := fe.NewServer(jsonRPC, store)
	frontendOpiSpdkServer := frontend.NewServer(jsonRPC, store)
	backendOpiSpdkServer := backend.NewServer(jsonRPC, store)
	middleendOpiIntelServer := me.NewServer(jsonRPC, store)

	pb.RegisterFrontendNvmeServiceServer(s, frontendOpiIntelServer)
	pb.RegisterFrontendVirtioBlkServiceServer(s, frontendOpiIntelServer)
	pb.RegisterFrontendVirtioScsiServiceServer(s, frontendOpiSpdkServer)
	pb.RegisterNvmeRemoteControllerServiceServer(s, backendOpiSpdkServer)
	pb.RegisterNullVolumeServiceServer(s, backendOpiSpdkServer)
	pb.RegisterAioVolumeServiceServer(s, backendOpiSpdkServer)
	pb.RegisterMiddleendEncryptionServiceServer(s, middleendOpiIntelServer)
	pb.RegisterMiddleendQosVolumeServiceServer(s, middleendOpiIntelServer)
	pc.RegisterInventorySvcServer(s, &inventory.Server{})
	ps.RegisterIPsecServer(s, &ipsec.Server{})

	reflection.Register(s)

	log.Printf("gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func runGatewayServer(grpcPort int, httpPort int) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := pc.RegisterInventorySvcHandlerFromEndpoint(ctx, mux, fmt.Sprintf(":%d", grpcPort), opts)
	if err != nil {
		log.Panic("cannot register handler server")
	}

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	log.Printf("HTTP Server listening at %v", httpPort)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", httpPort),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	err = server.ListenAndServe()
	if err != nil {
		log.Panic("cannot start HTTP gateway server")
	}
}
