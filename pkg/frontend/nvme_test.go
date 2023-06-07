// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2023 Intel Corporation

// Package frontend implements the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/opiproject/gospdk/spdk"
	pc "github.com/opiproject/opi-api/common/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/server"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	testSubsystem = pb.NvmeSubsystem{
		Name: server.ResourceIDToVolumeName("subsystem-test"),
		Spec: &pb.NvmeSubsystemSpec{
			Nqn: "nqn.2022-09.io.spdk:opi3",
		},
	}
	testControllerID         = "controller-test"
	testControllerName       = server.ResourceIDToVolumeName(testControllerID)
	testControllerWithMaxQos = pb.NvmeController{
		Spec: &pb.NvmeControllerSpec{
			SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
			PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
			NvmeControllerId: 1,
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
	tests := map[string]struct {
		in                 *pb.NvmeController
		out                *pb.NvmeController
		spdk               []string
		errCode            codes.Code
		errMsg             string
		start              bool
		existingController *pb.NvmeController
	}{
		"max_limit rw_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RwIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rw_iops_kiops is not supported",
			start:              false,
			existingController: nil,
		},
		"max_limit rw_bandwidth_mbs is not supported": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RwBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rw_bandwidth_mbs is not supported",
			start:              false,
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
			start:              true,
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
			start:              true,
			existingController: nil,
		},
		"allowed max qos limits": {
			in: &testControllerWithMaxQos,
			out: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
			spdk: []string{
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`,
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:            codes.OK,
			errMsg:             "",
			start:              true,
			existingController: nil,
		},
		"no qos limits are specified": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
				},
			},
			out: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
			spdk: []string{
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`,
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:            codes.OK,
			errMsg:             "",
			start:              true,
			existingController: nil,
		},
		"controller with the same qos limits exists": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
			},
			out: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode: codes.OK,
			errMsg:  "",
			start:   true,
			existingController: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
		},
		"controller with different max qos limits exists": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 12321, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.AlreadyExists,
			errMsg:  fmt.Sprintf("Controller %v exists with different QoS limits", testControllerName),
			start:   false,
			existingController: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
		},
		"controller with different min qos limits exists": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 12321, WrBandwidthMbs: 1},
				},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.AlreadyExists,
			errMsg:  fmt.Sprintf("Controller %v exists with different QoS limits", testControllerName),
			start:   false,
			existingController: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
		},
		"min_limit rw_bandwidth_mbs is not supported": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RwBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rw_bandwidth_mbs is not supported",
			start:              false,
			existingController: nil,
		},
		"min_limit rw_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RwIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rw_iops_kiops is not supported",
			start:              false,
			existingController: nil,
		},
		"min_limit rd_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rd_iops_kiops is not supported",
			start:              false,
			existingController: nil,
		},
		"min_limit wr_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{WrIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit wr_iops_kiops is not supported",
			start:              false,
			existingController: nil,
		},
		"allowed min qos limits": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{},
				},
			},
			out: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
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
			start:              true,
			existingController: nil,
		},
		"min_limit rd_bandwidth_mbs cannot be greater than max_limit rd_bandwidth_mbs": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 2},
					MaxLimit:         &pb.QosLimit{RdBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rd_bandwidth_mbs cannot be greater than max_limit rd_bandwidth_mbs",
			start:              false,
			existingController: nil,
		},
		"min_limit wr_bandwidth_mbs cannot be greater than max_limit wr_bandwidth_mbs": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{WrBandwidthMbs: 2},
					MaxLimit:         &pb.QosLimit{WrBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit wr_bandwidth_mbs cannot be greater than max_limit wr_bandwidth_mbs",
			start:              false,
			existingController: nil,
		},
		"allowed min and max qos limits": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 2, WrIopsKiops: 2, RdBandwidthMbs: 2, WrBandwidthMbs: 2},
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
			},
			out: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
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
			start:              true,
			existingController: nil,
		},
		"max_limit rd_iops_kiops is negative": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rd_iops_kiops cannot be negative",
			start:              false,
			existingController: nil,
		},
		"max_limit wr_iops_kiops is negative": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{WrIopsKiops: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit wr_iops_kiops cannot be negative",
			start:              false,
			existingController: nil,
		},
		"max_limit rd_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RdBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rd_bandwidth_mbs cannot be negative",
			start:              false,
			existingController: nil,
		},
		"max_limit wr_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{WrBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit wr_bandwidth_mbs cannot be negative",
			start:              false,
			existingController: nil,
		},
		"min_limit rd_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rd_bandwidth_mbs cannot be negative",
			start:              false,
			existingController: nil,
		},
		"min_limit wr_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{WrBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit wr_bandwidth_mbs cannot be negative",
			start:              false,
			existingController: nil,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			test.in = server.ProtoClone(test.in)
			testEnv := createTestEnvironment(test.start, test.spdk)
			defer testEnv.Close()
			testEnv.opiSpdkServer.nvme.Subsystems[testSubsystem.Name] = &testSubsystem
			if test.existingController != nil {
				test.existingController = server.ProtoClone(test.existingController)
				test.existingController.Name = testControllerName
				testEnv.opiSpdkServer.nvme.Controllers[test.existingController.Name] = test.existingController
			}
			if test.out != nil {
				test.out = server.ProtoClone(test.out)
				test.out.Name = testControllerName
			}

			response, err := testEnv.opiSpdkServer.CreateNvmeController(testEnv.ctx,
				&pb.CreateNvmeControllerRequest{
					NvmeController:   test.in,
					NvmeControllerId: testControllerID})

			marshalledOut, _ := proto.Marshal(test.out)
			marshalledResponse, _ := proto.Marshal(response)
			if !bytes.Equal(marshalledOut, marshalledResponse) {
				t.Error("response: expected", test.out, "received", response)
			}

			if er, ok := status.FromError(err); ok {
				if er.Code() != test.errCode {
					t.Error("error code: expected", test.errCode, "received", er.Code())
				}
				if er.Message() != test.errMsg {
					t.Error("error message: expected", test.errMsg, "received", er.Message())
				}
			} else {
				t.Errorf("expect grpc error status")
			}

			controller := testEnv.opiSpdkServer.nvme.Controllers[testControllerName]
			if test.existingController != nil {
				if !proto.Equal(test.existingController, controller) {
					t.Errorf("expect %v exists", test.existingController)
				}
			} else {
				if test.errCode == codes.OK {
					if !proto.Equal(test.in, controller) {
						t.Errorf("expect %v exists", test.in)
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
	tests := map[string]struct {
		in                 *pb.NvmeController
		out                *pb.NvmeController
		spdk               []string
		errCode            codes.Code
		errMsg             string
		start              bool
		existingController *pb.NvmeController
	}{
		"max_limit rw_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RwIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rw_iops_kiops is not supported",
			start:              false,
			existingController: nil,
		},
		"max_limit rw_bandwidth_mbs is not supported": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RwBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rw_bandwidth_mbs is not supported",
			start:              false,
			existingController: &testControllerWithMaxQos,
		},
		"min_limit rw_bandwidth_mbs is not supported": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RwBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rw_bandwidth_mbs is not supported",
			start:              false,
			existingController: &testControllerWithMaxQos,
		},
		"min_limit rw_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RwIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rw_iops_kiops is not supported",
			start:              false,
			existingController: &testControllerWithMaxQos,
		},
		"min_limit rd_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rd_iops_kiops is not supported",
			start:              false,
			existingController: &testControllerWithMaxQos,
		},
		"min_limit wr_iops_kiops is not supported": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{WrIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit wr_iops_kiops is not supported",
			start:              false,
			existingController: &testControllerWithMaxQos,
		},
		"min_limit rd_bandwidth_mbs cannot be greater than max_limit rd_bandwidth_mbs": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 2},
					MaxLimit:         &pb.QosLimit{RdBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rd_bandwidth_mbs cannot be greater than max_limit rd_bandwidth_mbs",
			start:              false,
			existingController: &testControllerWithMaxQos,
		},
		"min_limit wr_bandwidth_mbs cannot be greater than max_limit wr_bandwidth_mbs": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{WrBandwidthMbs: 2},
					MaxLimit:         &pb.QosLimit{WrBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit wr_bandwidth_mbs cannot be greater than max_limit wr_bandwidth_mbs",
			start:              false,
			existingController: &testControllerWithMaxQos,
		},
		"max_limit rd_iops_kiops is negative": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rd_iops_kiops cannot be negative",
			start:              false,
			existingController: &testControllerWithMaxQos,
		},
		"max_limit wr_iops_kiops is negative": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{WrIopsKiops: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit wr_iops_kiops cannot be negative",
			start:              false,
			existingController: &testControllerWithMaxQos,
		},
		"max_limit rd_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RdBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit rd_bandwidth_mbs cannot be negative",
			start:              false,
			existingController: &testControllerWithMaxQos,
		},
		"max_limit wr_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{WrBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS max_limit wr_bandwidth_mbs cannot be negative",
			start:              false,
			existingController: &testControllerWithMaxQos,
		},
		"min_limit rd_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit rd_bandwidth_mbs cannot be negative",
			start:              false,
			existingController: &testControllerWithMaxQos,
		},
		"min_limit wr_bandwidth_mbs is negative": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{WrBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS min_limit wr_bandwidth_mbs cannot be negative",
			start:              false,
			existingController: &testControllerWithMaxQos,
		},
		"no qos limits are specified": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
				},
			},
			out: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
				},
				Status: &pb.NvmeControllerStatus{Active: true},
			},
			spdk:               []string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:            codes.OK,
			errMsg:             "",
			start:              true,
			existingController: &testControllerWithMaxQos,
		},
		"set qos SPDK call failed": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 10},
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 5},
				},
			},
			out:                nil,
			spdk:               []string{`{"id":%d,"error":{"code":-1,"message":"some internal error"},"result":true}`},
			errCode:            status.Convert(spdk.ErrFailedSpdkCall).Code(),
			errMsg:             status.Convert(spdk.ErrFailedSpdkCall).Message(),
			start:              true,
			existingController: &testControllerWithMaxQos,
		},
		"set qos SPDK call result false": {
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Name},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 10},
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 5},
				},
			},
			out:                nil,
			spdk:               []string{`{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			errCode:            status.Convert(spdk.ErrUnexpectedSpdkCallResult).Code(),
			errMsg:             status.Convert(spdk.ErrUnexpectedSpdkCallResult).Message(),
			start:              true,
			existingController: &testControllerWithMaxQos,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			test.in = server.ProtoClone(test.in)
			testEnv := createTestEnvironment(test.start, test.spdk)
			defer testEnv.Close()
			testEnv.opiSpdkServer.nvme.Subsystems[testSubsystem.Name] = &testSubsystem
			if test.existingController != nil {
				test.existingController = server.ProtoClone(test.existingController)
				test.existingController.Name = testControllerName
				testEnv.opiSpdkServer.nvme.Controllers[test.existingController.Name] = test.existingController
			}

			response, err := testEnv.opiSpdkServer.UpdateNvmeController(testEnv.ctx,
				&pb.UpdateNvmeControllerRequest{NvmeController: test.in})

			marshalledOut, _ := proto.Marshal(test.out)
			marshalledResponse, _ := proto.Marshal(response)
			if !bytes.Equal(marshalledOut, marshalledResponse) {
				t.Error("response: expected", test.out, "received", response)
			}

			if er, ok := status.FromError(err); ok {
				if er.Code() != test.errCode {
					t.Error("error code: expected", test.errCode, "received", er.Code())
				}
				if er.Message() != test.errMsg {
					t.Error("error message: expected", test.errMsg, "received", er.Message())
				}
			} else {
				t.Errorf("expect grpc error status")
			}

			controller := testEnv.opiSpdkServer.nvme.Controllers[test.in.Name]
			if test.errCode == codes.OK {
				if !proto.Equal(test.in, controller) {
					t.Errorf("expect new %v exists, found %v", test.in, controller)
				}
			} else {
				if !proto.Equal(test.existingController, controller) {
					t.Errorf("expect original %v exists, found %v", test.existingController, controller)
				}
			}
		})
	}
}
