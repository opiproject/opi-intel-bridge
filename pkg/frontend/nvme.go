// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2023 Intel Corporation

// Package frontend implements the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/opiproject/gospdk/spdk"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-intel-bridge/pkg/models"
	"github.com/opiproject/opi-spdk-bridge/pkg/frontend"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type npiSubsystemListener struct {
}

// NewSubsystemListener creates a new instance of a SubsystemListener for npi transport
func NewSubsystemListener() frontend.SubsystemListener {
	return npiSubsystemListener{}
}

func (c npiSubsystemListener) Params(ctrlr *pb.NVMeController, nqn string) spdk.NvmfSubsystemAddListenerParams {
	result := spdk.NvmfSubsystemAddListenerParams{}
	result.Nqn = nqn
	result.ListenAddress.Trtype = "npi"
	result.ListenAddress.Traddr = calculateTransportAddr(ctrlr.Spec.PcieId)
	return result
}

func calculateTransportAddr(pci *pb.PciEndpoint) string {
	return strconv.Itoa(int(pci.PhysicalFunction)) +
		"." + strconv.Itoa(int(pci.VirtualFunction))
}

// CreateNVMeController creates an NVMe controller
func (s *Server) CreateNVMeController(ctx context.Context, in *pb.CreateNVMeControllerRequest) (*pb.NVMeController, error) {
	log.Printf("Intel bridge received from client: %v", in.NvMeController)
	if err := s.verifyNVMeController(in.NvMeController); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// opi-spdk-bridge checks only IDs and returns success if a controller
	// with such id exists ignoring any parameters. Check a controller
	// exists with the same max/min limits
	controller, ok := s.nvme.Controllers[in.NvMeController.Spec.Id.Value]
	if ok {
		if proto.Equal(controller.Spec.MaxLimit, in.NvMeController.Spec.MaxLimit) &&
			proto.Equal(controller.Spec.MinLimit, in.NvMeController.Spec.MinLimit) {
			log.Printf("Already existing NVMeController with id %v", in.NvMeController.Spec.Id.Value)
			return controller, nil
		}
		log.Printf("Existing NVMeController %v has different QoS limits",
			in.NvMeController)
		return nil, status.Errorf(codes.AlreadyExists,
			"Controller %v exists with different QoS limits", in.NvMeController.Spec.Id.Value)
	}

	log.Printf("Passing request to opi-spdk-bridge")
	response, err := s.FrontendNvmeServiceServer.CreateNVMeController(ctx, in)

	if err == nil {
		if qosErr := s.setNVMeQosLimit(in.NvMeController); qosErr != nil {
			s.cleanupNVMeControllerCreation(in.NvMeController.Spec.Id.Value)
			return nil, qosErr
		}
	}
	return response, err
}

func (s *Server) verifyNVMeController(controller *pb.NVMeController) error {
	maxLimit := controller.Spec.MaxLimit
	if err := s.verifyNVMeControllerMaxLimits(maxLimit); err != nil {
		return err
	}

	minLimit := controller.Spec.MinLimit
	if err := s.verifyNVMeControllerMinLimits(minLimit); err != nil {
		return err
	}

	return s.verifyNVMeControllerMinMaxLimitCorrespondence(minLimit, maxLimit)
}

func (s *Server) verifyNVMeControllerMaxLimits(maxLimit *pb.QosLimit) error {
	if maxLimit != nil {
		if maxLimit.RwIopsKiops != 0 {
			return fmt.Errorf("QoS limit_max rw_iops_kiops is not supported")
		}
		if maxLimit.RwBandwidthMbs != 0 {
			return fmt.Errorf("QoS limit_max rw_bandwidth_mbs is not supported")
		}

		if maxLimit.RdIopsKiops < 0 {
			return fmt.Errorf("QoS limit_max rd_iops_kiops cannot be negative")
		}
		if maxLimit.WrIopsKiops < 0 {
			return fmt.Errorf("QoS limit_max wr_iops_kiops cannot be negative")
		}
		if maxLimit.RdBandwidthMbs < 0 {
			return fmt.Errorf("QoS limit_max rd_bandwidth_mbs cannot be negative")
		}
		if maxLimit.WrBandwidthMbs < 0 {
			return fmt.Errorf("QoS limit_max wr_bandwidth_mbs cannot be negative")
		}
	}
	return nil
}

