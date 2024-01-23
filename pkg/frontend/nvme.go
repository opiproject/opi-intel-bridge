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
	"github.com/opiproject/opi-spdk-bridge/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type nvmeNpiTransport struct {
	rpc spdk.JSONRPC
}

// build time check that struct implements interface
var _ frontend.NvmeTransport = (*nvmeNpiTransport)(nil)

// NewNvmeNpiTransport creates a new instance of a NvmeTransport for npi
func NewNvmeNpiTransport(rpc spdk.JSONRPC) frontend.NvmeTransport {
	if rpc == nil {
		log.Panicf("rpc cannot be nil")
	}

	return &nvmeNpiTransport{
		rpc: rpc,
	}
}

func (c *nvmeNpiTransport) CreateController(
	ctx context.Context,
	ctrlr *pb.NvmeController,
	subsys *pb.NvmeSubsystem,
) error {
	if ctrlr.GetSpec().GetPcieId().GetPortId().GetValue() != 0 {
		return status.Error(codes.InvalidArgument, "only port 0 is supported")
	}

	if ctrlr.GetSpec().GetPcieId().GetPhysicalFunction().GetValue() != 0 {
		return status.Error(codes.InvalidArgument,
			"only physical_function 0 is supported")
	}

	if subsys.GetSpec().GetHostnqn() != "" {
		return status.Error(codes.InvalidArgument,
			"hostnqn for subsystem is not supported for npi")
	}

	maxNsq := ctrlr.GetSpec().GetMaxNsq()
	maxNcq := ctrlr.GetSpec().GetMaxNcq()
	if maxNsq != maxNcq {
		return status.Error(codes.InvalidArgument,
			"max_nsq and max_ncq must be equal")
	}

	params := c.params(ctrlr, subsys)
	if maxNsq > 0 {
		params.MaxQPairs = int(maxNsq) + 1 // + 1 admin queue
	}
	var result spdk.NvmfSubsystemAddListenerResult
	err := c.rpc.Call(ctx, "nvmf_subsystem_add_listener", &params, &result)
	if err != nil {
		return status.Error(codes.Unknown, err.Error())
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not create CTRL: %s", ctrlr.Name)
		return status.Errorf(codes.InvalidArgument, msg)
	}

	return nil
}

func (c *nvmeNpiTransport) DeleteController(
	ctx context.Context,
	ctrlr *pb.NvmeController,
	subsys *pb.NvmeSubsystem,
) error {
	params := c.params(ctrlr, subsys)
	var result spdk.NvmfSubsystemAddListenerResult
	err := c.rpc.Call(ctx, "nvmf_subsystem_remove_listener", &params, &result)
	if err != nil {
		return err
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not delete CTRL: %s", ctrlr.Name)
		return status.Errorf(codes.InvalidArgument, msg)
	}

	return nil
}

func (c *nvmeNpiTransport) params(
	ctrlr *pb.NvmeController,
	subsys *pb.NvmeSubsystem,
) models.NpiNvmfSubsystemAddListenerParams {
	result := models.NpiNvmfSubsystemAddListenerParams{}
	result.Nqn = subsys.GetSpec().GetNqn()
	result.ListenAddress.Trtype = "npi"
	result.ListenAddress.Traddr = calculateTransportAddr(ctrlr.GetSpec().GetPcieId())

	return result
}

func calculateTransportAddr(pci *pb.PciEndpoint) string {
	return strconv.Itoa(int(pci.PhysicalFunction.Value)) +
		"." + strconv.Itoa(int(pci.VirtualFunction.Value))
}

// CreateNvmeController creates an Nvme controller
func (s *Server) CreateNvmeController(ctx context.Context, in *pb.CreateNvmeControllerRequest) (*pb.NvmeController, error) {
	if err := s.verifyNvmeControllerOnCreate(in.NvmeController); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	log.Printf("Passing request to opi-spdk-bridge")
	response, err := s.FrontendNvmeServiceServer.CreateNvmeController(ctx, in)
	if err == nil && in.GetNvmeController().GetSpec().GetTrtype() == pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE {
		// response contains different QoS limits. It is an indication that
		// opi-spdk-bridge returned an already existing controller providing idempotence
		if !proto.Equal(response.Spec.MaxLimit, in.NvmeController.Spec.MaxLimit) ||
			!proto.Equal(response.Spec.MinLimit, in.NvmeController.Spec.MinLimit) {
			log.Printf("Existing NvmeController %v has different QoS limits",
				in.NvmeController)
			return nil, status.Errorf(codes.AlreadyExists,
				"Controller %v exists with different QoS limits", in.NvmeController.Name)
		}

		if qosErr := s.setNvmeQosLimit(ctx, in.NvmeController); qosErr != nil {
			s.cleanupNvmeControllerCreation(in.NvmeController.Name)
			return nil, qosErr
		}
	}

	return response, err
}

// UpdateNvmeController updates an Nvme controller
func (s *Server) UpdateNvmeController(ctx context.Context, in *pb.UpdateNvmeControllerRequest) (*pb.NvmeController, error) {
	if err := s.verifyNvmeControllerOnUpdate(in.NvmeController); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	originalNvmeController := s.nvme.Controllers[in.NvmeController.Name]
	log.Printf("Passing request to opi-spdk-bridge")
	response, err := s.FrontendNvmeServiceServer.UpdateNvmeController(ctx, in)
	if err == nil && in.GetNvmeController().GetSpec().GetTrtype() == pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE {
		if qosErr := s.setNvmeQosLimit(ctx, in.NvmeController); qosErr != nil {
			log.Println("Failed to set qos settings:", qosErr)
			log.Println("Restore original controller")
			s.nvme.Controllers[in.NvmeController.Name] = originalNvmeController
			return nil, qosErr
		}
	}
	return response, err
}

