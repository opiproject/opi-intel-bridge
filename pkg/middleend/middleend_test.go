// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2023 Intel Corporation

// Package middleend implements the FrontEnd APIs (host facing) of the storage Server
package middleend

import (
	"bytes"
	"context"
	"log"
	"net"
	"os"
	"testing"

	"github.com/opiproject/gospdk/spdk"
	pc "github.com/opiproject/opi-api/common/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

type middleendClient struct {
	pb.MiddleendEncryptionServiceClient
}

type testEnv struct {
	opiSpdkServer *Server
	client        *middleendClient
	ln            net.Listener
	testSocket    string
	ctx           context.Context
	conn          *grpc.ClientConn
	jsonRPC       spdk.JSONRPC
}

func (e *testEnv) Close() {
	server.CloseListener(e.ln)
	if err := os.RemoveAll(e.testSocket); err != nil {
		log.Fatal(err)
	}
	server.CloseGrpcConnection(e.conn)
}

func createTestEnvironment(startSpdkServer bool, spdkResponses []string) *testEnv {
	env := &testEnv{}
	env.testSocket = server.GenerateSocketName("middleend")
	env.ln, env.jsonRPC = server.CreateTestSpdkServer(env.testSocket, startSpdkServer, spdkResponses)
	env.opiSpdkServer = NewServer(env.jsonRPC)

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx,
		"",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer(env.opiSpdkServer)))
	if err != nil {
		log.Fatal(err)
	}
	env.ctx = ctx
	env.conn = conn

	env.client = &middleendClient{
		pb.NewMiddleendEncryptionServiceClient(env.conn),
	}

	return env
}

func dialer(opiSpdkServer *Server) func(context.Context, string) (net.Conn, error) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	pb.RegisterMiddleendEncryptionServiceServer(server, opiSpdkServer)

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

var (
	bdevName          = "bdev-42"
	foundBdevResponse = `{"id":%d,"error":{"code":0,"message":""},"result":[` +
		`{"name":"bdev-42","block_size":4096,"num_blocks":256,"uuid":"9d7988c6-4b42-4196-a46f-a656a89deb36"}]}`
	keyOf128Bits      = []byte("0123456789abcdef")
	keyOf192Bits      = []byte("0123456789abcdef01234567")
	keyOf256Bits      = []byte("0123456789abcdef0123456789abcdef")
	keyOf384Bits      = []byte("0123456789abcdef0123456789abcdef0123456789abcdef")
	keyOf512bits      = []byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	encryptedVolumeID = "encrypted-volume-id"
	// Volumes for supported algorithms
	encryptedVolumeAesXts256 = pb.EncryptedVolume{
		VolumeId: &pc.ObjectKey{Value: bdevName},
		Key:      keyOf512bits,
		Cipher:   pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_256,
	}
	encryptedVolumeAesXts256InResponse = pb.EncryptedVolume{
		VolumeId: encryptedVolumeAesXts256.VolumeId,
	}
	encryptedVolumeAesXts128 = pb.EncryptedVolume{
		VolumeId: &pc.ObjectKey{Value: bdevName},
		Key:      keyOf256Bits,
		Cipher:   pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_128,
	}
	encryptedVolumeAesXts128InResponse = pb.EncryptedVolume{
		VolumeId: encryptedVolumeAesXts128.VolumeId,
	}

	// Volumes for not supported algorithms
	encryptedVolumeAesXts192 = pb.EncryptedVolume{
		VolumeId: &pc.ObjectKey{Value: bdevName},
		Key:      keyOf384Bits,
		Cipher:   pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_192,
	}
	encryptedVolumeAesCbc128 = pb.EncryptedVolume{
		VolumeId: &pc.ObjectKey{Value: bdevName},
		Key:      keyOf128Bits,
		Cipher:   pb.EncryptionType_ENCRYPTION_TYPE_AES_CBC_128,
	}
	encryptedVolumeAesCbc192 = pb.EncryptedVolume{
		VolumeId: &pc.ObjectKey{Value: bdevName},
		Key:      keyOf192Bits,
		Cipher:   pb.EncryptionType_ENCRYPTION_TYPE_AES_CBC_192,
	}
	encryptedVolumeAesCbc256 = pb.EncryptedVolume{
		VolumeId: &pc.ObjectKey{Value: bdevName},
		Key:      keyOf256Bits,
		Cipher:   pb.EncryptionType_ENCRYPTION_TYPE_AES_CBC_256,
	}
)

