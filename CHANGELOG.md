# Changelog

This file lists all notable changes in the project per release. It is
also continously updated with already published but not yet released work.

Release versioning in this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Entries in this file are grouped into several categories:

* Added
* Changed
* Deprecated
* Fixed
* Removed
* Security

## [Unreleased]

## [0.2.0] - 2024-01-12

### Added

* Usage of SPDK Acceleration Framework for encryption ([#465](https://github.com/opiproject/opi-intel-bridge/pull/465)).
* Setting number of queues per NVMe controller ([#410](https://github.com/opiproject/opi-intel-bridge/pull/410)).
* Otel monitoring ([#317](https://github.com/opiproject/opi-intel-bridge/pull/317)).
* Nvme/TCP as a target ([#314](https://github.com/opiproject/opi-intel-bridge/pull/314)).
* HTTP gateway for inventory service ([264](https://github.com/opiproject/opi-intel-bridge/pull/264)).
* Virtio-blk support ([#234](https://github.com/opiproject/opi-intel-bridge/pull/234)).

### Security

* Nvme/TCP PSK support ([#318](https://github.com/opiproject/opi-intel-bridge/pull/318)).

## [0.1.0] - 2023-07-14

### Added

* Changelog file ([#167](https://github.com/opiproject/opi-intel-bridge/pull/167)).
* Documentation with usage examples using OPI commands ([#164](https://github.com/opiproject/opi-intel-bridge/pull/164)).
* Enablement of QoS on NVMe device level ([#103](https://github.com/opiproject/opi-intel-bridge/pull/103), [#104](https://github.com/opiproject/opi-intel-bridge/pull/104)).
* Enablement of QoS on volume level ([#92](https://github.com/opiproject/opi-intel-bridge/pull/92)).
* Enablement of HW-accelerated DARE crypto on volume level ([#79](https://github.com/opiproject/opi-intel-bridge/pull/79), [#84](https://github.com/opiproject/opi-intel-bridge/pull/84)).
* Dynamic exposition of NVMe storage devices to the host ([#15](https://github.com/opiproject/opi-intel-bridge/pull/15), [#43](https://github.com/opiproject/opi-intel-bridge/pull/43)).

### Security

* Enablement of mTLS-secured gRPC connection between host and IPU ([#165](https://github.com/opiproject/opi-intel-bridge/pull/165)).
* Hardening of project security through enablement of security scans ([#137](https://github.com/opiproject/opi-intel-bridge/pull/137), [#149](https://github.com/opiproject/opi-intel-bridge/pull/149)).

[unreleased]: https://github.com/opiproject/opi-intel-bridge/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/opiproject/opi-intel-bridge/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/opiproject/opi-intel-bridge/releases/tag/v0.1.0
