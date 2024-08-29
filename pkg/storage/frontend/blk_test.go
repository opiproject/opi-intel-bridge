// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2023 Intel Corporation

// Package frontend implements the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"reflect"
	"testing"

	"github.com/opiproject/gospdk/spdk"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-intel-bridge/pkg/models"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestFrontEnd_CreateBlkParams(t *testing.T) {
	tests := map[string]struct {
		in        *pb.VirtioBlk
		out       any
		expectErr bool
	}{
		"fail on virtual function": {
			in: &pb.VirtioBlk{
				PcieId: &pb.PciEndpoint{PhysicalFunction: wrapperspb.Int32(0), VirtualFunction: wrapperspb.Int32(1), PortId: wrapperspb.Int32(0)},
			},
			out:       nil,
			expectErr: true,
		},
		"fail on non zero port": {
			in: &pb.VirtioBlk{
				PcieId: &pb.PciEndpoint{PhysicalFunction: wrapperspb.Int32(0), VirtualFunction: wrapperspb.Int32(0), PortId: wrapperspb.Int32(1)},
			},
			out:       nil,
			expectErr: true,
		},
		"valid pf": {
			in: &pb.VirtioBlk{
				PcieId:        &pb.PciEndpoint{PhysicalFunction: wrapperspb.Int32(1), VirtualFunction: wrapperspb.Int32(0), PortId: wrapperspb.Int32(0)},
				VolumeNameRef: "volume42",
				MaxIoQps:      5,
			},
			out: models.MevVhostCreateBlkControllerParams{
				Ctrlr:     "h0-pf1-vf0-PF",
				DevName:   "volume42",
				Transport: blkTransport,
				VqCount:   5,
			},
			expectErr: false,
		},
		"empty max_io_qps": {
			in: &pb.VirtioBlk{
				PcieId:        &pb.PciEndpoint{PhysicalFunction: wrapperspb.Int32(3), VirtualFunction: wrapperspb.Int32(0), PortId: wrapperspb.Int32(0)},
				VolumeNameRef: "volume42",
			},
			out: models.MevVhostCreateBlkControllerParams{
				Ctrlr:     "h0-pf3-vf0-PF",
				DevName:   "volume42",
				Transport: blkTransport,
				VqCount:   1,
			},
			expectErr: false,
		},
	}
	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			transport := NewMevBlkTransport()

			params, err := transport.CreateParams(tt.in)

			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, received: %v", tt.expectErr, err)
			}
			if !reflect.DeepEqual(params, tt.out) {
				t.Errorf("Expected params: %v, received %v", tt.out, params)
			}
		})
	}
}

func TestFrontEnd_DeleteBlkParams(t *testing.T) {
	tests := map[string]struct {
		in        *pb.VirtioBlk
		out       any
		expectErr bool
	}{
		"fail on virtual function": {
			in: &pb.VirtioBlk{
				PcieId: &pb.PciEndpoint{PhysicalFunction: wrapperspb.Int32(0), VirtualFunction: wrapperspb.Int32(1), PortId: wrapperspb.Int32(0)},
			},
			out:       nil,
			expectErr: true,
		},
		"fail on non zero port": {
			in: &pb.VirtioBlk{
				PcieId: &pb.PciEndpoint{PhysicalFunction: wrapperspb.Int32(0), VirtualFunction: wrapperspb.Int32(0), PortId: wrapperspb.Int32(1)},
			},
			out:       nil,
			expectErr: true,
		},
		"valid pf": {
			in: &pb.VirtioBlk{
				PcieId:        &pb.PciEndpoint{PhysicalFunction: wrapperspb.Int32(1), VirtualFunction: wrapperspb.Int32(0), PortId: wrapperspb.Int32(0)},
				VolumeNameRef: "volume42",
			},
			out: spdk.VhostDeleteControllerParams{
				Ctrlr: "h0-pf1-vf0-PF",
			},
			expectErr: false,
		},
	}
	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			transport := NewMevBlkTransport()

			params, err := transport.DeleteParams(tt.in)

			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, received: %v", tt.expectErr, err)
			}
			if !reflect.DeepEqual(params, tt.out) {
				t.Errorf("Expected params: %v, received %v", tt.out, params)
			}
		})
	}
}