func (s *Server) verifyNVMeControllerMinLimits(minLimit *pb.QosLimit) error {
	if minLimit != nil {
		if minLimit.RwIopsKiops != 0 {
			return fmt.Errorf("QoS limit_min rw_iops_kiops is not supported")
		}
		if minLimit.RwBandwidthMbs != 0 {
			return fmt.Errorf("QoS limit_min rw_bandwidth_mbs is not supported")
		}
		if minLimit.RdIopsKiops != 0 {
			return fmt.Errorf("QoS limit_min rd_iops_kiops is not supported")
		}
		if minLimit.WrIopsKiops != 0 {
			return fmt.Errorf("QoS limit_min wr_iops_kiops is not supported")
		}

		if minLimit.RdBandwidthMbs < 0 {
			return fmt.Errorf("QoS limit_min rd_bandwidth_mbs cannot be negative")
		}
		if minLimit.WrBandwidthMbs < 0 {
			return fmt.Errorf("QoS limit_min wr_bandwidth_mbs cannot be negative")
		}
	}
	return nil
}

func (s *Server) verifyNVMeControllerMinMaxLimitCorrespondence(minLimit *pb.QosLimit, maxLimit *pb.QosLimit) error {
	if minLimit != nil && maxLimit != nil {
		if minLimit.RdBandwidthMbs > maxLimit.RdBandwidthMbs {
			return fmt.Errorf("QoS limit_min rd_bandwidth_mbs cannot be greater than limit_max rd_bandwidth_mbs")
		}
		if minLimit.WrBandwidthMbs > maxLimit.WrBandwidthMbs {
			return fmt.Errorf("QoS limit_min wr_bandwidth_mbs cannot be greater than limit_max wr_bandwidth_mbs")
		}
	}
	return nil
}

func (s *Server) setNVMeQosLimit(controller *pb.NVMeController) error {
	log.Printf("Setting QoS limits %v for %v", controller.Spec.MaxLimit, controller.Spec.Id.Value)
	params := models.NpiQosBwIopsLimitParams{
		Nqn: s.nvme.Subsystems[controller.Spec.SubsystemId.Value].Spec.Nqn,
	}

	maxLimit := controller.Spec.MaxLimit
	if maxLimit != nil {
		params.MaxReadIops = int(maxLimit.RdIopsKiops)
		params.MaxWriteIops = int(maxLimit.WrIopsKiops)
		params.MaxReadBw = int(maxLimit.RdBandwidthMbs)
		params.MaxWriteBw = int(maxLimit.WrBandwidthMbs)
	}

	minLimit := controller.Spec.MinLimit
	if minLimit != nil {
		params.MinReadBw = int(minLimit.RdBandwidthMbs)
		params.MinWriteBw = int(minLimit.WrBandwidthMbs)
	}

	var result models.NpiQosBwIopsLimitResult
	err := s.rpc.Call("npi_qos_bw_iops_limit", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return spdk.ErrFailedSpdkCall
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		log.Println("Could not set QoS for", controller)
		return spdk.ErrUnexpectedSpdkCallResult
	}
	return nil
}

func (s *Server) cleanupNVMeControllerCreation(id string) {
	log.Println("Cleanup failed NVMe controller creation for", id)
	_, err := s.FrontendNvmeServiceServer.DeleteNVMeController(context.TODO(),
		&pb.DeleteNVMeControllerRequest{Name: id})
	log.Println("Cleanup NVMe controller creation:", err)
}
