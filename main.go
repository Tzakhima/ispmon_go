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


	// get client and ISP info from ipinfo.io
	IpInfo, err := getIspInfo()
	if err !=nil {
		fmt.Printf("Error getting client IP info: %s", err )
	}
	fmt.Printf("%+v\n", IpInfo)


	// getting targets and interval parameters
	pingLinks, httpLinks, interval, err := getParameters()
	fmt.Println(pingLinks, httpLinks, interval)


	// http trace
	var httpResults []map[string]map[string]int64
	c := make(chan map[string]map[string]int64)

	for _, link := range httpLinks {
		go getHttpStat(link, c)
	}

	for i := 0; i < len(httpLinks); i++ {
		httpResults = append(httpResults, <-c)
	}

	fmt.Printf("%+v", httpResults)

	// ping test

	// speed test

	// push results

	//Get PING, HTTP and speed test parameters


}
