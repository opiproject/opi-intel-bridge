// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Intel Corporation, or its subsidiaries.
// Copyright (C) 2023 Nordix Foundation.
//
//nolint:all
package p4translation

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/opiproject/opi-evpn-bridge/pkg/infradb"
	netlink_polling "github.com/opiproject/opi-evpn-bridge/pkg/netlink"
	"github.com/opiproject/opi-evpn-bridge/pkg/utils"
	p4client "github.com/opiproject/opi-intel-bridge/pkg/evpn/vendor_plugins/intel-e2000/p4runtime/p4driverapi"
	binarypack "github.com/roman-kachanovsky/go-binary-pack/binary-pack"
)

// TcamPrefix structure of tcam type
var TcamPrefix = struct {
	GRD, VRF, P2P uint32
}{
	GRD: 0,
	VRF: 2, // taking const for now as not imported VRF
	P2P: 0x78654312,
}

// Direction structure of type rx, tx or rxtx
var Direction = struct {
	Rx, Tx int
}{
	Rx: 0,
	Tx: 1,
}

// Vlan structure of type grd phy port
var Vlan = struct {
	GRD, PHY0, PHY1, PHY2, PHY3 uint16
}{
	GRD:  4089,
	PHY0: 4090,
	PHY1: 4091,
	PHY2: 4092,
	PHY3: 4093,
}
var trueStr = "TRUE"
var grdStr = "GRD"
var intele2000Str = "intel-e2000"

// PortID structure of type phy port
var PortID = struct {
	PHY0, PHY1, PHY2, PHY3 int
}{
	PHY0: 0,
	PHY1: 1,
	PHY2: 2,
	PHY3: 3,
}

// EntryType structure of entry type
var EntryType = struct {
	BP, l3NH, l2Nh, trieIn uint32
}{
	BP:   1,
	l3NH: 2,
	l2Nh: 3,
}

// ModPointer structure of  mod ptr definitions
var ModPointer = struct {
	ignorePtr, l2FloodingPtr, ptrMinRange, ptrMaxRange uint32
}{
	ignorePtr:     0,
	l2FloodingPtr: 1,
	ptrMinRange:   2,
	ptrMaxRange:   uint32(math.Pow(2, 16)),
}

// TrieIndex structure of  tri index definitions
var TrieIndex = struct {
	triIdxMinRange, triIdxMaxRange uint32
}{
	triIdxMinRange: 1,
	triIdxMaxRange: uint32(math.Pow(2, 16)),
}

// EcmpIndex structure of ecmp definitions
var EcmpIndex = struct {
	ecmpIdxMinRange, ecmpIdxMaxRange uint32
}{
	ecmpIdxMinRange: 1,
	ecmpIdxMaxRange: uint32(math.Pow(2, 16)),
}

// ptrPool initialized variable
var ptrPool, _ = utils.IDPoolInit("mod_ptr", ModPointer.ptrMinRange, ModPointer.ptrMaxRange)

// trieIndexPool initialized variable
var trieIndexPool, _ = utils.IDPoolInit("trie_index", TrieIndex.triIdxMinRange, TrieIndex.triIdxMaxRange)

var ecmpIndexPool, _ = utils.IDPoolInit("ecmp", EcmpIndex.ecmpIdxMinRange, EcmpIndex.ecmpIdxMaxRange)

// Table of type string
type Table string

const (

	// l3Rt  evpn p4 table name
	l3Rt = "evpn_gw_control.l3_routing_table" // VRFs routing table in LPM
	//                            TableKeys (
	//                                ipv4_table_lpm_root2,  // Exact
	//                                vrf,                   // LPM
	//                                direction,             // LPM
	//                                dst_ip,                // LPM
	//                            )
	//                            Actions (
	//                                set_neighbor(neighbor, ecmp_on),
	//                            )

	// l3RtHost  evpn p4 table name
	l3RtHost = "evpn_gw_control.l3_lem_table"
	//                            TableKeys (
	//                                vrf,                   // Exact
	//                                direction,             // Exact
	//                                dst_ip,                // Exact
	//                            )
	//                            Actions (
	//                                set_neighbor(neighbor, ecmp_on)
	//                            )

	// l3P2PRt  evpn p4 table name
	l3P2PRt = "evpn_gw_control.l3_p2p_routing_table" // Special GRD routing table for VXLAN packets
	//                            TableKeys (
	//                                ipv4_table_lpm_root2,  # Exact
	//                                dst_ip,                # LPM
	//                            )
	//                            Actions (
	//                                set_p2p_neighbor(neighbor, ecmp_on),
	//

	// l3P2PRtHost  evpn p4 table name
	l3P2PRtHost = "evpn_gw_control.l3_p2p_lem_table"
	// Special LEM table for VXLAN packets
	//                            TableKeys (
	//                                vrf,                   # Exact
	//                                direction,             # Exact
	//                                dst_ip,                # Exact
	//                            )
	//                            Actions (
	//                                set_p2p_neighbor(neighbor, ecmp_on)
	//                            )

	// l3NHrx evpn p4 table name
	l3NhRx = "evpn_gw_control.l3_nexthop_table_rx" // LEM next hop table in rx direction
	//                            TableKeys (
	//                                neighbor,              # Exact
	//                                bit32_zeros,           # Exact
	//                            }
	//                            Actions (
	//                               push_dmac_vlan(mod_ptr, vport)
	//                               push_vlan(mod_ptr, vport)
	//                               push_mac(mod_ptr, vport)
	//                               push_outermac_vxlan_innermac(mod_ptr, vport)
	//                               push_mac_vlan(mod_ptr, vport)
	//                               send_p2p(dport, q_id)
	//                               sendp2p_push_mac(mod_ptr, dport, q_id)
	//                               send_p2p_push_outermac_vxlan_innermac(mod_ptr, vport, q_id)
	//                            )

	// l3NH  evpn p4 table name
	l3NhTx = "evpn_gw_control.l3_nexthop_table_tx" // LEM next hop table in tx direction
	//                            TableKeys (
	//                                neighbor,              // Exact
	//                                bit32_zeros,           // Exact
	//                            )
	//                            Actions (
	//                               push_dmac_vlan(mod_ptr, vport)
	//                               push_vlan(mod_ptr, vport)
	//                               push_mac(mod_ptr, vport)
	//                               push_outermac_vxlan_innermac(mod_ptr, vport)
	//                               push_mac_vlan(mod_ptr, vport)
	//                               send_p2p(dport, q_id)
	//                               sendp2p_push_mac(mod_ptr, dport, q_id)
	//                               send_p2p_push_outermac_vxlan_innermac(mod_ptr, vport, q_id)
	//                            )

	// l3EcmpSel evpn p4 table name
	l3EcmpSel = "evpn_gw_control.ecmp_selection_table" // SEM table for ECMP nexthop selection
	//                            TableKeys (
	//                                neighbor,              # Exact
	//                                hash,                  # Exact (4-bits)
	//                                bit32_zeros,           # Exact
	//                            )
	//                            Actions (
	//                                set_neighbor_withoutrec(neighbor)
	//
	//                            )

	// p2pIn  evpn p4 table name
	p2pIn = "evpn_gw_control.ingress_p2p_table"
	//                           TableKeys (
	//                               neighbor,              # Exact
	//                               bit32_zeros,           # Exact
	//                           )
	//                           Actions(
	//                               fwd_to_port(port)
	//

	// phyInIP  evpn p4 table name
	phyInIP = "evpn_gw_control.phy_ingress_ip_table" // PHY ingress table - IP traffic
	//                           TableKeys(
	//                               port_id,                // Exact
	//                               da,            // Exact
	//                           )
	//                           Actions(
	//                               set_vrf_id(tcam_prefix, vport, vrf),
	//                           )

	// phyInArp  evpn p4 table name
	phyInArp = "evpn_gw_control.phy_ingress_arp_table" // PHY ingress table - ARP traffic
	//                           TableKeys(
	//                               port_id,                // Exact
	//                               bit32_zeros,            // Exact
	//                           )
	//                           Actions(
	//                               fwd_to_port(port)
	//                           )

	// phyInVxlan  evpn p4 table name
	phyInVxlan = "evpn_gw_control.phy_ingress_vxlan_table" // PHY ingress table - VXLAN traffic
	//                           TableKeys(
	//                               dst_ip
	//                               vni,
	//                               da
	//                           )
	//                           Actions(
	//                               pop_vxlan_set_vrf_id(mod_ptr, tcam_prefix, vport, vrf),
	//                           )

	// phyInVxlanL2  evpn p4 table name
	phyInVxlanL2 = "evpn_gw_control.phy_ingress_vxlan_vlan_table"
	//                           Keys {
	//                               dst_ip                  // Exact
	//                               vni                     // Exact
	//                           }
	//                           Actions(
	//                               pop_vxlan_set_vlan_id(mod_ptr, vlan_id, vport)
	//                           )

	// podInArpAccess  evpn p4 table name
	podInArpAccess = "evpn_gw_control.vport_arp_ingress_table"
	//                       Keys {
	//                           vsi,                        // Exact
	//                           bit32_zeros                 // Exact
	//                       }
	//                       Actions(
	//                           fwd_to_port(port),
	//                           send_to_port_mux_access(mod_ptr, vport)
	//                       )

	// podInArpTrunk  evpn p4 table name
	podInArpTrunk = "evpn_gw_control.tagged_vport_arp_ingress_table"
	//                       Key {
	//                           vsi,                        // Exact
	//                           vid                         // Exact
	//                       }
	//                       Actions(
	//                           send_to_port_mux_trunk(mod_ptr, vport),
	//                           fwd_to_port(port),
	//                           pop_vlan(mod_ptr, vport)
	//                       )

	// podInIPAccess  evpn p4 table name
	podInIPAccess = "evpn_gw_control.vport_ingress_table"
	//                       Key {
	//                           vsi,                        // Exact
	//                           bit32_zeros                 // Exact
	//                       }
	//                       Actions(
	//                          fwd_to_port(port)
	//                          set_vlan(vlan_id, vport)
	//                       )

	// podInIPTrunk  evpn p4 table name
	podInIPTrunk = "evpn_gw_control.tagged_vport_ingress_table"
	//                       Key {
	//                           vsi,                        // Exact
	//                           vid                         // Exact
	//                       }
	//                       Actions(
	//                           //pop_vlan(mod_ptr, vport)
	//                           //pop_vlan_set_vrfid(mod_ptr, vport, tcam_prefix, vrf)
	//                           set_vlan_and_pop_vlan(mod_ptr, vlan_id, vport)
	//                       )

	// portInSviAccess  evpn p4 table name
	portInSviAccess = "evpn_gw_control.vport_svi_ingress_table"
	//                       Key {
	//                           vsi,                        // Exact
	//                           da                          // Exact
	//                       }
	//                       Actions(
	//                           set_vrf_id_tx(tcam_prefix, vport, vrf)
	//                           fwd_to_port(port)
	//                       )

	// portInSviTrunk  evpn p4 table name
	portInSviTrunk = "evpn_gw_control.tagged_vport_svi_ingress_table"
	//                       Key {
	//                           vsi,                        // Exact
	//                           vid,                        // Exact
	//                           da                          // Exact
	//                       }
	//                       Actions(
	//                           pop_vlan_set_vrf_id(tcam_prefix, mod_ptr, vport, vrf)
	//                       )

	// portMuxIn  evpn p4 table name
	portMuxIn = "evpn_gw_control.port_mux_ingress_table"
	//                       Key {
	//                           vsi,                        // Exact
	//                           vid                         // Exact
	//                       }
	//                       Actions(
	//                           set_def_vsi_loopback()
	//                           pop_ctag_stag_vlan(mod_ptr, vport),
	//                           pop_stag_vlan(mod_ptr, vport)
	//                       )
	//    PORT_MUX_RX        = "evpn_gw_control.port_mux_rx_table"
	//                       Key {
	//                           vid,                        // Exact
	//                           bit32_zeros                 // Exact
	//                       }
	//                       Actions(
	//                           pop_ctag_stag_vlan(mod_ptr, vport),
	//                           pop_stag_vlan(mod_ptr, vport)
	//                       )

	// portMuxFwd  evpn p4 table name
	portMuxFwd = "evpn_gw_control.port_mux_fwd_table"
	//                       Key {
	//                           bit32_zeros                 // Exact
	//                       }
	//                       Actions(
	//                           "evpn_gw_control.send_to_port_mux(vport)"
	//                       )

	// l2FwdLoop  evpn p4 table name
	l2FwdLoop = "evpn_gw_control.l2_fwd_rx_table"
	//                       Key {
	//                           da                          // Exact (MAC)
	//                       }
	//                       Actions(
	//                           l2_fwd(port)
	//                       )

	// l2Fwd  evpn p4 table name
	l2Fwd = "evpn_gw_control.l2_dmac_table"
	//                       Key {
	//                           vlan_id,                    // Exact
	//                           da,                         // Exact
	//                           direction                   // Exact
	//                       }
	//                       Actions(
	//                           set_neighbor(neighbor)
	//                       )

	// l2Nh  evpn p4 table name
	l2Nh = "evpn_gw_control.l2_nexthop_table"
	//                       Key {
	//                           neighbor                    // Exact
	//                           bit32_zeros                 // Exact
	//                       }
	//                       Actions(
	//                           //push_dmac_vlan(mod_ptr, vport)
	//                           push_stag_ctag(mod_ptr, vport)
	//                           push_vlan(mod_ptr, vport)
	//                           fwd_to_port(port)
	//                           push_outermac_vxlan(mod_ptr, vport)
	//                       )

	// tcamEntries  evpn p4 table name
	tcamEntries = "evpn_gw_control.ecmp_lpm_root_lut1"

	//                       Key {
	//                           tcam_prefix,                 // Exact
	//                           MATCH_PRIORITY,              // Exact
	//                       }
	//                       Actions(
	//                           None(ipv4_table_lpm_root1)
	//                       )

	// tcamEntries2  evpn p4 table name
	tcamEntries2 = "evpn_gw_control.ecmp_lpm_root_lut2"
	//                       Key {
	//                           tcamPrefix,                 # Exact
	//                           MATCH_PRIORITY,              # Exact
	//                       }
	//                       Actions(
	//                           None(ipv4_table_lpm_root2)
	//

)

