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
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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

func createTestEnvironment(spdkResponses []string) *testEnv {
	env := &testEnv{}
	env.testSocket = server.GenerateSocketName("middleend")
	env.ln, env.jsonRPC = server.CreateTestSpdkServer(env.testSocket, spdkResponses)
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
		VolumeNameRef: bdevName,
		Key:           keyOf512bits,
		Cipher:        pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_256,
	}
	encryptedVolumeAesXts256InResponse = pb.EncryptedVolume{
		VolumeNameRef: encryptedVolumeAesXts256.VolumeNameRef,
	}
	encryptedVolumeAesXts128 = pb.EncryptedVolume{
		VolumeNameRef: bdevName,
		Key:           keyOf256Bits,
		Cipher:        pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_128,
	}
	encryptedVolumeAesXts128InResponse = pb.EncryptedVolume{
		VolumeNameRef: encryptedVolumeAesXts128.VolumeNameRef,
	}

	// Volumes for not supported algorithms
	encryptedVolumeAesXts192 = pb.EncryptedVolume{
		VolumeNameRef: bdevName,
		Key:           keyOf384Bits,
		Cipher:        pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_192,
	}
	encryptedVolumeAesCbc128 = pb.EncryptedVolume{
		VolumeNameRef: bdevName,
		Key:           keyOf128Bits,
		Cipher:        pb.EncryptionType_ENCRYPTION_TYPE_AES_CBC_128,
	}
	encryptedVolumeAesCbc192 = pb.EncryptedVolume{
		VolumeNameRef: bdevName,
		Key:           keyOf192Bits,
		Cipher:        pb.EncryptionType_ENCRYPTION_TYPE_AES_CBC_192,
	}
	encryptedVolumeAesCbc256 = pb.EncryptedVolume{
		VolumeNameRef: bdevName,
		Key:           keyOf256Bits,
		Cipher:        pb.EncryptionType_ENCRYPTION_TYPE_AES_CBC_256,
	}
)

