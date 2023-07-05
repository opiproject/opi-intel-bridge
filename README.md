# OPI gRPC to Intel SDK bridge

[![Linters](https://github.com/opiproject/opi-intel-bridge/actions/workflows/linters.yml/badge.svg)](https://github.com/opiproject/opi-intel-bridge/actions/workflows/linters.yml)
[![CodeQL](https://github.com/opiproject/opi-intel-bridge/actions/workflows/codeql.yml/badge.svg)](https://github.com/opiproject/opi-intel-bridge/actions/workflows/codeql.yml)
[![tests](https://github.com/opiproject/opi-intel-bridge/actions/workflows/go.yml/badge.svg)](https://github.com/opiproject/opi-intel-bridge/actions/workflows/go.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/opiproject/opi-intel-bridge/badge)](https://securityscorecards.dev/viewer/?platform=github.com&org=opiproject&repo=opi-intel-bridge)
[![codecov](https://codecov.io/gh/opiproject/opi-intel-bridge/branch/main/graph/badge.svg)](https://codecov.io/gh/opiproject/opi-intel-bridge)
[![Go Report Card](https://goreportcard.com/badge/github.com/opiproject/opi-intel-bridge)](https://goreportcard.com/report/github.com/opiproject/opi-intel-bridge)

[Download 🚀](https://github.com/orgs/opiproject/packages?repo_name=opi-intel-bridge) ·
[Report issue 🐞](https://github.com/opiproject/opi-intel-bridge/issues/new/choose) ·
[Contribute 👋](#i-want-to-contribute)

----

This is an Intel bridge to OPI APIs. Currently it supports storage APIs and is subject to extension for other domains including inventory, ipsec and networking.
The intel-opi-bridge (further bridge) acts as a gRPC server for xPU management and configuration.

The diagram below illustrates main system components of an exemplary NVMe-oF initiator deployment. The bridge (in blue) runs on an xPU and translates OPI API commands to appropriate sequences of Intel SDK instructions. Ultimately, two emulated NVMe storage devices are exposed to the host. These devices are backed by an "over Fabrics"-connection to some remote storage backends while the host has an illusion of accessing locally attached storage and can run standard/unmodified apps/drivers to access it.

![opi-intel-bridge system overview](doc/images/opi-intel-bridge_system-overview.png "opi-intel-bridge system overview")\
*Fig. 1 - System components in NVMe-oF scenario*

## Quickstart

This section outlines the basic steps to get you up-and-running with the bridge and shows some examples of its usage to expose storage devices to the host, set bandwidth/rate limiters on them or enable data-at-rest crypto. The steps are mainly executed as gRPC commands to the bridge but may also involve some host-side interactions or initial xPU-side setup.

> **Note** \
It is assumed that Intel IPU is already properly set up to be used with Intel OPI bridge.

The following variables are used throughout this document:

| Variable    | Description                                                                                                                                      |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| BRIDGE_IP   | opi-intel-bridge gRPC listening IP address e.g. 10.10.10.10 or localhost                                                                         |
| BRIDGE_PORT | opi-intel-bridge gRPC listening port e.g. 50051                                                                                                  |
| BRIDGE_ADDR | BRIDGE_IP:BRIDGE_PORT                                                                                                                            |
| PF_BDF      | physical function PCI address e.g. 0000:3b:00.1                                                                                                  |
| VF_BDF      | virtual function PCI address e.g. 0000:40:00.0 can be found in pf's virtfn\<X\> where X equals to virtual_function in CreateNvmeController minus 1 |
| TARGET_IP   | storage target ip address                                                                                                                        |
| TARGET_PORT | storage target port                                                                                                                              |

### Build and import

To build the solution execute

```bash
go build -v -o /opi-intel-bridge ./cmd/...
```

To import the bridge within another go package or module use

```go
import "github.com/opiproject/opi-intel-bridge/pkg/frontend"
import "github.com/opiproject/opi-intel-bridge/pkg/middleend"
```

### Usage

On xPU run

```bash
$ docker run --rm -it -v /var/tmp/:/var/tmp/ -p $BRIDGE_PORT:$BRIDGE_PORT ghcr.io/opiproject/opi-intel-bridge:main

2023/07/03 11:04:30 Connection to SPDK will be via: unix detected from /var/tmp/spdk.sock
2023/07/03 11:04:30 server listening at [::]:50051
```

To send commands to the bridge, grpc_cli tool is used. It can be used as a containerized or a native version. If containerized version is preferable, then an alias can be defined as follows

```bash
alias grpc_cli="docker run --network=host --rm -it namely/grpc-cli"
```

On management machine run below command to check bridge availability and reflection capabilities

```bash
$ grpc_cli ls --json_input --json_output $BRIDGE_ADDR

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

or specify commands manually

```bash
# PF creation
grpc_cli call --json_input --json_output $BRIDGE_ADDR CreateNvmeSubsystem "{nvme_subsystem : {spec : {nqn: 'nqn.2022-09.io.spdk:opitest2', serial_number: 'myserial2', model_number: 'mymodel2', max_namespaces: 11} }, nvme_subsystem_id : 'subsystem2' }"
grpc_cli call --json_input --json_output $BRIDGE_ADDR ListNvmeSubsystems "{parent : 'todo'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR GetNvmeSubsystem "{name : '//storage.opiproject.org/volumes/subsystem2'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR CreateNvmeController "{nvme_controller : {spec : {nvme_controller_id: 2, subsystem_id : { value : '//storage.opiproject.org/volumes/subsystem2' }, pcie_id : {physical_function : 0}, max_nsq:5, max_ncq:5 } }, nvme_controller_id : 'controller1'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR ListNvmeControllers "{parent : '//storage.opiproject.org/volumes/subsystem2'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR GetNvmeController "{name : '//storage.opiproject.org/volumes/controller1'}"

# VF creation on PF0
grpc_cli call --json_input --json_output $BRIDGE_ADDR CreateNvmeSubsystem "{nvme_subsystem : {spec : {nqn: 'nqn.2022-09.io.spdk:opitest3', serial_number: 'mev-opi-serial', model_number: 'mev-opi-model', max_namespaces: 11} }, nvme_subsystem_id : 'subsystem03' }"
grpc_cli call --json_input --json_output $BRIDGE_ADDR CreateNvmeController "{nvme_controller : {spec : {nvme_controller_id: 2, subsystem_id : { value : '//storage.opiproject.org/volumes/subsystem03' }, pcie_id : {physical_function : 0, virtual_function : 3}, max_nsq:5, max_ncq:5 } }, nvme_controller_id : 'controller3'}"

# Connect to storage-target
grpc_cli call --json_input --json_output $BRIDGE_ADDR CreateNVMfRemoteController "{nv_mf_remote_controller : {multipath: 'NVME_MULTIPATH_MULTIPATH'}, nv_mf_remote_controller_id: 'nvmetcp12'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR ListNVMfRemoteControllers "{}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR GetNVMfRemoteController "{name: '//storage.opiproject.org/volumes/nvmetcp12'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR CreateNVMfPath "{nv_mf_path : {controller_id: {value: '//storage.opiproject.org/volumes/nvmetcp12'}, traddr:'11.11.11.2', subnqn:'nqn.2016-06.com.opi.spdk.target0', trsvcid:'4444', trtype:'NVME_TRANSPORT_TCP', adrfam:'NVMF_ADRFAM_IPV4', hostnqn:'nqn.2014-08.org.nvmexpress:uuid:feb98abe-d51f-40c8-b348-2753f3571d3c'}, nv_mf_path_id: 'nvmetcp12path0'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR ListNVMfPaths "{parent : 'todo'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR GetNVMfPath "{name: '//storage.opiproject.org/volumes/nvmetcp12path0'}"

# Create QoS volume
grpc_cli call --json_input --json_output $BRIDGE_ADDR CreateQosVolume "{'qos_volume' : {'volume_id' : { 'value':'nvmetcp12n1'}, 'max_limit' : { 'rw_iops_kiops': 3 } }, 'qos_volume_id' : 'qosnvmetcp12n1' }"

# Create encrypted volume
grpc_cli call --json_input --json_output $BRIDGE_ADDR CreateEncryptedVolume "{'encrypted_volume': { 'cipher': 'ENCRYPTION_TYPE_AES_XTS_128', 'volume_id': { 'value': 'nvmetcp12n1'}, 'key': 'MDAwMTAyMDMwNDA1MDYwNzA4MDkwYTBiMGMwZDBlMGY='}, 'encrypted_volume_id': 'encnvmetcp12n1' }"

# Create namespace
grpc_cli call --json_input --json_output $BRIDGE_ADDR CreateNvmeNamespace "{nvme_namespace : {spec : {subsystem_id : { value : '//storage.opiproject.org/volumes/subsystem2' }, volume_id : { value : 'nvmetcp12n1' }, 'host_nsid' : '10', uuid:{value : '1b4e28ba-2fa1-11d2-883f-b9a761bde3fb'}, nguid: '1b4e28ba-2fa1-11d2-883f-b9a761bde3fb', eui64: 1967554867335598546 } }, nvme_namespace_id: 'namespace1'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR ListNvmeNamespaces "{parent : '//storage.opiproject.org/volumes/subsystem2'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR GetNvmeNamespace "{name : '//storage.opiproject.org/volumes/namespace1'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR NvmeNamespaceStats "{namespace_id : {value : '//storage.opiproject.org/volumes/namespace1'} }"

# Delete namespace
grpc_cli call --json_input --json_output $BRIDGE_ADDR DeleteNvmeNamespace "{name : '//storage.opiproject.org/volumes/namespace1'}"

# Delete encrypted volume
grpc_cli call --json_input --json_output $BRIDGE_ADDR DeleteEncryptedVolume "{'name': '//storage.opiproject.org/volumes/encnvmetcp12n1'}"

# Delete QoS volume
grpc_cli call --json_input --json_output $BRIDGE_ADDR DeleteQosVolume "{name : '//storage.opiproject.org/volumes/qosnvmetcp12n1'}"

# Disconnect from storage-target
grpc_cli call --json_input --json_output $BRIDGE_ADDR DeleteNVMfPath "{name: '//storage.opiproject.org/volumes/nvmetcp12path0'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR DeleteNVMfRemoteController "{name: '//storage.opiproject.org/volumes/nvmetcp12'}"

# Delete VF
grpc_cli call --json_input --json_output $BRIDGE_ADDR DeleteNvmeController "{name : '//storage.opiproject.org/volumes/controller3'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR DeleteNvmeSubsystem "{name : '//storage.opiproject.org/volumes/subsystem03'}"

# Delete PF
grpc_cli call --json_input --json_output $BRIDGE_ADDR DeleteNvmeController "{name : '//storage.opiproject.org/volumes/controller1'}"
grpc_cli call --json_input --json_output $BRIDGE_ADDR DeleteNvmeSubsystem "{name : '//storage.opiproject.org/volumes/subsystem2'}"
```

To observe devices on host:

After PF is created

```bash
# Bind driver to PF
modprobe nvme
cd /sys/bus/pci/devices/$PF_BDF
echo 'nvme' > ./driver_override
echo $PF_BDF > /sys/bus/pci/drivers/nvme/bind

# Allocate resources and prepare for VF creation
echo 0 > ./sriov_drivers_autoprobe
echo 4 > ./sriov_numvfs
```

After VF is created

```bash
cd /sys/bus/pci/devices/$PF_BDF
echo 'nvme' > ./virtfn0/driver_override
echo $VF_BDF > /sys/bus/pci/drivers/nvme/bind
```

Before VF is deleted

```bash
cd /sys/bus/pci/devices/$PF_BDF
echo $VF_BDF > /sys/bus/pci/drivers/nvme/unbind
echo '(null)' > ./virtfn2/driver_override
```

Before PF is deleted

```bash
cd /sys/bus/pci/devices/$PF_BDF
echo $PF_BDF > /sys/bus/pci/drivers/nvme/unbind
echo '(null)' > ./driver_override
```

## I Want To Contribute

This project welcomes contributions and suggestions.  We are happy to have the Community involved via submission of **Issues and Pull Requests** (with substantive content or even just fixes). We are hoping for the documents, test framework, etc. to become a community process with active engagement.  PRs can be reviewed by any number of people, and a maintainer may accept.

See [CONTRIBUTING](https://github.com/opiproject/opi/blob/main/CONTRIBUTING.md) and [GitHub Basic Process](https://github.com/opiproject/opi/blob/main/doc-github-rules.md) for more details.
