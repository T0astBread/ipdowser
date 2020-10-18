package main

import (
	"fmt"
	"net"
)

type IPNetParticipation struct {
	Address net.IP
	Network *net.IPNet
}

func GetOwnNetworks(ifaceName string) ([]IPNetParticipation, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, err
	}
	netAddrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	if len(netAddrs) < 1 {
		return nil, fmt.Errorf("Interface has no addresses")
	}
	participations := make([]IPNetParticipation, len(netAddrs))
	for i, addr := range netAddrs {
		ipAddr, ipNet, err := net.ParseCIDR(addr.String())
		if err != nil {
			return nil, err
		}
		participations[i] = IPNetParticipation{ipAddr, ipNet}
	}
	return participations, nil
}