// ModTable string var of mod table
type ModTable string

const (

	// pushVlan evpn p4 table name
	pushVlan = "evpn_gw_control.vlan_push_mod_table"
	//                        src_action="push_vlan"
	//			  Actions(
	// 				vlan_push(pcp, dei, vlan_id),
	//                        )

	// pushMacVlan evpn p4 table name
	pushMacVlan = "evpn_gw_control.mac_vlan_push_mod_table"
	//                       src_action=""
	//                       Actions(
	//                          update_smac_dmac_vlan(src_mac_addr, dst_mac_addr, pcp, dei, vlan_id)

	// pushDmacVlan evpn p4 table name
	pushDmacVlan = "evpn_gw_control.dmac_vlan_push_mod_table"
	//                        src_action="push_dmac_vlan",
	//                       Actions(
	//                           dmac_vlan_push(pcp, dei, vlan_id, dst_mac_addr),
	//                        )

	// macMod evpn p4 table name
	macMod = "evpn_gw_control.mac_mod_table"
	//                       src_action="push_mac"
	//                        Actions(
	//                            update_smac_dmac(src_mac_addr, dst_mac_addr),
	//                        )

	// pushVxlanHdr evpn p4 table name
	pushVxlanHdr = "evpn_gw_control.omac_vxlan_imac_push_mod_table"
	//                       src_action="push_outermac_vxlan_innermac"
	//                       Actions(
	//                           omac_vxlan_imac_push(outer_smac_addr,
	//                                                outer_dmac_addr,
	//                                                src_addr,
	//                                                dst_addr,
	//                                                dst_port,
	//                                                vni,
	//                                                inner_smac_addr,
	//                                                inner_dmac_addr)
	//                       )

	// podOutAccess evpn p4 table name
	podOutAccess = "evpn_gw_control.vlan_encap_ctag_stag_mod_table"
	//                       src_actions="send_to_port_mux_access"
	//                       Actions(
	//                           vlan_push_access(pcp, dei, ctag_id, pcp_s, dei_s, stag_id, dst_mac)
	//                       )

	// podOutTrunk evpn p4 table name
	podOutTrunk = "evpn_gw_control.vlan_encap_stag_mod_table"
	//                       src_actions="send_to_port_mux_trunk"
	//                       Actions(
	//                           vlan_push_trunk(pcp, dei, stag_id, dst_mac)
	//                       )

	// popCtagStag evpn p4 table name
	popCtagStag = "evpn_gw_control.vlan_ctag_stag_pop_mod_table"
	//                       src_actions=""
	//                       Actions(
	//                           vlan_ctag_stag_pop()
	//                       )

	// popStag evpn p4 table name
	popStag = "evpn_gw_control.vlan_stag_pop_mod_table"
	//                       src_actions=""
	//                       Actions(
	//                           vlan_stag_pop()
	//                       )

	// pushQnQFlood evpn p4 table name
	pushQnQFlood = "evpn_gw_control.vlan_encap_ctag_stag_flood_mod_table"
	//                       src_action="l2_nexthop_table.push_stag_ctag()"
	//                       Action(
	//                           vlan_push_stag_ctag_flood()
	//                       )

	// pushVxlanOutHdr evpn p4 table name
	pushVxlanOutHdr = "evpn_gw_control.omac_vxlan_push_mod_table"

	//                      src_action="l2_nexthop_table.push_outermac_vxlan()"
	//			Action(
	//                           omac_vxlan_push(outer_smac_addr, outer_dmac_addr, src_addr, dst_addr, dst_port, vni)
	//                       )

)

// _isL3vpnEnabled check if l3 enabled
func _isL3vpnEnabled(vrf *infradb.Vrf) bool {
	return vrf.Spec.Vni != nil
}

// bigEndian16 convert uint32 to big endian number
func bigEndian16(id uint32) interface{} {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(id))
	unpackedData := binary.BigEndian.Uint16(buf)
	return unpackedData
}

// _bigEndian16 convert to big endian 16bit
func _bigEndian16(id interface{}) interface{} {
	var bp = new(binarypack.BinaryPack)
	var packFormat = []string{"H"}
	var value = []interface{}{id}
	var packedData, err = bp.Pack(packFormat, value)
	if err != nil {
		log.Printf("intel-e2000: error: %v\n", err)
	}
	var unpackedData = binary.BigEndian.Uint16(packedData)
	return unpackedData
}

// _toEgressVsi convert to vsi+16
func _toEgressVsi(vsiID int) int {
	return vsiID + 16
}

// _directionsOf get the direction
func _directionsOf(entry interface{}) []int {
	var directions []int
	var direction int

	switch e := entry.(type) {
	case netlink_polling.RouteStruct:
		direction, _ = e.Metadata["direction"].(int)
	case netlink_polling.FdbEntryStruct:
		direction, _ = e.Metadata["direction"].(int)
	}
	if direction == netlink_polling.TX || direction == netlink_polling.RXTX {
		directions = append(directions, Direction.Tx)
	}
	if direction == netlink_polling.RX || direction == netlink_polling.RXTX {
		directions = append(directions, Direction.Rx)
	}
	return directions
}

// _addTcamEntry adds the tcam entry
func _addTcamEntry(vrfID uint32, direction int, prefix interface{}) (p4client.TableEntry, uint32) {
	tcamPrefix := fmt.Sprintf("%d%d", vrfID, direction)
	var tblentry p4client.TableEntry
	var tcam, err = strconv.ParseUint(tcamPrefix, 10, 32)
	if err != nil {
		panic(err)
	}
	tidx, refCount := trieIndexPool.GetIDWithRef(tcam, prefix)
	if refCount == 1 {
		tblentry = p4client.TableEntry{
			Tablename: tcamEntries,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"user_meta.cmeta.tcam_prefix": {uint32(tcam), "ternary"},
				},
				Priority: int32(tidx),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.ecmp_lpm_root_lut1_action",
				Params:     []interface{}{tidx},
			},
		}
	}
	return tblentry, tidx
}

// _getTcamPrefix get the tcam prefix value
func _getTcamPrefix(vrfID uint32, direction int) (int, error) {
	tcamPrefix := fmt.Sprintf("%d%d", vrfID, direction)
	val, err := strconv.ParseInt(tcamPrefix, 10, 32)
	return int(val), err
}

// _deleteTcamEntry deletes the tcam entry
func _deleteTcamEntry(vrfID uint32, direction int, prefix interface{}) (p4client.TableEntry, uint32) {
	tcamPrefix := fmt.Sprintf("%d%d", vrfID, direction)
	var tblentry p4client.TableEntry
	var tcam, err = strconv.ParseUint(tcamPrefix, 10, 32)
	if err != nil {
		panic(err)
	}
	tidx, refCount := trieIndexPool.ReleaseIDWithRef(tcam, prefix)
	if refCount == 0 {
		tblentry = p4client.TableEntry{
			Tablename: tcamEntries,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"user_meta.cmeta.tcam_prefix": {uint32(tcam), "ternary"},
				},
				Priority: int32(tidx),
			},
		}
	}
	return tblentry, tidx
}

// PhyPort structure of phy ports
type PhyPort struct {
	id  int
	vsi int
	mac string
}

// PhyPortInit initializes the phy port
func (p PhyPort) PhyPortInit(id int, vsi string, mac string) PhyPort {
	p.id = id
	val, err := strconv.ParseUint(vsi, 10, 16)
	if err != nil {
		panic(err)
	}
	p.vsi = int(val)
	p.mac = mac

	return p
}

// _p4NexthopID get the p4 nexthop id
func _p4NexthopID(nh netlink_polling.NexthopStruct, direction int) int {
	nhID := nh.ID << 1

	if direction == Direction.Rx && (nh.NhType == netlink_polling.PHY || nh.NhType == netlink_polling.VXLAN) {
		nhID++
	}

	return nhID
}

func (e *EcmpDispatcher) _p4NexthopID(direction int) int {
	nhID := e.id << 1
	if direction == Direction.Rx {
		if e.dir == Direction.Tx {
			nhID++
		}
	}

	return int(nhID)
}

// _p2pQid get the qid for p2p port
func _p2pQid(pID int) int {
	if pID == PortID.PHY0 {
		return 0x87
	} else if pID == PortID.PHY1 {
		return 0x8d
	}

	return 0
}

// EcmpDispatcher structure
type EcmpDispatcher struct {
	Nexthop  []*netlink_polling.NexthopStruct
	key      string
	dir      int
	id       uint32
	hashmap  map[int]netlink_polling.NexthopStruct
	numslots int
}

