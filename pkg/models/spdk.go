// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2023 Intel Corporation

// Package models holds definitions for SPDK json RPC structs
package models

// NpiBdevSetKeysParams holds the parameters required to set crypto keys
type NpiBdevSetKeysParams struct {
	UUID   string `json:"uuid"`
	Key    string `json:"key"`
	Key2   string `json:"key2"`
	Cipher string `json:"cipher"`
	Tweak  string `json:"tweak"`
}

// NpiBdevSetKeysResult is the result of setting crypto keys
type NpiBdevSetKeysResult bool

// NpiBdevClearKeysParams holds the parameters required to clear crypto keys
type NpiBdevClearKeysParams struct {
	UUID string `json:"uuid"`
}

// NpiBdevClearKeysResult is the result of clearing crypto keys
type NpiBdevClearKeysResult bool

// NpiQosBwIopsLimitParams holds the parameters required to set QoS limits
type NpiQosBwIopsLimitParams struct {
	Nqn          string `json:"nqn"`
	MaxReadIops  int    `json:"max_read_iops"`
	MaxWriteIops int    `json:"max_write_iops"`
	MaxReadBw    int    `json:"max_read_bw"`
	MaxWriteBw   int    `json:"max_write_bw"`
	MinReadBw    int    `json:"min_read_bw"`
	MinWriteBw   int    `json:"min_write_bw"`
}

// NpiQosBwIopsLimitResult is the result of setting QoS limits
type NpiQosBwIopsLimitResult bool
