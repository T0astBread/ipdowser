package main

import (
	"context"
	"fmt"
	"net"
)

type EventLevel int

const (
	EventLevelCritical EventLevel = iota
	EventLevelWarning
	EventLevelInfo
)

type Event struct {
	Level            EventLevel
	Message          string
	AssociatedPacket TaggedPacket
}

type NotificationLevel int

const (
	NotificationLevelCritical NotificationLevel = iota
	NotificationLevelNormal
	NotificationLevelLow
)

type Notification struct {
	Level   EventLevel
	Message string
}

type IPReputation struct {
	IPAddress           net.IP
	IsTorGuard          bool
	IsGood              bool
	CapturedPacketCount int
}

type IPReputationMap map[string]*IPReputation

func AnalyzePackets(
	ctx context.Context,
	taggedChan chan TaggedPacket,
	initialTorDirectory *TorRelayDirectory,
	torDirectoryUpdateChan chan *TorRelayDirectory,
	eventChan chan Event,
	ipReputationChan chan *IPReputation,
) {
	defer close(eventChan)
	defer close(ipReputationChan)

	torDirectory := initialTorDirectory
	ipReputationMap := IPReputationMap{}
	eventChan <- Event{
		Level:   EventLevelInfo,
		Message: "Starting analysis",
	}

	// var taggedPacket TaggedPacket
	for {
		select {
		case <-ctx.Done():
			eventChan <- Event{
				Level:   EventLevelCritical,
				Message: "Stopping analysis - context cancelled",
			}
			return
		case torDirectory = <-torDirectoryUpdateChan:
			break
		// case taggedPacket, packetChanClosed := <-taggedChan:
		case taggedPacket := <-taggedChan:
			// case taggedChan <- taggedPacket:
			// fmt.Println("kk", taggedPacket)
			analyzePacket(taggedPacket, ipReputationMap, torDirectory, eventChan, ipReputationChan)
			// if packetChanClosed {
			// 	eventChan <- Event{
			// 		Level:   EventLevelCritical,
			// 		Message: "Stopping analysis - packet channel closed",
			// 	}
			// 	return
			// }
			break
		}
	}
}

func analyzePacket(
	pck TaggedPacket,
	ipReputationMap IPReputationMap,
	torDirectory *TorRelayDirectory,
	eventChan chan Event,
	ipReputationChan chan *IPReputation,
) {
	if pck.Error != nil {
		eventChan <- Event{
			Level:            EventLevelCritical,
			AssociatedPacket: pck,
			Message:          "Captured error packet",
		}
	}
	// ignore non-IP
	if !pck.IsIPv4 && !pck.IsIPv6 {
		return
	}
	// ignore loopback-only
	if pck.IsLoopbackOnly {
		return
	}
	// we snooped something that potentially doesn't belong to us
	if !pck.SrcIsUs && !pck.DstIsUs {
		if !pck.SrcIsInOurNetwork && !pck.DstIsInOurNetwork {
			eventChan <- Event{
				Level:            EventLevelCritical,
				AssociatedPacket: pck,
				Message:          "Neither src nor dst of captured packet are in our network",
			}
			return
		} else if pck.SrcIsMulticast || pck.DstIsMulticast {
			eventChan <- Event{
				Level:            EventLevelInfo,
				AssociatedPacket: pck,
				Message:          "Local network multicast packet was captured",
			}
		} else {
			eventChan <- Event{
				Level:            EventLevelWarning,
				AssociatedPacket: pck,
				Message:          "Non-multicast packet that does not involve one of our addresses was captured",
			}
			return
		}
	}
	if pck.ComPartnerIP == nil {
		eventChan <- Event{
			Level:            EventLevelCritical,
			AssociatedPacket: pck,
			Message:          "No com partner IP associated with tagged packet",
		}
	} else {
		ipStr := pck.ComPartnerIP.String()
		rep, isKnown := ipReputationMap[ipStr]
		if isKnown {
			rep.CapturedPacketCount++
			ipReputationChan <- rep
			if rep.IsGood {
				if rep.CapturedPacketCount%1000 == 0 {
					eventChan <- Event{
						Level:   EventLevelInfo,
						Message: fmt.Sprint("Captured ", rep.CapturedPacketCount, " good packets from/to ", rep.IPAddress),
					}
				}
			} else {
				if rep.CapturedPacketCount%10 == 0 {
					eventChan <- Event{
						Level:   EventLevelWarning,
						Message: fmt.Sprint("Captured ", rep.CapturedPacketCount, " BAD packets from/to ", rep.IPAddress),
					}
				}
			}
		} else {
			isTorGuard := torDirectory.AnyRelayHasIP(pck.ComPartnerIP)
			newRep := IPReputation{
				IPAddress:           pck.ComPartnerIP,
				IsGood:              isTorGuard,
				IsTorGuard:          isTorGuard,
				CapturedPacketCount: 1,
			}
			ipReputationMap[ipStr] = &newRep
			ipReputationChan <- &newRep
			if newRep.IsGood {
				eventChan <- Event{
					Level:            EventLevelInfo,
					AssociatedPacket: pck,
					Message:          fmt.Sprint("New good connection to ", pck.ComPartnerIP),
				}
			} else {
				eventChan <- Event{
					Level:            EventLevelCritical,
					AssociatedPacket: pck,
					Message:          fmt.Sprint("New BAD connection to ", pck.ComPartnerIP),
				}
			}
		}
	}
}

func DebugPrintEvents(
	ctx context.Context,
	eventChan chan Event,
	ipReputationChan chan *IPReputation,
) {
	for {
		select {
		case <-ctx.Done():
			return
		// case _ = <-eventChan:
		case event := <-eventChan:
			if event.Level == EventLevelCritical {
				print("CRIT")
			} else if event.Level == EventLevelWarning {
				print("WRN ")
			} else {
				print("I   ")
			}
			fmt.Println("  ", event.Message)
		// case ipRep := <-ipReputationChan:
		// 	fmt.Println(ipRep)
		case _ = <-ipReputationChan:
		}
	}
}
