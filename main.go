package main

import (
	"context"
	"os"
	"os/signal"
)

func main() {
	ctx := newInterruptContext()
	ownNets, err := GetOwnNetworks("enp0s31f6")
	if err != nil {
		panic(err)
	}

	torDirectory, err := GetFreshTorGuards(ctx)
	if err != nil {
		panic(err)
	}
	torDirectoryUpdateChan := make(chan *TorRelayDirectory)
	eventChan := make(chan Event)
	ipReputationChan := make(chan *IPReputation)

	taggedChan := make(chan TaggedPacket)
	go StartPcap(ctx, taggedChan, ownNets)
	// DebugPrintPackets(ctx, taggedChan)

	go AnalyzePackets(ctx, taggedChan,
		torDirectory, torDirectoryUpdateChan,
		eventChan, ipReputationChan)

	DebugPrintEvents(ctx, eventChan, ipReputationChan)
}

func newInterruptContext() context.Context {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for range sigChan {
			cancel()
			return
		}
	}()
	return ctx
}
