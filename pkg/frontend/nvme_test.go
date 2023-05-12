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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	testSubsystem = pb.NVMeSubsystem{
		Spec: &pb.NVMeSubsystemSpec{
			Id:  &pc.ObjectKey{Value: "subsystem-test"},
			Nqn: "nqn.2022-09.io.spdk:opi3",
		},
	}
	testControllerID         = "controller-test"
	testControllerWithMaxQos = pb.NVMeController{
		Spec: &pb.NVMeControllerSpec{
			Id:               &pc.ObjectKey{Value: testControllerID},
			SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
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

func TestFrontEnd_CreateNVMeController(t *testing.T) {
	tests := map[string]struct {
		in                 *pb.NVMeController
		out                *pb.NVMeController
		spdk               []string
		errCode            codes.Code
		errMsg             string
		start              bool
		existingController *pb.NVMeController
	}{
		"limit_max rw_iops_kiops is not supported": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RwIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_max rw_iops_kiops is not supported",
			start:              false,
			existingController: nil,
		},
		"limit_max rw_bandwidth_mbs is not supported": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RwBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_max rw_bandwidth_mbs is not supported",
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
			out: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NVMeControllerStatus{Active: true},
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
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
				},
			},
			out: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
				},
				Status: &pb.NVMeControllerStatus{Active: true},
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
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
			},
			out: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NVMeControllerStatus{Active: true},
			},
			spdk:    []string{},
			errCode: codes.OK,
			errMsg:  "",
			start:   false,
			existingController: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NVMeControllerStatus{Active: true},
			},
		},
		"controller with different max qos limits exists": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 12321, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.AlreadyExists,
			errMsg:  fmt.Sprintf("Controller %v exists with different QoS limits", testControllerID),
			start:   false,
			existingController: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 1, WrIopsKiops: 1, RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NVMeControllerStatus{Active: true},
			},
		},
		"controller with different min qos limits exists": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 12321, WrBandwidthMbs: 1},
				},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.AlreadyExists,
			errMsg:  fmt.Sprintf("Controller %v exists with different QoS limits", testControllerID),
			start:   false,
			existingController: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NVMeControllerStatus{Active: true},
			},
		},
		"limit_min rw_bandwidth_mbs is not supported": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RwBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_min rw_bandwidth_mbs is not supported",
			start:              false,
			existingController: nil,
		},
		"limit_min rw_iops_kiops is not supported": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RwIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_min rw_iops_kiops is not supported",
			start:              false,
			existingController: nil,
		},
		"limit_min rd_iops_kiops is not supported": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_min rd_iops_kiops is not supported",
			start:              false,
			existingController: nil,
		},
		"limit_min wr_iops_kiops is not supported": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{WrIopsKiops: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_min wr_iops_kiops is not supported",
			start:              false,
			existingController: nil,
		},
		"allowed min qos limits": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{},
				},
			},
			out: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
					MaxLimit:         &pb.QosLimit{},
				},
				Status: &pb.NVMeControllerStatus{Active: true},
			},
			spdk: []string{
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`,
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:            codes.OK,
			errMsg:             "",
			start:              true,
			existingController: nil,
		},
		"limit_min rd_bandwidth_mbs cannot be greater than limit_max rd_bandwidth_mbs": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 2},
					MaxLimit:         &pb.QosLimit{RdBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_min rd_bandwidth_mbs cannot be greater than limit_max rd_bandwidth_mbs",
			start:              false,
			existingController: nil,
		},
		"limit_min wr_bandwidth_mbs cannot be greater than limit_max wr_bandwidth_mbs": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{WrBandwidthMbs: 2},
					MaxLimit:         &pb.QosLimit{WrBandwidthMbs: 1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_min wr_bandwidth_mbs cannot be greater than limit_max wr_bandwidth_mbs",
			start:              false,
			existingController: nil,
		},
		"allowed min and max qos limits": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 2, WrIopsKiops: 2, RdBandwidthMbs: 2, WrBandwidthMbs: 2},
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
			},
			out: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: -1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: 2, WrIopsKiops: 2, RdBandwidthMbs: 2, WrBandwidthMbs: 2},
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: 1, WrBandwidthMbs: 1},
				},
				Status: &pb.NVMeControllerStatus{Active: true},
			},
			spdk: []string{
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`,
				`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:            codes.OK,
			errMsg:             "",
			start:              true,
			existingController: nil,
		},
		"limit_max rd_iops_kiops is negative": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RdIopsKiops: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_max rd_iops_kiops cannot be negative",
			start:              false,
			existingController: nil,
		},
		"limit_max wr_iops_kiops is negative": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{WrIopsKiops: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_max wr_iops_kiops cannot be negative",
			start:              false,
			existingController: nil,
		},
		"limit_max rd_bandwidth_mbs is negative": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{RdBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_max rd_bandwidth_mbs cannot be negative",
			start:              false,
			existingController: nil,
		},
		"limit_max wr_bandwidth_mbs is negative": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MaxLimit:         &pb.QosLimit{WrBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_max wr_bandwidth_mbs cannot be negative",
			start:              false,
			existingController: nil,
		},
		"limit_min rd_bandwidth_mbs is negative": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{RdBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_min rd_bandwidth_mbs cannot be negative",
			start:              false,
			existingController: nil,
		},
		"limit_min wr_bandwidth_mbs is negative": {
			in: &pb.NVMeController{
				Spec: &pb.NVMeControllerSpec{
					Id:               &pc.ObjectKey{Value: testControllerID},
					SubsystemId:      &pc.ObjectKey{Value: testSubsystem.Spec.Id.Value},
					PcieId:           &pb.PciEndpoint{PhysicalFunction: 0, VirtualFunction: 2},
					NvmeControllerId: 1,
					MinLimit:         &pb.QosLimit{WrBandwidthMbs: -1},
				},
			},
			out:                nil,
			spdk:               []string{},
			errCode:            codes.InvalidArgument,
			errMsg:             "QoS limit_min wr_bandwidth_mbs cannot be negative",
			start:              false,
			existingController: nil,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			testEnv := createTestEnvironment(test.start, test.spdk)
			defer testEnv.Close()
			testEnv.opiSpdkServer.nvme.Subsystems[testSubsystem.Spec.Id.Value] = &testSubsystem
			if test.existingController != nil {
				testEnv.opiSpdkServer.nvme.Controllers[test.existingController.Spec.Id.Value] = test.existingController
			}

			response, err := testEnv.opiSpdkServer.CreateNVMeController(testEnv.ctx,
				&pb.CreateNVMeControllerRequest{NvMeController: test.in})

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

			controller := testEnv.opiSpdkServer.nvme.Controllers[test.in.Spec.Id.Value]
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
