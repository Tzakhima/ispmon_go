package main

import (
	"crypto/sha1"
	"fmt"
)

const (
	ParametersUrl string = "http://ispmon.cloud/config"
	IpInfoUrl     string = "http://ipinfo.io/"
	//PushInfoUrl   string = "http://ispmon.cloud/gometrics"
)

func main() {

	// calculate user unique id based on MAC addresses. variable: uid
    mac, err := getMacAddr()

    if err != nil {
    	fmt.Printf("Error getting Mac address: %s", err )
	}

	var macString string
    for _, addr := range mac{
    	macString = macString + addr
	}

	macSha1  := sha1.Sum([]byte(macString))
    uid := fmt.Sprintf("%x\n", string(macSha1[:]))[0:10]
    fmt.Println(uid)


	// get isp info from ipinfo.io
	IpInfo, err := getIspInfo()
	if err !=nil {
		fmt.Printf("Error getting client IP info: %s", err )
	}
	fmt.Printf("%+v\n", IpInfo)


	// getting targets and interval parameters
	pingLinks, httpLinks, interval, err := getParameters()

	// ping the target

	// http trace
	schema := "https://"

	fmt.Println(pingLinks, httpLinks, interval)

	c := make(chan map[string]string)

	for _, link := range httpLinks {
		go httpStat(schema+link, c)
	}

	for i := 0; i < len(pingLinks); i++ {
		fmt.Println(<-c)
	}

	// speed test

	// push results

	//Get PING, HTTP and speed test parameters


}
