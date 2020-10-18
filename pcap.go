package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
)

type TaggedPacket struct {
	CapturedAt   time.Time
	Src          net.IP
	Dst          net.IP
	ComPartnerIP net.IP
	Error        error
	IsIPv4       bool
	IsIPv6       bool
	IsTCP        bool
	IsUDP        bool
	// "Truncated" means the packet is busted
	IsTruncated          bool
	IsLoopbackOnly       bool
	IsLoopbackAndOutside bool
	SrcIsUs              bool
	DstIsUs              bool
	SrcIsInOurNetwork    bool
	DstIsInOurNetwork    bool
}

func StartPcap(
	ctx context.Context,
	taggedChan chan TaggedPacket,
	ownNets []IPNetParticipation,
) {
	defer close(taggedChan)

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
		taggedPacket := TaggedPacket{
			CapturedAt: time.Now(),
		}
		if err != nil {
			taggedPacket.Error = fmt.Errorf("ReadPacketData error: %v", err)
			taggedChan <- taggedPacket
			continue
		}
		parser.DecodeLayers(data, &decodedLayers)
		for _, layerType := range decodedLayers {
			switch layerType {
			case layers.LayerTypeIPv4:
				taggedPacket.IsIPv4 = true
			case layers.LayerTypeIPv6:
				taggedPacket.IsIPv6 = true
			case layers.LayerTypeTCP:
				taggedPacket.IsTCP = true
			case layers.LayerTypeUDP:
				taggedPacket.IsUDP = true
			}
		}
		// "truncated" means the packet is busted
		taggedPacket.IsTruncated = parser.Truncated
		var src, dst net.IP
		if taggedPacket.IsIPv4 {
			src = ip4.SrcIP
			dst = ip4.DstIP
		} else {
			src = ip6.SrcIP
			dst = ip6.DstIP
		}
		taggedPacket.Src = src
		taggedPacket.Dst = dst
		taggedPacket.IsLoopbackOnly = src.IsLoopback() && dst.IsLoopback()
		taggedPacket.IsLoopbackAndOutside = src.IsLoopback() != dst.IsLoopback()
		// get the com partner or panic if the conversation is unrelated to us
		for _, netParticipation := range ownNets {
			if netParticipation.Address.Equal(src) {
				taggedPacket.SrcIsUs = true
				taggedPacket.SrcIsInOurNetwork = true
			} else if netParticipation.Network.Contains(src) {
				taggedPacket.SrcIsInOurNetwork = true
			}
			if netParticipation.Address.Equal(dst) {
				taggedPacket.DstIsUs = true
				taggedPacket.DstIsInOurNetwork = true
			} else if netParticipation.Network.Contains(dst) {
				taggedPacket.DstIsInOurNetwork = true
			}
		}
		if taggedPacket.SrcIsUs {
			taggedPacket.ComPartnerIP = dst
		} else if taggedPacket.DstIsUs {
			taggedPacket.ComPartnerIP = src
		}
		taggedChan <- taggedPacket
	}
}

func DebugPrintPackets(
	ctx context.Context,
	taggedChan chan TaggedPacket,
) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		pack := <-taggedChan
		fmt.Println(pack.CapturedAt.Format(time.Kitchen), pack.Src, pack.Dst, pack.IsIPv4, pack.IsTCP, pack.IsUDP, pack.IsLoopbackOnly)
	}
}