// using pointer
func (e *EcmpDispatcher) runWebsterAlg() {
	for i := 0; i < e.numslots; i++ {
		var maxNh *netlink_polling.NexthopStruct
		maxValue := float64(-math.MaxInt32)
		for _, nh := range e.Nexthop {
			if nh.Value > maxValue {
				maxValue = nh.Value
				maxNh = nh
			}
		}
		maxNh.Hashes = append(maxNh.Hashes, i)
		maxNh.Divisor += 2
		maxNh.Value = float64(maxNh.Weight) / float64(maxNh.Divisor)
		e.hashmap[i] = *maxNh
	}
}
func (e *EcmpDispatcher) getecmpnh(nexthop []*netlink_polling.NexthopStruct) {
	if e.Nexthop == nil {
		log.Println("Dcgw Ecmp:e.Nexthop is nil")
		return
	}

	for i, nh := range nexthop {
		if nh == nil {
			log.Printf("Dcgw Ecmp : nexthop[%d] is nil\n", i)
			continue
		}

		e.Nexthop[i].ID = nh.ID

		if nh.Metadata == nil {
			log.Printf("Dcgw Ecmp : nexthop[%d].Metadata is nil\n", i)
			continue
		}

		if direction, ok := nh.Metadata["direction"]; ok {
			if direction == netlink_polling.RX {
				e.Nexthop[i].Dir = Direction.Rx
			} else {
				e.Nexthop[i].Dir = Direction.Tx
			}
		} else {
			log.Printf("Dcgw Ecmp : nexthop[%d].Metadata[\"direction\"] not found\n", i)
		}

		e.Nexthop[i].Weight = nh.Weight
		e.Nexthop[i].Divisor = 1
		e.Nexthop[i].Value = float64(nh.Weight)
	}
}
func (e *EcmpDispatcher) getkeys(nexthop []*netlink_polling.NexthopStruct) string {
	var keys string
	for _, nh := range nexthop {
		keys += string(rune(nh.ID))
	}
	return keys
}
func (e *EcmpDispatcher) checkdir() bool {
	var rx, tx int
	rx = 0
	tx = 0
	for _, nh := range e.Nexthop {
		if nh.Dir == Direction.Rx {
			rx++
		} else {
			tx++
		}
	}
	if rx == len(e.Nexthop) {
		e.dir = Direction.Rx
		return true
	} else if tx == len(e.Nexthop) {
		e.dir = Direction.Tx
		return true
	}
	return false
}

// EcmpDispatcherInit function initializes the ecmp objects
func (e *EcmpDispatcher) EcmpDispatcherInit(nexthop []*netlink_polling.NexthopStruct, vrf *infradb.Vrf) bool {
	e.Nexthop = make([]*netlink_polling.NexthopStruct, len(nexthop))
	for i := range nexthop {
		e.Nexthop[i] = &netlink_polling.NexthopStruct{}
		e.Nexthop[i].ParseNexthop(vrf, netlink_polling.RouteCmdInfo{})
		e.Nexthop[i].NhType = netlink_polling.ECMP
	}
	e.getecmpnh(nexthop)
	e.key = e.getkeys(nexthop)
	if !e.checkdir() {
		return false
	}
	e.numslots = int(16)
	e.hashmap = make(map[int]netlink_polling.NexthopStruct, 0)
	return true
}

// GrpcPairPort structure
type GrpcPairPort struct {
	vsi  int
	mac  string
	peer map[string]string
}

// GrpcPairPortInit get the vsi+16
func (g GrpcPairPort) GrpcPairPortInit(vsi string, mac string) GrpcPairPort {
	val, err := strconv.ParseUint(vsi, 10, 16)
	if err != nil {
		panic(err)
	}
	g.vsi = int(val)
	g.mac = mac
	return g
}

// setRemotePeer set the remote peer
func (g GrpcPairPort) setRemotePeer(peer [2]string) GrpcPairPort {
	g.peer = make(map[string]string)
	g.peer["vsi"] = peer[0]
	g.peer["mac"] = peer[1]
	return g
}

// L3Decoder structure
type L3Decoder struct {
	_muxVsi     uint16
	_defaultVsi int
	_phyPorts   []PhyPort
	_grpcPorts  []GrpcPairPort
	PhyPort
	GrpcPairPort
}

// L3DecoderInit initialize the l3 decoder
func (l L3Decoder) L3DecoderInit(representors map[string][2]string) L3Decoder {
	s := L3Decoder{
		_muxVsi:     l.setMuxVsi(representors),
		_defaultVsi: 0x6,
		_phyPorts:   l._getPhyInfo(representors),
		_grpcPorts:  l._getGrpcInfo(representors),
	}
	return s
}

// setMuxVsi set the mux vsi
func (l L3Decoder) setMuxVsi(representors map[string][2]string) uint16 {
	a := representors["vrf_mux"][0]
	var muxVsi, err = strconv.ParseUint(a, 10, 16)
	if err != nil {
		panic(err)
	}
	return uint16(muxVsi)
}

// _getPhyInfo get the phy port info
func (l L3Decoder) _getPhyInfo(representors map[string][2]string) []PhyPort {
	var enabledPorts []PhyPort
	var vsi string
	var mac string
	var p = reflect.TypeOf(PortID)
	for i := 0; i < p.NumField(); i++ {
		var k = p.Field(i).Name
		var key = strings.ToLower(k) + "_rep"
		for k = range representors {
			if key == k {
				vsi = representors[key][0]
				mac = representors[key][1]
				enabledPorts = append(enabledPorts, l.PhyPortInit(i, vsi, mac))
			}
		}
	}
	return enabledPorts // should return tuple
}

// _getGrpcInfo get the grpc information
func (l L3Decoder) _getGrpcInfo(representors map[string][2]string) []GrpcPairPort {
	var accHost GrpcPairPort
	var hostPort GrpcPairPort
	var grpcPorts []GrpcPairPort

	accVsi := representors["grpc_acc"][0]
	accMac := representors["grpc_acc"][1]
	accHost = accHost.GrpcPairPortInit(accVsi, accMac) // ??

	hostVsi := representors["grpc_host"][0]
	hostMac := representors["grpc_host"][1]
	hostPort = hostPort.GrpcPairPortInit(hostVsi, hostMac) // ??

	accPeer := representors["grpc_host"]
	hostPeer := representors["grpc_acc"]
	accHost = accHost.setRemotePeer(accPeer)

	hostPort = hostPort.setRemotePeer(hostPeer)

	grpcPorts = append(grpcPorts, accHost, hostPort)
	return grpcPorts
}

// getVrfID get the vrf id from vni
func (l L3Decoder) getVrfID(route netlink_polling.RouteStruct) uint32 {
	if route.Vrf.Spec.Vni == nil {
		return 0
	}

	return *route.Vrf.Metadata.RoutingTable[0]
}

// _l3HostRoute gets the l3 host route
func (l L3Decoder) _l3HostRoute(route netlink_polling.RouteStruct, delete string, ecmpFlag bool, entries []interface{}, e EcmpDispatcher) []interface{} {
	var vrfID = l.getVrfID(route)
	var directions = _directionsOf(route)
	var host = route.Route0.Dst
	var ec uint16
	if ecmpFlag {
		ec = uint16(1)
	} else {
		ec = uint16(0)
	}

	if delete == trueStr {
		for _, dir := range directions {
			entries = append(entries, p4client.TableEntry{
				Tablename: l3RtHost,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vrf":       {_bigEndian16(vrfID), "exact"},
						"direction": {uint16(dir), "exact"},
						"dst_ip":    {host, "exact"},
					},
					Priority: int32(0),
				},
			})
		}
	} else {
		for _, dir := range directions {
			var neighbor int
			if ecmpFlag {
				neighbor = e._p4NexthopID(dir)
			} else {
				neighbor = _p4NexthopID(*route.Nexthops[0], dir)
			}

			entries = append(entries, p4client.TableEntry{
				Tablename: l3RtHost,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vrf":       {bigEndian16(vrfID), "exact"},
						"direction": {uint16(dir), "exact"},
						"dst_ip":    {host, "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.set_neighbor",
					Params:     []interface{}{uint16(neighbor), ec},
				},
			})
		}
	}
	if path.Base(route.Vrf.Name) == grdStr && route.Nexthops[0].NhType == netlink_polling.PHY {
		if delete == trueStr {
			entries = append(entries, p4client.TableEntry{
				Tablename: l3P2PRtHost,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vrf":       {_bigEndian16(vrfID), "exact"},
						"direction": {uint16(Direction.Rx), "exact"},
						"dst_ip":    {host, "exact"},
					},
					Priority: int32(0),
				},
			})
		} else {
			var neighbor int
			if ecmpFlag {
				neighbor = e._p4NexthopID(Direction.Rx)
			} else {
				neighbor = _p4NexthopID(*route.Nexthops[0], Direction.Rx)
			}

			entries = append(entries, p4client.TableEntry{
				Tablename: l3P2PRtHost,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vrf":       {bigEndian16(vrfID), "exact"},
						"direction": {uint16(Direction.Rx), "exact"},
						"dst_ip":    {host, "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.set_p2p_neighbor",
					Params:     []interface{}{uint16(neighbor), ec},
				},
			})
		}
	}
	return entries
}

// _l3Route generate the l3 route entries
func (l L3Decoder) _l3Route(route netlink_polling.RouteStruct, delete string, ecmpFlag bool, entries []interface{}, e EcmpDispatcher) []interface{} {
	var vrfID = l.getVrfID(route)
	var directions = _directionsOf(route)
	var addr = route.Route0.Dst.IP.String()
	var ec uint16
	if ecmpFlag {
		ec = uint16(1)
	} else {
		ec = uint16(0)
	}

	for _, dir := range directions {
		if delete == trueStr {
			var tblEntry, tIdx = _deleteTcamEntry(vrfID, dir, route.Route0.Dst)
			if !reflect.ValueOf(tblEntry).IsZero() {
				entries = append(entries, tblEntry)
			}
			entries = append(entries, p4client.TableEntry{
				Tablename: l3Rt,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"ipv4_table_lpm_root1": {tIdx, "exact"},
						"dst_ip":               {net.ParseIP(addr), "lpm"},
					},
					Priority: int32(1),
				},
			})
		} else {
			var neighbor int
			if ecmpFlag {
				neighbor = e._p4NexthopID(Direction.Rx)
			} else {
				neighbor = _p4NexthopID(*route.Nexthops[0], Direction.Rx)
			}

			var tblEntry, tIdx = _addTcamEntry(vrfID, dir, route.Route0.Dst)
			if !reflect.ValueOf(tblEntry).IsZero() {
				entries = append(entries, tblEntry)
			}
			entries = append(entries, p4client.TableEntry{
				Tablename: l3Rt,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"ipv4_table_lpm_root1": {tIdx, "exact"},
						"dst_ip":               {net.ParseIP(addr), "lpm"},
					},
					Priority: int32(1),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.set_neighbor",
					Params:     []interface{}{uint16(neighbor), ec},
				},
			})
		}
	}
	if path.Base(route.Vrf.Name) == grdStr && route.Nexthops[0].NhType == netlink_polling.PHY {
		tidx := trieIndexPool.GetID(TcamPrefix.P2P)
		if delete == trueStr {
			entries = append(entries, p4client.TableEntry{
				Tablename: l3P2PRt,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"ipv4_table_lpm_root2": {tidx, "exact"},
						"dst_ip":               {net.ParseIP(addr), "lpm"},
					},
					Priority: int32(1),
				},
			})
		} else {
			var neighbor int
			if ecmpFlag {
				neighbor = e._p4NexthopID(Direction.Rx)
			} else {
				neighbor = _p4NexthopID(*route.Nexthops[0], Direction.Rx)
			}

			entries = append(entries, p4client.TableEntry{
				Tablename: l3P2PRt,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"ipv4_table_lpm_root2": {tidx, "exact"},
						"dst_ip":               {net.ParseIP(addr), "lpm"},
					},
					Priority: int32(1),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.set_p2p_neighbor",
					Params:     []interface{}{uint16(neighbor), ec},
				},
			})
		}
	}
	return entries
}

func (e EcmpDispatcher) addEcmpDispatcher(entries []interface{}) []interface{} {
	var directions []int
	if e.dir == Direction.Rx {
		directions = append(directions, Direction.Rx)
	} else {
		directions = append(directions, Direction.Rx)
		directions = append(directions, Direction.Tx)
	}

	for i, nh := range e.hashmap {
		for dir := range directions {
			entries = append(entries, p4client.TableEntry{
				Tablename: l3EcmpSel,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(e._p4NexthopID(dir)), "exact"},
						"hash":        {uint16(i), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.set_neighbor_withoutrec",
					Params:     []interface{}{uint16(_p4NexthopID(nh, dir))},
				},
			})
		}
	}
	return entries
}

