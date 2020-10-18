package main

import (
	"context"
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
)

func StartPcap(
	ctx context.Context,
	ownNets []IPNetParticipation,
	torDirectory *TorRelayDirectory,
	ipReputationMap map[string]int,
) {
	var eth layers.Ethernet
	var ip4 layers.IPv4
	var ip6 layers.IPv6
	var tcp layers.TCP
	var udp layers.UDP

	tpacket, err := afpacket.NewTPacket()
	if err != nil {
		panic(err)
	}
	defer tpacket.Close()
	var source gopacket.PacketDataSource = tpacket

	parser := gopacket.NewDecodingLayerParser(
		layers.LayerTypeEthernet,
		&eth, &ip4, &ip6, &tcp, &udp)
	decodedLayers := make([]gopacket.LayerType, 0, 10)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		data, _, err := source.ReadPacketData()
		if err != nil {
			fmt.Println("packet error ", err)
			continue
		}
		isIPv4, isIPv6 := false, false
		parser.DecodeLayers(data, &decodedLayers)
		for _, layerType := range decodedLayers {
			switch layerType {
			case layers.LayerTypeIPv4:
				isIPv4 = true
			case layers.LayerTypeIPv6:
				isIPv6 = true
			case layers.LayerTypeTCP:
			case layers.LayerTypeUDP:
			}
		}
		// "truncated" means the packet is busted
		if parser.Truncated {
			panic(fmt.Errorf("Packet truncated"))
		}
		// is it not IP?
		if !(isIPv4 || isIPv6) {
			continue
		}
		// is it both IPv4 and IPv6?
		if isIPv4 && isIPv6 {
			panic(fmt.Errorf("Packet is both IPv4 and IPv6"))
		}
		var src, dst net.IP
		if isIPv4 {
			src = ip4.SrcIP
			dst = ip4.DstIP
		} else {
			src = ip6.SrcIP
			dst = ip6.DstIP
		}
		// is it only loopback?
		if src.IsLoopback() && dst.IsLoopback() {
			continue
		}
		// is ONLY ONE loopback?
		if src.IsLoopback() != dst.IsLoopback() {
			panic(fmt.Errorf("Only one loopback; Src:", src.String(), "Dst:", dst.String()))
		}
		// get the com partner or panic if the conversation is unrelated to us
		wereSrc, wereDst := false, false
		for _, netParticipation := range ownNets {
			if netParticipation.Address.Equal(src) {
				wereSrc = true
			}
			if netParticipation.Address.Equal(dst) {
				wereDst = true
			}
		}
		var comParterIP net.IP
		if wereSrc {
			comParterIP = dst
		} else if wereDst {
			comParterIP = src
		} else {
			panic(fmt.Errorf("We're neither src nor dst; Src:", src.String(), "Dst:", dst.String()))
		}
		comParterIPStr := comParterIP.String()
		score, isKnown := ipReputationMap[comParterIPStr]
		// if known
		if isKnown {
			if score > 0 {
				// ...and good log
				ipReputationMap[comParterIPStr] = score + 1
				if score%100 == 0 {
					fmt.Println("Good connection to", comParterIPStr, "continues; Captured packets:", score)
				}
				continue
			} else {
				// ...and bad log as well
				ipReputationMap[comParterIPStr] = score - 1
				if score%10 == 0 {
					fmt.Println("Bad connection to", comParterIPStr, "continues; Captured packets:", -score)
				}
				continue
			}
			// else...
		} else if torDirectory.AnyRelayHasIP(comParterIP) {
			// if it's a Tor guard mark it good
			fmt.Println("New good connection to", comParterIPStr)
			ipReputationMap[comParterIPStr] = 1
			continue
		} else {
			// if it's not, mark it bad
			ipReputationMap[comParterIPStr] = -1
			fmt.Println("New bad connection to", comParterIPStr)
			continue
		}
	}

	// _, stats, err := tpacket.SocketStats()
	// if err != nil {
	// 	panic(err)
	// }
	// println("stats from socket")
	// println("  drops:", stats.Drops())
	// println("  packets:", stats.Packets())
	// println("  queue freezes:", stats.QueueFreezes())
}
