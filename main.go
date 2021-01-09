// ISPMON Client
// Gets list of targets from http://ispmon.cloud/config and executing the following:
// HTTP Trace - measure the DNS time, TLS Handshake time, Connection time and Time To First Byte.
// PING       - running 60 PING against the list of targets. reports back the avg RTT and Packet Loss ratio.
// SpeedTest  - Although the results are not always accurate, running SpeedTest every <interval> and reports back
//              Download and Upload speed.
//
//  When starting, you should see your UID printed to STDOUT.
//  To see your results please navigate to https://ispmon.cloud Grafana dashboard. (look for your UID)

package main

import (
	"crypto/sha1"
	"fmt"
	"sync"
)

const (
	ParametersUrl string = "http://ispmon.cloud/config"
	IpInfoUrl     string = "http://ipinfo.io/"
	//PushInfoUrl   string = "http://ispmon.cloud/gometrics"
)

type pushResults struct {
	http		[]map[string]map[string]int64
	ping		[]map[string]map[string]float64
	speed       float64
	uid			string
	isp 		string
	country		string
	city		string
}

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


	// get client and ISP info from ipinfo.io
	IpInfo, err := getIspInfo()
	if err !=nil {
		fmt.Printf("Error getting client IP info: %s", err )
	}


	// getting targets and interval parameters
	pingLinks, httpLinks, interval, err := getParameters()
	fmt.Print(interval)


	// http trace
	var httpResults []map[string]map[string]int64
	c := make(chan map[string]map[string]int64) // Using channel and not WaitingGroup just for fun :-)

	for _, link := range httpLinks {
		go getHttpStat(link, c)
	}

	for i := 0; i < len(httpLinks); i++ {
		httpResults = append(httpResults, <-c)
	}


	// PING test
	var wg sync.WaitGroup
    var pingResults []map[string]map[string]float64

	for _, t := range pingLinks {
		wg.Add(1)
		go func (t string){
			result := getPingStat(t, &wg)
			pingResults = append(pingResults, result)
		}(t)
	}

	wg.Wait()


	// speed test
    var downloadSpeed float64
	downloadSpeed = getDownloadSpeed()


	// push results
	push := pushResults{}
	push.speed   = downloadSpeed
	push.http    = httpResults
	push.ping    = pingResults
	push.isp     = IpInfo["isp"]
	push.country = IpInfo["country"]
	push.city    = IpInfo["city"]
	push.uid     = uid

	fmt.Printf("%+v\n", push)


}