func (e EcmpDispatcher) delEcmpDispatcher(entries []interface{}) []interface{} {
	var directions []int
	if e.dir == Direction.Rx {
		directions = append(directions, Direction.Rx)
	} else {
		directions = append(directions, Direction.Rx)
		directions = append(directions, Direction.Tx)
	}

	for i := 0; i < e.numslots; i++ {
		for dir := range directions {
			entries = append(entries, p4client.TableEntry{
				Tablename: l3EcmpSel,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(e._p4NexthopID(dir)), "exact"},
						"hash":        {uint16(i), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
			})
		}
	}
	return entries
}

// translateAddedRoute translate the added route to p4 entries
func (l L3Decoder) translateAddedRoute(route netlink_polling.RouteStruct) []interface{} {
	var refCount uint32
	var entries = make([]interface{}, 0)
	var ecmpFlag bool
	ecmpFlag = false

	var ecmp EcmpDispatcher
	if len(route.Nexthops) > 1 {
		if !ecmp.EcmpDispatcherInit(route.Nexthops, route.Vrf) {
			return entries
		}
		ecmp.id, refCount = ecmpIndexPool.GetIDWithRef(ecmp.key, route.Key)
		if refCount == 1 {
			ecmp.runWebsterAlg()
			entries = ecmp.addEcmpDispatcher(entries)
		}
		route.Nexthops = []*netlink_polling.NexthopStruct{}
		route.Nexthops = ecmp.Nexthop
		ecmpFlag = true
	}
	var ipv4Net = route.Route0.Dst
	if net.IP(ipv4Net.Mask).String() == "255.255.255.255" {
		return l._l3HostRoute(route, "False", ecmpFlag, entries, ecmp)
	}
	return l._l3Route(route, "False", ecmpFlag, entries, ecmp)
}

// translateDeletedRoute translate the deleted route to p4 entries
func (l L3Decoder) translateDeletedRoute(route netlink_polling.RouteStruct) []interface{} {
	var refCount uint32
	var entries = make([]interface{}, 0)
	var ecmpFlag bool
	ecmpFlag = false

	var ecmp EcmpDispatcher
	if len(route.Nexthops) > 1 {
		if !ecmp.EcmpDispatcherInit(route.Nexthops, route.Vrf) {
			return entries
		}
		ecmp.id, refCount = ecmpIndexPool.ReleaseIDWithRef(ecmp.key, route.Key)
		if refCount == 0 {
			ecmp.runWebsterAlg()
			entries = ecmp.delEcmpDispatcher(entries)
		}
		route.Nexthops = []*netlink_polling.NexthopStruct{}
		route.Nexthops = ecmp.Nexthop
		ecmpFlag = true
	}
	var ipv4Net = route.Route0.Dst
	if net.IP(ipv4Net.Mask).String() == "255.255.255.255" {
		return l._l3HostRoute(route, "True", ecmpFlag, entries, ecmp)
	}
	return l._l3Route(route, "True", ecmpFlag, entries, ecmp)
}

// translateAddedNexthop translate the added nexthop to p4 entries
//
//nolint:funlen
func (l L3Decoder) translateAddedNexthop(nexthop netlink_polling.NexthopStruct) []interface{} {
	if nexthop.NhType == netlink_polling.VXLAN {
		var entries []interface{}
		return entries
	}
	key := fmt.Sprintf("%d-%s-%s-%d-%v", EntryType.l3NH, nexthop.Key.VrfName, nexthop.Key.Dst, nexthop.Key.Dev, nexthop.Key.Local)
	var modPtr = ptrPool.GetID(key)
	nhID := _p4NexthopID(nexthop, Direction.Tx)

	var entries = make([]interface{}, 0)
	switch nexthop.NhType {
	case netlink_polling.PHY:
		var smac, _ = net.ParseMAC(nexthop.Metadata["smac"].(string))
		var dmac, _ = net.ParseMAC(nexthop.Metadata["dmac"].(string))
		var portID = nexthop.Metadata["egress_vport"]

		entries = append(entries, p4client.TableEntry{
			Tablename: macMod,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"meta.common.mod_blob_ptr": {modPtr, "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.update_smac_dmac",
				Params:     []interface{}{smac, dmac},
			},
		},
			p4client.TableEntry{
				Tablename: l3NhTx,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(nhID), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.push_mac",
					Params:     []interface{}{modPtr, uint16(portID.(int))},
				},
			},
			p4client.TableEntry{
				Tablename: l3NhRx,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(_p4NexthopID(nexthop, Direction.Rx)), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.send_p2p_push_mac",
					Params:     []interface{}{modPtr, uint16(portID.(int)), uint16(_p2pQid(portID.(int)))},
				},
			},
			p4client.TableEntry{
				Tablename: p2pIn,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(_p4NexthopID(nexthop, Direction.Rx)), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.fwd_to_port",
					Params:     []interface{}{uint16(portID.(int))},
				},
			})
	case netlink_polling.ACC:
		var dmac, _ = net.ParseMAC(nexthop.Metadata["dmac"].(string))
		var vlanID = nexthop.Metadata["vlanID"].(uint32)
		var vport = _toEgressVsi(nexthop.Metadata["egress_vport"].(int))
		entries = append(entries, p4client.TableEntry{
			Tablename: pushDmacVlan,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"meta.common.mod_blob_ptr": {modPtr, "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.dmac_vlan_push",
				Params:     []interface{}{uint16(0), uint16(1), uint16(vlanID), dmac},
			},
		},
			p4client.TableEntry{
				Tablename: l3NhRx,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(nhID), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.push_dmac_vlan",
					Params:     []interface{}{modPtr, uint32(vport)},
				},
			},
			p4client.TableEntry{
				Tablename: l3NhTx,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(nhID), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.push_dmac_vlan",
					Params:     []interface{}{modPtr, uint32(vport)},
				},
			})
	case netlink_polling.SVI:
		var smac, _ = net.ParseMAC(nexthop.Metadata["smac"].(string))
		var dmac, _ = net.ParseMAC(nexthop.Metadata["dmac"].(string))
		var vlanID = nexthop.Metadata["vlanID"].(uint32)
		vp, err := strconv.Atoi(nexthop.Metadata["egress_vport"].(string))
		if err != nil {
			panic(err)
		}
		var vport = _toEgressVsi(vp)
		var Type = nexthop.Metadata["portType"].(infradb.BridgePortType)
		switch Type {
		case infradb.Trunk:
			entries = append(entries, p4client.TableEntry{
				Tablename: pushMacVlan,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"meta.common.mod_blob_ptr": {modPtr, "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.update_smac_dmac_vlan",
					Params:     []interface{}{smac, dmac, 0, 1, uint16(vlanID)},
				},
			},
				p4client.TableEntry{
					Tablename: l3NhRx,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"neighbor":    {uint16(nhID), "exact"},
							"bit32_zeros": {uint32(0), "exact"},
						},
						Priority: int32(0),
					},
					Action: p4client.Action{
						ActionName: "evpn_gw_control.push_mac_vlan",
						Params:     []interface{}{modPtr, uint32(vport)},
					},
				},
				p4client.TableEntry{
					Tablename: l3NhTx,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"neighbor":    {uint16(nhID), "exact"},
							"bit32_zeros": {uint32(0), "exact"},
						},
						Priority: int32(0),
					},
					Action: p4client.Action{
						ActionName: "evpn_gw_control.push_mac_vlan",
						Params:     []interface{}{modPtr, uint32(vport)},
					},
				})
		case infradb.Access:
			entries = append(entries, p4client.TableEntry{
				Tablename: macMod,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"meta.common.mod_blob_ptr": {modPtr, "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.update_smac_dmac",
					Params:     []interface{}{smac, dmac},
				},
			},
				p4client.TableEntry{
					Tablename: l3NhRx,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"neighbor":    {uint16(nhID), "exact"},
							"bit32_zeros": {uint32(0), "exact"},
						},
						Priority: int32(0),
					},
					Action: p4client.Action{
						ActionName: "evpn_gw_control.push_mac",
						Params:     []interface{}{modPtr, uint32(vport)},
					},
				},
				p4client.TableEntry{
					Tablename: l3NhTx,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"neighbor":    {uint16(nhID), "exact"},
							"bit32_zeros": {uint32(0), "exact"},
						},
						Priority: int32(0),
					},
					Action: p4client.Action{
						ActionName: "evpn_gw_control.push_mac",
						Params:     []interface{}{modPtr, uint32(vport)},
					},
				})
		default:
			return entries
		}
	default:
		return entries
	}

	return entries
}

// translateDeletedNexthop translate the deleted nexthop to p4 entries
//
//nolint:funlen
func (l L3Decoder) translateDeletedNexthop(nexthop netlink_polling.NexthopStruct) []interface{} {
	if nexthop.NhType == netlink_polling.VXLAN {
		var entries []interface{}
		return entries
	}
	key := fmt.Sprintf("%d-%s-%s-%d-%v", EntryType.l3NH, nexthop.Key.VrfName, nexthop.Key.Dst, nexthop.Key.Dev, nexthop.Key.Local)
	var modPtr = ptrPool.ReleaseID(key)
	nhID := _p4NexthopID(nexthop, Direction.Tx)
	var entries = make([]interface{}, 0)
	switch nexthop.NhType {
	case netlink_polling.PHY:
		// if nexthop.NhType == netlink_polling.PHY {
		entries = append(entries, p4client.TableEntry{
			Tablename: macMod,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"meta.common.mod_blob_ptr": {modPtr, "exact"},
				},
				Priority: int32(0),
			},
		},
			p4client.TableEntry{
				Tablename: l3NhTx,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(nhID), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
			},
			p4client.TableEntry{
				Tablename: l3NhRx,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(_p4NexthopID(nexthop, Direction.Rx)), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
			},
			p4client.TableEntry{
				Tablename: p2pIn,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(_p4NexthopID(nexthop, Direction.Rx)), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
			})
	case netlink_polling.ACC:
		entries = append(entries, p4client.TableEntry{
			Tablename: pushDmacVlan,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"meta.common.mod_blob_ptr": {modPtr, "exact"},
				},
				Priority: int32(0),
			},
		},
			p4client.TableEntry{
				Tablename: l3NhRx,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(nhID), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
			},
			p4client.TableEntry{
				Tablename: l3NhTx,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(nhID), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
			})
	case netlink_polling.SVI:
		var Type = nexthop.Metadata["portType"].(infradb.BridgePortType)
		switch Type {
		case infradb.Trunk:
			entries = append(entries, p4client.TableEntry{
				Tablename: pushMacVlan,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"meta.common.mod_blob_ptr": {modPtr, "exact"},
					},
					Priority: int32(0),
				},
			},
				p4client.TableEntry{
					Tablename: l3NhRx,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"neighbor":    {uint16(nhID), "exact"},
							"bit32_zeros": {uint32(0), "exact"},
						},
						Priority: int32(0),
					},
				},
				p4client.TableEntry{
					Tablename: l3NhTx,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"neighbor":    {uint16(nhID), "exact"},
							"bit32_zeros": {uint32(0), "exact"},
						},
						Priority: int32(0),
					},
				})
		case infradb.Access:
			entries = append(entries, p4client.TableEntry{
				Tablename: macMod,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"meta.common.mod_blob_ptr": {modPtr, "exact"},
					},
					Priority: int32(0),
				},
			},
				p4client.TableEntry{
					Tablename: l3NhRx,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"neighbor":    {uint16(nhID), "exact"},
							"bit32_zeros": {uint32(0), "exact"},
						},
						Priority: int32(0),
					},
				},
				p4client.TableEntry{
					Tablename: l3NhTx,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"neighbor":    {uint16(nhID), "exact"},
							"bit32_zeros": {uint32(0), "exact"},
						},
						Priority: int32(0),
					},
				})
		default:
			return entries
		}
	default:
		return entries
	}
	return entries
}

