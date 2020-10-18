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
	// torDirectory, err := GetFreshTorGuards(ctx)
	// if err != nil {
	// 	panic(err)
	// }
	taggedChan := make(chan TaggedPacket)
	go StartPcap(ctx, taggedChan, ownNets)
	DebugPrintPackets(ctx, taggedChan)
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