func TestMiddleEnd_CreateEncryptedVolume(t *testing.T) {
	tests := map[string]struct {
		in            *pb.CreateEncryptedVolumeRequest
		out           *pb.EncryptedVolume
		spdk          []string
		expectedInKey []byte
		errCode       codes.Code
		errMsg        string
		existBefore   bool
	}{
		"illegal resource_id": {
			in: &pb.CreateEncryptedVolumeRequest{
				EncryptedVolume: &encryptedVolumeAesXts256, EncryptedVolumeId: "CapitalLettersNotAllowed",
			},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			errCode:       status.Convert(errMalformedArgument).Code(),
			errMsg:        status.Convert(errMalformedArgument).Message(),
			existBefore:   false,
		},
		"nil request": {
			in:            nil,
			out:           nil,
			spdk:          []string{},
			expectedInKey: nil,
			errCode:       status.Convert(errMissingArgument).Code(),
			errMsg:        status.Convert(errMissingArgument).Message(),
			existBefore:   false,
		},
		"nil EncryptedVolume": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: nil, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: nil,
			errCode:       status.Convert(errMissingArgument).Code(),
			errMsg:        status.Convert(errMissingArgument).Message(),
			existBefore:   false,
		},
		"EncryptedVolume EncryptedVolumeId is ignored": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				Name:          "Some-ignored-id-value",
				VolumeNameRef: encryptedVolumeAesXts256.VolumeNameRef,
				Key:           encryptedVolumeAesXts256.Key,
				Cipher:        pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_256,
			}, EncryptedVolumeId: encryptedVolumeID},
			out: &pb.EncryptedVolume{
				Name:          encryptedVolumeID,
				VolumeNameRef: encryptedVolumeAesXts256.VolumeNameRef,
			},
			spdk:          []string{foundBdevResponse, `{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			errCode:       codes.OK,
			errMsg:        "",
			existBefore:   false,
		},
		"empty Key": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				VolumeNameRef: encryptedVolumeAesXts256.VolumeNameRef,
				Cipher:        pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_256,
				Key:           make([]byte, 0),
			}, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: nil,
			errCode:       status.Convert(errMissingArgument).Code(),
			errMsg:        status.Convert(errMissingArgument).Message(),
			existBefore:   false,
		},
		"nil VolumeId": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				Key:    encryptedVolumeAesXts256.Key,
				Cipher: pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_256,
			}, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			errCode:       status.Convert(errMissingArgument).Code(),
			errMsg:        status.Convert(errMissingArgument).Message(),
			existBefore:   false,
		},
		"empty VolumeId": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				VolumeNameRef: "",
				Key:           encryptedVolumeAesXts256.Key,
				Cipher:        pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_256,
			}, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			errCode:       status.Convert(errMissingArgument).Code(),
			errMsg:        status.Convert(errMissingArgument).Message(),
			existBefore:   false,
		},
		"use AES_XTS_128 cipher": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts128, EncryptedVolumeId: encryptedVolumeID},
			out:           &encryptedVolumeAesXts128InResponse,
			spdk:          []string{foundBdevResponse, `{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts128.Key)),
			errCode:       codes.OK,
			errMsg:        "",
			existBefore:   false,
		},
		"use AES_XTS_192 cipher": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts192, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts192.Key)),
			errCode:       status.Convert(errNotSupportedCipher).Code(),
			errMsg:        status.Convert(errNotSupportedCipher).Message(),
			existBefore:   false,
		},
		"use AES_XTS_256 cipher": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts256, EncryptedVolumeId: encryptedVolumeID},
			out:           &encryptedVolumeAesXts256InResponse,
			spdk:          []string{foundBdevResponse, `{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			errCode:       codes.OK,
			errMsg:        "",
			existBefore:   false,
		},
		"use AES_CBC_128 cipher": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesCbc128, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesCbc128.Key)),
			errCode:       status.Convert(errNotSupportedCipher).Code(),
			errMsg:        status.Convert(errNotSupportedCipher).Message(),
			existBefore:   false,
		},
		"use AES_CBC_192 cipher": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesCbc192, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesCbc192.Key)),
			errCode:       status.Convert(errNotSupportedCipher).Code(),
			errMsg:        status.Convert(errNotSupportedCipher).Message(),
			existBefore:   false,
		},
		"use AES_CBC_256 cipher": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesCbc256, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesCbc256.Key)),
			errCode:       status.Convert(errNotSupportedCipher).Code(),
			errMsg:        status.Convert(errNotSupportedCipher).Message(),
			existBefore:   false,
		},
		"use UNSPECIFIED cipher": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				VolumeNameRef: encryptedVolumeAesXts256.VolumeNameRef,
				Key:           encryptedVolumeAesXts256.Key,
				Cipher:        pb.EncryptionType_ENCRYPTION_TYPE_UNSPECIFIED,
			}, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			errCode:       status.Convert(errNotSupportedCipher).Code(),
			errMsg:        status.Convert(errNotSupportedCipher).Message(),
			existBefore:   false,
		},
		"key of wrong size for AEX_XTS_256": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				VolumeNameRef: encryptedVolumeAesXts256.VolumeNameRef,
				Key:           []byte("1"),
				Cipher:        pb.EncryptionType_ENCRYPTION_TYPE_AES_XTS_256,
			}, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, 1),
			errCode:       status.Convert(errWrongKeySize).Code(),
			errMsg:        status.Convert(errWrongKeySize).Message(),
			existBefore:   false,
		},
		"key of wrong size for AEX_XTS_128": {
			in: &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &pb.EncryptedVolume{
				VolumeNameRef: encryptedVolumeAesXts128.VolumeNameRef,
				Key:           []byte("1"),
				Cipher:        encryptedVolumeAesXts128.Cipher,
			}, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, 1),
			errCode:       status.Convert(errWrongKeySize).Code(),
			errMsg:        status.Convert(errWrongKeySize).Message(),
			existBefore:   false,
		},
		"find bdev uuid by name internal SPDK failure": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts256, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{`{"id":%d,"error":{"code":-19,"message":"No such device"},"result":null}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			errCode:       status.Convert(spdk.ErrFailedSpdkCall).Code(),
			errMsg:        status.Convert(spdk.ErrFailedSpdkCall).Message(),
			existBefore:   false,
		},
		"find no bdev uuid by name": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts256, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			errCode:       status.Convert(spdk.ErrUnexpectedSpdkCallResult).Code(),
			errMsg:        status.Convert(spdk.ErrUnexpectedSpdkCallResult).Message(),
			existBefore:   false,
		},
		"internal SPDK failure": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts256, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{foundBdevResponse, `{"id":%d,"error":{"code":1,"message":"some internal error"},"result":true}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			errCode:       status.Convert(spdk.ErrFailedSpdkCall).Code(),
			errMsg:        status.Convert(spdk.ErrFailedSpdkCall).Message(),
			existBefore:   false,
		},
		"SPDK result false": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts256, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{foundBdevResponse, `{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			errCode:       status.Convert(spdk.ErrUnexpectedSpdkCallResult).Code(),
			errMsg:        status.Convert(spdk.ErrUnexpectedSpdkCallResult).Message(),
			existBefore:   false,
		},
		"volume already exists": {
			in:            &pb.CreateEncryptedVolumeRequest{EncryptedVolume: &encryptedVolumeAesXts256, EncryptedVolumeId: encryptedVolumeID},
			out:           nil,
			spdk:          []string{},
			expectedInKey: make([]byte, len(encryptedVolumeAesXts256.Key)),
			errCode:       status.Convert(errAlreadyExists).Code(),
			errMsg:        status.Convert(errAlreadyExists).Message(),
			existBefore:   true,
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()
			var request *pb.CreateEncryptedVolumeRequest
			if tt.in != nil {
				var ok bool
				// make a copy to prevent key overwriting in the original structures
				request, ok = proto.Clone(tt.in).(*pb.CreateEncryptedVolumeRequest)
				if !ok {
					log.Panic("Failed to copy test structure for CreateEncryptedVolumeRequest")
				}
			}
			fullname := server.ResourceIDToVolumeName(encryptedVolumeID)
			if tt.out != nil {
				tt.out.Name = fullname
			}
			if tt.existBefore {
				testEnv.opiSpdkServer.volumes.encryptedVolumes[fullname] =
					request.EncryptedVolume.VolumeNameRef
			}

			response, err := testEnv.opiSpdkServer.CreateEncryptedVolume(testEnv.ctx, request)

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
				t.Errorf("expect grpc error status")
			}

			if request.GetEncryptedVolume() != nil {
				if !bytes.Equal(request.EncryptedVolume.Key, tt.expectedInKey) {
					t.Error("input key after operation expected",
						tt.expectedInKey, "received", request.EncryptedVolume.Key)
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
	fullname := server.ResourceIDToVolumeName(encryptedVolumeID)
	tests := map[string]struct {
		in          *pb.DeleteEncryptedVolumeRequest
		spdk        []string
		errCode     codes.Code
		errMsg      string
		existBefore bool
		existAfter  bool
	}{
		"nil request": {
			in:          nil,
			spdk:        []string{},
			errCode:     status.Convert(errMissingArgument).Code(),
			errMsg:      status.Convert(errMissingArgument).Message(),
			existBefore: false,
			existAfter:  false,
		},
		"valid delete encrypted volume request": {
			in:          &pb.DeleteEncryptedVolumeRequest{Name: fullname},
			spdk:        []string{foundBdevResponse, `{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode:     codes.OK,
			errMsg:      "",
			existBefore: true,
			existAfter:  false,
		},
		"find bdev uuid by name internal SPDK failure": {
			in:          &pb.DeleteEncryptedVolumeRequest{Name: fullname},
			spdk:        []string{`{"id":%d,"error":{"code":-19,"message":"No such device"},"result":null}`},
			errCode:     status.Convert(spdk.ErrFailedSpdkCall).Code(),
			errMsg:      status.Convert(spdk.ErrFailedSpdkCall).Message(),
			existBefore: true,
			existAfter:  true,
		},
		"find no bdev uuid by name": {
			in:          &pb.DeleteEncryptedVolumeRequest{Name: fullname},
			spdk:        []string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			errCode:     status.Convert(spdk.ErrUnexpectedSpdkCallResult).Code(),
			errMsg:      status.Convert(spdk.ErrUnexpectedSpdkCallResult).Message(),
			existBefore: true,
			existAfter:  true,
		},
		"internal SPDK failure": {
			in:          &pb.DeleteEncryptedVolumeRequest{Name: fullname},
			spdk:        []string{foundBdevResponse, `{"id":%d,"error":{"code":1,"message":"some internal error"},"result":true}`},
			errCode:     status.Convert(spdk.ErrFailedSpdkCall).Code(),
			errMsg:      status.Convert(spdk.ErrFailedSpdkCall).Message(),
			existBefore: true,
			existAfter:  true,
		},
		"SPDK result false": {
			in:          &pb.DeleteEncryptedVolumeRequest{Name: fullname},
			spdk:        []string{foundBdevResponse, `{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			errCode:     status.Convert(spdk.ErrUnexpectedSpdkCallResult).Code(),
			errMsg:      status.Convert(spdk.ErrUnexpectedSpdkCallResult).Message(),
			existBefore: true,
			existAfter:  true,
		},
		"delete non-existing encrypted volume with missing allowed": {
			in:          &pb.DeleteEncryptedVolumeRequest{Name: fullname, AllowMissing: true},
			spdk:        []string{},
			errCode:     codes.OK,
			errMsg:      "",
			existBefore: false,
			existAfter:  false,
		},
		"delete non-existing encrypted volume without missing allowed": {
			in:          &pb.DeleteEncryptedVolumeRequest{Name: fullname, AllowMissing: false},
			spdk:        []string{},
			errCode:     status.Convert(errVolumeNotFound).Code(),
			errMsg:      status.Convert(errVolumeNotFound).Message(),
			existBefore: false,
			existAfter:  false,
		},
		"malformed name": {
			in:          &pb.DeleteEncryptedVolumeRequest{Name: server.ResourceIDToVolumeName("-ABC-DEF"), AllowMissing: false},
			spdk:        []string{},
			errCode:     status.Convert(errMalformedArgument).Code(),
			errMsg:      status.Convert(errMalformedArgument).Message(),
			existBefore: false,
			existAfter:  false,
		},
	}
	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()
			if tt.existBefore {
				testEnv.opiSpdkServer.volumes.encryptedVolumes[fullname] = bdevName
			}
			request := server.ProtoClone(tt.in)

			_, err := testEnv.opiSpdkServer.DeleteEncryptedVolume(testEnv.ctx, request)

			if er, ok := status.FromError(err); ok {
				if er.Code() != tt.errCode {
					t.Error("error code: expected", tt.errCode, "received", er.Code())
				}
				if er.Message() != tt.errMsg {
					t.Error("error message: expected", tt.errMsg, "received", er.Message())
				}
			} else {
				t.Errorf("expect grpc error status")
			}
			_, ok := testEnv.opiSpdkServer.volumes.encryptedVolumes[tt.in.GetName()]
			if tt.existAfter != ok {
				t.Error("expect Encrypted volume exist", tt.existAfter, "received", ok)
			}
		})
	}
}