// StaticAdditions do the static additions for p4 tables
//
//nolint:funlen
func (l L3Decoder) StaticAdditions() []interface{} {
	var tcamPrefix = TcamPrefix.GRD
	var entries = make([]interface{}, 0)

	entries = append(entries, p4client.TableEntry{
		Tablename: podInIPTrunk,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"vsi": {l._muxVsi, "exact"},
				"vid": {Vlan.GRD, "exact"},
			},
			Priority: int32(0),
		},
		Action: p4client.Action{
			ActionName: "evpn_gw_control.pop_vlan_set_vrfid",
			Params:     []interface{}{ModPointer.ignorePtr, uint32(0), tcamPrefix, uint32(0)},
		},
	},
	)
	for _, port := range l._grpcPorts {
		var peerVsi, err = strconv.ParseUint(port.peer["vsi"], 10, 16)
		if err != nil {
			panic(err)
		}
		var peerDa, _ = net.ParseMAC(port.peer["mac"])
		var portDa, _ = net.ParseMAC(port.mac)
		entries = append(entries, p4client.TableEntry{
			Tablename: portInSviAccess,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"vsi": {uint16(port.vsi), "exact"},
					"da":  {peerDa, "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.fwd_to_port",
				Params:     []interface{}{uint32(_toEgressVsi(int(peerVsi)))},
			},
		},
			p4client.TableEntry{
				Tablename: l2FwdLoop,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"da": {portDa, "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.l2_fwd",
					Params:     []interface{}{uint32(_toEgressVsi(port.vsi))},
				},
			})
	}
	for _, port := range l._phyPorts {
		var portDa, _ = net.ParseMAC(port.mac)
		entries = append(entries, p4client.TableEntry{
			Tablename: phyInIP,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"port_id": {uint16(port.id), "exact"},
					"da":      {portDa, "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.set_vrf_id",
				Params:     []interface{}{tcamPrefix, uint32(_toEgressVsi(l._defaultVsi)), uint32(0)},
			},
		},
			p4client.TableEntry{
				Tablename: phyInArp,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"port_id":     {uint16(port.id), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.fwd_to_port",
					Params:     []interface{}{uint32(_toEgressVsi(port.vsi))},
				},
			},
			p4client.TableEntry{
				Tablename: podInIPAccess,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vsi":         {uint16(port.vsi), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.fwd_to_port",
					Params:     []interface{}{uint32(port.id)},
				},
			},
			p4client.TableEntry{
				Tablename: podInArpAccess,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vsi":         {uint16(port.vsi), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.fwd_to_port",
					Params:     []interface{}{uint32(port.id)},
				},
			})
	}
	tidx := trieIndexPool.GetID(TcamPrefix.P2P)
	entries = append(entries, p4client.TableEntry{
		Tablename: tcamEntries2,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"user_meta.cmeta.tcam_prefix": {TcamPrefix.P2P, "ternary"},
			},
			Priority: int32(tidx),
		},
		Action: p4client.Action{
			ActionName: "evpn_gw_control.ecmp_lpm_root_lut2_action",
			Params:     []interface{}{tidx},
		},
	})
	return entries
}

// StaticDeletions do the static deletion for p4 tables
func (l L3Decoder) StaticDeletions() []interface{} {
	var entries = make([]interface{}, 0)
	for _, port := range l._phyPorts {
		var portDa, _ = net.ParseMAC(port.mac)
		entries = append(entries, p4client.TableEntry{
			Tablename: phyInIP,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"port_id": {uint16(port.id), "exact"},
					"da":      {portDa, "exact"},
				},
				Priority: int32(0),
			},
		},
			p4client.TableEntry{
				Tablename: phyInArp,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"port_id":     {uint16(port.id), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
			},
			p4client.TableEntry{
				Tablename: podInIPAccess,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vsi":         {uint16(port.vsi), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
			},
			p4client.TableEntry{
				Tablename: podInArpAccess,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vsi":         {uint16(port.vsi), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
			})
	}
	for _, port := range l._grpcPorts {
		var peerDa, _ = net.ParseMAC(port.peer["mac"])
		var portDa, _ = net.ParseMAC(port.mac)
		entries = append(entries, p4client.TableEntry{
			Tablename: portInSviAccess,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"vsi": {uint16(port.vsi), "exact"},
					"da":  {peerDa, "exact"},
				},
				Priority: int32(0),
			},
		},
			p4client.TableEntry{
				Tablename: l2FwdLoop,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"da": {portDa, "exact"},
					},
					Priority: int32(0),
				},
			})
	}
	entries = append(entries, p4client.TableEntry{
		Tablename: podInIPTrunk,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"vsi": {l._muxVsi, "exact"},
				"vid": {Vlan.GRD, "exact"},
			},
			Priority: int32(0),
		},
	})
	tidx := trieIndexPool.ReleaseID(TcamPrefix.P2P)
	entries = append(entries, p4client.TableEntry{
		Tablename: tcamEntries2,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"user_meta.cmeta.tcam_prefix": {TcamPrefix.P2P, "ternary"},
			},
			Priority: int32(tidx),
		},
	})
	return entries
}

// VxlanDecoder structure
type VxlanDecoder struct {
	vxlanUDPPort uint32
	_muxVsi      int
	_defaultVsi  int
}

// VxlanDecoderInit initialize vxlan decoder
func (v VxlanDecoder) VxlanDecoderInit(representors map[string][2]string) VxlanDecoder {
	var muxVsi, err = strconv.ParseInt(representors["vrf_mux"][0], 10, 32)
	if err != nil {
		panic(err)
	}
	s := VxlanDecoder{
		vxlanUDPPort: 4789,
		_defaultVsi:  0xb,
		_muxVsi:      int(muxVsi),
	}
	return s
}

// _isL2vpnEnabled check s if l2evpn enabled
func _isL2vpnEnabled(lb *infradb.LogicalBridge) bool {
	return lb.Spec.Vni != nil
}

// translateAddedVrf translates the added vrf
func (v VxlanDecoder) translateAddedVrf(vrf *infradb.Vrf) []interface{} {
	var entries = make([]interface{}, 0)
	if !_isL3vpnEnabled(vrf) {
		return entries
	}
	var tcamPrefix, err = _getTcamPrefix(*vrf.Metadata.RoutingTable[0], Direction.Rx)
	if err != nil {
		return entries
	}
	G, _ := infradb.GetVrf(vrf.Name)
	var detail map[string]interface{}
	var Rmac net.HardwareAddr
	for _, com := range G.Status.Components {
		if com.Name == "frr" {
			err := json.Unmarshal([]byte(com.Details), &detail)
			if err != nil {
				log.Println("intel-e2000: Error: ", err)
			}
			rmac, found := detail["rmac"].(string)
			if !found {
				log.Println("intel-e2000: Key 'rmac' not found")
				break
			}
			Rmac, err = net.ParseMAC(rmac)
			if err != nil {
				log.Println("intel-e2000: Error parsing MAC address:", err)
			}
		}
	}
	if reflect.ValueOf(Rmac).IsZero() {
		log.Println("intel-e2000: Rmac not found for Vtep :", vrf.Spec.VtepIP.IP)

		return entries
	}
	entries = append(entries, p4client.TableEntry{
		Tablename: phyInVxlan,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"dst_ip": {vrf.Spec.VtepIP.IP, "exact"},
				"vni":    {*vrf.Spec.Vni, "exact"},
				"da":     {Rmac, "exact"},
			},
			Priority: int32(0),
		},
		Action: p4client.Action{
			ActionName: "evpn_gw_control.pop_vxlan_set_vrf_id",
			Params:     []interface{}{ModPointer.ignorePtr, uint32(tcamPrefix), uint32(_toEgressVsi(v._defaultVsi)), *vrf.Metadata.RoutingTable[0]},
		},
	})
	return entries
}

// translateDeletedVrf translates the deleted vrf
func (v VxlanDecoder) translateDeletedVrf(vrf *infradb.Vrf) []interface{} {
	var entries = make([]interface{}, 0)
	if !_isL3vpnEnabled(vrf) {
		return entries
	}
	G, _ := infradb.GetVrf(vrf.Name)
	var detail map[string]interface{}
	var Rmac net.HardwareAddr
	for _, com := range G.Status.Components {
		if com.Name == "frr" {
			err := json.Unmarshal([]byte(com.Details), &detail)
			if err != nil {
				log.Println("intel-e2000: Error: ", err)
			}
			rmac, found := detail["rmac"].(string)
			if !found {
				log.Println("intel-e2000: Key 'rmac' not found")
				break
			}
			Rmac, err = net.ParseMAC(rmac)
			if err != nil {
				log.Println("intel-e2000: Error parsing MAC address:", err)
			}
		}
	}
	if reflect.ValueOf(Rmac).IsZero() {
		log.Println("intel-e2000: Rmac not found for Vtep :", vrf.Spec.VtepIP.IP)

		return entries
	}
	entries = append(entries, p4client.TableEntry{
		Tablename: phyInVxlan,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"dst_ip": {vrf.Spec.VtepIP.IP, "exact"},
				"vni":    {*vrf.Spec.Vni, "exact"},
				"da":     {Rmac, "exact"},
			},
			Priority: int32(0),
		},
	})
	return entries
}

// translateAddedLb translates the added lb
func (v VxlanDecoder) translateAddedLb(lb *infradb.LogicalBridge) []interface{} {
	var entries = make([]interface{}, 0)
	if !(_isL2vpnEnabled(lb)) {
		return entries
	}
	entries = append(entries, p4client.TableEntry{
		Tablename: phyInVxlanL2,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"dst_ip": {lb.Spec.VtepIP.IP, "exact"},
				"vni":    {*lb.Spec.Vni, "exact"},
			},
			Priority: int32(0),
		},
		Action: p4client.Action{
			ActionName: "evpn_gw_control.pop_vxlan_set_vlan_id",
			Params:     []interface{}{ModPointer.ignorePtr, uint16(lb.Spec.VlanID), uint32(_toEgressVsi(v._defaultVsi))},
		},
	})
	return entries
}

// translateDeletedLb translates the deleted lb
func (v VxlanDecoder) translateDeletedLb(lb *infradb.LogicalBridge) []interface{} {
	var entries = make([]interface{}, 0)

	if !(_isL2vpnEnabled(lb)) {
		return entries
	}
	entries = append(entries, p4client.TableEntry{
		Tablename: phyInVxlanL2,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"dst_ip": {lb.Spec.VtepIP.IP, "exact"},
				"vni":    {*lb.Spec.Vni, "exact"},
			},
			Priority: int32(0),
		},
	})
	return entries
}

