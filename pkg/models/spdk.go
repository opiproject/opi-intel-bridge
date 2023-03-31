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