func (s *Server) verifyNvmeControllerOnCreate(controller *pb.NvmeController) error {
	return s.verifyNvmeController(controller)
}

func (s *Server) verifyNvmeControllerOnUpdate(controller *pb.NvmeController) error {
	if err := s.verifyNvmeController(controller); err != nil {
		return err
	}

	// Name had to be assigned on create
	if controller.Name == "" {
		return fmt.Errorf("name cannot be empty on update")
	}
	return nil
}

func (s *Server) verifyNvmeController(controller *pb.NvmeController) error {
	maxLimit := controller.Spec.MaxLimit
	if err := s.verifyNvmeControllerMaxLimits(maxLimit); err != nil {
		return err
	}

	minLimit := controller.Spec.MinLimit
	if err := s.verifyNvmeControllerMinLimits(minLimit); err != nil {
		return err
	}

	return s.verifyNvmeControllerMinMaxLimitCorrespondence(minLimit, maxLimit)
}

func (s *Server) verifyNvmeControllerMaxLimits(maxLimit *pb.QosLimit) error {
	if maxLimit != nil {
		if maxLimit.RwIopsKiops != 0 {
			return fmt.Errorf("QoS max_limit rw_iops_kiops is not supported")
		}
		if maxLimit.RwBandwidthMbs != 0 {
			return fmt.Errorf("QoS max_limit rw_bandwidth_mbs is not supported")
		}

		if maxLimit.RdIopsKiops < 0 {
			return fmt.Errorf("QoS max_limit rd_iops_kiops cannot be negative")
		}
		if maxLimit.WrIopsKiops < 0 {
			return fmt.Errorf("QoS max_limit wr_iops_kiops cannot be negative")
		}
		if maxLimit.RdBandwidthMbs < 0 {
			return fmt.Errorf("QoS max_limit rd_bandwidth_mbs cannot be negative")
		}
		if maxLimit.WrBandwidthMbs < 0 {
			return fmt.Errorf("QoS max_limit wr_bandwidth_mbs cannot be negative")
		}
	}
	return nil
}

func (s *Server) verifyNvmeControllerMinLimits(minLimit *pb.QosLimit) error {
	if minLimit != nil {
		if minLimit.RwIopsKiops != 0 {
			return fmt.Errorf("QoS min_limit rw_iops_kiops is not supported")
		}
		if minLimit.RwBandwidthMbs != 0 {
			return fmt.Errorf("QoS min_limit rw_bandwidth_mbs is not supported")
		}
		if minLimit.RdIopsKiops != 0 {
			return fmt.Errorf("QoS min_limit rd_iops_kiops is not supported")
		}
		if minLimit.WrIopsKiops != 0 {
			return fmt.Errorf("QoS min_limit wr_iops_kiops is not supported")
		}

		if minLimit.RdBandwidthMbs < 0 {
			return fmt.Errorf("QoS min_limit rd_bandwidth_mbs cannot be negative")
		}
		if minLimit.WrBandwidthMbs < 0 {
			return fmt.Errorf("QoS min_limit wr_bandwidth_mbs cannot be negative")
		}
	}
	return nil
}

func (s *Server) verifyNvmeControllerMinMaxLimitCorrespondence(minLimit *pb.QosLimit, maxLimit *pb.QosLimit) error {
	if minLimit != nil && maxLimit != nil {
		if maxLimit.RdBandwidthMbs != 0 && minLimit.RdBandwidthMbs > maxLimit.RdBandwidthMbs {
			return fmt.Errorf("QoS min_limit rd_bandwidth_mbs cannot be greater than max_limit rd_bandwidth_mbs")
		}
		if maxLimit.WrBandwidthMbs != 0 && minLimit.WrBandwidthMbs > maxLimit.WrBandwidthMbs {
			return fmt.Errorf("QoS min_limit wr_bandwidth_mbs cannot be greater than max_limit wr_bandwidth_mbs")
		}
	}
	return nil
}

func (s *Server) setNvmeQosLimit(ctx context.Context, controller *pb.NvmeController) error {
	log.Printf("Setting QoS limits %v for %v", controller.Spec.MaxLimit, controller.Name)
	subsysName := utils.ResourceIDToSubsystemName(
		utils.GetSubsystemIDFromNvmeName(controller.Name),
	)
	params := models.NpiQosBwIopsLimitParams{
		Nqn: s.nvme.Subsystems[subsysName].Spec.Nqn,
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
	err := s.rpc.Call(ctx, "npi_qos_bw_iops_limit", &params, &result)
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

func (s *Server) cleanupNvmeControllerCreation(id string) {
	log.Println("Cleanup failed Nvme controller creation for", id)
	_, err := s.FrontendNvmeServiceServer.DeleteNvmeController(context.TODO(),
		&pb.DeleteNvmeControllerRequest{Name: id})
	log.Println("Cleanup Nvme controller creation:", err)
}