// translateAddedNexthop translates the added nexthop
func (v VxlanDecoder) translateAddedNexthop(nexthop netlink_polling.NexthopStruct) []interface{} {
	var entries = make([]interface{}, 0)

	if nexthop.NhType != netlink_polling.VXLAN {
		return entries
	}
	key := fmt.Sprintf("%d-%s-%s-%d-%v", EntryType.l3NH, nexthop.Key.VrfName, nexthop.Key.Dst, nexthop.Key.Dev, nexthop.Key.Local)
	var modPtr = ptrPool.GetID(key)
	var vport = nexthop.Metadata["egress_vport"].(int)
	var smac, _ = net.ParseMAC(nexthop.Metadata["phy_smac"].(string))
	var dmac, _ = net.ParseMAC(nexthop.Metadata["phy_dmac"].(string))
	var srcAddr = nexthop.Metadata["local_vtep_ip"]
	var dstAddr = nexthop.Metadata["remote_vtep_ip"]
	var vni = nexthop.Metadata["vni"]
	var innerSmacAddr, _ = net.ParseMAC(nexthop.Metadata["inner_smac"].(string))
	var innerDmacAddr, _ = net.ParseMAC(nexthop.Metadata["inner_dmac"].(string))
	entries = append(entries, p4client.TableEntry{
		Tablename: pushVxlanHdr,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"meta.common.mod_blob_ptr": {modPtr, "exact"},
			},
			Priority: int32(0),
		},
		Action: p4client.Action{
			ActionName: "evpn_gw_control.omac_vxlan_imac_push",
			Params:     []interface{}{smac, dmac, net.ParseIP(srcAddr.(string)), net.ParseIP(dstAddr.(string)), v.vxlanUDPPort, vni.(uint32), innerSmacAddr, innerDmacAddr},
		},
	},
		p4client.TableEntry{
			Tablename: l3NhTx,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"neighbor":    {uint16(_p4NexthopID(nexthop, Direction.Tx)), "exact"},
					"bit32_zeros": {uint32(0), "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.push_outermac_vxlan_innermac",
				Params:     []interface{}{modPtr, uint32(vport)},
			},
		},
		p4client.TableEntry{
			Tablename: l3NhRx,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"neighbor":    {uint16(_p4NexthopID(nexthop, Direction.Rx)), "exact"},
					"bit32_zeros": {uint32(0), "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.send_p2p_push_outermac_vxlan_innermac",
				Params:     []interface{}{modPtr, uint32(vport), uint16(_p2pQid(vport))},
			},
		},
		p4client.TableEntry{
			Tablename: p2pIn,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"neighbor":    {uint16(_p4NexthopID(nexthop, Direction.Rx)), "exact"},
					"bit32_zeros": {uint32(0), "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.fwd_to_port",
				Params:     []interface{}{uint32(vport)},
			},
		})
	return entries
}

// translateDeletedNexthop translates the deleted nexthop
func (v VxlanDecoder) translateDeletedNexthop(nexthop netlink_polling.NexthopStruct) []interface{} {
	var entries = make([]interface{}, 0)

	if nexthop.NhType != netlink_polling.VXLAN {
		return entries
	}
	// var key []interface{}
	key := fmt.Sprintf("%d-%s-%s-%d-%v", EntryType.l2Nh, nexthop.Key.VrfName, nexthop.Key.Dst, nexthop.Key.Dev, nexthop.Key.Local)
	var modPtr = ptrPool.ReleaseID(key)
	entries = append(entries, p4client.TableEntry{
		Tablename: pushVxlanHdr,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"meta.common.mod_blob_ptr": {modPtr, "exact"},
			},
			Priority: int32(0),
		},
	},
		p4client.TableEntry{
			Tablename: l3NhTx,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"neighbor":    {uint16(_p4NexthopID(nexthop, Direction.Tx)), "exact"},
					"bit32_zeros": {uint32(0), "exact"},
				},
				Priority: int32(0),
			},
		},
		p4client.TableEntry{
			Tablename: l3NhRx,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"neighbor":    {uint16(_p4NexthopID(nexthop, Direction.Rx)), "exact"},
					"bit32_zeros": {uint32(0), "exact"},
				},
				Priority: int32(0),
			},
		},
		p4client.TableEntry{
			Tablename: p2pIn,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"neighbor":    {uint16(_p4NexthopID(nexthop, Direction.Rx)), "exact"},
					"bit32_zeros": {uint32(0), "exact"},
				},
				Priority: int32(0),
			},
		})
	return entries
}

// translateAddedL2Nexthop translates the added l2 nexthop
func (v VxlanDecoder) translateAddedL2Nexthop(nexthop netlink_polling.L2NexthopStruct) []interface{} {
	var entries = make([]interface{}, 0)

	if nexthop.Type != netlink_polling.VXLAN {
		return entries
	}
	key := fmt.Sprintf("%d-%s-%d-%s", EntryType.l2Nh, nexthop.Key.Dev, nexthop.Key.VlanID, nexthop.Key.Dst)
	var modPtr = ptrPool.GetID(key)
	var vport = nexthop.Metadata["egress_vport"].(int)
	var srcMac, _ = net.ParseMAC(nexthop.Metadata["phy_smac"].(string))
	var dstMac, _ = net.ParseMAC(nexthop.Metadata["phy_dmac"].(string))
	var srcIP = nexthop.Metadata["local_vtep_ip"]
	var dstIP = nexthop.Metadata["remote_vtep_ip"]
	var vni = nexthop.Metadata["vni"]
	var vsiOut = _toEgressVsi(vport)
	var neighbor = nexthop.ID
	entries = append(entries, p4client.TableEntry{
		Tablename: pushVxlanOutHdr,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"meta.common.mod_blob_ptr": {modPtr, "exact"},
			},
			Priority: int32(0),
		},
		Action: p4client.Action{
			ActionName: "evpn_gw_control.omac_vxlan_push",
			Params:     []interface{}{srcMac, dstMac, net.ParseIP(srcIP.(string)), net.ParseIP(dstIP.(string)), v.vxlanUDPPort, vni.(uint32)},
		},
	},
		p4client.TableEntry{
			Tablename: l2Nh,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"neighbor":    {uint16(neighbor), "exact"},
					"bit32_zeros": {uint32(0), "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.push_outermac_vxlan",
				Params:     []interface{}{modPtr, vsiOut},
			},
		})
	return entries
}

// translateDeletedL2Nexthop translates the deleted l2 nexthop
func (v VxlanDecoder) translateDeletedL2Nexthop(nexthop netlink_polling.L2NexthopStruct) []interface{} {
	var entries = make([]interface{}, 0)

	if nexthop.Type != netlink_polling.VXLAN {
		return entries
	}
	key := fmt.Sprintf("%d-%s-%d-%s", EntryType.l2Nh, nexthop.Key.Dev, nexthop.Key.VlanID, nexthop.Key.Dst)
	var modPtr = ptrPool.ReleaseID(key)
	var neighbor = nexthop.ID
	entries = append(entries, p4client.TableEntry{
		Tablename: pushVxlanOutHdr,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"meta.common.mod_blob_ptr": {modPtr, "exact"},
			},
			Priority: int32(0),
		},
	},
		p4client.TableEntry{
			Tablename: l2Nh,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"neighbor":    {uint16(neighbor), "exact"},
					"bit32_zeros": {uint32(0), "exact"},
				},
				Priority: int32(0),
			},
		})
	return entries
}

// translateAddedFdb translates the added fdb entry
func (v VxlanDecoder) translateAddedFdb(fdb netlink_polling.FdbEntryStruct) []interface{} {
	var entries = make([]interface{}, 0)

	if fdb.Type != netlink_polling.VXLAN {
		return entries
	}
	var mac, _ = net.ParseMAC(fdb.Mac)
	var directions = _directionsOf(fdb)

	for _, dir := range directions {
		entries = append(entries, p4client.TableEntry{
			Tablename: l2Fwd,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"vlan_id":   {uint16(fdb.VlanID), "exact"},
					"da":        {mac, "exact"},
					"direction": {uint16(dir), "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.set_neighbor",
				Params:     []interface{}{uint16(fdb.Metadata["nh_id"].(int))},
			},
		})
	}
	return entries
}

// translateDeletedFdb translates the deleted fdb entry
func (v VxlanDecoder) translateDeletedFdb(fdb netlink_polling.FdbEntryStruct) []interface{} {
	var entries = make([]interface{}, 0)

	if fdb.Type != netlink_polling.VXLAN {
		return entries
	}
	var mac, _ = net.ParseMAC(fdb.Mac)
	var directions = _directionsOf(fdb)

	for _, dir := range directions {
		entries = append(entries, p4client.TableEntry{
			Tablename: l2Fwd,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"vlan_id":   {uint16(fdb.VlanID), "exact"},
					"da":        {mac, "exact"},
					"direction": {uint16(dir), "exact"},
				},
				Priority: int32(0),
			},
		})
	}
	return entries
}

// PodDecoder structure for pod decode
type PodDecoder struct {
	portMuxIDs  [2]string
	_portMuxVsi int
	_portMuxMac string
	vrfMuxIDs   [2]string
	_vrfMuxVsi  int
	_vrfMuxMac  string
	floodModPtr uint32
	floodNhID   uint16
}

// PodDecoderInit initializes the pod decoder
func (p PodDecoder) PodDecoderInit(representors map[string][2]string) PodDecoder {
	p.portMuxIDs = representors["port_mux"]
	p.vrfMuxIDs = representors["vrf_mux"]

	portMuxVsi, err := strconv.ParseInt(p.portMuxIDs[0], 10, 32)
	if err != nil {
		panic(err)
	}
	vrfMuxVsi, err := strconv.ParseInt(p.vrfMuxIDs[0], 10, 32)
	if err != nil {
		panic(err)
	}
	p._portMuxVsi = int(portMuxVsi)
	p._portMuxMac = p.portMuxIDs[1]
	p._vrfMuxVsi = int(vrfMuxVsi)
	p._vrfMuxMac = p.vrfMuxIDs[1]
	p.floodModPtr = ModPointer.l2FloodingPtr
	p.floodNhID = uint16(0)
	return p
}

