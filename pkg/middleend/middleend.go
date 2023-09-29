// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2023 Intel Corporation

// Package middleend implements the MiddleEnd APIs (service) of the storage Server
package middleend

import (
	"context"
	"encoding/hex"
	"log"
	"runtime"
	"runtime/debug"

	"github.com/philippgille/gokv"

	"github.com/opiproject/gospdk/spdk"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-intel-bridge/pkg/models"
	"github.com/opiproject/opi-spdk-bridge/pkg/middleend"
	"github.com/opiproject/opi-spdk-bridge/pkg/utils"

	"go.einride.tech/aip/fieldbehavior"
	"go.einride.tech/aip/resourceid"
	"go.einride.tech/aip/resourcename"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	errMissingArgument    = status.Error(codes.InvalidArgument, "missing argument")
	errMalformedArgument  = status.Error(codes.InvalidArgument, "malformed argument")
	errAlreadyExists      = status.Error(codes.AlreadyExists, "volume already exists")
	errVolumeNotFound     = status.Error(codes.NotFound, "volume not found")
	errNotSupportedCipher = status.Error(codes.Unimplemented, "not supported cipher")
	errWrongKeySize       = status.Error(codes.InvalidArgument, "invalid key size")
)

type volumeParameters struct {
	encryptedVolumes map[string]string
}

// Server contains middleend related OPI services
type Server struct {
	pb.MiddleendEncryptionServiceServer
	pb.MiddleendQosVolumeServiceServer

	store   gokv.Store
	rpc     spdk.JSONRPC
	volumes volumeParameters
}

// NewServer creates initialized instance of middleend server
func NewServer(jsonRPC spdk.JSONRPC, store gokv.Store) *Server {
	if jsonRPC == nil {
		log.Panic("nil for JSONRPC is not allowed")
	}
	if store == nil {
		log.Panic("nil for Store is not allowed")
	}
	opiSpdkServer := middleend.NewServer(jsonRPC, store)
	return &Server{
		opiSpdkServer,
		opiSpdkServer,
		store,
		jsonRPC,
		volumeParameters{encryptedVolumes: make(map[string]string)},
	}
}

// CreateEncryptedVolume creates an encrypted volume
func (s *Server) CreateEncryptedVolume(_ context.Context, in *pb.CreateEncryptedVolumeRequest) (*pb.EncryptedVolume, error) {
	defer func() {
		if in != nil && in.EncryptedVolume != nil {
			for i := range in.EncryptedVolume.Key {
				in.EncryptedVolume.Key[i] = 0
			}
			in.EncryptedVolume.Cipher = pb.EncryptionType_ENCRYPTION_TYPE_UNSPECIFIED
		}
		// Run GC to free all variables which contained encryption keys
		runtime.GC()
		// Return allocated memory to OS, otherwise the memory which contained
		// keys can be kept for some time.
		debug.FreeOSMemory()
	}()

	err := verifyCreateEncryptedVolumeRequestArgs(in)
	if err != nil {
		return nil, err
	}

	resourceID := resourceid.NewSystemGenerated()
	if in.EncryptedVolumeId != "" {
		err := resourceid.ValidateUserSettable(in.EncryptedVolumeId)
		if err != nil {
			log.Printf("error: %v", err)
			return nil, errMalformedArgument
		}
		log.Printf("client provided the ID of a resource %v, ignoring the name field %v", in.EncryptedVolumeId, in.EncryptedVolume.Name)
		resourceID = in.EncryptedVolumeId
	}
	in.EncryptedVolume.Name = utils.ResourceIDToVolumeName(resourceID)

	_, ok := s.volumes.encryptedVolumes[in.EncryptedVolume.Name]
	if ok {
		log.Printf("Already existing EncryptedVolume with id %v", in.EncryptedVolume.Name)
		// it is not possible to check keys and algorithm. Always send error
		return nil, errAlreadyExists
	}

	bdevUUID, err := s.getBdevUUIDByName(in.EncryptedVolume.VolumeNameRef)
	if err != nil {
		log.Println("Failed to find UUID for bdev", in.EncryptedVolume.VolumeNameRef)
		return nil, err
	}

	half := len(in.EncryptedVolume.Key) / 2
	tweakMode := "A"
	params := &models.NpiBdevSetKeysParams{
		UUID:   bdevUUID,
		Key:    hex.EncodeToString(in.EncryptedVolume.Key[:half]),
		Key2:   hex.EncodeToString(in.EncryptedVolume.Key[half:]),
		Cipher: "AES_XTS",
		Tweak:  tweakMode,
	}
	defer func() {
		params = nil
	}()

	var result models.NpiBdevSetKeysResult
	err = s.rpc.Call("npi_bdev_set_keys", params, &result)
	if err != nil {
		log.Println("error:", err)
		return nil, spdk.ErrFailedSpdkCall
	}
	if !result {
		log.Println("Failed result on SPDK call:", result)
		return nil, spdk.ErrUnexpectedSpdkCallResult
	}

	s.volumes.encryptedVolumes[in.EncryptedVolume.Name] = in.EncryptedVolume.VolumeNameRef

	return &pb.EncryptedVolume{
		Name:          in.EncryptedVolume.Name,
		VolumeNameRef: in.EncryptedVolume.VolumeNameRef,
	}, nil
}

