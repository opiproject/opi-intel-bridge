// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2023 Intel Corporation

// Package frontend implements the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"fmt"
	"testing"

	"github.com/opiproject/gospdk/spdk"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var (
	testPciEndpoint = pb.PciEndpoint{
		PhysicalFunction: wrapperspb.Int32(0),
		VirtualFunction:  wrapperspb.Int32(2),
		PortId:           wrapperspb.Int32(0),
	}
	testSubsystemID = "subsystem-test"
	testSubsystem   = pb.NvmeSubsystem{
		Name: utils.ResourceIDToSubsystemName(testSubsystemID),
		Spec: &pb.NvmeSubsystemSpec{
			Nqn: "nqn.2022-09.io.spdk:opi3",
		},
	}
	testControllerID   = "controller-test"
	testControllerName = utils.ResourceIDToControllerName(
		testSubsystemID, testControllerID,
	)
	testControllerWithMaxQos = pb.NvmeController{
		Spec: &pb.NvmeControllerSpec{
			Endpoint: &pb.NvmeControllerSpec_PcieId{
				PcieId: &testPciEndpoint,
			},
			Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
			NvmeControllerId: proto.Int32(1),
			MaxLimit: &pb.QosLimit{
				RdIopsKiops:    1,
				WrIopsKiops:    1,
				RdBandwidthMbs: 1,
				WrBandwidthMbs: 1,
			},
		},
	}
)