// translateAddedBp translate the added bp
//
//nolint:funlen,gocognit
func (p PodDecoder) translateAddedBp(bp *infradb.BridgePort) ([]interface{}, error) {
	var entries = make([]interface{}, 0)

	var portMuxVsiOut = _toEgressVsi(p._portMuxVsi)
	port, err := strconv.ParseUint(bp.Metadata.VPort, 10, 16)
	if err != nil {
		return entries, err
	}
	key := fmt.Sprintf("%d-%d", EntryType.BP, port)
	key1 := fmt.Sprintf("%d-%v", EntryType.BP, *bp.Spec.MacAddress)
	var vsi = port
	var vsiOut = _toEgressVsi(int(vsi))
	var modPtr = ptrPool.GetID(key)
	var ignorePtr = ModPointer.ignorePtr
	var mac = *bp.Spec.MacAddress
	if p._portMuxVsi < 0 || p._portMuxVsi > math.MaxUint16 {
		return nil, errors.New("_portMuxVsi is not in range of uint16")
	}
	if bp.Spec.Ptype == infradb.Trunk {
		var modPtrD = ptrPool.GetID(key1)
		entries = append(entries, p4client.TableEntry{
			// From MUX
			Tablename: portMuxIn,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"vsi": {uint16(p._portMuxVsi), "exact"},
					"vid": {uint16(vsi), "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.pop_stag_vlan",
				Params:     []interface{}{modPtrD, uint32(vsiOut)},
			},
		},
			// From Rx-to-Tx-recirculate (pass 3) entry
			p4client.TableEntry{
				Tablename: popStag,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"meta.common.mod_blob_ptr": {modPtrD, "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.vlan_stag_pop",
					Params:     []interface{}{mac},
				},
			},
			p4client.TableEntry{
				Tablename: l2FwdLoop,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"da": {mac, "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.l2_fwd",
					Params:     []interface{}{uint32(vsiOut)},
				},
			},
			p4client.TableEntry{
				Tablename: podOutTrunk,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"meta.common.mod_blob_ptr": {modPtr, "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.vlan_push_trunk",
					Params:     []interface{}{uint16(0), uint16(0), uint32(vsi)},
				},
			})
		for _, vlan := range bp.Spec.LogicalBridges {
			BrObj, err := infradb.GetLB(vlan)
			if err != nil {
				log.Printf("intel-e2000: unable to find key %s and error is %v\n", vlan, err)
				return entries, err
			}
			if BrObj.Spec.VlanID > math.MaxUint16 {
				log.Printf("intel-e2000: VlanID %v value passed in Logical Bridge create is greater than 16 bit value\n", BrObj.Spec.VlanID)
				return entries, errors.New("VlanID value passed in Logical Bridge create is greater than 16 bit value")
			}

			vid := uint16(BrObj.Spec.VlanID)
			entries = append(entries, p4client.TableEntry{
				// To MUX PORT
				Tablename: podInArpTrunk,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vsi": {uint16(vsi), "exact"},
						"vid": {vid, "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.send_to_port_mux_trunk",
					Params:     []interface{}{modPtr, uint32(portMuxVsiOut)},
				},
			},
				// To L2 FWD
				p4client.TableEntry{
					Tablename: podInIPTrunk,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"vsi": {uint16(vsi), "exact"},
							"vid": {vid, "exact"},
						},
						Priority: int32(0),
					},
					Action: p4client.Action{
						ActionName: "evpn_gw_control.set_vlan_and_pop_vlan",
						Params:     []interface{}{ignorePtr, vid, uint32(0)},
					},
				})

			if BrObj.Svi != "" {
				SviObj, err := infradb.GetSvi(BrObj.Svi)
				if err != nil {
					log.Printf("intel-e2000: unable to find key %s and error is %v\n", BrObj.Svi, err)
					return entries, err
				}
				VrfObj, err := infradb.GetVrf(SviObj.Spec.Vrf)
				if err != nil {
					log.Printf("intel-e2000: unable to find key %s and error is %v\n", SviObj.Spec.Vrf, err)
					return entries, err
				}
				tcamPrefix, err := _getTcamPrefix(*VrfObj.Metadata.RoutingTable[0], Direction.Tx)
				if err != nil {
					return entries, err
				}
				// To VRF SVI
				var sviMac = *SviObj.Spec.MacAddress
				entries = append(entries, p4client.TableEntry{
					// From MUX
					Tablename: portInSviTrunk,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"vsi": {uint16(vsi), "exact"},
							"vid": {vid, "exact"},
							"da":  {sviMac, "exact"},
						},
						Priority: int32(0),
					},
					Action: p4client.Action{
						ActionName: "evpn_gw_control.pop_vlan_set_vrf_id",
						Params:     []interface{}{ignorePtr, uint32(tcamPrefix), uint32(0), uint16(*VrfObj.Metadata.RoutingTable[0])},
					},
				})
			} else {
				log.Println("intel-e2000: no associated SVI object created")
			}
		}
	} else if bp.Spec.Ptype == infradb.Access {
		BrObj, err := infradb.GetLB(bp.Spec.LogicalBridges[0])
		if err != nil {
			log.Printf("intel-e2000: unable to find key %s and error is %v\n", bp.Spec.LogicalBridges[0], err)
			return entries, err
		}
		if BrObj.Spec.VlanID > math.MaxUint16 {
			log.Printf("intel-e2000: VlanID %v value passed in Logical Bridge create is greater than 16 bit value\n", BrObj.Spec.VlanID)
			return entries, errors.New("VlanID value passed in Logical Bridge create is greater than 16 bit value")
		}
		var vid = uint16(BrObj.Spec.VlanID)
		var modPtrD = ptrPool.GetID(key1)
		var dstMacAddr = *bp.Spec.MacAddress
		entries = append(entries, p4client.TableEntry{
			// From MUX
			Tablename: portMuxIn,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"vsi": {uint16(p._portMuxVsi), "exact"},
					"vid": {uint16(vsi), "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.pop_ctag_stag_vlan",
				Params:     []interface{}{modPtrD, uint32(vsiOut)},
			},
		},
			p4client.TableEntry{
				Tablename: popCtagStag,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"meta.common.mod_blob_ptr": {modPtrD, "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.vlan_ctag_stag_pop",
					Params:     []interface{}{dstMacAddr},
				},
			},
			// From Rx-to-Tx-recirculate (pass 3) entry
			p4client.TableEntry{
				Tablename: l2FwdLoop,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"da": {dstMacAddr, "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.l2_fwd",
					Params:     []interface{}{uint32(vsiOut)},
				},
			},
			// To MUX PORT
			p4client.TableEntry{
				Tablename: podOutAccess,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"meta.common.mod_blob_ptr": {modPtr, "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.vlan_push_access",
					Params:     []interface{}{uint16(0), uint16(0), vid, uint16(0), uint16(0), uint16(vsi)},
				},
			},
			p4client.TableEntry{
				Tablename: podInArpAccess,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vsi":         {uint16(vsi), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.send_to_port_mux_access",
					Params:     []interface{}{modPtr, uint32(portMuxVsiOut)},
				},
			},
			// To L2 FWD
			p4client.TableEntry{
				Tablename: podInIPAccess,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vsi":         {uint16(vsi), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.set_vlan",
					Params:     []interface{}{vid, uint32(0)},
				},
			})
		if BrObj.Svi != "" {
			SviObj, err := infradb.GetSvi(BrObj.Svi)
			if err != nil {
				log.Printf("intel-e2000: unable to find key %s and error is %v\n", BrObj.Svi, err)
				return entries, err
			}
			VrfObj, err := infradb.GetVrf(SviObj.Spec.Vrf)
			if err != nil {
				log.Printf("intel-e2000: unable to find key %s and error is %v\n", SviObj.Spec.Vrf, err)
				return entries, err
			}
			tcamPrefix, err := _getTcamPrefix(*VrfObj.Metadata.RoutingTable[0], Direction.Tx)
			if err != nil {
				return entries, err
			}
			var sviMac = *SviObj.Spec.MacAddress
			entries = append(entries, p4client.TableEntry{
				// From MUX
				Tablename: portInSviAccess,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vsi": {uint16(vsi), "exact"},
						"da":  {sviMac, "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.set_vrf_id_tx",
					Params:     []interface{}{uint32(tcamPrefix), uint32(0), uint16(*VrfObj.Metadata.RoutingTable[0])},
				},
			})
		} else {
			log.Printf("no SVI for VLAN {vid} on BP {vsi}, skipping entry for SVI table")
		}
	}
	return entries, nil
}

// translateDeletedBp translate the deleted bp
//
//nolint:funlen
func (p PodDecoder) translateDeletedBp(bp *infradb.BridgePort) ([]interface{}, error) {
	var entries []interface{}
	port, err := strconv.ParseUint(bp.Metadata.VPort, 10, 16)
	if err != nil {
		return entries, err
	}
	key := fmt.Sprintf("%d-%d", EntryType.BP, port)
	key1 := fmt.Sprintf("%d-%v", EntryType.BP, *bp.Spec.MacAddress)
	var vsi = port
	var modPtr = ptrPool.ReleaseID(key)
	var mac = *bp.Spec.MacAddress
	var modPtrD = ptrPool.ReleaseID(key1)
	if p._portMuxVsi < 0 || p._portMuxVsi > math.MaxUint16 {
		return nil, errors.New("_portMuxVsi is not in range of uint16")
	}
	if bp.Spec.Ptype == infradb.Trunk {
		entries = append(entries, p4client.TableEntry{
			// From MUX
			Tablename: portMuxIn,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"vsi": {uint16(p._portMuxVsi), "exact"},
					"vid": {uint16(vsi), "exact"},
				},
				Priority: int32(0),
			},
		},
			// From Rx-to-Tx-recirculate (pass 3) entry
			p4client.TableEntry{
				Tablename: popStag,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"meta.common.mod_blob_ptr": {modPtrD, "exact"},
					},
					Priority: int32(0),
				},
			},
			p4client.TableEntry{
				Tablename: l2FwdLoop,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"da": {mac, "exact"},
					},
					Priority: int32(0),
				},
			},
			p4client.TableEntry{
				Tablename: podOutTrunk,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"meta.common.mod_blob_ptr": {modPtr, "exact"},
					},
					Priority: int32(0),
				},
			})
		for _, vlan := range bp.Spec.LogicalBridges {
			BrObj, err := infradb.GetLB(vlan)
			if err != nil {
				log.Printf("intel-e2000: unable to find key %s and error is %v\n", vlan, err)
				return entries, err
			}
			if BrObj.Spec.VlanID > math.MaxUint16 {
				log.Printf("intel-e2000: VlanID %v value passed in Logical Bridge create is greater than 16 bit value\n", BrObj.Spec.VlanID)
				return entries, errors.New("VlanID value passed in Logical Bridge create is greater than 16 bit value")
			}
			vid := uint16(BrObj.Spec.VlanID)
			entries = append(entries, p4client.TableEntry{
				// To MUX PORT
				Tablename: podInArpTrunk,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vsi": {uint16(vsi), "exact"},
						"vid": {vid, "exact"},
					},
					Priority: int32(0),
				},
			},
				// To L2 FWD
				p4client.TableEntry{
					Tablename: podInIPTrunk,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"vsi": {uint16(vsi), "exact"},
							"vid": {vid, "exact"},
						},
						Priority: int32(0),
					},
				})

			if BrObj.Svi != "" {
				SviObj, err := infradb.GetSvi(BrObj.Svi)
				if err != nil {
					log.Printf("intel-e2000: unable to find key %s and error is %v\n", BrObj.Svi, err)
					return entries, err
				}
				// To VRF SVI
				var sviMac = *SviObj.Spec.MacAddress
				entries = append(entries, p4client.TableEntry{
					// From MUX
					Tablename: portInSviTrunk,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"vsi": {uint16(vsi), "exact"},
							"vid": {vid, "exact"},
							"da":  {sviMac, "exact"},
						},
						Priority: int32(0),
					},
				})
			} else {
				log.Printf("no SVI for VLAN {vid} on BP {vsi}, skipping entry for SVI table")
			}
		}
	} else if bp.Spec.Ptype == infradb.Access {
		BrObj, err := infradb.GetLB(bp.Spec.LogicalBridges[0])
		if err != nil {
			log.Printf("intel-e2000: unable to find key %s and error is %v\n", bp.Spec.LogicalBridges[0], err)
			return entries, err
		}
		var dstMacAddr = *bp.Spec.MacAddress
		entries = append(entries, p4client.TableEntry{
			// From MUX
			Tablename: portMuxIn,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"vsi": {uint16(p._portMuxVsi), "exact"},
					"vid": {uint16(vsi), "exact"},
				},
				Priority: int32(0),
			},
		},
			p4client.TableEntry{
				Tablename: popCtagStag,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"meta.common.mod_blob_ptr": {modPtrD, "exact"},
					},
					Priority: int32(0),
				},
			},
			// From Rx-to-Tx-recirculate (pass 3) entry
			p4client.TableEntry{
				Tablename: l2FwdLoop,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"da": {dstMacAddr, "exact"},
					},
					Priority: int32(0),
				},
			},
			// To MUX PORT
			p4client.TableEntry{
				Tablename: podOutAccess,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"meta.common.mod_blob_ptr": {modPtr, "exact"},
					},
					Priority: int32(0),
				},
			},
			p4client.TableEntry{
				Tablename: podInArpAccess,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vsi":         {uint16(vsi), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
			},
			// To L2 FWD
			p4client.TableEntry{
				Tablename: podInIPAccess,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vsi":         {uint16(vsi), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
			})
		if BrObj.Svi != "" {
			SviObj, err := infradb.GetSvi(BrObj.Svi)
			if err != nil {
				log.Printf("intel-e2000: unable to find key %s and error is %v\n", BrObj.Svi, err)
				return entries, err
			}
			var sviMac = *SviObj.Spec.MacAddress
			entries = append(entries, p4client.TableEntry{
				// From MUX
				Tablename: portInSviAccess,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"vsi": {uint16(vsi), "exact"},
						"da":  {sviMac, "exact"},
					},
					Priority: int32(0),
				},
			})
		} else {
			log.Printf("no SVI for VLAN {vid} on BP {vsi}, skipping entry for SVI table")
		}
	}
	return entries, nil
}