func TestMiddleEnd_CreateEncryptedVolume(t *testing.T) {
	tests := map[string]struct {
		in            *pb.CreateEncryptedVolumeRequest
		out           *pb.EncryptedVolume
		spdk          []string
		expectedInKey []byte
		expectedErr   error
		start         bool
	}{
		"nil request": {
			in:            nil,
			out:           nil,
			spdk:          []string{},
			expectedInKey: nil,
			expectedErr:   errMissingArgument,
			start:         false,
		},
		"nil EncryptedVolume": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: nil, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: nil,
			expectedErr:   errMissingArgument,
			start:         false,
		},
		"EncryptedVolume EncryptedVolumeId is ignored": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				Name:     "Some-ignored-id-value",
				VolumeId: encryptedVolumeAesXts256.VolumeId,
				Key:      encryptedVolumeAesXts256.Key,
				Cipher:   pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_256,
			}, EncryptedVolumeId: encryptedVolumeID},
			out: &pb.EncryptedVolume{
				Name:     encryptedVolumeID,
				VolumeId: encryptedVolumeAesXts256.VolumeId,
			},
			spdk:          []string{foundBdevResponse, `{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			expectedErr:   nil,
			start:         true,
		},
		"empty Key": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				VolumeId: encryptedVolumeAesXts256.VolumeId,
				Cipher:   pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_256,
				Key:      make([]byte, 0),
			}, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: nil,
			expectedErr:   errMissingArgument,
			start:         false,
		},
		"nil VolumeId": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				Key:    encryptedVolumeAesXts256.Key,
				Cipher: pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_256,
			}, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			expectedErr:   errMissingArgument,
			start:         false,
		},
		"empty VolumeId": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				VolumeId: &pc.ObjectKey{Value: ""},
				Key:      encryptedVolumeAesXts256.Key,
				Cipher:   pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_256,
			}, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			expectedErr:   errMissingArgument,
			start:         false,
		},
		"use AES_XTS_128 cipher": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts128, EncryptedVolumeId: encryptedVolumeID},
			out:           &encryptedVolumeAesXts128InResponse,
			spdk:          []string{foundBdevResponse, `{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts128.Key)),
			expectedErr:   nil,
			start:         true,
		},
		"use AES_XTS_192 cipher": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts192, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts192.Key)),
			expectedErr:   errNotSupportedCipher,
			start:         false,
		},
		"use AES_XTS_256 cipher": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts256, EncryptedVolumeId: encryptedVolumeID},
			out:           &encryptedVolumeAesXts256InResponse,
			spdk:          []string{foundBdevResponse, `{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			expectedErr:   nil,
			start:         true,
		},
		"use AES_CBC_128 cipher": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesCbc128, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesCbc128.Key)),
			expectedErr:   errNotSupportedCipher,
			start:         false,
		},
		"use AES_CBC_192 cipher": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesCbc192, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesCbc192.Key)),
			expectedErr:   errNotSupportedCipher,
			start:         false,
		},
		"use AES_CBC_256 cipher": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesCbc256, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesCbc256.Key)),
			expectedErr:   errNotSupportedCipher,
			start:         false,
		},
		"use UNSPECIFIED cipher": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				VolumeId: encryptedVolumeAesXts256.VolumeId,
				Key:      encryptedVolumeAesXts256.Key,
				Cipher:   pb.EncryptionType_ENCRYPTION_TYPE_UNSPECIFIED,
			}, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			expectedErr:   errNotSupportedCipher,
			start:         false,
		},
		"key of wrong size for AEX_XTS_256": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				VolumeId: encryptedVolumeAesXts256.VolumeId,
				Key:      []byte("1"),
				Cipher:   pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_256,
			}, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, 1),
			expectedErr:   errWrongKeySize,
			start:         false,
		},
		"key of wrong size for AEX_XTS_128": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				VolumeId: encryptedVolumeAesXts128.VolumeId,
				Key:      []byte("1"),
				Cipher:   encryptedVolumeAesXts128.Cipher,
			}, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, 1),
			expectedErr:   errWrongKeySize,
			start:         false,
		},
		"find bdev uuid by name internal SPDK failure": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts256, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{`{"id":%d,"error":{"code":-19,"message":"No such device"},"result":null}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			expectedErr:   spdk.ErrFailedSpdkCall,
			start:         true,
		},
		"find no bdev uuid by name": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts256, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			expectedErr:   spdk.ErrUnexpectedSpdkCallResult,
			start:         true,
		},
		"internal SPDK failure": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts256, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{foundBdevResponse, `{"id":%d,"error":{"code":1,"message":"some internal error"},"result":true}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			expectedErr:   spdk.ErrFailedSpdkCall,
			start:         true,
		},
		"SPDK result false": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts256, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{foundBdevResponse, `{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			expectedErr:   spdk.ErrUnexpectedSpdkCallResult,
			start:         true,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			testEnv := createTestEnvironment(test.start, test.spdk)
			defer testEnv.Close()
			var request *pb.CreateEncryptedVolumeRequest
			if test.in != nil {
				var ok bool
				// make a copy to prevent key overwriting in the original structures
				request, ok = proto.Clone(test.in).(*pb.CreateEncryptedVolumeRequest)
				if !ok {
					log.Panic("Failed to copy test structure for CreateEncryptedVolumeRequest")
				}
			}
			if test.out != nil {
				test.out.Name = encryptedVolumeID
			}

			response, err := testEnv.opiSpdkServer.CreateEncryptedVolume(testEnv.ctx, request)

			wantOut, _ := proto.Marshal(test.out)
			gotOut, _ := proto.Marshal(response)
			if !bytes.Equal(wantOut, gotOut) {
				t.Error("response: expected", test.out, "received", response)
			}
			if err != test.expectedErr {
				t.Error("error: expected", test.expectedErr, "received", err)
			}
			if request != nil && request.EncryptedVolume != nil {
				if !bytes.Equal(request.EncryptedVolume.Key, test.expectedInKey) {
					t.Error("input key after operation expected",
						test.expectedInKey, "received", request.EncryptedVolume.Key)
				}
				if request.EncryptedVolume.Cipher != pb.EncryptionType_ENCRYPTION_TYPE_UNSPECIFIED {
					t.Error("Expect in cipher set to EncryptionType_ENCRYPTION_TYPE_UNSPECIFIED, received",
						request.EncryptedVolume.Cipher)
				}
			}
		})
	}
}

