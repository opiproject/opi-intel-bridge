// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2023 Intel Corporation

// Package frontend implements the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"context"
	"fmt"
	"log"
	"strconv"

	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/models"
	"github.com/ulule/deepcopier"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	mevTransportType = "npi"
)

// CreateNVMeController creates an NVMe Controller
func (s *Server) CreateNVMeController(ctx context.Context, in *pb.CreateNVMeControllerRequest) (*pb.NVMeController, error) {
	log.Printf("CreateNVMeController: Received from client: %v", in)

	err := verifyCreateNvmeControllerRequestArgs(in)
	if err != nil {
		return nil, err
	}

	subsys, ok := s.nvme.Subsystems[in.NvMeController.Spec.SubsystemId.Value]
	if !ok {
		msg := fmt.Sprintf("unable to find subsystem with id %v", in.NvMeController.Spec.SubsystemId.Value)
		log.Printf("error: %v", msg)
		return nil, status.Error(codes.FailedPrecondition, msg)
	}
	log.Printf("Found subsystem: %v", subsys.Spec.Nqn)

	controllerID := in.NvMeController.Spec.Id.Value
	_, ok = s.nvme.Controllers[controllerID]
	if ok {
		msg := fmt.Sprintf("Already existing controller with id %v", controllerID)
		log.Printf("error: %v", msg)
		return nil, status.Error(codes.AlreadyExists, msg)
	}

	params := models.NvmfSubsystemAddListenerParams{}
	params.Nqn = subsys.Spec.Nqn
	params.ListenAddress.Trtype = mevTransportType
	params.ListenAddress.Traddr = calculateTransportAddr(in.NvMeController.Spec.PcieId)

	var result models.NvmfSubsystemAddListenerResult
	err = s.rpc.Call("nvmf_subsystem_add_listener", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, status.Errorf(codes.FailedPrecondition, "failed to execute SPDK call")
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("could not create NVMe controller: %v", controllerID)
		log.Print(msg)
		return nil, status.Errorf(codes.FailedPrecondition, msg)
	}

	in.NvMeController.Spec.NvmeControllerId = -1
	in.NvMeController.Status = &pb.NVMeControllerStatus{Active: true}
	s.nvme.Controllers[controllerID] = in.NvMeController

	response := &pb.NVMeController{}
	err = deepcopier.Copy(in.NvMeController).To(response)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to create response")
	}
	return response, nil
}

// DeleteNVMeController deletes an NVMe Controller
func (s *Server) DeleteNVMeController(ctx context.Context, in *pb.DeleteNVMeControllerRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteNVMeController: Received from client: %v", in)
	controller, ok := s.nvme.Controllers[in.Name]
	if !ok {
		msg := fmt.Sprintf("unable to find controller with id %v", in.Name)
		log.Printf("error: %v", msg)
		return nil, status.Error(codes.FailedPrecondition, msg)
	}

	subsys, ok := s.nvme.Subsystems[controller.Spec.SubsystemId.Value]
	if !ok {
		msg := fmt.Sprintf("unable to find subsystem with id %v for controller %v",
			controller.Spec.SubsystemId.Value,
			in.Name)
		log.Printf("error: %v", msg)
		return nil, status.Error(codes.FailedPrecondition, msg)
	}

	params := models.NvmfSubsystemAddListenerParams{}
	params.Nqn = subsys.Spec.Nqn
	params.ListenAddress.Trtype = mevTransportType
	params.ListenAddress.Traddr = calculateTransportAddr(controller.Spec.PcieId)
	var result models.NvmfSubsystemAddListenerResult
	err := s.rpc.Call("nvmf_subsystem_remove_listener", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, status.Errorf(codes.FailedPrecondition, "Failed to execute SPDK call")
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not delete controller with id %v on subsystem %v",
			controller.Spec.NvmeControllerId,
			subsys.Spec.Nqn)
		log.Print(msg)
		return nil, status.Errorf(codes.FailedPrecondition, msg)
	}

	delete(s.nvme.Controllers, controller.Spec.Id.Value)
	return &emptypb.Empty{}, nil
}

// UpdateNVMeController updates an NVMe Controller
func (s *Server) UpdateNVMeController(ctx context.Context, in *pb.UpdateNVMeControllerRequest) (*pb.NVMeController, error) {
	log.Printf("UpdateNVMeController: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "UpdateNVMeController method is not implemented")
}

func verifyCreateNvmeControllerRequestArgs(in *pb.CreateNVMeControllerRequest) error {
	var err error
	switch {
	case in.NvMeController == nil:
		err = status.Error(codes.InvalidArgument, "NvMeController field should be specified")
	case in.NvMeController.Spec == nil:
		err = status.Error(codes.InvalidArgument, "Spec field should be specified")
	case in.NvMeController.Spec.Id == nil || in.NvMeController.Spec.Id.Value == "":
		err = status.Error(codes.InvalidArgument, "ControllerId field should be specified")
	case in.NvMeController.Spec.SubsystemId == nil || in.NvMeController.Spec.SubsystemId.Value == "":
		err = status.Error(codes.InvalidArgument, "SubsystemId field should be specified")
	case in.NvMeController.Spec.PcieId == nil:
		err = status.Error(codes.InvalidArgument, "PcieId field should be specified")
	}

	if err != nil {
		log.Printf("error: %v", err)
	}

	return err
}

func calculateTransportAddr(pci *pb.PciEndpoint) string {
	return strconv.Itoa(int(pci.PhysicalFunction)) +
		"." + strconv.Itoa(int(pci.VirtualFunction))
}
