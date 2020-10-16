package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type TorRelayDirectory struct {
	RelaysPublishedStr string `json:"relays_published"`
	RelaysPublished    time.Time
	Relays             []TorRelay
}

func (self TorRelayDirectory) IsFresh() bool {
	return self.RelaysPublished.Add(30 * time.Minute).After(time.Now())
}

func (self TorRelayDirectory) AnyRelayHasIP(ip net.IP) bool {
	ipStr := ip.String()
	for _, relay := range self.Relays {
		for _, relayIP := range relay.OrAddresses {
			// if relayIP == ipStr {
			if strings.Contains(relayIP, ipStr) { // TODO: This sucks
				return true
			}
		}
	}
	return false
}

type TorRelay struct {
	Nickname    string
	OrAddresses []string `json:"or_addresses"`
}

func GetFreshTorGuards(ctx context.Context) (*TorRelayDirectory, error) {
	fmt.Println("Checking Tor guards directory...")
	currentDirectory, err := loadGuardsJSON()
	if err != nil {
		return nil, err
	}

	// return currentDirectory, nil

	if currentDirectory.IsFresh() {
		fmt.Println("Guards directory is fresh")
		return currentDirectory, nil
	}
	fmt.Println("Guards directory is not fresh - reloading...")
	newDirectoryData, err := fetchGuardsJSON(ctx)
	if err != nil {
		return nil, err
	}
	return parseGuardsJSON(newDirectoryData)
}

func loadGuardsJSON() (*TorRelayDirectory, error) {
	data, err := ioutil.ReadFile("guards.json")
	if err != nil {
		return nil, err
	}
	return parseGuardsJSON(data)
}

func parseGuardsJSON(data []byte) (*TorRelayDirectory, error) {
	var directory TorRelayDirectory
	if err := json.Unmarshal(data, &directory); err != nil {
		return nil, err
	}
	// I hate this.
	// Why can't I set the time format and location globally for the unmarshaller?
	// And why do I have to use these weird random numbers to specify the format?
	publishTime, err := time.ParseInLocation("2006-01-02 15:04:05", directory.RelaysPublishedStr, time.Now().Location())
	if err != nil {
		return nil, err
	}
	directory.RelaysPublished = publishTime
	return &directory, nil
}

func fetchGuardsJSON(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://onionoo.torproject.org/details?search=flag:Guard", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ioutil.WriteFile("guards.json", data, os.ModePerm)
	return data, nil
}