func TestFrontEnd_CreateNvmeController(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in                 *pb.NvmeController
		out                *pb.NvmeController
		spdk               []string
		errCode            codes.Code
		errMsg             string
		existingController *pb.NvmeController
		hostnqn            string
	}{
		"max_limit rw_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{RwIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rw_iops_kiops is not supported",
			existingController: nil,
		},
		"max_limit rw_bandwidth_mbs is not supported": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{RwBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rw_bandwidth_mbs is not supported",
			existingController: nil,
		},
		"set qos SPDK call failed": {
			in:  &testControllerWithMaxQos,
			out: nil,
			spdk: []string{
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`,
				`{"id":%d,"error":{"code":-1,"message":"some internal error"},"result":true}`,
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:            status.Convert(spdk.ErrFailedSpdkCall).Code(),
			errMsg:             status.Convert(spdk.ErrFailedSpdkCall).Message(),
			existingController: nil,
		},
		"set qos SPDK call result false": {
			in:  &testControllerWithMaxQos,
			out: nil,
			spdk: []string{
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`,
				`{"id":%d,"error":{"code":0,"message":""},"result":false}`,
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:            status.Convert(spdk.ErrUnexpectedSpdkCallResult).Code(),
			errMsg:             status.Convert(spdk.ErrUnexpectedSpdkCallResult).Message(),
			existingController: nil,
		},
		"allowed max qos limits": {
			in: &testControllerWithMaxQos,
			out: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(-1),
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
			spdk: []string{
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`,
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:            codes.OK,
			errMsg:             "",
			existingController: nil,
		},
		"no qos limits are specified": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
				},
			},
			out: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(-1),
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
			spdk: []string{
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`,
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:            codes.OK,
			errMsg:             "",
			existingController: nil,
		},
		"controller with the same qos limits exists": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
			},
			out: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(-1),
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode: codes.OK,
			errMsg:  "",
			existingController: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(-1),
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
		},
		"controller with different max qos limits exists": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 12321, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.AlreadyExists,
			errMsg:  fmt.Sprintf("Controller %v exists with different QoS limits", testControllerName),
			existingController: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(-1),
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
		},
		"controller with different min qos limits exists": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 12321, WrBandwidthMbs: 1},
				},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.AlreadyExists,
			errMsg:  fmt.Sprintf("Controller %v exists with different QoS limits", testControllerName),
			existingController: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(-1),
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
		},
		"min_limit rw_bandwidth_mbs is not supported": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{RwBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rw_bandwidth_mbs is not supported",
			existingController: nil,
		},
		"min_limit rw_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{RwIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rw_iops_kiops is not supported",
			existingController: nil,
		},
		"min_limit rd_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{RdIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rd_iops_kiops is not supported",
			existingController: nil,
		},
		"min_limit wr_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{WrIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit wr_iops_kiops is not supported",
			existingController: nil,
		},
		"allowed min qos limits": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{},
				},
			},
			out: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(-1),
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{},
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
			spdk: []string{
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`,
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:            codes.OK,
			errMsg:             "",
			existingController: nil,
		},
		"min_limit rd_bandwidth_mbs cannot be greater than max_limit rd_bandwidth_mbs": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 2},
					MaxLimit:         &pb.QosLimit{RdBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rd_bandwidth_mbs cannot be greater than max_limit rd_bandwidth_mbs",
			existingController: nil,
		},
		"min_limit wr_bandwidth_mbs cannot be greater than max_limit wr_bandwidth_mbs": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{WrBandwidthMbs: 2},
					MaxLimit:         &pb.QosLimit{WrBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit wr_bandwidth_mbs cannot be greater than max_limit wr_bandwidth_mbs",
			existingController: nil,
		},
		"allowed min and max qos limits": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 2, WrIopsKiops: 2, RdBandwidthMbs: 2, WrBandwidthMbs: 2},
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
			},
			out: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(-1),
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 2, WrIopsKiops: 2, RdBandwidthMbs: 2, WrBandwidthMbs: 2},
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
			spdk: []string{
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`,
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:            codes.OK,
			errMsg:             "",
			existingController: nil,
		},
		"max_limit rd_iops_kiops is negative": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{RdIopsKiops: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rd_iops_kiops cannot be negative",
			existingController: nil,
		},
		"max_limit wr_iops_kiops is negative": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{WrIopsKiops: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit wr_iops_kiops cannot be negative",
			existingController: nil,
		},
		"max_limit rd_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{RdBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rd_bandwidth_mbs cannot be negative",
			existingController: nil,
		},
		"max_limit wr_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{WrBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit wr_bandwidth_mbs cannot be negative",
			existingController: nil,
		},
		"min_limit rd_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rd_bandwidth_mbs cannot be negative",
			existingController: nil,
		},
		"min_limit wr_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{WrBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit wr_bandwidth_mbs cannot be negative",
			existingController: nil,
		},
		"non 0 port id": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint: &pb.NvmeControllerSpec_PcieId{
						PcieId: &pb.PciEndpoint{
							PortId:           wrapperspb.Int32(1),
							PhysicalFunction: wrapperspb.Int32(0),
							VirtualFunction:  wrapperspb.Int32(0),
						},
					},
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "only port 0 is supported",
			existingController: nil,
		},
		"non 0 physical_function": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint: &pb.NvmeControllerSpec_PcieId{
						PcieId: &pb.PciEndpoint{
							PortId:           wrapperspb.Int32(0),
							PhysicalFunction: wrapperspb.Int32(1),
							VirtualFunction:  wrapperspb.Int32(0),
						},
					},
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "only physical_function 0 is supported",
			existingController: nil,
		},
		"non-empty hostnqn": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint: &pb.NvmeControllerSpec_PcieId{
						PcieId: &pb.PciEndpoint{
							PortId:           wrapperspb.Int32(0),
							PhysicalFunction: wrapperspb.Int32(0),
							VirtualFunction:  wrapperspb.Int32(0),
						},
					},
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "hostnqn for subsystem is not supported for npi",
			existingController: nil,
			hostnqn:            "nqn.2014-08.org.nvmexpress:uuid:feb98abe-d51f-40c8-b348-2753f3571d3c",
		},
		"valid request with empty SPDK response": {
			in:                 &testControllerWithMaxQos,
			out:                nil,
			spdk:               []string{""},
			errCode:            codes.Unknown,
			errMsg:             fmt.Sprintf("nvmf_subsystem_add_listener: %v", "EOF"),
			existingController: nil,
			hostnqn:            "",
		},
		"valid request with invalid SPDK response": {
			in:                 &testControllerWithMaxQos,
			out:                nil,
			spdk:               []string{`{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			errCode:            codes.InvalidArgument,
			errMsg:             fmt.Sprintf("Could not create CTRL: %v", testControllerName),
			existingController: nil,
			hostnqn:            "",
		},
		"non-equal max_nsq and max_ncq": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint: &pb.NvmeControllerSpec_PcieId{
						PcieId: &pb.PciEndpoint{
							PortId:           wrapperspb.Int32(0),
							PhysicalFunction: wrapperspb.Int32(0),
							VirtualFunction:  wrapperspb.Int32(0),
						},
					},
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxNsq:           17,
					MaxNcq:           18,
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "max_nsq and max_ncq must be equal",
			existingController: nil,
			hostnqn:            "",
		},
		"non-zero max_nsq and max_ncq": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint: testControllerWithMaxQos.Spec.Endpoint,
					Trtype:   pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					MaxNsq:   17,
					MaxNcq:   17,
				},
			},
			out: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(-1),
					MaxNsq:           17,
					MaxNcq:           17,
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
			spdk: []string{
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`,
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:            codes.OK,
			errMsg:             "",
			existingController: nil,
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			tt.in = utils.ProtoClone(tt.in)
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()
			testEnv.opiSpdkServer.nvme.Subsystems[testSubsystem.Name] = utils.ProtoClone(&testSubsystem)
			testEnv.opiSpdkServer.nvme.Subsystems[testSubsystem.Name].Spec.Hostnqn = tt.hostnqn
			if tt.existingController != nil {
				tt.existingController = utils.ProtoClone(tt.existingController)
				tt.existingController.Name = testControllerName
				testEnv.opiSpdkServer.nvme.Controllers[tt.existingController.Name] = tt.existingController
			}
			if tt.out != nil {
				tt.out = utils.ProtoClone(tt.out)
				tt.out.Name = testControllerName
			}

			response, err := testEnv.opiSpdkServer.CreateNvmeController(testEnv.ctx,
				&pb.CreateNvmeControllerRequest{
					Parent:           testSubsystem.Name,
					NvmeController:   tt.in,
					NvmeControllerId: testControllerID})

			if !proto.Equal(response, tt.out) {
				t.Error("response: expected", tt.out, "received", response)
			}

			if er, ok := status.FromError(err); ok {
				if er.Code() != tt.errCode {
					t.Error("error code: expected", tt.errCode, "received", er.Code())
				}
				if er.Message() != tt.errMsg {
					t.Error("error message: expected", tt.errMsg, "received", er.Message())
				}
			} else {
				t.Errorf("expect grpc error status, received %v", err)
			}

			controller := testEnv.opiSpdkServer.nvme.Controllers[testControllerName]
			if tt.existingController != nil {
				if !proto.Equal(tt.existingController, controller) {
					t.Errorf("expect %v exists", tt.existingController)
				}
			} else {
				if tt.errCode == codes.OK {
					if !proto.Equal(tt.out, controller) {
						t.Errorf("expect %v exists", tt.out)
					}
				} else {
					if controller != nil {
						t.Errorf("expect no controller exists")
					}
				}
			}
		})
	}
}

