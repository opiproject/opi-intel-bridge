// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2023 Intel Corporation

// Package frontend implements the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"fmt"

	"github.com/opiproject/gospdk/spdk"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-intel-bridge/pkg/models"
	"github.com/opiproject/opi-spdk-bridge/pkg/frontend"
)

const (
	blkTransport = "mev_blk_transport"
)

type mevBlkTransport struct{}

func (v mevBlkTransport) CreateParams(virtioBlk *pb.VirtioBlk) (any, error) {
	ctrlr, err := v.getCtrlr(virtioBlk.PcieId)
	if err != nil {
		return nil, err
	}

	vqCount := 1
	if virtioBlk.GetMaxIoQps() != 0 {
		vqCount = int(virtioBlk.GetMaxIoQps())
	}

	return models.MevVhostCreateBlkControllerParams{
		Ctrlr:     ctrlr,
		DevName:   virtioBlk.VolumeNameRef,
		Transport: blkTransport,
		VqCount:   vqCount,
	}, nil
}

func (v mevBlkTransport) DeleteParams(virtioBlk *pb.VirtioBlk) (any, error) {
	ctrlr, err := v.getCtrlr(virtioBlk.PcieId)
	if err != nil {
		return nil, err
	}

	return spdk.VhostDeleteControllerParams{Ctrlr: ctrlr}, nil
}

func (v mevBlkTransport) getCtrlr(pci *pb.PciEndpoint) (string, error) {
	if pci.PortId.Value != 0 {
		return "", fmt.Errorf("only port 0 is supported")
	}

	if pci.VirtualFunction.Value != 0 {
		return "", fmt.Errorf("virtual functions are not supported")
	}

	return fmt.Sprintf("h0-pf%v-vf0-PF", pci.PhysicalFunction.Value), nil
}

// NewMevBlkTransport creates an isntance of mevBlkTransport
func NewMevBlkTransport() frontend.VirtioBlkTransport {
	return &mevBlkTransport{}
}
