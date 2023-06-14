# OPI gRPC to Intel SDK bridge third party repo

[![Linters](https://github.com/opiproject/opi-intel-bridge/actions/workflows/linters.yml/badge.svg)](https://github.com/opiproject/opi-intel-bridge/actions/workflows/linters.yml)
[![tests](https://github.com/opiproject/opi-intel-bridge/actions/workflows/go.yml/badge.svg)](https://github.com/opiproject/opi-intel-bridge/actions/workflows/go.yml)
[![Docker](https://github.com/opiproject/opi-intel-bridge/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/opiproject/opi-intel-bridge/actions/workflows/docker-publish.yml)
[![License](https://img.shields.io/github/license/opiproject/opi-intel-bridge?style=flat-square&color=blue&label=License)](https://github.com/opiproject/opi-intel-bridge/blob/master/LICENSE)
[![codecov](https://codecov.io/gh/opiproject/opi-intel-bridge/branch/main/graph/badge.svg)](https://codecov.io/gh/opiproject/opi-intel-bridge)
[![Go Report Card](https://goreportcard.com/badge/github.com/opiproject/opi-intel-bridge)](https://goreportcard.com/report/github.com/opiproject/opi-intel-bridge)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/opiproject/opi-intel-bridge)
[![Pulls](https://img.shields.io/docker/pulls/opiproject/opi-intel-bridge.svg?logo=docker&style=flat&label=Pulls)](https://hub.docker.com/r/opiproject/opi-intel-bridge)
[![Last Release](https://img.shields.io/github/v/release/opiproject/opi-intel-bridge?label=Latest&style=flat-square&logo=go)](https://github.com/opiproject/opi-intel-bridge/releases)
[![GitHub stars](https://img.shields.io/github/stars/opiproject/opi-intel-bridge.svg?style=flat-square&label=github%20stars)](https://github.com/opiproject/opi-intel-bridge)
[![GitHub Contributors](https://img.shields.io/github/contributors/opiproject/opi-intel-bridge.svg?style=flat-square)](https://github.com/opiproject/opi-intel-bridge/graphs/contributors)

This is a Intel app (bridge) to OPI APIs for storage, inventory, ipsec and networking (future).

## I Want To Contribute

This project welcomes contributions and suggestions.  We are happy to have the Community involved via submission of **Issues and Pull Requests** (with substantive content or even just fixes). We are hoping for the documents, test framework, etc. to become a community process with active engagement.  PRs can be reviewed by by any number of people, and a maintainer may accept.

See [CONTRIBUTING](https://github.com/opiproject/opi/blob/main/CONTRIBUTING.md) and [GitHub Basic Process](https://github.com/opiproject/opi/blob/main/doc-github-rules.md) for more details.

## Getting started

build like this:

```bash
go build -v -o /opi-intel-bridge ./cmd/...
```

import like this:

```go
import "github.com/opiproject/opi-intel-bridge/pkg/frontend"
```

## Using docker

on DPU/IPU (i.e. with IP=10.10.10.1) run

```bash
$ docker run --rm -it -v /var/tmp/:/var/tmp/ -p 50051:50051 ghcr.io/opiproject/opi-intel-bridge:main
2022/11/29 00:03:55 plugin serevr is &{{}}
2022/11/29 00:03:55 server listening at [::]:50051
```

on X86 management VM run

reflection

```bash
$ docker run --network=host --rm -it namely/grpc-cli ls --json_input --json_output localhost:50051
grpc.reflection.v1alpha.ServerReflection
opi_api.inventory.v1.InventorySvc
opi_api.security.v1.IPsec
opi_api.storage.v1.AioControllerService
opi_api.storage.v1.FrontendNvmeService
opi_api.storage.v1.FrontendVirtioBlkService
opi_api.storage.v1.FrontendVirtioScsiService
opi_api.storage.v1.MiddleendEncryptionService
opi_api.storage.v1.MiddleendQosVolumeService
opi_api.storage.v1.NVMfRemoteControllerService
opi_api.storage.v1.NullDebugService
```

full test suite

```bash
docker run --rm -it --network=host docker.io/opiproject/godpu:main get --addr="10.10.10.10:50051"
docker run --rm -it --network=host docker.io/opiproject/godpu:main storagetest --addr="10.10.10.10:50051"
docker run --rm -it --network=host docker.io/opiproject/godpu:main test --addr=10.10.10.10:50151 --pingaddr=8.8.8.1"
```

or manually

```bash
docker run --network=host --rm -it namely/grpc-cli ls   --json_input --json_output 10.10.10.10:50051 -l
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNvmeSubsystem "{nvme_subsystem : {spec : {nqn: 'nqn.2022-09.io.spdk:opitest2', serial_number: 'myserial2', model_number: 'mymodel2', max_namespaces: 11} }, nvme_subsystem_id : 'subsystem2' }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNvmeSubsystems "{}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNvmeSubsystem "{name : '//storage.opiproject.org/volumes/subsystem2'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNvmeController "{nvme_controller : {spec : {nvme_controller_id: 2, subsystem_id : { value : '//storage.opiproject.org/volumes/subsystem2' }, pcie_id : {physical_function : 0}, max_nsq:5, max_ncq:5 } }, nvme_controller_id : 'controller1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNvmeControllers "{parent : '//storage.opiproject.org/volumes/subsystem2'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNvmeController "{name : '//storage.opiproject.org/volumes/controller1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNvmeNamespace "{nvme_namespace : {spec : {subsystem_id : { value : '//storage.opiproject.org/volumes/subsystem2' }, volume_id : { value : 'Malloc0' }, 'host_nsid' : '10', uuid:{value : '1b4e28ba-2fa1-11d2-883f-b9a761bde3fb'}, nguid: '1b4e28ba-2fa1-11d2-883f-b9a761bde3fb', eui64: 1967554867335598546 } }, nvme_namespace_id: 'namespace1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNvmeNamespaces "{parent : '//storage.opiproject.org/volumes/subsystem2'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNvmeNamespace "{name : '//storage.opiproject.org/volumes/namespace1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 NvmeNamespaceStats "{namespace_id : {value : '//storage.opiproject.org/volumes/namespace1'} }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNVMfRemoteController "{nv_mf_remote_controller : {traddr:'11.11.11.2', subnqn:'nqn.2016-06.com.opi.spdk.target0', trsvcid:'4444', trtype:'NVME_TRANSPORT_TCP', adrfam:'NVMF_ADRFAM_IPV4', hostnqn:'nqn.2014-08.org.nvmexpress:uuid:feb98abe-d51f-40c8-b348-2753f3571d3c'}, nv_mf_remote_controller_id: 'NvmeTcp12'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNVMfRemoteControllers "{}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNVMfRemoteController "{name: '//storage.opiproject.org/volumes/NvmeTcp12'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNVMfRemoteController "{name: '//storage.opiproject.org/volumes/NvmeTcp12'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNvmeNamespace "{name : '//storage.opiproject.org/volumes/namespace1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNvmeController "{name : '//storage.opiproject.org/volumes/controller1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNvmeSubsystem "{name : '//storage.opiproject.org/volumes/subsystem2'}"
```