// translateAddedSvi translate the added svi
func (p PodDecoder) translateAddedSvi(svi *infradb.Svi) ([]interface{}, error) {
	var ignorePtr = int(ModPointer.ignorePtr)
	var mac = *svi.Spec.MacAddress
	var entries = make([]interface{}, 0)

	BrObj, err := infradb.GetLB(svi.Spec.LogicalBridge)
	if err != nil {
		log.Printf("intel-e2000: unable to find key %s and error is %v\n", svi.Spec.LogicalBridge, err)
		return entries, err
	}
	for k, v := range BrObj.BridgePorts {
		if !v {
			PortObj, err := infradb.GetBP(k)
			if err != nil {
				log.Printf("intel-e2000: unable to find key %s and error is %v\n", k, err)
				return entries, err
			}
			port, err := strconv.ParseUint(PortObj.Metadata.VPort, 10, 16)
			if err != nil {
				return entries, err
			}
			VrfObj, err := infradb.GetVrf(svi.Spec.Vrf)
			if err != nil {
				log.Printf("intel-e2000: unable to find key %s and error is %v", svi.Spec.Vrf, err)
				return entries, err
			}
			tcamPrefix, err := _getTcamPrefix(*VrfObj.Metadata.RoutingTable[0], Direction.Tx)
			if err != nil {
				return entries, err
			}
			if PortObj.Spec.Ptype == infradb.Access {
				entries = append(entries, p4client.TableEntry{
					Tablename: portInSviAccess,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"vsi": {uint16(port), "exact"},
							"da":  {mac, "exact"},
						},
						Priority: int32(0),
					},
					Action: p4client.Action{
						ActionName: "evpn_gw_control.set_vrf_id_tx",
						Params:     []interface{}{uint32(tcamPrefix), uint32(0), uint16(*VrfObj.Metadata.RoutingTable[0])},
					},
				})
			} else if PortObj.Spec.Ptype == infradb.Trunk {
				entries = append(entries, p4client.TableEntry{
					Tablename: portInSviTrunk,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"vsi": {uint16(port), "exact"},
							"vid": {uint16(BrObj.Spec.VlanID), "exact"},
							"da":  {mac, "exact"},
						},
						Priority: int32(0),
					},
					Action: p4client.Action{
						ActionName: "evpn_gw_control.pop_vlan_set_vrf_id",
						Params:     []interface{}{ignorePtr, uint32(tcamPrefix), uint32(0), uint16(*VrfObj.Spec.Vni)},
					},
				})
			}
		}
	}
	return entries, nil
}

// translateDeletedSvi translate the deleted svi
func (p PodDecoder) translateDeletedSvi(svi *infradb.Svi) ([]interface{}, error) {
	var mac = *svi.Spec.MacAddress
	var entries = make([]interface{}, 0)

	BrObj, err := infradb.GetLB(svi.Spec.LogicalBridge)
	if err != nil {
		log.Printf("intel-e2000: unable to find key %s and error is %v\n", svi.Spec.LogicalBridge, err)
		return entries, err
	}

	for k, v := range BrObj.BridgePorts {
		if !v {
			PortObj, err := infradb.GetBP(k)
			if err != nil {
				log.Printf("unable to find key %s and error is %v", k, err)
				return entries, err
			}
			port, err := strconv.ParseUint(PortObj.Metadata.VPort, 10, 16)
			if err != nil {
				return entries, err
			}
			if PortObj.Spec.Ptype == infradb.Access {
				entries = append(entries, p4client.TableEntry{
					Tablename: portInSviAccess,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"vsi": {uint16(port), "exact"},
							"da":  {mac, "exact"},
						},
						Priority: int32(0),
					},
				})
			} else if PortObj.Spec.Ptype == infradb.Trunk {
				entries = append(entries, p4client.TableEntry{
					Tablename: portInSviTrunk,
					TableField: p4client.TableField{
						FieldValue: map[string][2]interface{}{
							"vsi": {uint16(port), "exact"},
							"vid": {uint16(BrObj.Spec.VlanID), "exact"},
							"da":  {mac, "exact"},
						},
						Priority: int32(0),
					},
				})
			}
		}
	}
	return entries, nil
}

// translateAddedFdb translate the added fdb entry
func (p PodDecoder) translateAddedFdb(fdb netlink_polling.FdbEntryStruct) []interface{} {
	var entries = make([]interface{}, 0)

	var fdbMac, _ = net.ParseMAC(fdb.Mac)
	if fdb.Type != netlink_polling.BRIDGEPORT {
		return entries
	}
	for dir := range _directionsOf(fdb) {
		entries = append(entries, p4client.TableEntry{
			Tablename: l2Fwd,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"vlan_id":   {uint16(fdb.VlanID), "exact"},
					"da":        {fdbMac, "exact"},
					"direction": {uint16(dir), "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.set_neighbor",
				Params:     []interface{}{uint16(fdb.Nexthop.ID)},
			},
		})
	}
	return entries
}

// translateDeletedFdb translate the deleted fdb entry
func (p PodDecoder) translateDeletedFdb(fdb netlink_polling.FdbEntryStruct) []interface{} {
	var entries = make([]interface{}, 0)

	var fdbMac, _ = net.ParseMAC(fdb.Mac)
	if fdb.Type != netlink_polling.BRIDGEPORT {
		return entries
	}
	for dir := range _directionsOf(fdb) {
		entries = append(entries, p4client.TableEntry{
			Tablename: l2Fwd,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"vlan_id":   {uint16(fdb.VlanID), "exact"},
					"da":        {fdbMac, "exact"},
					"direction": {uint16(dir), "exact"},
				},
				Priority: int32(0),
			},
		})
	}
	return entries
}

// translateAddedL2Nexthop translate the added l2 nexthop entry
func (p PodDecoder) translateAddedL2Nexthop(nexthop netlink_polling.L2NexthopStruct) []interface{} {
	var entries = make([]interface{}, 0)

	if nexthop.Type != netlink_polling.BRIDGEPORT {
		return entries
	}
	var neighbor = nexthop.ID
	var portType = nexthop.Metadata["portType"].(infradb.BridgePortType)
	var portID, err = strconv.Atoi(nexthop.Metadata["vport_id"].(string))
	if err != nil {
		panic(err)
	}
	if portType == infradb.Access {
		entries = append(entries, p4client.TableEntry{
			Tablename: l2Nh,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"neighbor":    {uint16(neighbor), "exact"},
					"bit32_zeros": {uint32(0), "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.fwd_to_port",
				Params:     []interface{}{uint32(_toEgressVsi(portID))},
			},
		})
	} else if portType == infradb.Trunk {
		key := fmt.Sprintf("%d-%s-%d-%s", EntryType.l2Nh, nexthop.Key.Dev, nexthop.Key.VlanID, nexthop.Key.Dst)
		var modPtr = ptrPool.GetID(key)
		entries = append(entries, p4client.TableEntry{
			Tablename: pushVlan,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"meta.common.mod_blob_ptr": {modPtr, "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.vlan_push",
				Params:     []interface{}{uint16(0), uint16(0), uint16(nexthop.VlanID)},
			},
		},
			p4client.TableEntry{
				Tablename: l2Nh,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(neighbor), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
				Action: p4client.Action{
					ActionName: "evpn_gw_control.push_vlan_l2",
					Params:     []interface{}{modPtr, uint32(_toEgressVsi(portID))},
				},
			})
	}
	return entries
}

// translateDeletedL2Nexthop translate the deleted l2 nexthop entry
func (p PodDecoder) translateDeletedL2Nexthop(nexthop netlink_polling.L2NexthopStruct) []interface{} {
	var entries = make([]interface{}, 0)

	var modPtr uint32
	if nexthop.Type != netlink_polling.BRIDGEPORT {
		return entries
	}
	var neighbor = nexthop.ID
	var portType = nexthop.Metadata["portType"].(infradb.BridgePortType)

	if portType == infradb.Access {
		entries = append(entries, p4client.TableEntry{
			Tablename: l2Nh,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"neighbor":    {uint16(neighbor), "exact"},
					"bit32_zeros": {uint32(0), "exact"},
				},
				Priority: int32(0),
			},
		})
	} else if portType == infradb.Trunk {
		key := fmt.Sprintf("%d-%s-%d-%s", EntryType.l2Nh, nexthop.Key.Dev, nexthop.Key.VlanID, nexthop.Key.Dst)
		modPtr = ptrPool.ReleaseID(key)
		entries = append(entries, p4client.TableEntry{
			Tablename: pushVlan,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"meta.common.mod_blob_ptr": {modPtr, "exact"},
				},
				Priority: int32(0),
			},
		},
			p4client.TableEntry{
				Tablename: l2Nh,
				TableField: p4client.TableField{
					FieldValue: map[string][2]interface{}{
						"neighbor":    {uint16(neighbor), "exact"},
						"bit32_zeros": {uint32(0), "exact"},
					},
					Priority: int32(0),
				},
			})
	}
	return entries
}

// StaticAdditions static additions
func (p PodDecoder) StaticAdditions() []interface{} {
	var portMuxDa, _ = net.ParseMAC(p._portMuxMac)
	var vrfMuxDa, _ = net.ParseMAC(p._vrfMuxMac)
	var entries = make([]interface{}, 0)

	entries = append(entries, p4client.TableEntry{
		Tablename: portMuxFwd,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"bit32_zeros": {uint32(0), "exact"},
			},
			Priority: int32(0),
		},
		Action: p4client.Action{
			ActionName: "evpn_gw_control.send_to_port_mux",
			Params:     []interface{}{uint32(_toEgressVsi(p._portMuxVsi))},
		},
	},
		p4client.TableEntry{
			Tablename: l2FwdLoop,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"da": {portMuxDa, "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.l2_fwd",
				Params:     []interface{}{uint32(_toEgressVsi(p._portMuxVsi))},
			},
		},
		p4client.TableEntry{
			Tablename: l2FwdLoop,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"da": {vrfMuxDa, "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.l2_fwd",
				Params:     []interface{}{uint32(_toEgressVsi(p._vrfMuxVsi))},
			},
		},
		// NH entry for flooding
		p4client.TableEntry{
			Tablename: pushQnQFlood,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"meta.common.mod_blob_ptr": {p.floodModPtr, "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.vlan_push_stag_ctag_flood",
				Params:     []interface{}{uint32(0)},
			},
		},
		p4client.TableEntry{
			Tablename: l2Nh,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"neighbor":    {p.floodNhID, "exact"},
					"bit32_zeros": {uint32(0), "exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				ActionName: "evpn_gw_control.push_stag_ctag",
				Params:     []interface{}{p.floodModPtr, uint32(_toEgressVsi(p._vrfMuxVsi))},
			},
		})
	return entries
}

// StaticDeletions static deletions
func (p PodDecoder) StaticDeletions() []interface{} {
	var entries = make([]interface{}, 0)

	var portMuxDa, _ = net.ParseMAC(p._portMuxMac)
	var vrfMuxDa, _ = net.ParseMAC(p._vrfMuxMac)
	entries = append(entries, p4client.TableEntry{
		Tablename: portMuxFwd,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"bit32_zeros": {uint32(0), "exact"},
			},
			Priority: int32(0),
		},
	},
		p4client.TableEntry{
			Tablename: l2FwdLoop,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"da": {portMuxDa, "exact"},
				},
				Priority: int32(0),
			},
		},
		p4client.TableEntry{
			Tablename: l2FwdLoop,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"da": {vrfMuxDa, "exact"},
				},
				Priority: int32(0),
			},
		},
		// NH entry for flooding
		p4client.TableEntry{
			Tablename: pushQnQFlood,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"meta.common.mod_blob_ptr": {p.floodModPtr, "exact"},
				},
				Priority: int32(0),
			},
		},
		p4client.TableEntry{
			Tablename: l2Nh,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"neighbor":    {p.floodNhID, "exact"},
					"bit32_zeros": {uint32(0), "exact"},
				},
				Priority: int32(0),
			},
		})
	return entries
}