func verifyCreateEncryptedVolumeRequestArgs(in *pb.CreateEncryptedVolumeRequest) error {
	if err := fieldbehavior.ValidateRequiredFields(in); err != nil {
		log.Printf("error: %v", err)
		return errMissingArgument
	}

	keyLengthInBits := len(in.EncryptedVolume.Key) * 8
	expectedKeyLengthInBits := 0
	switch {
	case in.EncryptedVolume.Cipher == pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_256:
		expectedKeyLengthInBits = 512
	case in.EncryptedVolume.Cipher == pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_128:
		expectedKeyLengthInBits = 256
	default:
		log.Println("only AES_XTS_128 and AES_XTS_256 are supported")
		return errNotSupportedCipher
	}

	if keyLengthInBits != expectedKeyLengthInBits {
		log.Printf("expected key size %vb, provided size %vb",
			expectedKeyLengthInBits, keyLengthInBits)
		return errWrongKeySize
	}

	return nil
}

// DeleteEncryptedVolume deletes an encrypted volume
func (s *Server) DeleteEncryptedVolume(_ context.Context, in *pb.DeleteEncryptedVolumeRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteEncryptedVolume: Received from client: %v", in)

	if err := fieldbehavior.ValidateRequiredFields(in); err != nil {
		log.Printf("error: %v", err)
		return nil, errMissingArgument
	}

	if err := resourcename.Validate(in.Name); err != nil {
		log.Printf("error: %v", err)
		return nil, errMalformedArgument
	}

	underlyingBdev, ok := s.volumes.encryptedVolumes[in.Name]
	if !ok {
		if in.AllowMissing {
			return &emptypb.Empty{}, nil
		}
		log.Printf("error: unable to find key %s", in.Name)
		return nil, errVolumeNotFound
	}

	bdevUUID, err := s.getBdevUUIDByName(underlyingBdev)
	if err != nil {
		log.Println("Failed to find UUID for bdev", in.Name)
		return nil, err
	}
	params := models.NpiBdevClearKeysParams{
		UUID: bdevUUID,
	}
	var result models.NpiBdevClearKeysResult
	err = s.rpc.Call("npi_bdev_clear_keys", params, &result)
	if err != nil {
		log.Println(err)
		return nil, spdk.ErrFailedSpdkCall
	}
	if !result {
		log.Println("Failed result on SPDK call:", result)
		return nil, spdk.ErrUnexpectedSpdkCallResult
	}
	delete(s.volumes.encryptedVolumes, in.Name)
	return &emptypb.Empty{}, nil
}

func (s *Server) getBdevUUIDByName(name string) (string, error) {
	params := spdk.BdevGetBdevsParams{Name: name}
	var result []spdk.BdevGetBdevsResult
	err := s.rpc.Call("bdev_get_bdevs", params, &result)
	if err != nil {
		log.Println("error:", err)
		return "", spdk.ErrFailedSpdkCall
	}
	if len(result) != 1 {
		log.Println("Found bdevs:", result, "under the name", params.Name)
		return "", spdk.ErrUnexpectedSpdkCallResult
	}
	return result[0].UUID, nil
}