func TestFrontEnd_UpdateNvmeController(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in                 *pb.NvmeController
		out                *pb.NvmeController
		spdk               []string
		errCode            codes.Code
		errMsg             string
		existingController *pb.NvmeController
	}{
		"max_limit rw_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{RwIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rw_iops_kiops is not supported",
			existingController: nil,
		},
		"max_limit rw_bandwidth_mbs is not supported": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{RwBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rw_bandwidth_mbs is not supported",
			existingController: &testControllerWithMaxQos,
		},
		"min_limit rw_bandwidth_mbs is not supported": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{RwBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rw_bandwidth_mbs is not supported",
			existingController: &testControllerWithMaxQos,
		},
		"min_limit rw_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{RwIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rw_iops_kiops is not supported",
			existingController: &testControllerWithMaxQos,
		},
		"min_limit rd_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{RdIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rd_iops_kiops is not supported",
			existingController: &testControllerWithMaxQos,
		},
		"min_limit wr_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{WrIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit wr_iops_kiops is not supported",
			existingController: &testControllerWithMaxQos,
		},
		"min_limit rd_bandwidth_mbs cannot be greater than max_limit rd_bandwidth_mbs": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 2},
					MaxLimit:         &pb.QosLimit{RdBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rd_bandwidth_mbs cannot be greater than max_limit rd_bandwidth_mbs",
			existingController: &testControllerWithMaxQos,
		},
		"min_limit wr_bandwidth_mbs cannot be greater than max_limit wr_bandwidth_mbs": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{WrBandwidthMbs: 2},
					MaxLimit:         &pb.QosLimit{WrBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit wr_bandwidth_mbs cannot be greater than max_limit wr_bandwidth_mbs",
			existingController: &testControllerWithMaxQos,
		},
		"max_limit rd_iops_kiops is negative": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{RdIopsKiops: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rd_iops_kiops cannot be negative",
			existingController: &testControllerWithMaxQos,
		},
		"max_limit wr_iops_kiops is negative": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{WrIopsKiops: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit wr_iops_kiops cannot be negative",
			existingController: &testControllerWithMaxQos,
		},
		"max_limit rd_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{RdBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rd_bandwidth_mbs cannot be negative",
			existingController: &testControllerWithMaxQos,
		},
		"max_limit wr_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{WrBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit wr_bandwidth_mbs cannot be negative",
			existingController: &testControllerWithMaxQos,
		},
		"min_limit rd_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rd_bandwidth_mbs cannot be negative",
			existingController: &testControllerWithMaxQos,
		},
		"min_limit wr_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MinLimit:         &pb.QosLimit{WrBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit wr_bandwidth_mbs cannot be negative",
			existingController: &testControllerWithMaxQos,
		},
		"no qos limits are specified": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
				},
			},
			out: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
			spdk:               []string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:            codes.OK,
			errMsg:             "",
			existingController: &testControllerWithMaxQos,
		},
		"set qos SPDK call failed": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 10},
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 5},
				},
			},
			out:                nil,
			spdk:               []string{`{"id":%d,"error":{"code":-1,"message":"some internal error"},"result":true}`},
			errCode:            status.Convert(spdk.ErrFailedSpdkCall).Code(),
			errMsg:             status.Convert(spdk.ErrFailedSpdkCall).Message(),
			existingController: &testControllerWithMaxQos,
		},
		"set qos SPDK call result false": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testControllerWithMaxQos.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TYPE_PCIE,
					NvmeControllerId: proto.Int32(1),
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 10},
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 5},
				},
			},
			out:                nil,
			spdk:               []string{`{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			errCode:            status.Convert(spdk.ErrUnexpectedSpdkCallResult).Code(),
			errMsg:             status.Convert(spdk.ErrUnexpectedSpdkCallResult).Message(),
			existingController: &testControllerWithMaxQos,
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			tt.in = utils.ProtoClone(tt.in)
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()
			testEnv.opiSpdkServer.nvme.Subsystems[testSubsystem.Name] = &testSubsystem
			if tt.existingController != nil {
				tt.existingController = utils.ProtoClone(tt.existingController)
				tt.existingController.Name = testControllerName
				testEnv.opiSpdkServer.nvme.Controllers[tt.existingController.Name] = tt.existingController
			}

			response, err := testEnv.opiSpdkServer.UpdateNvmeController(testEnv.ctx,
				&pb.UpdateNvmeControllerRequest{NvmeController: tt.in})

			if !proto.Equal(response, tt.out) {
				t.Error("response: expected", tt.out, "received", response)
			}

			if er, ok := status.FromError(err); ok {
				if er.Code() != tt.errCode {
					t.Error("error code: expected", tt.errCode, "received", er.Code())
				}
				if er.Message() != tt.errMsg {
					t.Error("error message: expected", tt.errMsg, "received", er.Message())
				}
			} else {
				t.Errorf("expect grpc error status, received %v", err)
			}

			controller := testEnv.opiSpdkServer.nvme.Controllers[tt.in.Name]
			if tt.errCode == codes.OK {
				if !proto.Equal(tt.out, controller) {
					t.Errorf("expect new %v exists, found %v", tt.out, controller)
				}
			} else {
				if !proto.Equal(tt.existingController, controller) {
					t.Errorf("expect original %v exists, found %v", tt.existingController, controller)
				}
			}
		})
	}
}
