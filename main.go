package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"regexp"
)

func main() {
	ctx := newInterruptContext()
	ownIPAddrs, err := getOwnIPAddrs()
	if err != nil {
		panic(err)
	}
	torDirectory, err := GetFreshTorGuards(ctx)
	if err != nil {
		panic(err)
	}
	StartPcap(ctx, ownIPAddrs, torDirectory, make(map[string]int))
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

// func getOwnIP() string {
// 	resp, err := http.Get("https://checkip.amazonaws.com")
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer resp.Body.Close()
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return strings.Trim(string(body), "\n \t")
// }

func getOwnIPAddrs() ([]string, error) {
	iface, err := net.InterfaceByName("enp0s31f6")
	if err != nil {
		return nil, err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	if len(addrs) < 1 {
		return nil, fmt.Errorf("Interface has no addresses")
	}
	convertedAddrs := make([]string, len(addrs))
	subnetLengthRgx := regexp.MustCompile("/\\d{1,3}$")
	for i, addr := range addrs {
		addrStr := addr.String()
		convertedAddr := subnetLengthRgx.ReplaceAllLiteralString(addrStr, "")
		convertedAddrs[i] = convertedAddr
		fmt.Println("Own IP:", convertedAddr)
	}
	return convertedAddrs, nil
}