func TestMiddleEnd_DeleteEncryptedVolume(t *testing.T) {
	tests := map[string]struct {
		in          *pb.DeleteEncryptedVolumeRequest
		spdk        []string
		expectedErr error
		start       bool
	}{

		"nil request": {
			in:          nil,
			spdk:        []string{},
			expectedErr: errMissingArgument,
			start:       false,
		},
		"valid delete encrypted volume request": {
			in:          &pb.DeleteEncryptedVolumeRequest{Name: bdevName},
			spdk:        []string{foundBdevResponse, `{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			expectedErr: nil,
			start:       true,
		},
		"find bdev uuid by name internal SPDK failure": {
			in:          &pb.DeleteEncryptedVolumeRequest{Name: bdevName},
			spdk:        []string{`{"id":%d,"error":{"code":-19,"message":"No such device"},"result":null}`},
			expectedErr: spdk.ErrFailedSpdkCall,
			start:       true,
		},
		"find no bdev uuid by name": {
			in:          &pb.DeleteEncryptedVolumeRequest{Name: bdevName},
			spdk:        []string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			expectedErr: spdk.ErrUnexpectedSpdkCallResult,
			start:       true,
		},
		"internal SPDK failure": {
			in:          &pb.DeleteEncryptedVolumeRequest{Name: bdevName},
			spdk:        []string{foundBdevResponse, `{"id":%d,"error":{"code":1,"message":"some internal error"},"result":true}`},
			expectedErr: spdk.ErrFailedSpdkCall,
			start:       true,
		},
		"SPDK result false": {
			in:          &pb.DeleteEncryptedVolumeRequest{Name: bdevName},
			spdk:        []string{foundBdevResponse, `{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			expectedErr: spdk.ErrUnexpectedSpdkCallResult,
			start:       true,
		},
		"delete non-existing encrypted volume with missing allowed": {
			in: &pb.DeleteEncryptedVolumeRequest{Name: bdevName, AllowMissing: true},
			spdk: []string{foundBdevResponse,
				`{"id":%d,"error":{"code":-32603,"message":"Could not find a crypto object for a given bdev, set using npi_bdev_set_keys"},"result":null}`},
			expectedErr: nil,
			start:       true,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			testEnv := createTestEnvironment(test.start, test.spdk)
			defer testEnv.Close()

			_, err := testEnv.opiSpdkServer.DeleteEncryptedVolume(testEnv.ctx, test.in)

			if err != test.expectedErr {
				t.Error("error: expected", test.expectedErr, "received", err)
			}
		})
	}
}
