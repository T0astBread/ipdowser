package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
)

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

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

	// ring, err := pfring.NewRing("enp0s31f6", 65536, pfring.FlagPromisc)
	// if err != nil {
	// 	panic(err)
	// }
	// defer ring.Close()
	// ring.Enable()
	// var source gopacket.PacketDataSource = ring

	parser := gopacket.NewDecodingLayerParser(
		layers.LayerTypeEthernet,
		&eth, &ip4, &ip6, &tcp, &udp)
	decodedLayers := make([]gopacket.LayerType, 0, 10)
	var sb strings.Builder
	var isLocal, isIP = false, false
	var ipPackCounter, packCounter = 0, 0
	for {
		var done = false
		select {
		case <-sigChan:
			done = true
		default:
		}
		if done {
			break
		}

		data, _, err := source.ReadPacketData()
		isLocal, isIP = false, false
		sb.Reset()
		sb.WriteString("P----------------------\n")
		if err != nil {
			fmt.Println("packet error ", err)
			continue
		}
		parser.DecodeLayers(data, &decodedLayers)
		for _, layerType := range decodedLayers {
			switch layerType {
			case layers.LayerTypeEthernet:
				sb.WriteString(fmt.Sprintln("  eth", eth.SrcMAC, eth.DstMAC))
			case layers.LayerTypeIPv4:
				isIP = true
				if ip4.SrcIP.IsLoopback() || ip4.DstIP.IsLoopback() {
					isLocal = true
					break
				}
				sb.WriteString(fmt.Sprintln("  ip4", ip4.SrcIP.String(), ip4.DstIP.String()))
			case layers.LayerTypeIPv6:
				isIP = true
				if ip6.SrcIP.IsLinkLocalUnicast() || ip6.SrcIP.IsLinkLocalMulticast() || ip6.DstIP.IsLinkLocalUnicast() || ip6.DstIP.IsLinkLocalMulticast() {
					isLocal = true
					break
				}
				sb.WriteString(fmt.Sprintln("  ip6", ip6.SrcIP.String(), ip6.DstIP.String()))
			case layers.LayerTypeTCP:
				sb.WriteString(fmt.Sprintln("  tcp", tcp.SrcPort, tcp.DstPort))
			case layers.LayerTypeUDP:
				sb.WriteString(fmt.Sprintln("  udp", udp.SrcPort, udp.DstPort))
			}
		}
		if !isLocal && isIP {
			print(sb.String())
			ipPackCounter++
		}
		packCounter++
		if parser.Truncated {
			println("packet truncated")
		}
	}

	println()
	fmt.Println(ipPackCounter, "counted non-local IP packets")
	fmt.Println(packCounter, "counted packets")

	_, stats, err := tpacket.SocketStats()
	if err != nil {
		panic(err)
	}
	println("stats from socket")
	println("  drops:", stats.Drops())
	println("  packets:", stats.Packets())
	println("  queue freezes:", stats.QueueFreezes())

	// ringStats, err := ring.Stats()
	// if err != nil {
	// 	panic(err)
	// }
	// println("stats from ring")
	// println("  dropped:", ringStats.Dropped)
	// println("  recieved:", ringStats.Received)
}
